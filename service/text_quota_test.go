package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestCalculateTextQuotaSummaryUnifiedForClaudeSemantic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 200,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         100,
			CachedCreationTokens: 50,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	}

	priceData := types.PriceData{
		ModelRatio:           1,
		CompletionRatio:      2,
		CacheRatio:           0.1,
		CacheCreationRatio:   1.25,
		CacheCreation5mRatio: 1.25,
		CacheCreation1hRatio: 2,
		GroupRatioInfo: types.GroupRatioInfo{
			GroupRatio: 1,
		},
	}

	chatRelayInfo := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-3-7-sonnet",
		PriceData:               priceData,
		StartTime:               time.Now(),
	}
	messageRelayInfo := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatClaude,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-3-7-sonnet",
		PriceData:               priceData,
		StartTime:               time.Now(),
	}

	chatSummary := calculateTextQuotaSummary(ctx, chatRelayInfo, usage)
	messageSummary := calculateTextQuotaSummary(ctx, messageRelayInfo, usage)

	require.Equal(t, messageSummary.Quota, chatSummary.Quota)
	require.Equal(t, messageSummary.CacheCreationTokens5m, chatSummary.CacheCreationTokens5m)
	require.Equal(t, messageSummary.CacheCreationTokens1h, chatSummary.CacheCreationTokens1h)
	require.True(t, chatSummary.IsClaudeUsageSemantic)
	require.Equal(t, 1487, chatSummary.Quota)
}

func TestCalculateTextQuotaSummaryUsesSplitClaudeCacheCreationRatios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-3-7-sonnet",
		PriceData: types.PriceData{
			ModelRatio:           1,
			CompletionRatio:      1,
			CacheRatio:           0,
			CacheCreationRatio:   1,
			CacheCreation5mRatio: 2,
			CacheCreation1hRatio: 3,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 0,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedCreationTokens: 10,
		},
		ClaudeCacheCreation5mTokens: 2,
		ClaudeCacheCreation1hTokens: 3,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// 100 + remaining(5)*1 + 2*2 + 3*3 = 118
	require.Equal(t, 118, summary.Quota)
}

func TestCalculateTextQuotaSummaryTruncatesHalfQuotaForClaudeTextSettlement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatClaude,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-sonnet-4-6",
		PriceData: types.PriceData{
			ModelRatio:           1.5,
			CompletionRatio:      5,
			CacheCreationRatio:   1.25,
			CacheCreation5mRatio: 1.25,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     3,
		CompletionTokens: 20,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedCreationTokens: 28872,
		},
		ClaudeCacheCreation5mTokens: 28872,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// (3 + 28872*1.25 + 20*5) * 1.5 = 54289.5, text settlement should truncate.
	require.Equal(t, 54289, summary.Quota)
}

func TestCalculateTextQuotaSummaryPreservesMinimumOneQuotaForUsePriceTextSettlement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "tiny-price-text-model",
		PriceData: types.PriceData{
			UsePrice:       true,
			ModelPrice:     0.0000015,
			GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens: 1,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// 0.0000015 * 500000 = 0.75 quota, price-based text settlement should still bill 1 quota.
	require.Equal(t, 1, summary.Quota)
}

func TestCalculateTextQuotaSummaryPreservesMinimumOneQuotaForAdvancedNonTokenTextSettlement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	liveDurationSecs := 1
	startTime := time.Unix(time.Now().Unix()-1, 0)
	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "tiny-advanced-text-model",
		PriceData: types.PriceData{
			BillingMode:      types.BillingModeAdvanced,
			AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
			GroupRatioInfo:   types.GroupRatioInfo{GroupRatio: 1},
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
				RuleType:    types.AdvancedRuleTypeTextSegment,
				BillingUnit: types.AdvancedBillingUnitPerSecond,
				PriceSnapshot: types.AdvancedRulePriceSnapshot{
					InputPrice: common.GetPointer(0.0000015),
				},
			},
			AdvancedPricingContext: &types.AdvancedPricingContextSnapshot{
				BillingUnit:      types.AdvancedBillingUnitPerSecond,
				LiveDurationSecs: &liveDurationSecs,
			},
		},
		StartTime: startTime,
	}

	usage := &dto.Usage{
		PromptTokens: 1,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// 1 second * 0.0000015 * 500000 = 0.75 quota, advanced non-token text settlement should still bill 1 quota.
	require.Equal(t, 1, summary.Quota)
}

