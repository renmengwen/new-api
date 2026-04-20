package helper

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
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

func TestModelPriceHelperReturnsErrorWhenAdvancedRuleDoesNotMatch(t *testing.T) {
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

	_, err := ModelPriceHelper(c, info, 256, &types.TokenCountMeta{MaxTokens: 2048})
	require.Error(t, err)
	require.ErrorContains(t, err, "advanced pricing")
	require.ErrorContains(t, err, "fallback-model")
}

func TestModelPriceHelperUsesDefaultAdvancedTextSegmentWhenNoConditionalRuleMatches(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"default-advanced-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"default-advanced-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 100,
					"service_tier": "priority",
					"input_price": 4,
					"output_price": 10
				},
				{
					"priority": 99,
					"input_price": 3,
					"output_price": 9
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "default-advanced-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
	}

	priceData, err := ModelPriceHelper(c, info, 256, &types.TokenCountMeta{MaxTokens: 2048})
	require.NoError(t, err)
	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 1.5, priceData.ModelRatio)
	require.Equal(t, 3.0, priceData.CompletionRatio)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentServiceTierCaseInsensitively(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"service-tier-case-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"service-tier-case-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"service_tier": "Default",
					"input_price": 4,
					"output_price": 8
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "service-tier-case-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "default",
		},
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{MaxTokens: 256})
	require.NoError(t, err)
	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 2.0, priceData.ModelRatio)
	require.Equal(t, 2.0, priceData.CompletionRatio)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentWithOpenEndedRanges(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"gemini-3.1-pro-preview":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"gemini-3.1-pro-preview": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 1,
					"input_min": 0,
					"input_max": 200000,
					"output_min": 0,
					"output_max": 200000,
					"input_price": 2,
					"output_price": 12,
					"cache_read_price": 0.2
				},
				{
					"priority": 2,
					"input_min": 200001,
					"output_min": 200001,
					"input_price": 4,
					"output_price": 18,
					"cache_read_price": 0.4
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3.1-pro-preview",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	priceData, err := ModelPriceHelper(c, info, 210000, &types.TokenCountMeta{MaxTokens: 210001})
	require.NoError(t, err)
	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 2.0, priceData.ModelRatio)
	require.Equal(t, 4.5, priceData.CompletionRatio)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.NotNil(t, priceData.AdvancedRuleSnapshot.Priority)
	require.Equal(t, 2, *priceData.AdvancedRuleSnapshot.Priority)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentByModalityAndCapturesRuntimeContext(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"gpt-4o-realtime-preview":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"gpt-4o-realtime-preview": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"input_modality": "audio",
					"output_modality": "audio",
					"input_price": 12,
					"output_price": 24
				},
				{
					"priority": 99,
					"input_price": 3,
					"output_price": 9
				}
			]
		}
	}`))

	message := dto.Message{Role: "user"}
	message.SetMediaContent([]dto.MediaContent{
		{
			Type: dto.ContentTypeText,
			Text: "Summarize the meeting",
		},
		{
			Type: dto.ContentTypeInputAudio,
			InputAudio: &dto.MessageInputAudio{
				Data:   "UklGRg==",
				Format: "wav",
			},
		},
	})

	request := &dto.GeneralOpenAIRequest{
		Model:      "gpt-4o-realtime-preview",
		Messages:   []dto.Message{message},
		Modalities: []byte(`["audio"]`),
	}

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o-realtime-preview",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         request,
	}

	priceData, err := ModelPriceHelper(c, info, 128, request.GetTokenCountMeta())
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 6.0, priceData.ModelRatio)
	require.Equal(t, 2.0, priceData.CompletionRatio)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "audio", priceData.AdvancedRuleSnapshot.InputModality)
	require.Equal(t, "audio", priceData.AdvancedRuleSnapshot.OutputModality)
	require.Equal(t, "per_million_tokens", priceData.AdvancedRuleSnapshot.BillingUnit)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "input_modalities=audio,text")
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "output_modalities=audio")
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, "per_million_tokens", priceData.AdvancedPricingContext.BillingUnit)
	require.Equal(t, []string{"audio", "text"}, priceData.AdvancedPricingContext.InputModalities)
	require.Equal(t, []string{"audio"}, priceData.AdvancedPricingContext.OutputModalities)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentByResponsesGoogleSearchUsageWithGroundingRule(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"responses-grounding-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"responses-grounding-model": {
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
					"tool_usage_count": 2,
					"input_price": 14
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "responses-grounding-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         &dto.OpenAIResponsesRequest{},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolWebSearchPreview: {
					ToolName:          dto.BuildInToolWebSearchPreview,
					CallCount:         3,
					SearchContextSize: "medium",
				},
			},
		},
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{})
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 7.0, priceData.ModelRatio)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, types.AdvancedBillingUnitPer1000Calls, priceData.AdvancedRuleSnapshot.BillingUnit)
	require.Equal(t, "google_search", priceData.AdvancedRuleSnapshot.ToolUsageType)
	require.NotNil(t, priceData.AdvancedRuleSnapshot.ThresholdSnapshot.ToolUsageCount)
	require.Equal(t, 2, *priceData.AdvancedRuleSnapshot.ThresholdSnapshot.ToolUsageCount)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "tool_usage_type=google_search")
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "tool_usage_count=3")
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, types.AdvancedBillingUnitPer1000Calls, priceData.AdvancedPricingContext.BillingUnit)
	require.Equal(t, "google_search", priceData.AdvancedPricingContext.ToolUsageType)
	require.NotNil(t, priceData.AdvancedPricingContext.ToolUsageCount)
	require.Equal(t, 3, *priceData.AdvancedPricingContext.ToolUsageCount)
}

func TestModelPriceHelper_MatchesGroundingRuleCarriesDedicatedToolOveragePrice(t *testing.T) {
	restoreRatioSettings(t)

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
					"tool_usage_count": 2,
					"free_quota": 1000,
					"overage_threshold": 500,
					"input_price": 2,
					"tool_overage_price": 14
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "responses-grounding-overage-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         &dto.OpenAIResponsesRequest{},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolWebSearchPreview: {
					ToolName:          dto.BuildInToolWebSearchPreview,
					CallCount:         3,
					SearchContextSize: "medium",
				},
			},
		},
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{})
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "google_search", priceData.AdvancedRuleSnapshot.ToolUsageType)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "tool_usage_type=google_search")
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "tool_usage_count=3")
	require.NotNil(t, priceData.AdvancedRuleSnapshot.ThresholdSnapshot.FreeQuota)
	require.Equal(t, 1000, *priceData.AdvancedRuleSnapshot.ThresholdSnapshot.FreeQuota)
	require.NotNil(t, priceData.AdvancedRuleSnapshot.ThresholdSnapshot.OverageThreshold)
	require.Equal(t, 500, *priceData.AdvancedRuleSnapshot.ThresholdSnapshot.OverageThreshold)
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, "google_search", priceData.AdvancedPricingContext.ToolUsageType)
	require.NotNil(t, priceData.AdvancedPricingContext.ToolUsageCount)
	require.Equal(t, 3, *priceData.AdvancedPricingContext.ToolUsageCount)
	require.NotNil(t, priceData.AdvancedPricingContext.FreeQuota)
	require.Equal(t, 1000, *priceData.AdvancedPricingContext.FreeQuota)
	require.NotNil(t, priceData.AdvancedPricingContext.OverageThreshold)
	require.Equal(t, 500, *priceData.AdvancedPricingContext.OverageThreshold)

	snapshotJSON, err := common.Marshal(priceData.AdvancedRuleSnapshot)
	require.NoError(t, err)
	require.Contains(t, string(snapshotJSON), `"tool_overage_price":14`)

	contextJSON, err := common.Marshal(priceData.AdvancedPricingContext)
	require.NoError(t, err)
	require.Contains(t, string(contextJSON), `"tool_overage_price":14`)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentByResponsesGoogleSearchUsageWithWebSearchRule(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"responses-web-search-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"responses-web-search-model": {
			"rule_type": "text_segment",
			"billing_unit": "per_1000_calls",
			"segments": [
				{
					"priority": 10,
					"input_price": 2
				},
				{
					"priority": 20,
					"tool_usage_type": "web_search",
					"tool_usage_count": 2,
					"input_price": 14
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "responses-web-search-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         &dto.OpenAIResponsesRequest{},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				dto.BuildInToolWebSearchPreview: {
					ToolName:          dto.BuildInToolWebSearchPreview,
					CallCount:         3,
					SearchContextSize: "medium",
				},
			},
		},
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{})
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 7.0, priceData.ModelRatio)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "google_search", priceData.AdvancedRuleSnapshot.ToolUsageType)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "tool_usage_type=google_search")
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, "google_search", priceData.AdvancedPricingContext.ToolUsageType)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentByClaudeGoogleSearchUsageWithGroundingRule(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"claude-grounding-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"claude-grounding-model": {
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
					"tool_usage_count": 2,
					"input_price": 14
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	c.Set("claude_web_search_requests", 2)
	info := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatClaude,
		FinalRequestRelayFormat: types.RelayFormatClaude,
		OriginModelName:         "claude-grounding-model",
		UsingGroup:              "default",
		UserGroup:               "default",
		Request:                 &dto.ClaudeRequest{},
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{})
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 7.0, priceData.ModelRatio)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "google_search", priceData.AdvancedRuleSnapshot.ToolUsageType)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "tool_usage_type=google_search")
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, "google_search", priceData.AdvancedPricingContext.ToolUsageType)
	require.NotNil(t, priceData.AdvancedPricingContext.ToolUsageCount)
	require.Equal(t, 2, *priceData.AdvancedPricingContext.ToolUsageCount)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentByResponsesGoogleMapsUsage(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"responses-google-maps-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"responses-google-maps-model": {
			"rule_type": "text_segment",
			"billing_unit": "per_1000_calls",
			"segments": [
				{
					"priority": 10,
					"input_price": 1
				},
				{
					"priority": 20,
					"tool_usage_type": "google_maps",
					"tool_usage_count": 2,
					"input_price": 5
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "responses-google-maps-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         &dto.OpenAIResponsesRequest{},
		ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{
			BuiltInTools: map[string]*relaycommon.BuildInToolInfo{
				"google_maps": {
					ToolName:  "google_maps",
					CallCount: 3,
				},
			},
		},
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{})
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "google_maps", priceData.AdvancedRuleSnapshot.ToolUsageType)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "tool_usage_type=google_maps")
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "tool_usage_count=3")
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, "google_maps", priceData.AdvancedPricingContext.ToolUsageType)
	require.NotNil(t, priceData.AdvancedPricingContext.ToolUsageCount)
	require.Equal(t, 3, *priceData.AdvancedPricingContext.ToolUsageCount)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentPerSecondForGeminiLiveAudio(t *testing.T) {
	restoreRatioSettings(t)

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
					"input_price": 3,
					"output_price": 12
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

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3.1-flash-live-preview",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         request,
	}

	priceData, err := ModelPriceHelper(c, info, 128, request.GetTokenCountMeta())
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedBillingUnitPerSecond, priceData.AdvancedRuleSnapshot.BillingUnit)
	require.Equal(t, "audio", priceData.AdvancedRuleSnapshot.InputModality)
	require.Equal(t, "audio", priceData.AdvancedRuleSnapshot.OutputModality)
	require.Equal(t, []string{"audio"}, priceData.AdvancedPricingContext.OutputModalities)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentPerImageForGeminiImageGeneration(t *testing.T) {
	restoreRatioSettings(t)

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

	request := &dto.GeneralOpenAIRequest{
		Model:  "gemini-3.1-flash-image-preview",
		Prompt: "Generate a stylized skyline illustration",
		Size:   "1024x1024",
	}

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3.1-flash-image-preview",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         request,
		RequestURLPath:  "/v1/images/generations",
	}

	priceData, err := ModelPriceHelper(c, info, 64, request.GetTokenCountMeta())
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedBillingUnitPerImage, priceData.AdvancedRuleSnapshot.BillingUnit)
	require.Equal(t, "image", priceData.AdvancedRuleSnapshot.OutputModality)
	require.Equal(t, []string{"image"}, priceData.AdvancedPricingContext.OutputModalities)
}

func TestModelPriceHelperCapturesImageCountForChatBasedPerImageBilling(t *testing.T) {
	restoreRatioSettings(t)

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

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3.1-flash-image-chat-preview",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         request,
		RequestURLPath:  "/v1/chat/completions",
	}

	priceData, err := ModelPriceHelper(c, info, 64, request.GetTokenCountMeta())
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, types.AdvancedBillingUnitPerImage, priceData.AdvancedRuleSnapshot.BillingUnit)
	require.Equal(t, "2k", priceData.AdvancedRuleSnapshot.ImageSizeTier)
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, "2k", priceData.AdvancedPricingContext.ImageSizeTier)
	require.NotNil(t, priceData.AdvancedPricingContext.ImageCount)
	require.Equal(t, 2, *priceData.AdvancedPricingContext.ImageCount)
}

func TestModelPriceHelperMatchesAdvancedTextSegmentByImageSizeTier(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"gemini-3-pro-image-preview":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"gemini-3-pro-image-preview": {
			"rule_type": "text_segment",
			"billing_unit": "per_image",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000000,
					"output_modality": "image",
					"image_size_tier": "1k",
					"input_price": 2,
					"output_price": 120
				},
				{
					"priority": 20,
					"input_min": 0,
					"input_max": 1000000,
					"output_modality": "image",
					"image_size_tier": "2k",
					"input_price": 2,
					"output_price": 134
				},
				{
					"priority": 30,
					"input_min": 0,
					"input_max": 1000000,
					"output_modality": "image",
					"image_size_tier": "4k",
					"input_price": 2,
					"output_price": 240
				}
			]
		}
	}`))

	request := &dto.GeneralOpenAIRequest{
		Model:  "gemini-3-pro-image-preview",
		Prompt: "Generate a photorealistic product render",
		Size:   "2048x2048",
	}

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3-pro-image-preview",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request:         request,
		RequestURLPath:  "/v1/images/generations",
	}

	priceData, err := ModelPriceHelper(c, info, 64, request.GetTokenCountMeta())
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "2k", priceData.AdvancedRuleSnapshot.ImageSizeTier)
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, "2k", priceData.AdvancedPricingContext.ImageSizeTier)
	require.Equal(t, 1.0, priceData.ModelRatio)
	require.Equal(t, 67.0, priceData.CompletionRatio)
}

