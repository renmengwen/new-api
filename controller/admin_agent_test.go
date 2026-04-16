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

type adminAgentAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type adminAgentPageResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    common.PageInfo `json:"data"`
}

func setupAdminAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:admin_agent?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.AgentProfile{},
		&model.AgentQuotaPolicy{},
		&model.QuotaAccount{},
		&model.QuotaLedger{},
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

func newAdminAgentContext(t *testing.T, method string, target string, body any) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestCreateAgentCreatesOpeningLedgerEntry(t *testing.T) {
	db := setupAdminAgentTestDB(t)
	previousNewUserQuota := common.QuotaForNewUser
	common.QuotaForNewUser = 72
	t.Cleanup(func() {
		common.QuotaForNewUser = previousNewUserQuota
	})

	ctx, recorder := newAdminAgentContext(t, http.MethodPost, "/api/admin/agents", map[string]any{
		"username":      "agent_created_1",
		"password":      "12345678",
		"agent_name":    "Agent Created",
		"company_name":  "Shenzhou",
		"contact_phone": "13800000000",
	})
	CreateAgent(ctx)

	var response adminAgentAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var user model.User
	require.NoError(t, db.Where("username = ?", "agent_created_1").First(&user).Error)
	require.Equal(t, model.UserTypeAgent, user.GetUserType())
	require.Equal(t, common.RoleCommonUser, user.Role)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, user.Id, account.OwnerId)
	require.Equal(t, common.QuotaForNewUser, account.Balance)

	ledgers := listQuotaLedgersForUser(t, db, user.Id)
	require.Len(t, ledgers, 1)
	require.Equal(t, "opening", ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, common.QuotaForNewUser, ledgers[0].Amount)
	require.Equal(t, 0, ledgers[0].BalanceBefore)
	require.Equal(t, common.QuotaForNewUser, ledgers[0].BalanceAfter)
	require.Equal(t, "agent_create", ledgers[0].SourceType)
	require.Equal(t, user.Id, ledgers[0].SourceId)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND target_type = ? AND target_id = ?", "agent", "user", user.Id).First(&audit).Error)
	require.Equal(t, "create", audit.ActionType)
}

func TestCreateAgentPersistsRequestedGroup(t *testing.T) {
	db := setupAdminAgentTestDB(t)

	ctx, recorder := newAdminAgentContext(t, http.MethodPost, "/api/admin/agents", map[string]any{
		"username":      "agent_group_1",
		"password":      "12345678",
		"agent_name":    "Agent Group",
		"company_name":  "Shenzhou",
		"contact_phone": "13800000000",
		"group":         "EZModel",
	})
	CreateAgent(ctx)

	var response adminAgentAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var user model.User
	require.NoError(t, db.Where("username = ?", "agent_group_1").First(&user).Error)
	require.Equal(t, "EZModel", user.Group)
}

func TestGetAgentsReturnsCreatedAgent(t *testing.T) {
	db := setupAdminAgentTestDB(t)

	user := model.User{
		Username:    "agent_list_1",
		Password:    "hashed-password",
		DisplayName: "Agent List",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.AgentProfile{
		UserId:      user.Id,
		AgentName:   "Agent List",
		CompanyName: "Shenzhou",
		Status:      model.CommonStatusEnabled,
	}).Error)

	ctx, recorder := newAdminAgentContext(t, http.MethodGet, "/api/admin/agents?p=1&page_size=10", nil)
	GetAgents(ctx)

	var response adminAgentPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
}