func TestCalculateTextQuotaSummaryUsesAnthropicUsageSemanticFromUpstreamUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		RelayFormat:     types.RelayFormatOpenAI,
		OriginModelName: "claude-3-7-sonnet",
		PriceData: types.PriceData{
			ModelRatio:           1,
			CompletionRatio:      2,
			CacheRatio:           0.1,
			CacheCreationRatio:   1.25,
			CacheCreation5mRatio: 1.25,
			CacheCreation1hRatio: 2,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 200,
		UsageSemantic:    "anthropic",
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         100,
			CachedCreationTokens: 50,
		},
		ClaudeCacheCreation5mTokens: 10,
		ClaudeCacheCreation1hTokens: 20,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.True(t, summary.IsClaudeUsageSemantic)
	require.Equal(t, "anthropic", summary.UsageSemantic)
	require.Equal(t, 1487, summary.Quota)
}

func TestCacheWriteTokensTotal(t *testing.T) {
	t.Run("split cache creation", func(t *testing.T) {
		summary := textQuotaSummary{
			CacheCreationTokens:   50,
			CacheCreationTokens5m: 10,
			CacheCreationTokens1h: 20,
		}
		require.Equal(t, 50, cacheWriteTokensTotal(summary))
	})

	t.Run("legacy cache creation", func(t *testing.T) {
		summary := textQuotaSummary{CacheCreationTokens: 50}
		require.Equal(t, 50, cacheWriteTokensTotal(summary))
	})

	t.Run("split cache creation without aggregate remainder", func(t *testing.T) {
		summary := textQuotaSummary{
			CacheCreationTokens5m: 10,
			CacheCreationTokens1h: 20,
		}
		require.Equal(t, 30, cacheWriteTokensTotal(summary))
	})
}

func TestCalculateTextQuotaSummaryHandlesLegacyClaudeDerivedOpenAIUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		RelayFormat:     types.RelayFormatOpenAI,
		OriginModelName: "claude-3-7-sonnet",
		PriceData: types.PriceData{
			ModelRatio:           1,
			CompletionRatio:      5,
			CacheRatio:           0.1,
			CacheCreationRatio:   1.25,
			CacheCreation5mRatio: 1.25,
			CacheCreation1hRatio: 2,
			GroupRatioInfo:       types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     62,
		CompletionTokens: 95,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 3544,
		},
		ClaudeCacheCreation5mTokens: 586,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// 62 + 3544*0.1 + 586*1.25 + 95*5 = 1623.9 => 1623
	require.Equal(t, 1623, summary.Quota)
}

func TestCalculateTextQuotaSummarySeparatesOpenRouterCacheReadFromPromptBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "openai/gpt-4.1",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenRouter,
		},
		PriceData: types.PriceData{
			ModelRatio:         1,
			CompletionRatio:    1,
			CacheRatio:         0.1,
			CacheCreationRatio: 1.25,
			GroupRatioInfo:     types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     2604,
		CompletionTokens: 383,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 2432,
		},
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// OpenRouter OpenAI-format display keeps prompt_tokens as total input,
	// but billing still separates normal input from cache read tokens.
	// quota = (2604 - 2432) + 2432*0.1 + 383 = 798.2 => 798
	require.Equal(t, 2604, summary.PromptTokens)
	require.Equal(t, 798, summary.Quota)
}

func TestCalculateTextQuotaSummarySeparatesOpenRouterCacheCreationFromPromptBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "openai/gpt-4.1",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenRouter,
		},
		PriceData: types.PriceData{
			ModelRatio:         1,
			CompletionRatio:    1,
			CacheCreationRatio: 1.25,
			GroupRatioInfo:     types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     2604,
		CompletionTokens: 383,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedCreationTokens: 100,
		},
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// prompt_tokens is still logged as total input, but cache creation is billed separately.
	// quota = (2604 - 100) + 100*1.25 + 383 = 3012
	require.Equal(t, 2604, summary.PromptTokens)
	require.Equal(t, 3012, summary.Quota)
}

