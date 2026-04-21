package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestGetChannelAffinityUsageCacheStatsDeniesAgent(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)

	agent := model.User{
		Id:          8201,
		Username:    "affinity_agent",
		Password:    "hashed-password",
		DisplayName: "Affinity Agent",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
		AffCode:     "affinity_agent_aff",
	}
	require.NoError(t, db.Create(&agent).Error)

	ctx, recorder := newAdminAnalyticsContext(
		t,
		http.MethodGet,
		"/api/log/channel_affinity_usage_cache?rule_name=test_rule&key_fp=test_fp",
		nil,
		agent.Id,
		agent.Role,
	)
	GetChannelAffinityUsageCacheStats(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "permission denied", response.Message)
}
