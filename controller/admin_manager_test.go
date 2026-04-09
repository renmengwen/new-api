package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type adminManagerAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type adminManagerPageResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func setupAdminManagerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:admin_manager?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.QuotaAccount{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.AdminAuditLog{},
	))

	require.NoError(t, db.Create(&model.User{
		Id:          999,
		Username:    "root-admin-manager",
		Password:    "hashed-password",
		DisplayName: "Root",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeRoot,
		Group:       "default",
		AffCode:     "rootamgr",
	}).Error)

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newAdminManagerContext(t *testing.T, method string, target string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		payload, err := common.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequest(method, target, io.NopCloser(bytes.NewReader(payload)))
		req.Header.Set("Content-Type", "application/json")
	}
	ctx.Request = req
	ctx.Set("id", 999)
	ctx.Set("role", common.RoleRootUser)
	return ctx, recorder
}

func TestCreateAdminManagerCreatesAdminUser(t *testing.T) {
	db := setupAdminManagerTestDB(t)

	ctx, recorder := newAdminManagerContext(t, http.MethodPost, "/api/admin/admin-users", map[string]any{
		"username":     "admin_created_1",
		"password":     "12345678",
		"display_name": "Admin Created",
		"remark":       "ops",
	})

	CreateAdminManager(ctx)

	var response adminManagerAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var user model.User
	require.NoError(t, db.Where("username = ?", "admin_created_1").First(&user).Error)
	require.Equal(t, common.RoleAdminUser, user.Role)
	require.Equal(t, model.UserTypeAdmin, user.GetUserType())

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND target_type = ? AND target_id = ?", "admin_management", "user", user.Id).First(&audit).Error)
	require.Equal(t, "create", audit.ActionType)
}

func TestGetAdminManagersReturnsAdminUsers(t *testing.T) {
	db := setupAdminManagerTestDB(t)

	require.NoError(t, db.Create(&model.User{
		Username:    "admin_list_1",
		Password:    "hashed-password",
		DisplayName: "Admin List 1",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "adminl1",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Username:    "agent_hidden_1",
		Password:    "hashed-password",
		DisplayName: "Agent Hidden",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
		AffCode:     "agenth1",
	}).Error)

	ctx, recorder := newAdminManagerContext(t, http.MethodGet, "/api/admin/admin-users?p=1&page_size=10", nil)
	GetAdminManagers(ctx)

	var response adminManagerPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	items, ok := response.Data["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	item := items[0].(map[string]interface{})
	require.Equal(t, "admin_list_1", item["username"])
	require.Equal(t, model.UserTypeAdmin, item["user_type"])
}

func TestUpdateAdminManagerUpdatesProfileAndWritesAudit(t *testing.T) {
	db := setupAdminManagerTestDB(t)

	user := model.User{
		Username:    "admin_edit_1",
		Password:    "hashed-password",
		DisplayName: "Before",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "admine1",
	}
	require.NoError(t, db.Create(&user).Error)

	ctx, recorder := newAdminManagerContext(t, http.MethodPut, "/api/admin/admin-users/"+strconv.Itoa(user.Id), map[string]any{
		"display_name": "After",
		"remark":       "updated",
	})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}

	UpdateAdminManager(ctx)

	var response adminManagerAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var updated model.User
	require.NoError(t, db.First(&updated, user.Id).Error)
	require.Equal(t, "After", updated.DisplayName)
	require.Equal(t, "updated", updated.Remark)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND target_type = ? AND target_id = ?", "admin_management", "user", user.Id).Order("id desc").First(&audit).Error)
	require.Equal(t, "update", audit.ActionType)
}

func TestUpdateAdminManagerStatusDisablesAdmin(t *testing.T) {
	db := setupAdminManagerTestDB(t)

	user := model.User{
		Username:    "admin_status_1",
		Password:    "hashed-password",
		DisplayName: "Admin Status",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "admins1",
	}
	require.NoError(t, db.Create(&user).Error)

	ctx, recorder := newAdminManagerContext(t, http.MethodPost, "/api/admin/admin-users/"+strconv.Itoa(user.Id)+"/disable", nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	DisableAdminManager(ctx)

	var response adminManagerAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var updated model.User
	require.NoError(t, db.First(&updated, user.Id).Error)
	require.Equal(t, common.UserStatusDisabled, updated.Status)
}