func TestCalculateTextQuotaSummaryKeepsPrePRClaudeOpenRouterBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	relayInfo := &relaycommon.RelayInfo{
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "anthropic/claude-3.7-sonnet",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenRouter,
		},
		PriceData: types.PriceData{
			ModelRatio:         1,
			CompletionRatio:    1,
			CacheRatio:         0.1,
			CacheCreationRatio: 1.25,
			GroupRatioInfo:     types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     2604,
		CompletionTokens: 383,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 2432,
		},
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	// Pre-PR PostClaudeConsumeQuota behavior for OpenRouter:
	// prompt = 2604 - 2432 = 172
	// quota = 172 + 2432*0.1 + 383 = 798.2 => 798
	require.True(t, summary.IsClaudeUsageSemantic)
	require.Equal(t, 172, summary.PromptTokens)
	require.Equal(t, 798, summary.Quota)
}

func TestCalculateTextQuotaSummaryRebuildsAdvancedTextPricingFromActualOutputTokens(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"advanced-output-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"advanced-output-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"output_min": 0,
					"output_max": 100,
					"input_price": 1,
					"output_price": 1
				},
				{
					"priority": 20,
					"output_min": 101,
					"output_max": 1000,
					"input_price": 4,
					"output_price": 4
				}
			]
		}
	}`))

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "advanced-output-model",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
		PriceData: types.PriceData{
			BillingMode:          types.BillingModeAdvanced,
			ModelRatio:           2,
			CompletionRatio:      1,
			AdvancedRuleType:     types.AdvancedRuleTypeTextSegment,
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{MatchSummary: "output_tokens=1000"},
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     20,
		CompletionTokens: 50,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 35, summary.Quota)
	require.Equal(t, 0.5, summary.ModelRatio)
	require.Equal(t, 1.0, summary.CompletionRatio)
	require.NotNil(t, relayInfo.PriceData.AdvancedRuleSnapshot)
	require.Contains(t, relayInfo.PriceData.AdvancedRuleSnapshot.MatchSummary, "output_tokens=50")
}

func TestCalculateTextQuotaSummarySettlementCanNewlyResolveAdvancedTextPricing(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"advanced-newly-resolved-model":6}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"advanced-newly-resolved-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"advanced-newly-resolved-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"advanced-newly-resolved-model": {
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

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "advanced-newly-resolved-model",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
		PriceData: types.PriceData{
			BillingMode:     types.BillingModePerToken,
			ModelRatio:      6,
			CompletionRatio: 2.5,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     20,
		CompletionTokens: 50,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 35, summary.Quota)
	require.Equal(t, types.BillingModeAdvanced, relayInfo.PriceData.BillingMode)
	require.Equal(t, types.AdvancedRuleTypeTextSegment, relayInfo.PriceData.AdvancedRuleType)
	require.Equal(t, 0.5, summary.ModelRatio)
	require.Equal(t, 1.0, summary.CompletionRatio)
	require.NotNil(t, relayInfo.PriceData.AdvancedRuleSnapshot)
	require.Contains(t, relayInfo.PriceData.AdvancedRuleSnapshot.MatchSummary, "output_tokens=50")
}

func TestCalculateTextQuotaSummarySettlementFallsBackWhenAdvancedTextRuleNoLongerMatches(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"advanced-fallback-model":6}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"advanced-fallback-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"advanced-fallback-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"advanced-fallback-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"output_min": 101,
					"output_max": 1000,
					"input_price": 4,
					"output_price": 4
				}
			]
		}
	}`))

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "advanced-fallback-model",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
		PriceData: types.PriceData{
			BillingMode:          types.BillingModeAdvanced,
			ModelRatio:           2,
			CompletionRatio:      1,
			AdvancedRuleType:     types.AdvancedRuleTypeTextSegment,
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{MatchSummary: "output_tokens=500"},
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     20,
		CompletionTokens: 50,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 870, summary.Quota)
	require.Equal(t, types.BillingModePerToken, relayInfo.PriceData.BillingMode)
	require.Equal(t, 6.0, summary.ModelRatio)
	require.Equal(t, 2.5, summary.CompletionRatio)
	require.Nil(t, relayInfo.PriceData.AdvancedRuleSnapshot)
}

