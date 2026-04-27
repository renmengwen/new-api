package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
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

func TestNotifyAdminUsersByEmailSkipsDisabledModelMonitorRecipients(t *testing.T) {
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
		require.NoError(t, operation_setting.UpdateModelMonitorSettingFromMap(map[string]string{
			"notification_disabled_user_ids": "[]",
		}))
	})

	enabledAdmin := model.User{
		Username: "model_monitor_notify_enabled_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Email:    "enabled@example.com",
		Group:    "default",
		AffCode:  "notify_enabled",
	}
	disabledAdmin := model.User{
		Username: "model_monitor_notify_disabled_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Email:    "disabled@example.com",
		Group:    "default",
		AffCode:  "notify_disabled",
	}
	require.NoError(t, db.Create(&enabledAdmin).Error)
	require.NoError(t, db.Create(&disabledAdmin).Error)
	bindModelMonitorPermission(t, enabledAdmin, ActionRead)
	bindModelMonitorPermission(t, disabledAdmin, ActionRead)
	disabledIds, err := common.Marshal([]int{disabledAdmin.Id})
	require.NoError(t, err)
	require.NoError(t, operation_setting.UpdateModelMonitorSettingFromMap(map[string]string{
		"notification_disabled_user_ids": string(disabledIds),
	}))

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

	require.ElementsMatch(t, []string{"enabled@example.com"}, recipients)
}

func TestListModelMonitorNotificationUsersReturnsEligibleAdminsWithSelectionState(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)
	t.Cleanup(func() {
		require.NoError(t, operation_setting.UpdateModelMonitorSettingFromMap(map[string]string{
			"notification_disabled_user_ids": "[]",
		}))
	})

	root := model.User{
		Username:    "model_monitor_candidate_root",
		DisplayName: "Root Candidate",
		Password:    "hashed-password",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeRoot,
		Email:       "root-candidate@example.com",
		Group:       "default",
		AffCode:     "candidate_root",
	}
	readAdmin := model.User{
		Username:    "model_monitor_candidate_read_admin",
		DisplayName: "Read Candidate",
		Password:    "hashed-password",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Email:       "read-candidate@example.com",
		Group:       "default",
		AffCode:     "candidate_read",
	}
	noEmailAdmin := model.User{
		Username: "model_monitor_candidate_no_email",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Group:    "default",
		AffCode:  "candidate_no_email",
	}
	testOnlyAdmin := model.User{
		Username: "model_monitor_candidate_test_only",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Email:    "test-only-candidate@example.com",
		Group:    "default",
		AffCode:  "candidate_test_only",
	}
	disabledAccountAdmin := model.User{
		Username: "model_monitor_candidate_disabled_account",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusDisabled,
		UserType: model.UserTypeAdmin,
		Email:    "disabled-account@example.com",
		Group:    "default",
		AffCode:  "candidate_disabled_account",
	}
	require.NoError(t, db.Create(&root).Error)
	require.NoError(t, db.Create(&readAdmin).Error)
	require.NoError(t, db.Create(&noEmailAdmin).Error)
	require.NoError(t, db.Create(&testOnlyAdmin).Error)
	require.NoError(t, db.Create(&disabledAccountAdmin).Error)
	bindModelMonitorPermission(t, readAdmin, ActionRead)
	bindModelMonitorPermission(t, noEmailAdmin, ActionRead)
	bindModelMonitorPermission(t, testOnlyAdmin, ActionTest)
	bindModelMonitorPermission(t, disabledAccountAdmin, ActionRead)
	disabledIds, err := common.Marshal([]int{readAdmin.Id})
	require.NoError(t, err)
	require.NoError(t, operation_setting.UpdateModelMonitorSettingFromMap(map[string]string{
		"notification_disabled_user_ids": string(disabledIds),
	}))

	users, err := ListModelMonitorNotificationUsers(operation_setting.GetModelMonitorSetting())

	require.NoError(t, err)
	require.Len(t, users, 4)
	require.Equal(t, root.Id, users[0].Id)
	require.Equal(t, "root-candidate@example.com", users[0].Email)
	require.True(t, users[0].CanReceive)
	require.True(t, users[0].NotificationEnabled)
	require.Equal(t, readAdmin.Id, users[1].Id)
	require.Equal(t, "read-candidate@example.com", users[1].Email)
	require.True(t, users[1].CanReceive)
	require.False(t, users[1].NotificationEnabled)
	require.Equal(t, noEmailAdmin.Id, users[2].Id)
	require.Empty(t, users[2].Email)
	require.False(t, users[2].CanReceive)
	require.Equal(t, "no_email", users[2].DisabledReason)
	require.False(t, users[2].NotificationEnabled)
	require.Equal(t, testOnlyAdmin.Id, users[3].Id)
	require.Equal(t, "test-only-candidate@example.com", users[3].Email)
	require.False(t, users[3].CanReceive)
	require.Equal(t, "no_model_monitor_read_permission", users[3].DisabledReason)
	require.False(t, users[3].NotificationEnabled)
}
