package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type getSelfResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func setupGetSelfTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:get_self_user_type?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestGetSelfIncludesUserType(t *testing.T) {
	db := setupGetSelfTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
	))

	user := model.User{
		Username:    "agent-demo",
		Password:    "hashed-password",
		DisplayName: "Agent Demo",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		UserType:    model.UserTypeAgent,
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	ctx.Set("id", user.Id)
	ctx.Set("role", user.Role)

	GetSelf(ctx)

	var response getSelfResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, model.UserTypeAgent, response.Data["user_type"])
}

func TestGetSelfIncludesActionPermissions(t *testing.T) {
	db := setupGetSelfTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
	))

	user := model.User{
		Username:    "permission-self-demo",
		Password:    "hashed-password",
		DisplayName: "Permission Self Demo",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		UserType:    model.UserTypeAdmin,
		AffCode:     "selfperm",
	}
	require.NoError(t, db.Create(&user).Error)
	grantPermissionActions(t, db, user.Id, "admin",
		permissionGrant{Resource: "quota_management", Action: "adjust"},
		permissionGrant{Resource: "user_management", Action: "read"},
	)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	ctx.Set("id", user.Id)
	ctx.Set("role", user.Role)

	GetSelf(ctx)

	var response getSelfResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	permissions, ok := response.Data["permissions"].(map[string]any)
	require.True(t, ok)
	actions, ok := permissions["actions"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, actions["quota_management.adjust"])
	require.Equal(t, true, actions["user_management.read"])
}

func TestGetSelfReturnsCompleteSidebarPermissionsForAgentTemplate(t *testing.T) {
	db := setupGetSelfTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
	))

	sidebarConfigBytes, err := common.Marshal(map[string]any{
		"chat": map[string]any{
			"enabled":    true,
			"playground": true,
			"chat":       true,
		},
		"console": map[string]any{
			"enabled":    true,
			"detail":     true,
			"token":      true,
			"log":        true,
			"midjourney": true,
			"task":       true,
		},
		"personal": map[string]any{
			"enabled":  true,
			"topup":    true,
			"personal": true,
		},
		"admin": map[string]any{
			"enabled": true,
			"user":    true,
		},
	})
	require.NoError(t, err)

	user := model.User{
		Username:    "agent-sidebar-demo",
		Password:    "hashed-password",
		DisplayName: "Agent Sidebar Demo",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		UserType:    model.UserTypeAgent,
		Setting:     string(sidebarConfigBytes),
	}
	require.NoError(t, db.Create(&user).Error)
	grantPermissionActions(t, db, user.Id, model.UserTypeAgent,
		permissionGrant{Resource: "quota_management", Action: "adjust"},
		permissionGrant{Resource: "user_management", Action: "read"},
	)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	ctx.Set("id", user.Id)
	ctx.Set("role", user.Role)

	GetSelf(ctx)

	var response getSelfResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	permissions, ok := response.Data["permissions"].(map[string]any)
	require.True(t, ok)
	sidebar, ok := permissions["sidebar_modules"].(map[string]any)
	require.True(t, ok)

	chatSection, ok := sidebar["chat"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, chatSection["enabled"])
	require.Equal(t, true, chatSection["chat"])

	consoleSection, ok := sidebar["console"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, consoleSection["enabled"])
	require.Equal(t, true, consoleSection["token"])

	personalSection, ok := sidebar["personal"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, personalSection["enabled"])
	require.Equal(t, true, personalSection["personal"])

	require.Equal(t, false, sidebar["admin"])
}

func TestGetSelfAppliesTemplateMenuVisibilityDefaults(t *testing.T) {
	db := setupGetSelfTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
	))

	sidebarConfigBytes, err := common.Marshal(map[string]any{
		"chat": map[string]any{
			"enabled":    true,
			"playground": true,
			"chat":       true,
		},
		"console": map[string]any{
			"enabled":    true,
			"detail":     true,
			"token":      true,
			"log":        true,
			"midjourney": true,
			"task":       true,
		},
		"personal": map[string]any{
			"enabled":  true,
			"topup":    true,
			"personal": true,
		},
		"admin": map[string]any{
			"enabled":      true,
			"user":         true,
			"quota-ledger": false,
		},
	})
	require.NoError(t, err)

	user := model.User{
		Username:    "agent-template-menu-demo",
		Password:    "hashed-password",
		DisplayName: "Agent Template Menu Demo",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		UserType:    model.UserTypeAgent,
		Setting:     string(sidebarConfigBytes),
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Agent Template Menu",
		ProfileType: model.UserTypeAgent,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: "user_management",
		ActionKey:   "read",
		Allowed:     true,
		ScopeType:   model.ScopeTypeAll,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: "__menu__.admin",
		ActionKey:   "quota-ledger",
		Allowed:     true,
		ScopeType:   model.ScopeTypeAll,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:        user.Id,
		ProfileId:     profile.Id,
		Status:        model.CommonStatusEnabled,
		EffectiveFrom: common.GetTimestamp(),
		CreatedAtTs:   common.GetTimestamp(),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	ctx.Set("id", user.Id)
	ctx.Set("role", user.Role)

	GetSelf(ctx)

	var response getSelfResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	permissions, ok := response.Data["permissions"].(map[string]any)
	require.True(t, ok)
	sidebar, ok := permissions["sidebar_modules"].(map[string]any)
	require.True(t, ok)
	adminSection, ok := sidebar["admin"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, adminSection["quota-ledger"])
	require.Equal(t, false, adminSection["user"])
}

func TestGetSelfTreatsLegacyRootUserTypeAsRoot(t *testing.T) {
	db := setupGetSelfTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
	))

	user := model.User{
		Username:    "legacy-root",
		Password:    "hashed-password",
		DisplayName: "Legacy Root",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		UserType:    model.UserTypeEndUser,
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	ctx.Set("id", user.Id)
	ctx.Set("role", user.Role)

	GetSelf(ctx)

	var response getSelfResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, model.UserTypeRoot, response.Data["user_type"])

	permissions, ok := response.Data["permissions"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, model.UserTypeRoot, permissions["profile_type"])

	sidebar, ok := permissions["sidebar_modules"].(map[string]any)
	require.True(t, ok)

	chatSection, ok := sidebar["chat"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, chatSection["enabled"])
	require.Equal(t, true, chatSection["chat"])

	consoleSection, ok := sidebar["console"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, consoleSection["enabled"])
	require.Equal(t, true, consoleSection["token"])

	personalSection, ok := sidebar["personal"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, personalSection["enabled"])
	require.Equal(t, true, personalSection["personal"])

	adminSection, ok := sidebar["admin"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, adminSection["enabled"])
	require.Equal(t, true, adminSection["setting"])
}
