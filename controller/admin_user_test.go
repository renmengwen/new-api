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
	"github.com/QuantumNous/new-api/service"
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
		&model.UserPermissionOverride{},
		&model.UserMenuOverride{},
		&model.UserDataScopeOverride{},
		&model.AgentUserRelation{},
		&model.AgentQuotaPolicy{},
		&model.QuotaAccount{},
		&model.QuotaTransferOrder{},
		&model.QuotaLedger{},
		&model.Log{},
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

func newAdminUserJSONContext(t *testing.T, method string, target string, body any, operatorId int, role int) (*gin.Context, *httptest.ResponseRecorder) {
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

func listQuotaLedgersForUser(t *testing.T, db *gorm.DB, userId int) []model.QuotaLedger {
	t.Helper()

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userId)
	require.NoError(t, err)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	return ledgers
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

func TestGetAdminUsersIncludesInviteOwnerUsernames(t *testing.T) {
	db := setupAdminUserTestDB(t)
	agent := seedManagedUser(t, db, "agent_parent", model.UserTypeAgent, common.RoleAdminUser, 0, 0)
	inviter := seedManagedUser(t, db, "direct_inviter", model.UserTypeEndUser, common.RoleCommonUser, 0, 0)
	owned := seedManagedUser(t, db, "managed_end_user", model.UserTypeEndUser, common.RoleCommonUser, 300, agent.Id)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", owned.Id).Updates(map[string]any{
		"parent_agent_id": agent.Id,
		"inviter_id":      inviter.Id,
	}).Error)
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
	require.Equal(t, "direct_inviter", firstItem["inviter_username"])
	require.Equal(t, "agent_parent", firstItem["parent_agent_username"])
}

func TestGetAdminUsersForAgentAllowsAllScopeOverride(t *testing.T) {
	db := setupAdminUserTestDB(t)
	agent := seedManagedUser(t, db, "agent_scope_all_operator", model.UserTypeAgent, common.RoleAdminUser, 0, 0)
	owned := seedManagedUser(t, db, "owned_scope_all_end_user", model.UserTypeEndUser, common.RoleCommonUser, 300, agent.Id)
	_ = seedManagedUser(t, db, "unowned_scope_all_end_user", model.UserTypeEndUser, common.RoleCommonUser, 500, 0)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   owned.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.UserDataScopeOverride{
		UserId:      agent.Id,
		ResourceKey: "user_management",
		ScopeType:   model.ScopeTypeAll,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "user_management", Action: "read"},
	)

	ctx, recorder := newAdminUserContext(t, http.MethodGet, "/api/admin/users?p=1&page_size=10", agent.Id, common.RoleAdminUser)
	GetAdminUsers(ctx)

	var response adminUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 2, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
}

func TestGetAdminUsersForAgentAppliesTemplateAssignedScope(t *testing.T) {
	db := setupAdminUserTestDB(t)
	agent := seedManagedUser(t, db, "agent_template_scope_operator", model.UserTypeAgent, common.RoleAdminUser, 0, 0)
	allowed := seedManagedUser(t, db, "allowed_template_scope_user", model.UserTypeEndUser, common.RoleCommonUser, 300, agent.Id)
	_ = seedManagedUser(t, db, "blocked_template_scope_user", model.UserTypeEndUser, common.RoleCommonUser, 500, agent.Id)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   allowed.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)

	profile := model.PermissionProfile{
		ProfileName: "Agent Assigned Scope Template",
		ProfileType: model.UserTypeAgent,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: service.ResourceUserManagement,
		ActionKey:   service.ActionRead,
		Allowed:     true,
		ScopeType:   model.ScopeTypeAll,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:      profile.Id,
		ResourceKey:    service.ResourceUserManagement,
		ActionKey:      "__scope__",
		Allowed:        true,
		ScopeType:      model.ScopeTypeAssigned,
		ExtraScopeJSON: "[" + strconv.Itoa(allowed.Id) + "]",
		CreatedAtTs:    common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:        agent.Id,
		ProfileId:     profile.Id,
		Status:        model.CommonStatusEnabled,
		EffectiveFrom: common.GetTimestamp(),
		CreatedAtTs:   common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminUserContext(t, http.MethodGet, "/api/admin/users?p=1&page_size=10", agent.Id, common.RoleAdminUser)
	GetAdminUsers(ctx)

	var response adminUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(allowed.Id), firstItem["id"])
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

