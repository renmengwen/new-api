package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAdminPermissionServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:admin_permission_service?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.UserPermissionOverride{},
		&model.UserMenuOverride{},
		&model.UserDataScopeOverride{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestBuildUserPermissionsMergesAllowAndDenyOverrides(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)

	user := model.User{
		Username: "permission_merge_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Group:    "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Admin Base",
		ProfileType: model.UserTypeAdmin,
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: ResourceUserManagement,
		ActionKey:   ActionRead,
		Allowed:     true,
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:    user.Id,
		ProfileId: profile.Id,
		Status:    model.CommonStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionOverride{
		UserId:      user.Id,
		ResourceKey: ResourceUserManagement,
		ActionKey:   ActionRead,
		Effect:      "deny",
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionOverride{
		UserId:      user.Id,
		ResourceKey: ResourceQuotaManagement,
		ActionKey:   ActionLedgerRead,
		Effect:      "allow",
	}).Error)

	permissions := BuildUserPermissions(user.Id, user.Role)
	actions, ok := permissions["actions"].(map[string]bool)
	require.True(t, ok)
	require.False(t, actions[permissionActionKey(ResourceUserManagement, ActionRead)])
	require.True(t, actions[permissionActionKey(ResourceQuotaManagement, ActionLedgerRead)])
}

func TestBuildUserPermissionsMergesMenuOverrides(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)

	user := model.User{
		Username: "permission_merge_menu_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Group:    "default",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.UserMenuOverride{
		UserId:     user.Id,
		SectionKey: "admin",
		ModuleKey:  "setting",
		Effect:     "show",
	}).Error)
	require.NoError(t, db.Create(&model.UserMenuOverride{
		UserId:     user.Id,
		SectionKey: "admin",
		ModuleKey:  "channel",
		Effect:     "hide",
	}).Error)

	permissions := BuildUserPermissions(user.Id, user.Role)
	sidebar, ok := permissions["sidebar_modules"].(map[string]any)
	require.True(t, ok)

	adminSection, ok := sidebar["admin"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, adminSection["setting"])
	require.Equal(t, false, adminSection["channel"])
}

func TestBuildUserPermissionsKeepsBaseSidebarWhenTemplateHasNoMenuItems(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)

	user := model.User{
		Username: "permission_template_inherit_menu_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Group:    "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Admin No Menu Items",
		ProfileType: model.UserTypeAdmin,
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: ResourceUserManagement,
		ActionKey:   ActionRead,
		Allowed:     true,
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:    user.Id,
		ProfileId: profile.Id,
		Status:    model.CommonStatusEnabled,
	}).Error)

	permissions := BuildUserPermissions(user.Id, user.Role)
	sidebar, ok := permissions["sidebar_modules"].(map[string]any)
	require.True(t, ok)

	adminSection, ok := sidebar["admin"].(map[string]any)
	require.True(t, ok)
	require.NotEqual(t, false, adminSection["enabled"])
	require.Equal(t, false, adminSection["setting"])
}

func TestBuildUserPermissionsExpandsAdminSidebarSnapshotForLegacyAdmins(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)

	user := model.User{
		Username: "permission_explicit_sidebar_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Group:    "default",
	}
	require.NoError(t, db.Create(&user).Error)

	permissions := BuildUserPermissions(user.Id, user.Role)
	sidebar, ok := permissions["sidebar_modules"].(map[string]any)
	require.True(t, ok)

	chatSection, ok := sidebar["chat"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, chatSection["enabled"])
	require.Equal(t, true, chatSection["playground"])
	require.Equal(t, true, chatSection["chat"])

	consoleSection, ok := sidebar["console"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, consoleSection["enabled"])
	require.Equal(t, true, consoleSection["detail"])
	require.Equal(t, true, consoleSection["token"])
	require.Equal(t, true, consoleSection["log"])
	require.Equal(t, true, consoleSection["midjourney"])
	require.Equal(t, true, consoleSection["task"])

	personalSection, ok := sidebar["personal"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, personalSection["enabled"])
	require.Equal(t, true, personalSection["topup"])
	require.Equal(t, true, personalSection["personal"])

	adminSection, ok := sidebar["admin"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, adminSection["enabled"])
	require.Equal(t, true, adminSection["channel"])
	require.Equal(t, true, adminSection["subscription"])
	require.Equal(t, true, adminSection["models"])
	require.Equal(t, true, adminSection["deployment"])
	require.Equal(t, true, adminSection["redemption"])
	require.Equal(t, true, adminSection["user"])
	require.Equal(t, true, adminSection["admin-users"])
	require.Equal(t, true, adminSection["agents"])
	require.Equal(t, true, adminSection["permission-templates"])
	require.Equal(t, true, adminSection["user-permissions"])
	require.Equal(t, true, adminSection["quota-ledger"])
	require.Equal(t, true, adminSection["audit-logs"])
	require.Equal(t, false, adminSection["setting"])
}

func TestBuildUserPermissionsExpandsAdminSidebarSnapshotForRootUsers(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)

	user := model.User{
		Username: "permission_explicit_sidebar_root",
		Password: "hashed-password",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeRoot,
		Group:    "default",
	}
	require.NoError(t, db.Create(&user).Error)

	permissions := BuildUserPermissions(user.Id, user.Role)
	sidebar, ok := permissions["sidebar_modules"].(map[string]any)
	require.True(t, ok)

	chatSection, ok := sidebar["chat"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, chatSection["enabled"])
	require.Equal(t, true, chatSection["playground"])
	require.Equal(t, true, chatSection["chat"])

	consoleSection, ok := sidebar["console"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, consoleSection["enabled"])
	require.Equal(t, true, consoleSection["detail"])
	require.Equal(t, true, consoleSection["token"])
	require.Equal(t, true, consoleSection["log"])
	require.Equal(t, true, consoleSection["midjourney"])
	require.Equal(t, true, consoleSection["task"])

	personalSection, ok := sidebar["personal"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, personalSection["enabled"])
	require.Equal(t, true, personalSection["topup"])
	require.Equal(t, true, personalSection["personal"])

	adminSection, ok := sidebar["admin"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, adminSection["enabled"])
	require.Equal(t, true, adminSection["channel"])
	require.Equal(t, true, adminSection["subscription"])
	require.Equal(t, true, adminSection["models"])
	require.Equal(t, true, adminSection["deployment"])
	require.Equal(t, true, adminSection["redemption"])
	require.Equal(t, true, adminSection["user"])
	require.Equal(t, true, adminSection["admin-users"])
	require.Equal(t, true, adminSection["agents"])
	require.Equal(t, true, adminSection["permission-templates"])
	require.Equal(t, true, adminSection["user-permissions"])
	require.Equal(t, true, adminSection["quota-ledger"])
	require.Equal(t, true, adminSection["audit-logs"])
	require.Equal(t, true, adminSection["setting"])
}

func TestRequirePermissionActionRespectsDenyOverride(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)

	user := model.User{
		Username: "permission_require_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Group:    "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Admin Read",
		ProfileType: model.UserTypeAdmin,
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: ResourcePermissionManagement,
		ActionKey:   ActionRead,
		Allowed:     true,
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:    user.Id,
		ProfileId: profile.Id,
		Status:    model.CommonStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionOverride{
		UserId:      user.Id,
		ResourceKey: ResourcePermissionManagement,
		ActionKey:   ActionRead,
		Effect:      "deny",
	}).Error)

	err := RequirePermissionAction(user.Id, user.Role, ResourcePermissionManagement, ActionRead)
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}
