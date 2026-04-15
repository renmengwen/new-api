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

type adminPermissionAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type adminPermissionPageResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    common.PageInfo `json:"data"`
}

func setupAdminPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:admin_permission?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.AgentProfile{},
		&model.AgentUserRelation{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.UserPermissionOverride{},
		&model.UserMenuOverride{},
		&model.UserDataScopeOverride{},
		&model.AdminAuditLog{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newAdminPermissionContext(t *testing.T, method string, target string, body any) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestGetPermissionProfilesReturnsProfiles(t *testing.T) {
	db := setupAdminPermissionTestDB(t)
	require.NoError(t, db.Create(&model.PermissionProfile{
		ProfileName: "Agent Ops",
		ProfileType: "agent",
		Status:      model.CommonStatusEnabled,
	}).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodGet, "/api/admin/permission/profiles", nil)
	GetPermissionProfiles(ctx)

	var response adminPermissionPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
}

func TestUpdateUserPermissionBindingUpsertsBinding(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "admin_permission_user",
		Password:    "hashed-password",
		DisplayName: "Admin Permission User",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Admin Basic",
		ProfileType: "admin",
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, db.Create(&profile).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodPut, "/api/admin/permission/users/"+strconv.Itoa(user.Id), map[string]any{
		"profile_id": profile.Id,
	})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}

	UpdateUserPermissionBinding(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var binding model.UserPermissionBinding
	require.NoError(t, db.Where("user_id = ? AND status = ?", user.Id, model.CommonStatusEnabled).First(&binding).Error)
	require.Equal(t, profile.Id, binding.ProfileId)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND target_type = ? AND target_id = ?", "permission", "user", user.Id).First(&audit).Error)
	require.Equal(t, "bind_profile", audit.ActionType)
}

func TestUpdateUserPermissionBindingRejectsMismatchedProfileType(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "agent_permission_user",
		Password:    "hashed-password",
		DisplayName: "Agent Permission User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Admin Only",
		ProfileType: "admin",
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, db.Create(&profile).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodPut, "/api/admin/permission/users/"+strconv.Itoa(user.Id), map[string]any{
		"profile_id": profile.Id,
	})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}

	UpdateUserPermissionBinding(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Contains(t, response.Message, "profile type")

	var bindingCount int64
	require.NoError(t, db.Model(&model.UserPermissionBinding{}).Where("user_id = ?", user.Id).Count(&bindingCount).Error)
	require.Zero(t, bindingCount)
}

func TestUpdateUserPermissionBindingSupportsEndUserTargetWithMatchingProfileType(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "end_user_permission_target",
		Password:    "hashed-password",
		DisplayName: "End User Permission Target",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "End User Ops",
		ProfileType: model.UserTypeEndUser,
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, db.Create(&profile).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodPut, "/api/admin/permission/users/"+strconv.Itoa(user.Id), map[string]any{
		"profile_id": profile.Id,
	})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}

	UpdateUserPermissionBinding(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var binding model.UserPermissionBinding
	require.NoError(t, db.Where("user_id = ? AND status = ?", user.Id, model.CommonStatusEnabled).First(&binding).Error)
	require.Equal(t, profile.Id, binding.ProfileId)
}

func TestGetPermissionUsersIncludesCurrentProfile(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "admin_permission_list",
		Password:    "hashed-password",
		DisplayName: "Admin Permission List",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Admin Advanced",
		ProfileType: "admin",
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:    user.Id,
		ProfileId: profile.Id,
		Status:    model.CommonStatusEnabled,
	}).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodGet, "/api/admin/permission/users?p=1&page_size=10", nil)
	GetPermissionUsers(ctx)

	var response adminPermissionPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, profile.ProfileName, firstItem["profile_name"])
}

func TestGetPermissionProfilesRequiresActionPermissionForAdmin(t *testing.T) {
	db := setupAdminPermissionTestDB(t)
	operator := model.User{
		Username:    "permission_operator_no_grant",
		Password:    "hashed-password",
		DisplayName: "Permission Operator",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "permdeny",
	}
	require.NoError(t, db.Create(&operator).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodGet, "/api/admin/permission/profiles", nil)
	ctx.Set("id", operator.Id)
	ctx.Set("role", operator.Role)
	GetPermissionProfiles(ctx)

	var response adminPermissionPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
}