func TestCreateAdminUserForAgentCreatesOpeningLedgerEntry(t *testing.T) {
	db := setupAdminUserTestDB(t)
	previousNewUserQuota := common.QuotaForNewUser
	common.QuotaForNewUser = 64
	t.Cleanup(func() {
		common.QuotaForNewUser = previousNewUserQuota
	})
	agent := seedManagedUser(t, db, "agent_create_operator", model.UserTypeAgent, common.RoleAdminUser, 0, 0)
	grantPermissionActions(t, db, agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceUserManagement, Action: service.ActionCreate},
	)

	ctx, recorder := newAdminUserJSONContext(t, http.MethodPost, "/api/admin/users", map[string]any{
		"username":     "agent_created_user",
		"password":     "12345678",
		"display_name": "Agent Created User",
		"remark":       "created by agent",
	}, agent.Id, common.RoleAdminUser)

	CreateAdminUser(ctx)

	var response adminUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var user model.User
	require.NoError(t, db.Where("username = ?", "agent_created_user").First(&user).Error)
	require.Equal(t, common.RoleCommonUser, user.Role)
	require.Equal(t, model.UserTypeEndUser, user.GetUserType())
	require.Equal(t, agent.Id, user.ParentAgentId)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, common.QuotaForNewUser, account.Balance)

	ledgers := listQuotaLedgersForUser(t, db, user.Id)
	require.Len(t, ledgers, 1)
	require.Equal(t, "opening", ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, common.QuotaForNewUser, ledgers[0].Amount)
	require.Equal(t, 0, ledgers[0].BalanceBefore)
	require.Equal(t, common.QuotaForNewUser, ledgers[0].BalanceAfter)
	require.Equal(t, "admin_user_create", ledgers[0].SourceType)
	require.Equal(t, user.Id, ledgers[0].SourceId)

	var relation model.AgentUserRelation
	require.NoError(t, db.Where("agent_user_id = ? AND end_user_id = ? AND status = ?", agent.Id, user.Id, model.CommonStatusEnabled).First(&relation).Error)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ? AND target_type = ? AND target_id = ?", service.ResourceUserManagement, service.ActionCreate, "user", user.Id).First(&audit).Error)
}

func TestUpdateAdminUserForAgentRejectsUserOutsideManagedScope(t *testing.T) {
	db := setupAdminUserTestDB(t)
	agent := seedManagedUser(t, db, "agent_update_operator", model.UserTypeAgent, common.RoleAdminUser, 0, 0)
	target := seedManagedUser(t, db, "unmanaged_update_target", model.UserTypeEndUser, common.RoleCommonUser, 120, 0)
	grantPermissionActions(t, db, agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceUserManagement, Action: service.ActionUpdate},
	)

	ctx, recorder := newAdminUserJSONContext(t, http.MethodPut, "/api/admin/users/"+strconv.Itoa(target.Id), map[string]any{
		"display_name": "Should Fail",
		"remark":       "forbidden",
	}, agent.Id, common.RoleAdminUser)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}

	UpdateAdminUser(ctx)

	var response adminUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, target.Id).Error)
	require.Equal(t, target.DisplayName, reloaded.DisplayName)
}

func TestDeleteAdminUserForAgentDeletesManagedUser(t *testing.T) {
	db := setupAdminUserTestDB(t)
	agent := seedManagedUser(t, db, "agent_delete_operator", model.UserTypeAgent, common.RoleAdminUser, 0, 0)
	target := seedManagedUser(t, db, "managed_delete_target", model.UserTypeEndUser, common.RoleCommonUser, 120, agent.Id)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   target.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceUserManagement, Action: service.ActionDelete},
	)

	ctx, recorder := newAdminUserJSONContext(t, http.MethodDelete, "/api/admin/users/"+strconv.Itoa(target.Id), nil, agent.Id, common.RoleAdminUser)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}

	DeleteAdminUser(ctx)

	var response adminUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var deleted model.User
	require.Error(t, db.First(&deleted, target.Id).Error)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ? AND target_type = ? AND target_id = ?", service.ResourceUserManagement, service.ActionDelete, "user", target.Id).First(&audit).Error)
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

