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