func TestModelPriceHelperHonorsExplicitPerTokenModeWhenPriceAndRatioBothExist(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"explicit-per-token-model":6}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"explicit-per-token-model":0.25}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"explicit-per-token-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"explicit-per-token-model":"per_token"}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "explicit-per-token-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{MaxTokens: 512})
	require.NoError(t, err)

	require.Equal(t, types.BillingModePerToken, priceData.BillingMode)
	require.False(t, priceData.UsePrice)
	require.Equal(t, 0.0, priceData.ModelPrice)
	require.Equal(t, 6.0, priceData.ModelRatio)
	require.Equal(t, 2.5, priceData.CompletionRatio)
}

func TestModelPriceHelperHonorsExplicitPerRequestModeWhenPriceAndRatioBothExist(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"explicit-per-request-model":6}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"explicit-per-request-model":0.25}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"explicit-per-request-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"explicit-per-request-model":"per_request"}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "explicit-per-request-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{MaxTokens: 512})
	require.NoError(t, err)

	require.Equal(t, types.BillingModePerRequest, priceData.BillingMode)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.25, priceData.ModelPrice)
	require.Equal(t, 0.0, priceData.ModelRatio)
}

func TestModelPriceHelperReturnsErrorWhenExplicitPerTokenModeHasNoRatio(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"missing-ratio-model":0.25}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"missing-ratio-model":"per_token"}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "missing-ratio-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	_, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{MaxTokens: 512})
	require.Error(t, err)
	require.ErrorContains(t, err, "per_token")
	require.ErrorContains(t, err, "ratio")
}

