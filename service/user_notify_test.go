package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func bindModelMonitorPermission(t *testing.T, user model.User, actionKey string) {
	t.Helper()

	profile := model.PermissionProfile{
		ProfileName: user.Username + "_" + actionKey,
		ProfileType: user.GetUserType(),
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, model.DB.Create(&profile).Error)
	require.NoError(t, model.DB.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: ResourceModelMonitorManagement,
		ActionKey:   actionKey,
		Allowed:     true,
	}).Error)
	require.NoError(t, model.DB.Create(&model.UserPermissionBinding{
		UserId:    user.Id,
		ProfileId: profile.Id,
		Status:    model.CommonStatusEnabled,
	}).Error)
}

func TestNotifyAdminUsersByEmailRequiresModelMonitorReadPermission(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)
	notifyLimitStore.Range(func(key any, _ any) bool {
		notifyLimitStore.Delete(key)
		return true
	})
	originalNotifyLimitCount := constant.NotifyLimitCount
	originalNotifyLimitDurationMinute := constant.NotificationLimitDurationMinute
	constant.NotifyLimitCount = 100
	constant.NotificationLimitDurationMinute = 10
	t.Cleanup(func() {
		constant.NotifyLimitCount = originalNotifyLimitCount
		constant.NotificationLimitDurationMinute = originalNotifyLimitDurationMinute
	})

	root := model.User{
		Username: "model_monitor_notify_root",
		Password: "hashed-password",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeRoot,
		Email:    "root@example.com",
		Group:    "default",
		AffCode:  "notify_root",
	}
	readAdmin := model.User{
		Username: "model_monitor_notify_read_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Email:    "read@example.com",
		Group:    "default",
		AffCode:  "notify_read",
	}
	testOnlyAdmin := model.User{
		Username: "model_monitor_notify_test_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Email:    "test-only@example.com",
		Group:    "default",
		AffCode:  "notify_test_only",
	}
	noPermissionAdmin := model.User{
		Username: "model_monitor_notify_no_permission_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Email:    "no-permission@example.com",
		Group:    "default",
		AffCode:  "notify_no_permission",
	}
	require.NoError(t, db.Create(&root).Error)
	require.NoError(t, db.Create(&readAdmin).Error)
	require.NoError(t, db.Create(&testOnlyAdmin).Error)
	require.NoError(t, db.Create(&noPermissionAdmin).Error)
	bindModelMonitorPermission(t, readAdmin, ActionRead)
	bindModelMonitorPermission(t, testOnlyAdmin, ActionTest)

	originalSendEmailNotify := sendEmailNotify
	t.Cleanup(func() {
		sendEmailNotify = originalSendEmailNotify
	})

	recipients := make([]string, 0)
	sendEmailNotify = func(userEmail string, data dto.Notify) error {
		recipients = append(recipients, userEmail)
		return nil
	}

	NotifyAdminUsersByEmail(dto.NotifyTypeModelMonitor, "subject", "content")

	require.ElementsMatch(t, []string{"root@example.com", "read@example.com"}, recipients)
}