func TestGetAgentReturnsProfileAndQuotaSummary(t *testing.T) {
	db := setupAdminAgentTestDB(t)

	user := model.User{
		Username:    "agent_detail_1",
		Password:    "hashed-password",
		DisplayName: "Agent Detail",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.AgentProfile{
		UserId:       user.Id,
		AgentName:    "Agent Detail",
		CompanyName:  "Shenzhou",
		ContactPhone: "13800000000",
		Status:       model.CommonStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.QuotaAccount{
		OwnerType:   model.QuotaOwnerTypeUser,
		OwnerId:     user.Id,
		Balance:     5000,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminAgentContext(t, http.MethodGet, "/api/admin/agents/"+strconv.Itoa(user.Id), nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	GetAgent(ctx)

	var response adminAgentAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, "Agent Detail", response.Data["agent_name"])

	quotaSummary, ok := response.Data["quota_summary"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(5000), quotaSummary["balance"])
}

func TestUpdateAgentUpdatesProfileAndWritesAudit(t *testing.T) {
	db := setupAdminAgentTestDB(t)

	user := model.User{
		Username:    "agent_update_1",
		Password:    "hashed-password",
		DisplayName: "Agent Before",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.AgentProfile{
		UserId:       user.Id,
		AgentName:    "Before Agent",
		CompanyName:  "Old Co",
		ContactPhone: "13800000000",
		Remark:       "old remark",
		Status:       model.CommonStatusEnabled,
		CreatedAtTs:  common.GetTimestamp(),
		UpdatedAtTs:  common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminAgentContext(t, http.MethodPut, "/api/admin/agents/"+strconv.Itoa(user.Id), map[string]any{
		"display_name":  "Agent After",
		"agent_name":    "After Agent",
		"company_name":  "New Co",
		"contact_phone": "13900000000",
		"remark":        "new remark",
	})
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	UpdateAgent(ctx)

	var response adminAgentAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	require.Equal(t, "Agent After", reloaded.DisplayName)
	require.Equal(t, "13900000000", reloaded.Phone)

	var profile model.AgentProfile
	require.NoError(t, db.Where("user_id = ?", user.Id).First(&profile).Error)
	require.Equal(t, "After Agent", profile.AgentName)
	require.Equal(t, "New Co", profile.CompanyName)
	require.Equal(t, "new remark", profile.Remark)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ? AND target_id = ?", "agent", "update", user.Id).First(&audit).Error)
}

func TestUpdateAgentStatusDisablesAgent(t *testing.T) {
	db := setupAdminAgentTestDB(t)

	user := model.User{
		Username:    "agent_disable_1",
		Password:    "hashed-password",
		DisplayName: "Agent Disable",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.AgentProfile{
		UserId:      user.Id,
		AgentName:   "Agent Disable",
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminAgentContext(t, http.MethodPost, "/api/admin/agents/"+strconv.Itoa(user.Id)+"/disable", nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	DisableAgent(ctx)

	var response adminAgentAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	require.Equal(t, common.UserStatusDisabled, reloaded.Status)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ? AND target_type = ? AND target_id = ?", "agent", "disable", "user", user.Id).First(&audit).Error)
}

func TestUpdateAgentStatusRejectsNonAgentTarget(t *testing.T) {
	db := setupAdminAgentTestDB(t)

	user := model.User{
		Username:    "normal_user_disable_1",
		Password:    "hashed-password",
		DisplayName: "Normal User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	ctx, recorder := newAdminAgentContext(t, http.MethodPost, "/api/admin/agents/"+strconv.Itoa(user.Id)+"/disable", nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	DisableAgent(ctx)

	var response adminAgentAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	require.Equal(t, common.UserStatusEnabled, reloaded.Status)
}

func TestCreateAgentRequiresActionPermissionForAdmin(t *testing.T) {
	db := setupAdminAgentTestDB(t)
	operator := model.User{
		Username:    "agent_operator_no_grant",
		Password:    "hashed-password",
		DisplayName: "Agent Operator",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "agentdeny",
	}
	require.NoError(t, db.Create(&operator).Error)

	ctx, recorder := newAdminAgentContext(t, http.MethodPost, "/api/admin/agents", map[string]any{
		"username":   "agent_should_fail",
		"password":   "12345678",
		"agent_name": "No Permission",
	})
	ctx.Set("id", operator.Id)
	ctx.Set("role", operator.Role)
	CreateAgent(ctx)

	var response adminAgentAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
}