func TestModelPriceHelperReturnsErrorWhenExplicitPerRequestModeHasNoModelPrice(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"missing-price-model":6}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"missing-price-model":"per_request"}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "missing-price-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	_, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{MaxTokens: 512})
	require.Error(t, err)
	require.ErrorContains(t, err, "per_request")
	require.ErrorContains(t, err, "model_price")
}

func TestModelPriceHelperUsesWildcardExplicitPerTokenModeForCompactModel(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"*-openai-compact":6}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"*-openai-compact":0.25}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"foo-openai-compact":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"*-openai-compact":"per_token"}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "foo-openai-compact",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	priceData, err := ModelPriceHelper(c, info, 128, &types.TokenCountMeta{MaxTokens: 512})
	require.NoError(t, err)

	require.Equal(t, types.BillingModePerToken, priceData.BillingMode)
	require.False(t, priceData.UsePrice)
	require.Equal(t, 0.0, priceData.ModelPrice)
	require.Equal(t, 6.0, priceData.ModelRatio)
}

func TestModelPriceHelperUsesWildcardAdvancedRuleForCompactModel(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"foo-openai-compact":7}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"foo-openai-compact":4}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"*-openai-compact":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"*-openai-compact": {
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
		OriginModelName: "foo-openai-compact",
		UsingGroup:      "default",
		UserGroup:       "default",
		Request: &dto.OpenAIResponsesRequest{
			ServiceTier: "priority",
		},
	}

	priceData, err := ModelPriceHelper(c, info, 256, &types.TokenCountMeta{MaxTokens: 2048})
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, 2.0, priceData.ModelRatio)
	require.Equal(t, 3.0, priceData.CompletionRatio)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
}

