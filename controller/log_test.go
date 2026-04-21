package controller

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type usageLogPageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Page     int         `json:"page"`
		PageSize int         `json:"page_size"`
		Total    int         `json:"total"`
		Items    []model.Log `json:"items"`
	} `json:"data"`
}

type usageLogStatResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Quota int `json:"quota"`
		Rpm   int `json:"rpm"`
		Tpm   int `json:"tpm"`
	} `json:"data"`
}

type agentUsageLogFixture struct {
	Agent            model.User
	Owned            model.User
	Other            model.User
	OwnedToken       model.Token
	SelfConsume      model.Log
	OwnedConsume     model.Log
	BorrowedTokenLog model.Log
	OtherConsume     model.Log
	StartAt          int64
	EndAt            int64
}

func TestGetAllLogsScopesAgentToSelfAndManagedUsers(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAgentUsageLogFixture(t, db)

	target := "/api/log/?p=1&page_size=10&type=" + strconv.Itoa(model.LogTypeConsume) +
		"&start_timestamp=" + strconv.FormatInt(fixture.StartAt, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.EndAt, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Agent.Id, fixture.Agent.Role)
	GetAllLogs(ctx)

	var response usageLogPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 3, response.Data.Total)
	require.Len(t, response.Data.Items, 3)
	require.Equal(t, fixture.Other.Username, response.Data.Items[0].Username)
	require.Equal(t, fixture.BorrowedTokenLog.Content, response.Data.Items[0].Content)
	require.Equal(t, fixture.Owned.Username, response.Data.Items[1].Username)
	require.Equal(t, fixture.Agent.Username, response.Data.Items[2].Username)
	for _, item := range response.Data.Items {
		require.NotEqual(t, fixture.OtherConsume.Content, item.Content)
	}
}

func TestGetLogsStatScopesAgentToSelfAndManagedUsers(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAgentUsageLogFixture(t, db)

	target := "/api/log/stat?type=" + strconv.Itoa(model.LogTypeConsume) +
		"&start_timestamp=" + strconv.FormatInt(fixture.StartAt, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.EndAt, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Agent.Id, fixture.Agent.Role)
	GetLogsStat(ctx)

	var response usageLogStatResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, fixture.SelfConsume.Quota+fixture.OwnedConsume.Quota+fixture.BorrowedTokenLog.Quota, response.Data.Quota)
	require.NotEqual(t, fixture.SelfConsume.Quota+fixture.OwnedConsume.Quota+fixture.BorrowedTokenLog.Quota+fixture.OtherConsume.Quota, response.Data.Quota)
}

func TestGetAllLogsRequiresLedgerReadPermissionForAgent(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAgentUsageLogFixture(t, db)
	grantPermissionActions(t, db, fixture.Agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceQuotaManagement, Action: service.ActionReadSummary},
	)

	target := "/api/log/?p=1&page_size=10&type=" + strconv.Itoa(model.LogTypeConsume) +
		"&start_timestamp=" + strconv.FormatInt(fixture.StartAt, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.EndAt, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Agent.Id, fixture.Agent.Role)
	GetAllLogs(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "permission denied", response.Message)
}

func TestGetLogsStatRequiresReadSummaryPermissionForAgent(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAgentUsageLogFixture(t, db)
	grantPermissionActions(t, db, fixture.Agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceQuotaManagement, Action: service.ActionLedgerRead},
	)

	target := "/api/log/stat?type=" + strconv.Itoa(model.LogTypeConsume) +
		"&start_timestamp=" + strconv.FormatInt(fixture.StartAt, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.EndAt, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Agent.Id, fixture.Agent.Role)
	GetLogsStat(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "permission denied", response.Message)
}

func seedAgentUsageLogFixture(t *testing.T, db *gorm.DB) agentUsageLogFixture {
	t.Helper()

	agent := model.User{
		Id:          8101,
		Username:    "usage_agent",
		Password:    "hashed-password",
		DisplayName: "Usage Agent",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
		AffCode:     "usage_agent_aff",
	}
	owned := model.User{
		Id:            8102,
		Username:      "usage_owned_user",
		Password:      "hashed-password",
		DisplayName:   "Usage Owned User",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		UserType:      model.UserTypeEndUser,
		ParentAgentId: agent.Id,
		Group:         "default",
		AffCode:       "usage_owned_aff",
	}
	other := model.User{
		Id:          8103,
		Username:    "usage_other_user",
		Password:    "hashed-password",
		DisplayName: "Usage Other User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		AffCode:     "usage_other_aff",
	}
	require.NoError(t, db.Create(&[]model.User{agent, owned, other}).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   owned.Id,
		BindSource:  "manual",
		BindAt:      1812000000,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: 1812000000,
	}).Error)
	ownedToken := model.Token{
		UserId:      owned.Id,
		Key:         "usage-token-owned-fixture-key",
		Status:      common.TokenStatusEnabled,
		Name:        "usage-owned-token",
		CreatedTime: 1812000000,
		Group:       "default",
	}
	require.NoError(t, db.Create(&ownedToken).Error)

	selfConsume := model.Log{
		UserId:           agent.Id,
		Username:         agent.Username,
		CreatedAt:        1812000001,
		Type:             model.LogTypeConsume,
		Content:          "agent consume",
		TokenName:        "agent-token",
		ModelName:        "agent-model",
		Quota:            40,
		PromptTokens:     10,
		CompletionTokens: 20,
		Group:            "default",
	}
	ownedConsume := model.Log{
		UserId:           owned.Id,
		Username:         owned.Username,
		CreatedAt:        1812000002,
		Type:             model.LogTypeConsume,
		Content:          "owned consume",
		TokenName:        "owned-token",
		ModelName:        "owned-model",
		Quota:            60,
		PromptTokens:     12,
		CompletionTokens: 18,
		Group:            "default",
	}
	otherConsume := model.Log{
		UserId:           other.Id,
		Username:         other.Username,
		CreatedAt:        1812000003,
		Type:             model.LogTypeConsume,
		Content:          "other consume",
		TokenName:        "other-token",
		ModelName:        "other-model",
		Quota:            90,
		PromptTokens:     8,
		CompletionTokens: 22,
		Group:            "default",
	}
	borrowedTokenLog := model.Log{
		UserId:           other.Id,
		Username:         other.Username,
		CreatedAt:        1812000005,
		Type:             model.LogTypeConsume,
		Content:          "borrowed managed token consume",
		TokenId:          ownedToken.Id,
		TokenName:        ownedToken.Name,
		ModelName:        "borrowed-model",
		Quota:            70,
		PromptTokens:     6,
		CompletionTokens: 14,
		Group:            "default",
	}
	ownedError := model.Log{
		UserId:           owned.Id,
		Username:         owned.Username,
		CreatedAt:        1812000004,
		Type:             model.LogTypeError,
		Content:          "owned error",
		TokenName:        "owned-token",
		ModelName:        "owned-model",
		PromptTokens:     1,
		CompletionTokens: 0,
		Group:            "default",
	}
	require.NoError(t, db.Create(&[]model.Log{selfConsume, ownedConsume, otherConsume, borrowedTokenLog, ownedError}).Error)

	return agentUsageLogFixture{
		Agent:            agent,
		Owned:            owned,
		Other:            other,
		OwnedToken:       ownedToken,
		SelfConsume:      selfConsume,
		OwnedConsume:     ownedConsume,
		BorrowedTokenLog: borrowedTokenLog,
		OtherConsume:     otherConsume,
		StartAt:          1812000000,
		EndAt:            1812000010,
	}
}
