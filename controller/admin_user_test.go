package controller

import (
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

type adminUserAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type adminUserPageResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    common.PageInfo `json:"data"`
}

func setupAdminUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:admin_user?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.AgentUserRelation{},
		&model.QuotaAccount{},
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

func newAdminUserContext(t *testing.T, method string, target string, operatorId int, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	ctx.Set("id", operatorId)
	ctx.Set("role", role)
	return ctx, recorder
}

func seedManagedUser(t *testing.T, db *gorm.DB, username string, userType string, role int, quota int, parentAgentId int) model.User {
	t.Helper()

	user := model.User{
		Username:      username,
		Password:      "hashed-password",
		DisplayName:   username,
		Role:          role,
		Status:        common.UserStatusEnabled,
		UserType:      userType,
		ParentAgentId: parentAgentId,
		Group:         "default",
		AffCode:       username + "_aff",
		Quota:         quota,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.QuotaAccount{
		OwnerType:      model.QuotaOwnerTypeUser,
		OwnerId:        user.Id,
		Balance:        quota,
		TotalRecharged: quota,
		Status:         model.CommonStatusEnabled,
		CreatedAtTs:    common.GetTimestamp(),
		UpdatedAtTs:    common.GetTimestamp(),
	}).Error)
	return user
}

func TestGetAdminUsersReturnsOnlyEndUsers(t *testing.T) {
	db := setupAdminUserTestDB(t)
	_ = seedManagedUser(t, db, "admin_users_end", model.UserTypeEndUser, common.RoleCommonUser, 600, 0)
	_ = seedManagedUser(t, db, "admin_users_agent", model.UserTypeAgent, common.RoleAdminUser, 0, 0)

	ctx, recorder := newAdminUserContext(t, http.MethodGet, "/api/admin/users?p=1&page_size=10", 999, common.RoleRootUser)
	GetAdminUsers(ctx)

	var response adminUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, model.UserTypeEndUser, firstItem["user_type"])
}

func TestGetAdminUsersIncludesLegacyCommonUsersWithoutUserType(t *testing.T) {
	db := setupAdminUserTestDB(t)
	legacyUser := seedManagedUser(t, db, "legacy_end_user", "", common.RoleCommonUser, 600, 0)
	_ = seedManagedUser(t, db, "admin_users_agent", model.UserTypeAgent, common.RoleCommonUser, 0, 0)

	ctx, recorder := newAdminUserContext(t, http.MethodGet, "/api/admin/users?p=1&page_size=10", 999, common.RoleRootUser)
	GetAdminUsers(ctx)

	var response adminUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(legacyUser.Id), firstItem["id"])
}

func TestGetAdminUsersForAgentFiltersOwnedUsers(t *testing.T) {
	db := setupAdminUserTestDB(t)
	agent := seedManagedUser(t, db, "agent_operator", model.UserTypeAgent, common.RoleAdminUser, 0, 0)
	owned := seedManagedUser(t, db, "owned_end_user", model.UserTypeEndUser, common.RoleCommonUser, 300, agent.Id)
	_ = seedManagedUser(t, db, "other_end_user", model.UserTypeEndUser, common.RoleCommonUser, 500, 0)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   owned.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "user_management", Action: "read"},
	)

	ctx, recorder := newAdminUserContext(t, http.MethodGet, "/api/admin/users?p=1&page_size=10", agent.Id, common.RoleAdminUser)
	GetAdminUsers(ctx)

	var response adminUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(owned.Id), firstItem["id"])
}

func TestGetAdminUserReturnsQuotaSummary(t *testing.T) {
	db := setupAdminUserTestDB(t)
	user := seedManagedUser(t, db, "admin_user_detail", model.UserTypeEndUser, common.RoleCommonUser, 900, 0)

	ctx, recorder := newAdminUserContext(t, http.MethodGet, "/api/admin/users/"+strconv.Itoa(user.Id), 999, common.RoleRootUser)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	GetAdminUser(ctx)

	var response adminUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, float64(user.Id), response.Data["id"])

	quotaSummary, ok := response.Data["quota_summary"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(900), quotaSummary["balance"])
}

func TestUpdateAdminUserStatusWritesAudit(t *testing.T) {
	db := setupAdminUserTestDB(t)
	user := seedManagedUser(t, db, "admin_user_disable", model.UserTypeEndUser, common.RoleCommonUser, 200, 0)

	ctx, recorder := newAdminUserContext(t, http.MethodPost, "/api/admin/users/"+strconv.Itoa(user.Id)+"/disable", 999, common.RoleRootUser)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	DisableAdminUser(ctx)

	var response adminUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	require.Equal(t, common.UserStatusDisabled, reloaded.Status)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ? AND target_type = ? AND target_id = ?", "user_management", "disable", "user", user.Id).First(&audit).Error)
}

func TestGetAdminUsersRequiresActionPermissionForAdmin(t *testing.T) {
	db := setupAdminUserTestDB(t)
	operator := seedManagedUser(t, db, "admin_users_operator_no_grant", model.UserTypeAdmin, common.RoleAdminUser, 0, 0)
	_ = seedManagedUser(t, db, "visible_end_user", model.UserTypeEndUser, common.RoleCommonUser, 100, 0)

	ctx, recorder := newAdminUserContext(t, http.MethodGet, "/api/admin/users?p=1&page_size=10", operator.Id, operator.Role)
	GetAdminUsers(ctx)

	var response adminUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
}