func TestModelPriceHelperPerCallHonorsExplicitPerTokenMode(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"task-per-token-model":6}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"task-per-token-model":0.25}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"task-per-token-model":2.5}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-per-token-model":"per_token"}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-per-token-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModePerToken, priceData.BillingMode)
	require.False(t, priceData.UsePrice)
	require.Equal(t, 6.0, priceData.ModelRatio)
	require.Equal(t, 0.0, priceData.ModelPrice)
	require.Greater(t, priceData.Quota, 0)
}

func TestModelPriceHelperPerCallHonorsExplicitPerRequestMode(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"task-per-request-model":6}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"task-per-request-model":0.25}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-per-request-model":"per_request"}`))

	c, _ := gin.CreateTestContext(nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-per-request-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModePerRequest, priceData.BillingMode)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.25, priceData.ModelPrice)
	require.Equal(t, 0.0, priceData.ModelRatio)
	require.Greater(t, priceData.Quota, 0)
}

func TestModelPriceHelperPerCallReturnsAdvancedMediaTaskPriceDataWhenRuleMatches(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-model": {
			"rule_type": "media_task",
			"task_type": "video_generation",
			"segments": [
				{
					"priority": 10,
					"resolution": "720p",
					"output_duration_min": 5,
					"output_duration_max": 5,
					"unit_price": 8.8,
					"min_tokens": 194400,
					"remark": "task advanced rule"
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Size:     "1280x720",
		Duration: 5,
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: constant.TaskActionGenerate}

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.True(t, priceData.UsePrice)
	require.Equal(t, types.AdvancedRuleTypeMediaTask, priceData.AdvancedRuleType)
	require.Equal(t, 8.8, priceData.ModelPrice)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "video_generation", priceData.AdvancedRuleSnapshot.TaskType)
	require.NotNil(t, priceData.AdvancedRuleSnapshot.PriceSnapshot.InputPrice)
	require.Equal(t, 8.8, *priceData.AdvancedRuleSnapshot.PriceSnapshot.InputPrice)
	require.NotNil(t, priceData.AdvancedRuleSnapshot.ThresholdSnapshot.MinTokens)
	require.Equal(t, 194400, *priceData.AdvancedRuleSnapshot.ThresholdSnapshot.MinTokens)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "task_type=video_generation")
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "raw_action=generate")
	require.NotNil(t, priceData.AdvancedPricingContext)
	require.Equal(t, types.AdvancedBillingUnitPerMillionTokens, priceData.AdvancedPricingContext.BillingUnit)
	require.NotNil(t, priceData.AdvancedPricingContext.LiveDurationSecs)
	require.Equal(t, 5, *priceData.AdvancedPricingContext.LiveDurationSecs)
	require.Greater(t, priceData.Quota, 0)
}

