package helper

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperReturnsAdvancedTextSegmentPriceDataWhenRuleMatches(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"advanced-text-model":7}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"advanced-text-model":4}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"advanced-text-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"advanced-text-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000,
					"output_min": 0,
					"output_max": 4096,
					"service_tier": "priority",
					"input_price": 4,
					"output_price": 12
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "advanced-text-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "priority",
		},
	}
	meta := &types.TokenCountMeta{MaxTokens: 2048}

	priceData, err := ModelPriceHelper(c, info, 256, meta)
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedRuleTypeTextSegment, priceData.AdvancedRuleType)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, types.AdvancedRuleTypeTextSegment, priceData.AdvancedRuleSnapshot.RuleType)
	require.Equal(t, 2.0, priceData.ModelRatio)
	require.Equal(t, 3.0, priceData.CompletionRatio)
}

func TestModelPriceHelperKeepsFixedPerTokenLogicWhenModeIsPerToken(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"fixed-token-model":6}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"fixed-token-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"fixed-token-model":"per_token"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"fixed-token-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000,
					"input_price": 4,
					"output_price": 8
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "fixed-token-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{MaxTokens: 512})
	require.NoError(t, err)

	require.Equal(t, types.BillingModePerToken, priceData.BillingMode)
	require.Equal(t, 6.0, priceData.ModelRatio)
	require.Equal(t, 2.5, priceData.CompletionRatio)
	require.Nil(t, priceData.AdvancedRuleSnapshot)
}

func TestModelPriceHelperFallsBackToFixedLogicWhenAdvancedRuleDoesNotMatch(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"fallback-model":9}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"fallback-model":5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"fallback-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"fallback-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 100,
					"service_tier": "priority",
					"input_price": 4,
					"output_price": 10
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "fallback-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
	}

	priceData, err := ModelPriceHelper(c, info, 256, &types.TokenCountMeta{MaxTokens: 2048})
	require.NoError(t, err)

	require.Equal(t, types.BillingModePerToken, priceData.BillingMode)
	require.Equal(t, 9.0, priceData.ModelRatio)
	require.Equal(t, 5.0, priceData.CompletionRatio)
	require.Nil(t, priceData.AdvancedRuleSnapshot)
}

func restoreRatioSettings(t *testing.T) {
	t.Helper()

	modelRatioJSON := ratio_setting.ModelRatio2JSONString()
	completionRatioJSON := ratio_setting.CompletionRatio2JSONString()
	modelPriceJSON := ratio_setting.ModelPrice2JSONString()
	cacheRatioJSON := ratio_setting.CacheRatio2JSONString()
	createCacheRatioJSON := ratio_setting.CreateCacheRatio2JSONString()
	groupRatioJSON := ratio_setting.GroupRatio2JSONString()
	groupGroupRatioJSON := ratio_setting.GroupGroupRatio2JSONString()
	advancedModeJSON := ratio_setting.AdvancedPricingMode2JSONString()
	advancedRulesJSON := ratio_setting.AdvancedPricingRules2JSONString()

	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(modelRatioJSON))
		require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(completionRatioJSON))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(modelPriceJSON))
		require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(cacheRatioJSON))
		require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(createCacheRatioJSON))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(groupRatioJSON))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(groupGroupRatioJSON))
		require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(advancedModeJSON))
		require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(advancedRulesJSON))
	})
}