func TestCalculateTextQuotaSummarySettlementPreservesOriginalRequestGroupRatioInfo(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"advanced-group-ratio-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"advanced-group-ratio-model": {
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

	originalGroupRatio := types.GroupRatioInfo{
		GroupRatio:        2,
		GroupSpecialRatio: 2,
		HasSpecialRatio:   true,
	}
	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "advanced-group-ratio-model",
		UsingGroup:      "vip",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
		PriceData: types.PriceData{
			BillingMode:          types.BillingModeAdvanced,
			ModelRatio:           2,
			CompletionRatio:      1,
			AdvancedRuleType:     types.AdvancedRuleTypeTextSegment,
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{MatchSummary: "output_tokens=500"},
			GroupRatioInfo:       originalGroupRatio,
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     20,
		CompletionTokens: 50,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 70, summary.Quota)
	require.Equal(t, originalGroupRatio, relayInfo.PriceData.GroupRatioInfo)
	require.Equal(t, originalGroupRatio.GroupRatio, summary.GroupRatio)
}

func TestCalculateTextQuotaSummarySettlementFallsBackWhenAdvancedConfigDriftsToLegacy(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"advanced-drift-model":6}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"advanced-drift-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{}`))

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "advanced-drift-model",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
		PriceData: types.PriceData{
			BillingMode:          types.BillingModeAdvanced,
			ModelRatio:           2,
			CompletionRatio:      1,
			AdvancedRuleType:     types.AdvancedRuleTypeTextSegment,
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{MatchSummary: "output_tokens=500"},
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     20,
		CompletionTokens: 50,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 870, summary.Quota)
	require.Equal(t, types.BillingModePerToken, relayInfo.PriceData.BillingMode)
	require.Equal(t, 6.0, summary.ModelRatio)
	require.Equal(t, 2.5, summary.CompletionRatio)
	require.Nil(t, relayInfo.PriceData.AdvancedRuleSnapshot)
}

func TestCalculateTextQuotaSummarySettlementDoesNotRefreshForCurrentAdvancedMediaTaskConfig(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"advanced-media-task-model":6}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"advanced-media-task-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"advanced-media-task-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"advanced-media-task-model": {
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

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "advanced-media-task-model",
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
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     20,
		CompletionTokens: 50,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 285, summary.Quota)
	require.Equal(t, types.BillingModePerToken, relayInfo.PriceData.BillingMode)
	require.Equal(t, 3.0, summary.ModelRatio)
	require.Equal(t, 1.5, summary.CompletionRatio)
}