func TestModelPriceHelperPerCallMatchesAdvancedMediaTaskWhenMetadataUsesRatioKey(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-ratio-key-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-ratio-key-model": {
			"rule_type": "media_task",
			"task_type": "video_generation",
			"segments": [
				{
					"priority": 10,
					"input_video": true,
					"resolution": "720p",
					"aspect_ratio": "16:9",
					"output_duration_min": 5,
					"output_duration_max": 5,
					"input_video_duration_min": 2,
					"input_video_duration_max": 15,
					"unit_price": 8.8
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Duration: 5,
		Metadata: map[string]interface{}{
			"input_video":          true,
			"input_video_duration": 3,
			"resolution":           "720p",
			"ratio":                "16:9",
		},
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-ratio-key-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: constant.TaskActionGenerate}

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedRuleTypeMediaTask, priceData.AdvancedRuleType)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 8.8, priceData.ModelPrice)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "aspect_ratio=16:9")
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "input_video=true")
}

func TestModelPriceHelperPerCallReturnsErrorWhenAdvancedMediaTaskDoesNotMatch(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"task-advanced-media-fallback-model":0.25}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-fallback-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-fallback-model": {
			"rule_type": "media_task",
			"task_type": "image_generation",
			"segments": [
				{
					"priority": 10,
					"resolution": "1080p",
					"unit_price": 8.8
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Size: "1280x720",
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-fallback-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: constant.TaskActionGenerate}

	_, err := ModelPriceHelperPerCall(c, info)
	require.Error(t, err)
	require.ErrorContains(t, err, "advanced pricing")
	require.ErrorContains(t, err, "task-advanced-media-fallback-model")
}

