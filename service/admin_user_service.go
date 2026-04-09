package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type AdminUserListItem struct {
	Id            int    `json:"id"`
	Username      string `json:"username"`
	DisplayName   string `json:"display_name"`
	Status        int    `json:"status"`
	UserType      string `json:"user_type"`
	ParentAgentId int    `json:"parent_agent_id"`
	Quota         int    `json:"quota"`
	LastActiveAt  int64  `json:"last_active_at"`
}

func ListAdminUsers(pageInfo *common.PageInfo, keyword string, operatorUserId int, operatorRole int) ([]AdminUserListItem, int64, error) {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, 0, err
	}

	var items []AdminUserListItem
	var total int64

	query := model.DB.Model(&model.User{}).
		Select("users.id, users.username, users.display_name, users.status, users.user_type, users.parent_agent_id, users.quota, users.last_active_at")
	query = ApplyManagedEndUserScope(query, operator)
	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("users.username LIKE ? OR users.display_name LIKE ? OR users.email LIKE ?", like, like, like)
	}

	countQuery := model.DB.Model(&model.User{})
	countQuery = ApplyManagedEndUserScope(countQuery, operator)
	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		countQuery = countQuery.Where("users.username LIKE ? OR users.display_name LIKE ? OR users.email LIKE ?", like, like, like)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("users.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func GetAdminUserDetail(targetUserId int, operatorUserId int, operatorRole int) (map[string]any, error) {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, err
	}

	var user model.User
	query := model.DB.Model(&model.User{}).Where("users.id = ?", targetUserId)
	query = ApplyManagedEndUserScope(query, operator)
	if err := query.First(&user).Error; err != nil {
		return nil, err
	}

	quotaSummary, err := GetUserQuotaSummary(user.Id)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":              user.Id,
		"username":        user.Username,
		"display_name":    user.DisplayName,
		"status":          user.Status,
		"user_type":       user.GetUserType(),
		"parent_agent_id": user.ParentAgentId,
		"phone":           user.Phone,
		"email":           user.Email,
		"quota":           user.Quota,
		"last_active_at":  user.LastActiveAt,
		"quota_summary":   quotaSummary,
	}, nil
}

func UpdateAdminUserStatus(targetUserId int, status int, operatorUserId int, operatorRole int, ip string) error {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return err
	}

	var user model.User
	query := model.DB.Model(&model.User{}).Where("users.id = ?", targetUserId)
	query = ApplyManagedEndUserScope(query, operator)
	if err := query.First(&user).Error; err != nil {
		return err
	}

	if user.Status == status {
		return nil
	}

	beforeJSON, _ := common.Marshal(map[string]any{"status": user.Status})
	afterJSON, _ := common.Marshal(map[string]any{"status": status})
	actionType := "disable"
	if status == common.UserStatusEnabled {
		actionType = "enable"
	}

	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("status", status).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operator.Id,
		OperatorUserType: operator.GetUserType(),
		ActionModule:     "user_management",
		ActionType:       actionType,
		ActionDesc:       "update managed user status",
		TargetType:       "user",
		TargetId:         user.Id,
		BeforeJSON:       string(beforeJSON),
		AfterJSON:        string(afterJSON),
		IP:               ip,
	}); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