func TestCalculateTextQuotaSummarySettlementUsesLiveDurationForAdvancedPerSecond(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"gemini-3.1-flash-live-preview":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"gemini-3.1-flash-live-preview": {
			"rule_type": "text_segment",
			"billing_unit": "per_second",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000000,
					"input_modality": "audio",
					"output_modality": "audio",
					"input_price": 0.5,
					"output_price": 1.5
				}
			]
		}
	}`))

	message := dto.Message{Role: "user"}
	message.SetMediaContent([]dto.MediaContent{
		{
			Type: dto.ContentTypeInputAudio,
			InputAudio: &dto.MessageInputAudio{
				Data:   "UklGRg==",
				Format: "wav",
			},
		},
	})

	request := &dto.GeneralOpenAIRequest{
		Model:    "gemini-3.1-flash-live-preview",
		Messages: []dto.Message{message},
		ExtraBody: []byte(`{
			"google": {
				"generation_config": {
					"response_modalities": ["AUDIO"]
				}
			}
		}`),
	}

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3.1-flash-live-preview",
		Request:         request,
		PriceData: types.PriceData{
			BillingMode:      types.BillingModeAdvanced,
			AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
			GroupRatioInfo:   types.GroupRatioInfo{GroupRatio: 1},
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
				RuleType:    types.AdvancedRuleTypeTextSegment,
				BillingUnit: types.AdvancedBillingUnitPerSecond,
				PriceSnapshot: types.AdvancedRulePriceSnapshot{
					InputPrice:  common.GetPointer(0.5),
					OutputPrice: common.GetPointer(1.5),
				},
			},
			AdvancedPricingContext: &types.AdvancedPricingContextSnapshot{
				BillingUnit: types.AdvancedBillingUnitPerSecond,
			},
		},
		StartTime: time.Now().Add(-3 * time.Second),
	}

	usage := &dto.Usage{
		PromptTokens:     10,
		CompletionTokens: 0,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 3000000, summary.Quota)
	require.Equal(t, types.BillingModeAdvanced, relayInfo.PriceData.BillingMode)
	require.Equal(t, types.AdvancedBillingUnitPerSecond, relayInfo.PriceData.AdvancedRuleSnapshot.BillingUnit)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext.LiveDurationSecs)
	require.GreaterOrEqual(t, *relayInfo.PriceData.AdvancedPricingContext.LiveDurationSecs, 3)
}

func TestCalculateTextQuotaSummarySettlementUsesImageCountForAdvancedPerImage(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"gemini-3.1-flash-image-preview":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"gemini-3.1-flash-image-preview": {
			"rule_type": "text_segment",
			"billing_unit": "per_image",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000000,
					"output_modality": "image",
					"input_price": 0.5,
					"output_price": 3
				}
			]
		}
	}`))

	imageCount := 2
	request := &dto.GeneralOpenAIRequest{
		Model:  "gemini-3.1-flash-image-preview",
		Prompt: "Generate a stylized skyline illustration",
		Size:   "1024x1024",
		N:      &imageCount,
	}

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3.1-flash-image-preview",
		Request:         request,
		RequestURLPath:  "/v1/images/generations",
		PriceData: types.PriceData{
			BillingMode:      types.BillingModeAdvanced,
			AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
			GroupRatioInfo:   types.GroupRatioInfo{GroupRatio: 1},
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
				RuleType:    types.AdvancedRuleTypeTextSegment,
				BillingUnit: types.AdvancedBillingUnitPerImage,
				PriceSnapshot: types.AdvancedRulePriceSnapshot{
					InputPrice:  common.GetPointer(0.5),
					OutputPrice: common.GetPointer(3.0),
				},
			},
			AdvancedPricingContext: &types.AdvancedPricingContextSnapshot{
				BillingUnit: types.AdvancedBillingUnitPerImage,
				ImageCount:  &imageCount,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     516,
		CompletionTokens: 0,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 3500000, summary.Quota)
	require.Equal(t, types.BillingModeAdvanced, relayInfo.PriceData.BillingMode)
	require.Equal(t, types.AdvancedBillingUnitPerImage, relayInfo.PriceData.AdvancedRuleSnapshot.BillingUnit)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext.ImageCount)
	require.Equal(t, 2, *relayInfo.PriceData.AdvancedPricingContext.ImageCount)
}