func TestModelPriceHelperPerCallDoesNotDropAdvancedMediaTaskWhenMinTokensConfigured(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-min-tokens-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-min-tokens-model": {
			"rule_type": "media_task",
			"task_type": "video_generation",
			"segments": [
				{
					"priority": 10,
					"unit_price": 8.8,
					"min_tokens": 194400
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{})
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-min-tokens-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: constant.TaskActionGenerate}

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedRuleTypeMediaTask, priceData.AdvancedRuleType)
	require.True(t, priceData.UsePrice)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.NotNil(t, priceData.AdvancedRuleSnapshot.ThresholdSnapshot.MinTokens)
	require.Equal(t, 194400, *priceData.AdvancedRuleSnapshot.ThresholdSnapshot.MinTokens)
}

func TestModelPriceHelperPerCallDoesNotMatchAdvancedMediaTaskWhenCanonicalTaskTypeDiffers(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"task-advanced-media-task-type-model":0.3}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-task-type-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-task-type-model": {
			"rule_type": "media_task",
			"task_type": "image_generation",
			"segments": [
				{
					"priority": 10,
					"unit_price": 8.8
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{})
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-task-type-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: constant.TaskActionGenerate}

	_, err := ModelPriceHelperPerCall(c, info)
	require.Error(t, err)
	require.Contains(t, err.Error(), "advanced pricing did not match any active rule")
}

