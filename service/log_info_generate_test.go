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

func TestGenerateTextOtherInfo_IncludesAdvancedBillingContextSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	imageCount := 2
	toolUsageCount := 1200
	freeQuota := 1000
	relayInfo := &relaycommon.RelayInfo{
		StartTime:         time.UnixMilli(1000),
		FirstResponseTime: time.UnixMilli(1500),
		ChannelMeta:       &relaycommon.ChannelMeta{},
		PriceData: types.PriceData{
			BillingMode:      types.BillingModeAdvanced,
			AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
			AdvancedPricingContext: &types.AdvancedPricingContextSnapshot{
				BillingUnit:    types.AdvancedBillingUnitPerImage,
				ImageSizeTier:  "2k",
				ImageCount:     &imageCount,
				ToolUsageType:  "google_search",
				ToolUsageCount: &toolUsageCount,
				FreeQuota:      &freeQuota,
			},
		},
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, 1.0, 1.0, 1.0, 0, 0, 0, 1.0)

	contextSnapshot, ok := other["advanced_pricing_context"].(*types.AdvancedPricingContextSnapshot)
	require.True(t, ok)
	assert.Equal(t, types.AdvancedBillingUnitPerImage, contextSnapshot.BillingUnit)
	assert.Equal(t, "2k", contextSnapshot.ImageSizeTier)
	require.NotNil(t, contextSnapshot.ImageCount)
	assert.Equal(t, 2, *contextSnapshot.ImageCount)
	assert.Equal(t, "google_search", contextSnapshot.ToolUsageType)
	require.NotNil(t, contextSnapshot.ToolUsageCount)
	assert.Equal(t, 1200, *contextSnapshot.ToolUsageCount)
	require.NotNil(t, contextSnapshot.FreeQuota)
	assert.Equal(t, 1000, *contextSnapshot.FreeQuota)

	otherJSON := common.MapToJsonStr(other)
	assert.Contains(t, otherJSON, "\"advanced_pricing_context\":")
	assert.Contains(t, otherJSON, "\"billing_unit\":\"per_image\"")
	assert.Contains(t, otherJSON, "\"image_size_tier\":\"2k\"")
	assert.Contains(t, otherJSON, "\"image_count\":2")
	assert.Contains(t, otherJSON, "\"tool_usage_type\":\"google_search\"")
	assert.Contains(t, otherJSON, "\"tool_usage_count\":1200")
	assert.Contains(t, otherJSON, "\"free_quota\":1000")
	assert.NotContains(t, otherJSON, "\"BillingUnit\"")
	assert.NotContains(t, otherJSON, "\"ToolUsageType\"")
}