func TestUpdateAdminUserForAgentQuotaDecreaseReturnsBalanceAndCreatesLedger(t *testing.T) {
	db := setupAdminUserTestDB(t)
	agent := seedManagedUser(t, db, "agent_quota_op", model.UserTypeAgent, common.RoleCommonUser, 200, 0)
	target := seedManagedUser(t, db, "managed_quota_tgt", model.UserTypeEndUser, common.RoleCommonUser, 140, agent.Id)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   target.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.AgentQuotaPolicy{
		AgentUserId:           agent.Id,
		AllowRechargeUser:     true,
		AllowReclaimQuota:     true,
		MaxSingleAdjustAmount: 0,
		Status:                model.CommonStatusEnabled,
		UpdatedAtTs:           common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: "user_management", Action: "update"},
		permissionGrant{Resource: "quota_management", Action: "adjust"},
	)

	ctx, recorder := newAdminUserJSONContext(t, http.MethodPut, "/api/admin/users/"+strconv.Itoa(target.Id), map[string]any{
		"username":     target.Username,
		"display_name": "after-quota-update",
		"password":     "",
		"group":        "vip",
		"remark":       "quota-adjusted",
		"email":        "",
		"quota":        80,
	}, agent.Id, common.RoleCommonUser)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}
	UpdateAdminUser(ctx)

	var response adminUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	agentAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, agent.Id)
	require.NoError(t, err)
	targetAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, target.Id)
	require.NoError(t, err)
	require.Equal(t, 260, agentAccount.Balance)
	require.Equal(t, 80, targetAccount.Balance)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, target.Id).Error)
	require.Equal(t, "after-quota-update", reloaded.DisplayName)
	require.Equal(t, "vip", reloaded.Group)
	require.Equal(t, "quota-adjusted", reloaded.Remark)
	require.Equal(t, 80, reloaded.Quota)

	var order model.QuotaTransferOrder
	require.NoError(t, db.Where("transfer_type = ?", model.TransferTypeAgentReclaim).First(&order).Error)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("transfer_order_id = ?", order.Id).Order("account_id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 2)
}

func TestUpdateAdminUserForAgentQuotaIncreaseConsumesAgentBalanceAndCreatesLedger(t *testing.T) {
	db := setupAdminUserTestDB(t)
	agent := seedManagedUser(t, db, "agent_quota_inc", model.UserTypeAgent, common.RoleCommonUser, 200, 0)
	target := seedManagedUser(t, db, "managed_quota_inc", model.UserTypeEndUser, common.RoleCommonUser, 140, agent.Id)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   target.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.AgentQuotaPolicy{
		AgentUserId:           agent.Id,
		AllowRechargeUser:     true,
		AllowReclaimQuota:     true,
		MaxSingleAdjustAmount: 0,
		Status:                model.CommonStatusEnabled,
		UpdatedAtTs:           common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: "user_management", Action: "update"},
		permissionGrant{Resource: "quota_management", Action: "adjust"},
	)

	ctx, recorder := newAdminUserJSONContext(t, http.MethodPut, "/api/admin/users/"+strconv.Itoa(target.Id), map[string]any{
		"username":     target.Username,
		"display_name": target.DisplayName,
		"password":     "",
		"group":        target.Group,
		"remark":       target.Remark,
		"email":        "",
		"quota":        210,
	}, agent.Id, common.RoleCommonUser)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(target.Id)}}
	UpdateAdminUser(ctx)

	var response adminUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	agentAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, agent.Id)
	require.NoError(t, err)
	targetAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, target.Id)
	require.NoError(t, err)
	require.Equal(t, 130, agentAccount.Balance)
	require.Equal(t, 210, targetAccount.Balance)

	var order model.QuotaTransferOrder
	require.NoError(t, db.Where("transfer_type = ?", model.TransferTypeAgentRecharge).First(&order).Error)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("transfer_order_id = ?", order.Id).Order("account_id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 2)
}