func TestModelPriceHelperPerCallMatchesAdvancedMediaTaskWhenLegacyRawActionTaskTypeConfigured(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-legacy-task-type-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-legacy-task-type-model": {
			"rule_type": "media_task",
			"task_type": "generate",
			"segments": [
				{
					"priority": 10,
					"unit_price": 6.6
				}
			]
		}
	}`))

	c, _ := gin.CreateTestContext(nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Size:     "1280x720",
		Duration: 5,
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-legacy-task-type-model",
		UsingGroup:      "default",
		UserGroup:       "default",
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: constant.TaskActionGenerate}

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedRuleTypeMediaTask, priceData.AdvancedRuleType)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 6.6, priceData.ModelPrice)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "generate", priceData.AdvancedRuleSnapshot.TaskType)
}

func TestModelPriceHelperPerCallMatchesAdvancedMediaTaskWhenInfoActionProvidesSubmitTimeRawAction(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-info-action-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-info-action-model": {
			"rule_type": "media_task",
			"task_type": "video_generation",
			"segments": [
				{
					"priority": 10,
					"unit_price": 5.5
				}
			]
		}
	}`))

	body := `{"model":"task-advanced-media-info-action-model","prompt":"video prompt","images":["test-image"],"size":"1280x720","duration":5}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-info-action-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, relaycommon.ValidateMultipartDirect(c, info))
	require.Equal(t, constant.TaskActionGenerate, info.Action)

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedRuleTypeMediaTask, priceData.AdvancedRuleType)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 5.5, priceData.ModelPrice)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "video_generation", priceData.AdvancedRuleSnapshot.TaskType)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "raw_action=generate")
}

func TestModelPriceHelperPerCallMatchesAdvancedMediaTaskWhenGeminiLegacyRawActionNeedsTaskRequestInference(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-gemini-infer-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-gemini-infer-model": {
			"rule_type": "media_task",
			"task_type": "generate",
			"segments": [
				{
					"priority": 10,
					"unit_price": 4.4
				}
			]
		}
	}`))

	body := `{"model":"task-advanced-media-gemini-infer-model","prompt":"video prompt","images":["test-image"],"size":"1280x720","duration":5}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-gemini-infer-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeGemini},
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionTextGenerate))
	require.Equal(t, constant.TaskActionTextGenerate, info.Action)

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedRuleTypeMediaTask, priceData.AdvancedRuleType)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 4.4, priceData.ModelPrice)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "generate", priceData.AdvancedRuleSnapshot.TaskType)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "raw_action=generate")
}

func TestResolveRawTaskActionInfersGeminiGenerateBeforeBuildRequestBodyUsingProviderDefaultAction(t *testing.T) {
	body := `{"model":"veo-3.0-generate-001","prompt":"video prompt","images":["test-image"]}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeGemini},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionTextGenerate))
	require.Equal(t, constant.TaskActionTextGenerate, info.Action)
	require.Equal(t, constant.TaskActionGenerate, resolveRawTaskAction(info, c))
}

func TestResolveRawTaskActionInfersVertexTextGenerateWithoutImageBeforeBuildRequestBodyUsingProviderDefaultAction(t *testing.T) {
	body := `{"model":"veo-3.0-generate-001","prompt":"video prompt"}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeVertexAi},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionTextGenerate))
	require.Equal(t, constant.TaskActionTextGenerate, resolveRawTaskAction(info, c))
}

func TestResolveRawTaskActionInfersKlingTextGenerateWithoutImageBeforeBuildRequestBodyUsingProviderDefaultAction(t *testing.T) {
	body := `{"model":"kling-v1","prompt":"video prompt"}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeKling},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate))
	require.Equal(t, constant.TaskActionGenerate, info.Action)
	require.Equal(t, constant.TaskActionTextGenerate, resolveRawTaskAction(info, c))
}

func TestResolveRawTaskActionInfersDoubaoTextGenerateWithoutReferences(t *testing.T) {
	body := `{"model":"doubao-seedance-2-0-260128","prompt":"video prompt","size":"1280x720","duration":5,"metadata":{"input_video":false}}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeDoubaoVideo},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate))
	require.Equal(t, constant.TaskActionGenerate, info.Action)
	require.Equal(t, constant.TaskActionTextGenerate, resolveRawTaskAction(info, c))
}

func TestResolveRawTaskActionInfersDoubaoGenerateWithReferenceVideo(t *testing.T) {
	body := `{"model":"doubao-seedance-2-0-260128","prompt":"video prompt","metadata":{"videos":["https://example.com/reference.mp4"]}}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeDoubaoVideo},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate))
	require.Equal(t, constant.TaskActionGenerate, resolveRawTaskAction(info, c))
}

