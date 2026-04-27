package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	AnnouncementEmailSourceNotice       = "notice"
	AnnouncementEmailSourceAnnouncement = "announcement"
	AnnouncementEmailTargetAgent        = "agent"
	AnnouncementEmailTargetEndUser      = "end_user"
	AnnouncementEmailTargetAll          = "all"
)

type AnnouncementEmailBroadcastRequest struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type AnnouncementEmailBroadcastResult struct {
	SentCount    int `json:"sent_count"`
	SkippedCount int `json:"skipped_count"`
	FailedCount  int `json:"failed_count"`
}

var sendAnnouncementBroadcastEmail = common.SendEmail

func BroadcastAnnouncementEmail(req AnnouncementEmailBroadcastRequest) (AnnouncementEmailBroadcastResult, error) {
	req.Source = strings.TrimSpace(req.Source)
	req.Target = strings.TrimSpace(req.Target)
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)

	if req.Source != AnnouncementEmailSourceNotice && req.Source != AnnouncementEmailSourceAnnouncement {
		return AnnouncementEmailBroadcastResult{}, fmt.Errorf("invalid source: %s", req.Source)
	}
	if req.Target != AnnouncementEmailTargetAgent && req.Target != AnnouncementEmailTargetEndUser && req.Target != AnnouncementEmailTargetAll {
		return AnnouncementEmailBroadcastResult{}, fmt.Errorf("invalid target: %s", req.Target)
	}
	if req.Title == "" {
		return AnnouncementEmailBroadcastResult{}, errors.New("title is required")
	}
	if req.Content == "" {
		return AnnouncementEmailBroadcastResult{}, errors.New("content is required")
	}

	users, err := listAnnouncementEmailRecipients(req.Target)
	if err != nil {
		return AnnouncementEmailBroadcastResult{}, err
	}

	result := AnnouncementEmailBroadcastResult{}
	for _, user := range users {
		email := strings.TrimSpace(user.Email)
		if email == "" {
			result.SkippedCount++
			continue
		}
		if err := sendAnnouncementBroadcastEmail(req.Title, email, req.Content); err != nil {
			result.FailedCount++
			common.SysLog(fmt.Sprintf("failed to send announcement email to user %d: %s", user.Id, err.Error()))
			continue
		}
		result.SentCount++
	}
	common.SysLog(fmt.Sprintf("announcement email broadcast source=%s target=%s sent=%d skipped=%d failed=%d", req.Source, req.Target, result.SentCount, result.SkippedCount, result.FailedCount))
	return result, nil
}

func listAnnouncementEmailRecipients(target string) ([]model.User, error) {
	query := model.DB.
		Select("id", "email", "role", "status", "user_type").
		Where("status = ?", common.UserStatusEnabled)

	switch target {
	case AnnouncementEmailTargetAgent:
		query = query.Where("COALESCE(user_type, '') = ?", model.UserTypeAgent)
	case AnnouncementEmailTargetEndUser:
		query = query.Where("role < ? AND COALESCE(user_type, '') <> ?", common.RoleAdminUser, model.UserTypeAgent)
	case AnnouncementEmailTargetAll:
		query = query.Where("role < ?", common.RoleAdminUser)
	default:
		return nil, fmt.Errorf("invalid target: %s", target)
	}

	var users []model.User
	if err := query.Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