func TestCalculateTextQuotaSummarySettlementUsesImageCountForChatBasedAdvancedPerImage(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"gemini-3.1-flash-image-chat-preview":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"gemini-3.1-flash-image-chat-preview": {
			"rule_type": "text_segment",
			"billing_unit": "per_image",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000000,
					"output_modality": "image",
					"image_size_tier": "2k",
					"input_price": 0.5,
					"output_price": 3
				}
			]
		}
	}`))

	imageCount := 2
	request := &dto.GeneralOpenAIRequest{
		Model:      "gemini-3.1-flash-image-chat-preview",
		Prompt:     "Generate two stylized skyline illustrations",
		Size:       "2048x2048",
		N:          &imageCount,
		Modalities: []byte(`["image"]`),
	}

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3.1-flash-image-chat-preview",
		Request:         request,
		RequestURLPath:  "/v1/chat/completions",
		PriceData: types.PriceData{
			BillingMode:      types.BillingModeAdvanced,
			AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
			GroupRatioInfo:   types.GroupRatioInfo{GroupRatio: 1},
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
				RuleType:    types.AdvancedRuleTypeTextSegment,
				BillingUnit: types.AdvancedBillingUnitPerImage,
			},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     516,
		CompletionTokens: 0,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 3500000, summary.Quota)
	require.Equal(t, types.BillingModeAdvanced, relayInfo.PriceData.BillingMode)
	require.NotNil(t, relayInfo.PriceData.AdvancedRuleSnapshot)
	require.Equal(t, types.AdvancedBillingUnitPerImage, relayInfo.PriceData.AdvancedRuleSnapshot.BillingUnit)
	require.Equal(t, "2k", relayInfo.PriceData.AdvancedRuleSnapshot.ImageSizeTier)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext)
	require.Equal(t, "2k", relayInfo.PriceData.AdvancedPricingContext.ImageSizeTier)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext.ImageCount)
	require.Equal(t, 2, *relayInfo.PriceData.AdvancedPricingContext.ImageCount)
}

func TestCalculateAdvancedNonTokenQuotaUsesFreeQuotaAndThresholdForPer1000Calls(t *testing.T) {
	priceData := types.PriceData{
		BillingMode:      types.BillingModeAdvanced,
		AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
		AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
			RuleType:    types.AdvancedRuleTypeTextSegment,
			BillingUnit: types.AdvancedBillingUnitPer1000Calls,
			PriceSnapshot: types.AdvancedRulePriceSnapshot{
				InputPrice: common.GetPointer(14.0),
			},
		},
		AdvancedPricingContext: &types.AdvancedPricingContextSnapshot{
			BillingUnit:      types.AdvancedBillingUnitPer1000Calls,
			ToolUsageType:    "web_search",
			ToolUsageCount:   common.GetPointer(2500),
			FreeQuota:        common.GetPointer(500),
			OverageThreshold: common.GetPointer(2000),
		},
	}

	quota, used := calculateAdvancedNonTokenQuota(textQuotaSummary{}, priceData, decimal.NewFromInt(1), decimal.NewFromFloat(common.QuotaPerUnit))

	require.True(t, used)
	require.True(t, quota.Equal(decimal.NewFromInt(14).Mul(decimal.NewFromFloat(common.QuotaPerUnit))))
}

func TestCalculateTextQuotaSummaryToolOverageUsesDedicatedPriceBeyondFreeQuota(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"responses-grounding-overage-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"responses-grounding-overage-model": {
			"rule_type": "text_segment",
			"billing_unit": "per_1000_calls",
			"segments": [
				{
					"priority": 10,
					"input_price": 2
				},
				{
					"priority": 20,
					"tool_usage_type": "grounding",
					"tool_usage_count": 1,
					"free_quota": 5000,
					"overage_threshold": 1000,
					"input_price": 2,
					"tool_overage_price": 14
				}
			]
		}
	}`))

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "responses-grounding-overage-model",
		Request: &dto.OpenAIResponsesRequest{
			Model: "responses-grounding-overage-model",
		},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolWebSearchPreview: {
					ToolName:          dto.BuildInToolWebSearchPreview,
					CallCount:         6200,
					SearchContextSize: "medium",
				},
			},
		},
		PriceData: types.PriceData{
			BillingMode:     types.BillingModePerToken,
			ModelRatio:      6,
			CompletionRatio: 2.5,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     10,
		CompletionTokens: 0,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	expectedQuota := decimal.NewFromFloat(16.8).Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	require.Equal(t, int(expectedQuota.IntPart()), summary.Quota)
	require.Equal(t, types.BillingModeAdvanced, relayInfo.PriceData.BillingMode)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext.FreeQuota)
	require.Equal(t, 5000, *relayInfo.PriceData.AdvancedPricingContext.FreeQuota)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext.OverageThreshold)
	require.Equal(t, 1000, *relayInfo.PriceData.AdvancedPricingContext.OverageThreshold)

	contextJSON, err := common.Marshal(relayInfo.PriceData.AdvancedPricingContext)
	require.NoError(t, err)
	require.Contains(t, string(contextJSON), `"tool_overage_price":14`)
}

