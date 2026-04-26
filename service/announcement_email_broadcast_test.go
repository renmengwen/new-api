package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func seedBroadcastUser(t *testing.T, user model.User) model.User {
	t.Helper()
	if user.Password == "" {
		user.Password = "hashed-password"
	}
	if user.Group == "" {
		user.Group = "default"
	}
	if user.AffCode == "" {
		user.AffCode = user.Username
	}
	require.NoError(t, model.DB.Create(&user).Error)
	return user
}

func TestBroadcastAnnouncementEmailTargetsAgentsOnly(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)
	seedBroadcastUser(t, model.User{Username: "broadcast_agent", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAgent, Email: "agent@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_end_user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "user@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAdmin, Email: "admin@example.com"})

	originalSender := sendAnnouncementBroadcastEmail
	t.Cleanup(func() { sendAnnouncementBroadcastEmail = originalSender })
	recipients := []string{}
	sendAnnouncementBroadcastEmail = func(subject string, receiver string, content string) error {
		recipients = append(recipients, receiver)
		require.Equal(t, "Subject", subject)
		require.Equal(t, "<p>Body</p>", content)
		return nil
	}

	result, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{
		Source:  AnnouncementEmailSourceNotice,
		Target:  AnnouncementEmailTargetAgent,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.SentCount)
	require.Equal(t, 0, result.SkippedCount)
	require.Equal(t, 0, result.FailedCount)
	require.Equal(t, []string{"agent@example.com"}, recipients)
}

func TestBroadcastAnnouncementEmailTargetsEndUsersAndLegacyBlankUserType(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)
	seedBroadcastUser(t, model.User{Username: "broadcast_end_user_current", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "current@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_end_user_legacy", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: "", Email: "legacy@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_agent_skip", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAgent, Email: "agent-skip@example.com"})

	originalSender := sendAnnouncementBroadcastEmail
	t.Cleanup(func() { sendAnnouncementBroadcastEmail = originalSender })
	recipients := []string{}
	sendAnnouncementBroadcastEmail = func(subject string, receiver string, content string) error {
		recipients = append(recipients, receiver)
		return nil
	}

	result, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{
		Source:  AnnouncementEmailSourceAnnouncement,
		Target:  AnnouncementEmailTargetEndUser,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.SentCount)
	require.Equal(t, 0, result.SkippedCount)
	require.Equal(t, 0, result.FailedCount)
	require.ElementsMatch(t, []string{"current@example.com", "legacy@example.com"}, recipients)
}

func TestBroadcastAnnouncementEmailTargetsAllNonAdminUsers(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)
	seedBroadcastUser(t, model.User{Username: "broadcast_all_agent", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAgent, Email: "agent@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_all_user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "user@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_all_admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, UserType: model.UserTypeAdmin, Email: "admin@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_all_root", Role: common.RoleRootUser, Status: common.UserStatusEnabled, UserType: model.UserTypeRoot, Email: "root@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_all_disabled", Role: common.RoleCommonUser, Status: common.UserStatusDisabled, UserType: model.UserTypeEndUser, Email: "disabled@example.com"})

	originalSender := sendAnnouncementBroadcastEmail
	t.Cleanup(func() { sendAnnouncementBroadcastEmail = originalSender })
	recipients := []string{}
	sendAnnouncementBroadcastEmail = func(subject string, receiver string, content string) error {
		recipients = append(recipients, receiver)
		return nil
	}

	result, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{
		Source:  AnnouncementEmailSourceNotice,
		Target:  AnnouncementEmailTargetAll,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.SentCount)
	require.Equal(t, 0, result.SkippedCount)
	require.Equal(t, 0, result.FailedCount)
	require.ElementsMatch(t, []string{"agent@example.com", "user@example.com"}, recipients)
}

func TestBroadcastAnnouncementEmailSkipsEmptyEmailAndContinuesAfterFailure(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)
	seedBroadcastUser(t, model.User{Username: "broadcast_skip_no_email", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser})
	seedBroadcastUser(t, model.User{Username: "broadcast_fail_email", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "fail@example.com"})
	seedBroadcastUser(t, model.User{Username: "broadcast_ok_email", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, UserType: model.UserTypeEndUser, Email: "ok@example.com"})

	originalSender := sendAnnouncementBroadcastEmail
	t.Cleanup(func() { sendAnnouncementBroadcastEmail = originalSender })
	sendAnnouncementBroadcastEmail = func(subject string, receiver string, content string) error {
		if receiver == "fail@example.com" {
			return errors.New("smtp failed")
		}
		return nil
	}

	result, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{
		Source:  AnnouncementEmailSourceNotice,
		Target:  AnnouncementEmailTargetEndUser,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.SentCount)
	require.Equal(t, 1, result.SkippedCount)
	require.Equal(t, 1, result.FailedCount)
}

func TestBroadcastAnnouncementEmailRejectsInvalidInput(t *testing.T) {
	setupAdminPermissionServiceTestDB(t)

	_, err := BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{Source: "bad", Target: AnnouncementEmailTargetAll, Title: "Subject", Content: "<p>Body</p>"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid source")

	_, err = BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{Source: AnnouncementEmailSourceNotice, Target: "bad", Title: "Subject", Content: "<p>Body</p>"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid target")

	_, err = BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{Source: AnnouncementEmailSourceNotice, Target: AnnouncementEmailTargetAll, Title: " ", Content: "<p>Body</p>"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "title is required")

	_, err = BroadcastAnnouncementEmail(AnnouncementEmailBroadcastRequest{Source: AnnouncementEmailSourceNotice, Target: AnnouncementEmailTargetAll, Title: "Subject", Content: " "})
	require.Error(t, err)
	require.Contains(t, err.Error(), "content is required")
}