func TestModelPriceHelperPerCallMatchesAdvancedMediaTaskWhenJimengMultipartInfersFirstTailGenerateBeforeBuildRequestBody(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"task-advanced-media-jimeng-first-tail-model":0.33}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"task-advanced-media-jimeng-first-tail-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"task-advanced-media-jimeng-first-tail-model": {
			"rule_type": "media_task",
			"task_type": "firstTailGenerate",
			"segments": [
				{
					"priority": 10,
					"unit_price": 7.7
				}
			]
		}
	}`))

	c := newMultipartTaskContext(t, "/v1/videos", map[string]string{
		"model":  "task-advanced-media-jimeng-first-tail-model",
		"prompt": "video prompt",
	}, "input_reference", 2)
	info := &relaycommon.RelayInfo{
		OriginModelName: "task-advanced-media-jimeng-first-tail-model",
		UsingGroup:      "default",
		UserGroup:       "default",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeJimeng},
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}

	require.Nil(t, relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate))
	require.Equal(t, constant.TaskActionGenerate, info.Action)

	priceData, err := ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)

	require.Equal(t, types.BillingModeAdvanced, priceData.BillingMode)
	require.Equal(t, types.AdvancedRuleTypeMediaTask, priceData.AdvancedRuleType)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 7.7, priceData.ModelPrice)
	require.NotNil(t, priceData.AdvancedRuleSnapshot)
	require.Equal(t, "firstTailGenerate", priceData.AdvancedRuleSnapshot.TaskType)
	require.Contains(t, priceData.AdvancedRuleSnapshot.MatchSummary, "raw_action=firstTailGenerate")
}

func newMultipartTaskContext(t *testing.T, path string, fields map[string]string, fileField string, fileCount int) *gin.Context {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	for i := 0; i < fileCount; i++ {
		part, err := writer.CreateFormFile(fileField, fmt.Sprintf("%s-%d.png", fileField, i))
		require.NoError(t, err)
		_, err = part.Write([]byte("test-image"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}

func TestContainPriceOrRatioReturnsTrueForAdvancedOnlyModel(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"advanced-only-model":"advanced"}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(`{
		"advanced-only-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000,
					"input_price": 4
				}
			]
		}
	}`))

	require.True(t, ContainPriceOrRatio("advanced-only-model"))
}

func TestContainPriceOrRatioRespectsExplicitModesAndWildcardModes(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"explicit-per-token-model":6,"*-openai-compact":5}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"explicit-per-request-model":0.25}`))
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{
		"explicit-per-token-model":"per_token",
		"explicit-per-request-model":"per_request",
		"missing-price-model":"per_request",
		"missing-ratio-model":"per_token",
		"*-openai-compact":"per_token"
	}`))

	require.True(t, ContainPriceOrRatio("explicit-per-token-model"))
	require.True(t, ContainPriceOrRatio("explicit-per-request-model"))
	require.True(t, ContainPriceOrRatio("foo-openai-compact"))
	require.False(t, ContainPriceOrRatio("missing-price-model"))
	require.False(t, ContainPriceOrRatio("missing-ratio-model"))
}

func TestValidateAdvancedPricingRulesRejectsUnsupportedCacheConditions(t *testing.T) {
	err := ratio_setting.ValidateAdvancedPricingRulesJSONString(`{
		"cache-condition-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000,
					"cache_read": true,
					"input_price": 4
				}
			]
		}
	}`)

	require.Error(t, err)
	require.ErrorContains(t, err, "cache")
	require.ErrorContains(t, err, "not supported")
}

func TestValidateAdvancedPricingRulesRejectsPositiveDependentPricesWhenInputPriceIsZero(t *testing.T) {
	err := ratio_setting.ValidateAdvancedPricingRulesJSONString(`{
		"zero-input-price-model": {
			"rule_type": "text_segment",
			"segments": [
				{
					"priority": 10,
					"input_min": 0,
					"input_max": 1000,
					"input_price": 0,
					"output_price": 1
				}
			]
		}
	}`)

	require.Error(t, err)
	require.ErrorContains(t, err, "input_price")
	require.ErrorContains(t, err, "zero")
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
