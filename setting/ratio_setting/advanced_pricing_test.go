package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAdvancedPricingConfigValidatesTextAndMediaRules(t *testing.T) {
	jsonStr := `{
      "billing_mode": {"doubao-seed-2.0-pro":"advanced"},
      "rules": {
        "doubao-seed-2.0-pro": {
          "rule_type": "text_segment",
          "segments": [
            {
              "priority": 10,
              "input_min": 0,
              "input_max": 32,
              "output_min": 0,
              "output_max": 8192,
              "service_tier": "default",
              "cache_read": true,
              "cache_create": false,
              "input_price": 3.2,
              "output_price": 16,
              "cache_price": 1.6
            }
          ]
        },
        "doubao-seedance-2.0": {
          "rule_type": "media_task",
          "segments": [
            {
              "priority": 10,
              "inference_mode": "online",
              "audio": true,
              "input_video": false,
              "resolution": "720p",
              "aspect_ratio": "16:9",
              "output_duration": 5,
              "input_video_duration": 0,
              "draft": true,
              "draft_coefficient": 0.5,
              "remark": "fast lane",
              "unit_price": 28,
              "min_tokens": 194400
            }
          ]
        }
      }
    }`

	cfg, err := ParseAdvancedPricingConfig(jsonStr)
	require.NoError(t, err)
	require.Equal(t, BillingModeAdvanced, cfg.ModelModes["doubao-seed-2.0-pro"])
	require.Equal(t, RuleTypeTextSegment, cfg.ModelRules["doubao-seed-2.0-pro"].RuleType)
	require.Equal(t, RuleTypeMediaTask, cfg.ModelRules["doubao-seedance-2.0"].RuleType)
	require.Equal(t, 0, *cfg.ModelRules["doubao-seed-2.0-pro"].Segments[0].OutputMin)
	require.Equal(t, 8192, *cfg.ModelRules["doubao-seed-2.0-pro"].Segments[0].OutputMax)
	require.Equal(t, "default", cfg.ModelRules["doubao-seed-2.0-pro"].Segments[0].ServiceTier)
	require.Equal(t, true, *cfg.ModelRules["doubao-seed-2.0-pro"].Segments[0].CacheRead)
	require.Equal(t, false, *cfg.ModelRules["doubao-seed-2.0-pro"].Segments[0].CacheCreate)
	require.Equal(t, 1.6, *cfg.ModelRules["doubao-seed-2.0-pro"].Segments[0].CachePrice)
	require.Equal(t, true, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].Audio)
	require.Equal(t, "16:9", cfg.ModelRules["doubao-seedance-2.0"].Segments[0].AspectRatio)
	require.Equal(t, 5, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].OutputDuration)
	require.Equal(t, 0, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].InputVideoDuration)
	require.Equal(t, true, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].Draft)
	require.Equal(t, 0.5, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].DraftCoefficient)
	require.Equal(t, "fast lane", cfg.ModelRules["doubao-seedance-2.0"].Segments[0].Remark)
}

func TestParseAdvancedPricingConfigRejectsOverlappingTextSegments(t *testing.T) {
	_, err := ParseAdvancedPricingConfig(`{
      "rules": {
        "gemini-2.5-pro": {
          "rule_type": "text_segment",
          "segments": [
            {"priority": 10, "input_min": 0, "input_max": 32, "input_price": 1, "output_price": 2},
            {"priority": 20, "input_min": 16, "input_max": 64, "input_price": 2, "output_price": 4}
          ]
        }
      }
    }`)
	require.ErrorContains(t, err, "区间")
}

func TestParseAdvancedPricingConfigAllowsTextRuleWithoutOutputPrice(t *testing.T) {
	cfg, err := ParseAdvancedPricingConfig(`{
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "segments": [
            {"priority": 10, "service_tier": "priority", "input_price": 1.2}
          ]
        }
      }
    }`)
	require.NoError(t, err)
	require.Nil(t, cfg.ModelRules["gpt-5"].Segments[0].OutputPrice)
	require.Equal(t, 1.2, *cfg.ModelRules["gpt-5"].Segments[0].InputPrice)
}

func TestParseAdvancedPricingConfigRejectsTextRuleWithoutConditions(t *testing.T) {
	_, err := ParseAdvancedPricingConfig(`{
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "segments": [
            {"priority": 10, "input_price": 1.2}
          ]
        }
      }
    }`)
	require.ErrorContains(t, err, "condition")
}

func TestParseAdvancedPricingConfigAllowsMediaRuleWithoutMinTokens(t *testing.T) {
	cfg, err := ParseAdvancedPricingConfig(`{
      "rules": {
        "veo-3.1-fast-generate-preview": {
          "rule_type": "media_task",
          "segments": [
            {"priority": 10, "audio": true, "input_video": false, "aspect_ratio": "16:9", "draft": false, "unit_price": 8}
          ]
        }
      }
    }`)
	require.NoError(t, err)
	require.Nil(t, cfg.ModelRules["veo-3.1-fast-generate-preview"].Segments[0].MinTokens)
}

func TestParseAdvancedPricingConfigRejectsDuplicatePriority(t *testing.T) {
	_, err := ParseAdvancedPricingConfig(`{
      "rules": {
        "gemini-2.5-pro": {
          "rule_type": "text_segment",
          "segments": [
            {"priority": 10, "input_min": 0, "input_max": 32, "input_price": 1},
            {"priority": 10, "input_min": 32, "input_max": 64, "input_price": 2}
          ]
        }
      }
    }`)
	require.ErrorContains(t, err, "priority")
}

func TestValidateAdvancedPricingRulesJSONStringDoesNotMutateRuntimeMap(t *testing.T) {
	original := advancedPricingRulesMap.ReadAll()
	advancedPricingRulesMap.Clear()
	advancedPricingRulesMap.Set("existing-model", AdvancedPricingRuleSet{
		RuleType: RuleTypeTextSegment,
		Segments: []AdvancedPriceRule{
			{
				Priority:    intPtr(1),
				ServiceTier: "default",
				InputPrice:  float64Ptr(1),
			},
		},
	})
	t.Cleanup(func() {
		advancedPricingRulesMap.Clear()
		advancedPricingRulesMap.AddAll(original)
	})

	err := ValidateAdvancedPricingRulesJSONString(`{
      "new-model": {
        "rule_type": "media_task",
        "segments": [
          {"priority": 10, "unit_price": 9.9}
        ]
      }
    }`)
	require.NoError(t, err)

	current := advancedPricingRulesMap.ReadAll()
	require.Len(t, current, 1)
	require.Contains(t, current, "existing-model")
	require.NotContains(t, current, "new-model")
}

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
