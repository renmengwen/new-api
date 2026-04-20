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

func TestRefreshTextPriceDataForSettlementReturnsResolvedPriceDataWithoutMutatingRelayInfo(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"settlement-refresh-model":6}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"settlement-refresh-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"settlement-refresh-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"settlement-refresh-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"output_min": 0,
					"output_max": 100,
					"input_price": 1,
					"output_price": 1
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	originalGroupRatio := types.GroupRatioInfo{
		GroupRatio:        2,
		GroupSpecialRatio: 2,
		HasSpecialRatio:   true,
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "settlement-refresh-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
		PriceData: types.PriceData{
			BillingMode:     types.BillingModePerToken,
			ModelRatio:      6,
			CompletionRatio: 2.5,
			GroupRatioInfo:  originalGroupRatio,
		},
	}
	before := info.PriceData

	priceData, ok, err := RefreshTextPriceDataForSettlement(c, info, 20, 50)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 0.5, priceData.ModelRatio)
	require.Equal(t, 1.0, priceData.CompletionRatio)
	require.Equal(t, originalGroupRatio, priceData.GroupRatioInfo)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "output_tokens=50")
	require.Equal(t, before, info.PriceData)
}

func TestRefreshTextPriceDataForSettlementSkipsCurrentAdvancedMediaTaskConfig(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"settlement-media-task-model":6}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"settlement-media-task-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"settlement-media-task-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"settlement-media-task-model": {
			"rule_type": "media_task",
			"segments": [
				{
					"priority": 10,
					"unit_price": 8.8,
					"remark": "media task"
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "settlement-media-task-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
		PriceData: types.PriceData{
			BillingMode:     types.BillingModePerToken,
			ModelRatio:      3,
			CompletionRatio: 1.5,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}
	before := info.PriceData

	priceData, ok, err := RefreshTextPriceDataForSettlement(c, info, 20, 50)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, types.PriceData{}, priceData)
	require.Equal(t, before, info.PriceData)
}

func TestRefreshTextPriceDataForSettlementReusesExistingAdvancedTextSnapshotWhenCurrentRulesNoLongerMatch(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"stale-advanced-text-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"stale-advanced-text-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"output_min": 1000,
					"output_max": 2000,
					"input_price": 9,
					"output_price": 9
				}
			]
		}
	}`))

	inputPrice := 14.0
	toolUsageCount := 3
	freeQuota := 1000
	overageThreshold := 500
	existingPriceData := types.PriceData{
		BillingMode:      types.BillingModeAdvanced,
		AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
		ModelRatio:       7,
		CompletionRatio:  1,
		GroupRatioInfo: types.GroupRatioInfo{
			GroupRatio:        2,
			GroupSpecialRatio: 2,
			HasSpecialRatio:   true,
		},
		AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
			RuleType:      types.AdvancedRuleTypeTextSegment,
			MatchSummary:  "priority=20, output_tokens=20, tool_usage_type=google_search",
			BillingUnit:   types.AdvancedBillingUnitPer1000Calls,
			ToolUsageType: "google_search",
			PriceSnapshot: types.AdvancedRulePriceSnapshot{
				InputPrice: &inputPrice,
			},
			ThresholdSnapshot: types.AdvancedRuleThresholdSnapshot{
				FreeQuota:        &freeQuota,
				OverageThreshold: &overageThreshold,
			},
		},
		AdvancedPricingContext: &types.AdvancedPricingContextSnapshot{
			BillingUnit:      types.AdvancedBillingUnitPer1000Calls,
			ToolUsageType:    "google_search",
			ToolUsageCount:   &toolUsageCount,
			FreeQuota:        &freeQuota,
			OverageThreshold: &overageThreshold,
		},
	}

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "stale-advanced-text-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
		PriceData: existingPriceData,
	}

	priceData, ok, err := RefreshTextPriceDataForSettlement(c, info, 20, 50)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, existingPriceData, priceData)
	require.Equal(t, existingPriceData, info.PriceData)
}