func TestCalculateTextQuotaSummarySettlementSkipsLegacyWebSearchQuotaWhenAdvancedPer1000CallsMatchesGroundingAlias(t *testing.T) {
	restoreTextQuotaRatioSettings(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("chat_completion_web_search_context_size", "medium")

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"grounding-search-preview":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"grounding-search-preview": {
			"rule_type": "text_segment",
			"billing_unit": "per_1000_calls",
			"segments": [
				{
					"priority": 10,
					"input_price": 2
				},
				{
					"priority": 20,
					"tool_usage_type": "grounding",
					"tool_usage_count": 1,
					"overage_threshold": 1,
					"input_price": 14
				}
			]
		}
	}`))

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "grounding-search-preview",
		Request: &dto.GeneralOpenAIRequest{
			Model: "grounding-search-preview",
		},
		PriceData: types.PriceData{
			BillingMode:     types.BillingModePerToken,
			ModelRatio:      6,
			CompletionRatio: 2.5,
			GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		},
		StartTime: time.Now(),
	}

	usage := &dto.Usage{
		PromptTokens:     10,
		CompletionTokens: 0,
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Equal(t, 1, summary.WebSearchCallCount)
	require.Greater(t, summary.WebSearchPrice, 0.0)
	require.Equal(t, int(14*common.QuotaPerUnit), summary.Quota)
	require.Equal(t, types.BillingModeAdvanced, relayInfo.PriceData.BillingMode)
	require.NotNil(t, relayInfo.PriceData.AdvancedRuleSnapshot)
	require.Equal(t, types.AdvancedBillingUnitPer1000Calls, relayInfo.PriceData.AdvancedRuleSnapshot.BillingUnit)
	require.Equal(t, "google_search", relayInfo.PriceData.AdvancedRuleSnapshot.ToolUsageType)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext)
	require.Equal(t, "google_search", relayInfo.PriceData.AdvancedPricingContext.ToolUsageType)
	require.NotNil(t, relayInfo.PriceData.AdvancedPricingContext.ToolUsageCount)
	require.Equal(t, 1, *relayInfo.PriceData.AdvancedPricingContext.ToolUsageCount)
}

func TestShouldSkipLegacyWebSearchQuotaForAdvancedBillingTreatsSearchAliasesAsEquivalent(t *testing.T) {
	aliases := []string{"grounding", "web_search", "google_search"}

	for _, alias := range aliases {
		priceData := types.PriceData{
			BillingMode:      types.BillingModeAdvanced,
			AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
				RuleType:    types.AdvancedRuleTypeTextSegment,
				BillingUnit: types.AdvancedBillingUnitPer1000Calls,
			},
			AdvancedPricingContext: &types.AdvancedPricingContextSnapshot{
				BillingUnit:   types.AdvancedBillingUnitPer1000Calls,
				ToolUsageType: alias,
			},
		}

		require.True(t, shouldSkipLegacyWebSearchQuotaForAdvancedBilling(priceData), alias)
	}
}

func TestShouldSkipLegacyWebSearchQuotaForAdvancedBillingUsesSnapshotWhenContextMissing(t *testing.T) {
	priceData := types.PriceData{
		BillingMode:      types.BillingModeAdvanced,
		AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
		AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
			RuleType:      types.AdvancedRuleTypeTextSegment,
			BillingUnit:   types.AdvancedBillingUnitPer1000Calls,
			ToolUsageType: "google_search",
		},
	}

	require.True(t, shouldSkipLegacyWebSearchQuotaForAdvancedBilling(priceData))
}

func restoreTextQuotaRatioSettings(t *testing.T) {
	t.Helper()

	advancedModeJSON := ratio_setting.AdvancedPricingMode2JSONString()
	advancedRulesJSON := ratio_setting.AdvancedPricingRules2JSONString()
	groupRatioJSON := ratio_setting.GroupRatio2JSONString()
	groupGroupRatioJSON := ratio_setting.GroupGroupRatio2JSONString()
	modelRatioJSON := ratio_setting.ModelRatio2JSONString()
	completionRatioJSON := ratio_setting.CompletionRatio2JSONString()
	cacheRatioJSON := ratio_setting.CacheRatio2JSONString()
	createCacheRatioJSON := ratio_setting.CreateCacheRatio2JSONString()

	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(advancedModeJSON))
		require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(advancedRulesJSON))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(groupRatioJSON))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(groupGroupRatioJSON))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(modelRatioJSON))
		require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(completionRatioJSON))
		require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(cacheRatioJSON))
		require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(createCacheRatioJSON))
	})
}
