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

type adminAuditLogResponseItem struct {
	Id                  int    `json:"id"`
	OperatorUserId      int    `json:"operator_user_id"`
	OperatorUserType    string `json:"operator_user_type"`
	ActionModule        string `json:"action_module"`
	ActionType          string `json:"action_type"`
	TargetType          string `json:"target_type"`
	TargetId            int    `json:"target_id"`
	OperatorUsername    string `json:"operator_username"`
	OperatorDisplayName string `json:"operator_display_name"`
	TargetUsername      string `json:"target_username"`
	TargetDisplayName   string `json:"target_display_name"`
}

type adminAuditLogPageData struct {
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
	Total    int                         `json:"total"`
	Items    []adminAuditLogResponseItem `json:"items"`
}

type adminAuditPageResponse struct {
	Success bool                  `json:"success"`
	Message string                `json:"message"`
	Data    adminAuditLogPageData `json:"data"`
}

type adminAuditPageRawResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Items []map[string]any `json:"items"`
	} `json:"data"`
}

func setupAdminAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open("file:admin_audit?mode=memory&cache=shared"), &gorm.Config{})
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
		&model.AgentUserRelation{},
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

func TestGetAdminAuditLogsReturnsLogs(t *testing.T) {
	db := setupAdminAuditTestDB(t)
	require.NoError(t, db.Create(&model.AdminAuditLog{
		OperatorUserId:   1,
		OperatorUserType: model.UserTypeAdmin,
		ActionModule:     "quota",
		ActionType:       "adjust",
		TargetType:       "user",
		TargetId:         2,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=10", nil)
	ctx.Set("id", 999)
	ctx.Set("role", common.RoleRootUser)

	GetAdminAuditLogs(ctx)

	var response adminAuditPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
}

func TestGetAdminAuditLogsReturnsEnrichedUserIdentityFields(t *testing.T) {
	db := setupAdminAuditTestDB(t)

	operator := model.User{
		Username:    "audit_operator_user",
		Password:    "hashed-password",
		DisplayName: "Audit Operator User",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "auditopuser",
	}
	target := model.User{
		Username:    "audit_target_user",
		Password:    "hashed-password",
		DisplayName: "Audit Target User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		AffCode:     "audittarget",
	}
	require.NoError(t, db.Create(&operator).Error)
	require.NoError(t, db.Create(&target).Error)
	require.NoError(t, db.Create(&model.AdminAuditLog{
		OperatorUserId:   operator.Id,
		OperatorUserType: operator.UserType,
		ActionModule:     "user_management",
		ActionType:       "update",
		TargetType:       "user",
		TargetId:         target.Id,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=10", nil)
	ctx.Set("id", 999)
	ctx.Set("role", common.RoleRootUser)

	GetAdminAuditLogs(ctx)

	var response adminAuditPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, operator.Username, response.Data.Items[0].OperatorUsername)
	require.Equal(t, operator.DisplayName, response.Data.Items[0].OperatorDisplayName)
	require.Equal(t, target.Username, response.Data.Items[0].TargetUsername)
	require.Equal(t, target.DisplayName, response.Data.Items[0].TargetDisplayName)
}

func TestGetAdminAuditLogsLeavesTargetIdentityEmptyForNonUserTargets(t *testing.T) {
	db := setupAdminAuditTestDB(t)

	operator := model.User{
		Username:    "audit_batch_operator",
		Password:    "hashed-password",
		DisplayName: "Audit Batch Operator",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "auditbatchop",
	}
	userTarget := model.User{
		Username:    "audit_batch_user_target",
		Password:    "hashed-password",
		DisplayName: "Audit Batch User Target",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		AffCode:     "auditbatchtg",
	}
	require.NoError(t, db.Create(&operator).Error)
	require.NoError(t, db.Create(&userTarget).Error)
	require.NoError(t, db.Create(&model.AdminAuditLog{
		OperatorUserId:   operator.Id,
		OperatorUserType: operator.UserType,
		ActionModule:     "quota",
		ActionType:       "adjust",
		TargetType:       "batch",
		TargetId:         userTarget.Id,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=10", nil)
	ctx.Set("id", 999)
	ctx.Set("role", common.RoleRootUser)

	GetAdminAuditLogs(ctx)

	var response adminAuditPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, "", response.Data.Items[0].TargetUsername)
	require.Equal(t, "", response.Data.Items[0].TargetDisplayName)
}

func TestGetAdminAuditLogsAllowsAgentUsersWithReadGrantAndScopesToSelf(t *testing.T) {
	db := setupAdminAuditTestDB(t)
	operator := model.User{
		Username:    "audit_agent_with_grant",
		Password:    "hashed-password",
		DisplayName: "Audit Agent With Grant",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
		AffCode:     "auditagentgr",
	}
	require.NoError(t, db.Create(&operator).Error)
	otherOperator := model.User{
		Username:    "audit_other_operator",
		Password:    "hashed-password",
		DisplayName: "Audit Other Operator",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "auditother",
	}
	managedUser := model.User{
		Username:    "audit_managed_user",
		Password:    "hashed-password",
		DisplayName: "Audit Managed User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		AffCode:     "auditmanaged",
	}
	require.NoError(t, db.Create(&otherOperator).Error)
	require.NoError(t, db.Create(&managedUser).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: operator.Id,
		EndUserId:   managedUser.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, operator.Id, model.UserTypeAgent, permissionGrant{
		Resource: "audit_management",
		Action:   "read",
	})
	require.NoError(t, db.Create(&[]model.AdminAuditLog{
		{
			OperatorUserId:   operator.Id,
			OperatorUserType: model.UserTypeAgent,
			ActionModule:     "user_management",
			ActionType:       "update",
			TargetType:       "user",
			TargetId:         operator.Id,
			CreatedAtTs:      common.GetTimestamp(),
		},
		{
			OperatorUserId:   managedUser.Id,
			OperatorUserType: model.UserTypeEndUser,
			ActionModule:     "quota",
			ActionType:       "adjust",
			TargetType:       "user",
			TargetId:         managedUser.Id,
			CreatedAtTs:      common.GetTimestamp() + 1,
		},
		{
			OperatorUserId:   otherOperator.Id,
			OperatorUserType: model.UserTypeAdmin,
			ActionModule:     "quota",
			ActionType:       "adjust",
			TargetType:       "user",
			TargetId:         otherOperator.Id,
			CreatedAtTs:      common.GetTimestamp() + 2,
		},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=10", nil)
	ctx.Set("id", operator.Id)
	ctx.Set("role", operator.Role)

	GetAdminAuditLogs(ctx)

	var response adminAuditPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 2, response.Data.Total)
	require.Len(t, response.Data.Items, 2)
	require.Equal(t, managedUser.Id, response.Data.Items[0].OperatorUserId)
	require.Equal(t, operator.Id, response.Data.Items[1].OperatorUserId)
}

func TestGetAdminAuditLogsListResponseOmitsHeavyFields(t *testing.T) {
	db := setupAdminAuditTestDB(t)
	require.NoError(t, db.Create(&model.AdminAuditLog{
		OperatorUserId:   1,
		OperatorUserType: model.UserTypeAdmin,
		ActionModule:     "quota",
		ActionType:       "adjust",
		ActionDesc:       "should stay out of list rows",
		TargetType:       "user",
		TargetId:         2,
		BeforeJSON:       `{"before":"value"}`,
		AfterJSON:        `{"after":"value"}`,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=10", nil)
	ctx.Set("id", 999)
	ctx.Set("role", common.RoleRootUser)

	GetAdminAuditLogs(ctx)

	var response adminAuditPageRawResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Items, 1)
	_, hasBeforeJSON := response.Data.Items[0]["before_json"]
	_, hasAfterJSON := response.Data.Items[0]["after_json"]
	_, hasActionDesc := response.Data.Items[0]["action_desc"]
	require.False(t, hasBeforeJSON)
	require.False(t, hasAfterJSON)
	require.False(t, hasActionDesc)
}

func TestGetAdminAuditLogsRequiresActionPermissionForAdmin(t *testing.T) {
	db := setupAdminAuditTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	operator := model.User{
		Username:    "audit_operator_no_grant",
		Password:    "hashed-password",
		DisplayName: "Audit Operator",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "auditdeny",
	}
	require.NoError(t, db.Create(&operator).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=10", nil)
	ctx.Set("id", operator.Id)
	ctx.Set("role", operator.Role)

	GetAdminAuditLogs(ctx)

	var response adminAuditPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
}
