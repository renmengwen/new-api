package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTextOtherInfo_IncludesAdvancedBillingSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	snapshot := &types.AdvancedRuleSnapshot{
		RuleType:     types.AdvancedRuleTypeTextSegment,
		MatchSummary: "input_tokens<=1000",
		ConditionTags: []string{
			"input",
		},
	}
	relayInfo := &relaycommon.RelayInfo{
		StartTime:         time.UnixMilli(1000),
		FirstResponseTime: time.UnixMilli(1500),
		ChannelMeta:       &relaycommon.ChannelMeta{},
		PriceData: types.PriceData{
			BillingMode:          types.BillingModeAdvanced,
			AdvancedRuleType:     types.AdvancedRuleTypeTextSegment,
			AdvancedRuleSnapshot: snapshot,
		},
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, 2.0, 1.0, 1.5, 0, 0, 0, 1.0)

	assert.Equal(t, string(types.BillingModeAdvanced), other["billing_mode"])
	assert.Equal(t, string(types.AdvancedRuleTypeTextSegment), other["advanced_rule_type"])
	rule, ok := other["advanced_rule"].(*types.AdvancedRuleSnapshot)
	require.True(t, ok)
	assert.Same(t, snapshot, rule)

	otherJSON := common.MapToJsonStr(other)
	assert.Contains(t, otherJSON, "\"rule_type\":\"text_segment\"")
	assert.Contains(t, otherJSON, "\"match_summary\":")
	assert.Contains(t, otherJSON, "\"condition_tags\":[\"input\"]")
	assert.NotContains(t, otherJSON, "\"RuleType\"")
	assert.NotContains(t, otherJSON, "\"MatchSummary\"")
}
