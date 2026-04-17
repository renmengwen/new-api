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
              "input_price": 3.2,
              "output_price": 16,
              "cache_read_price": 1.6,
              "cache_create_price": 2.4
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
              "output_duration_min": 5,
              "output_duration_max": 5,
              "input_video_duration_min": 0,
              "input_video_duration_max": 0,
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
	require.Equal(t, 1.6, *cfg.ModelRules["doubao-seed-2.0-pro"].Segments[0].CacheReadPrice)
	require.Equal(t, 2.4, *cfg.ModelRules["doubao-seed-2.0-pro"].Segments[0].CacheCreatePrice)
	require.Equal(t, true, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].Audio)
	require.Equal(t, "16:9", cfg.ModelRules["doubao-seedance-2.0"].Segments[0].AspectRatio)
	require.Equal(t, 5, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].OutputDurationMin)
	require.Equal(t, 5, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].OutputDurationMax)
	require.Equal(t, 0, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].InputVideoDurationMin)
	require.Equal(t, 0, *cfg.ModelRules["doubao-seedance-2.0"].Segments[0].InputVideoDurationMax)
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

func TestParseAdvancedPricingConfigAllowsExactRanges(t *testing.T) {
	cfg, err := ParseAdvancedPricingConfig(`{
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "segments": [
            {"priority": 10, "input_min": 32, "input_max": 32, "input_price": 1}
          ]
        },
        "veo-3.1-fast-generate-preview": {
          "rule_type": "media_task",
          "segments": [
            {"priority": 10, "unit_price": 8, "output_duration_min": 5, "output_duration_max": 5}
          ]
        }
      }
    }`)
	require.NoError(t, err)
	require.Equal(t, 32, *cfg.ModelRules["gpt-5"].Segments[0].InputMin)
	require.Equal(t, 32, *cfg.ModelRules["gpt-5"].Segments[0].InputMax)
	require.Equal(t, 5, *cfg.ModelRules["veo-3.1-fast-generate-preview"].Segments[0].OutputDurationMin)
	require.Equal(t, 5, *cfg.ModelRules["veo-3.1-fast-generate-preview"].Segments[0].OutputDurationMax)
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

func TestParseAdvancedPricingConfigNormalizesLegacyTextShellRule(t *testing.T) {
	cfg, err := ParseAdvancedPricingConfig(`{
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "display_name": "ignored shell field",
          "segments_text": "0-100: 1.2\n101-200: 2.4",
          "note": "ignored note"
        }
      }
    }`)
	require.NoError(t, err)

	ruleSet := cfg.ModelRules["gpt-5"]
	require.Equal(t, RuleTypeTextSegment, ruleSet.RuleType)
	require.Len(t, ruleSet.Segments, 2)
	require.Equal(t, 10, *ruleSet.Segments[0].Priority)
	require.Equal(t, 0, *ruleSet.Segments[0].InputMin)
	require.Equal(t, 100, *ruleSet.Segments[0].InputMax)
	require.Equal(t, 1.2, *ruleSet.Segments[0].InputPrice)
	require.Equal(t, 20, *ruleSet.Segments[1].Priority)
	require.Equal(t, 101, *ruleSet.Segments[1].InputMin)
	require.Equal(t, 200, *ruleSet.Segments[1].InputMax)
	require.Equal(t, 2.4, *ruleSet.Segments[1].InputPrice)
}

func TestParseAdvancedPricingConfigNormalizesLegacyMediaShellRule(t *testing.T) {
	cfg, err := ParseAdvancedPricingConfig(`{
      "rules": {
        "veo-3.1-fast-generate-preview": {
          "rule_type": "media_task",
          "display_name": "ignored shell field",
          "task_type": "video_generation",
          "billing_unit": "task",
          "unit_price": "8.8",
          "note": "preserved as remark"
        }
      }
    }`)
	require.NoError(t, err)

	ruleSet := cfg.ModelRules["veo-3.1-fast-generate-preview"]
	require.Equal(t, RuleTypeMediaTask, ruleSet.RuleType)
	require.Len(t, ruleSet.Segments, 1)
	require.Equal(t, 10, *ruleSet.Segments[0].Priority)
	require.Equal(t, 8.8, *ruleSet.Segments[0].UnitPrice)
	require.Equal(t, "preserved as remark", ruleSet.Segments[0].Remark)
}

func TestUpdateAdvancedPricingRulesByJSONStringNormalizesLegacyShellToCanonical(t *testing.T) {
	original := advancedPricingRulesMap.ReadAll()
	advancedPricingRulesMap.Clear()
	t.Cleanup(func() {
		advancedPricingRulesMap.Clear()
		advancedPricingRulesMap.AddAll(original)
	})

	err := UpdateAdvancedPricingRulesByJSONString(`{
      "gpt-5": {
        "rule_type": "text_segment",
        "display_name": "ignored shell field",
        "segments_text": "0-100: 1.2\n101-200: 2.4"
      }
    }`)
	require.NoError(t, err)

	ruleSet, ok := GetAdvancedPricingRuleSet("gpt-5")
	require.True(t, ok)
	require.Equal(t, RuleTypeTextSegment, ruleSet.RuleType)
	require.Len(t, ruleSet.Segments, 2)
	require.Equal(t, 10, *ruleSet.Segments[0].Priority)
	require.Equal(t, 20, *ruleSet.Segments[1].Priority)

	require.JSONEq(t, `{
      "gpt-5": {
        "rule_type": "text_segment",
        "segments": [
          {
            "priority": 10,
            "input_min": 0,
            "input_max": 100,
            "input_price": 1.2
          },
          {
            "priority": 20,
            "input_min": 101,
            "input_max": 200,
            "input_price": 2.4
          }
        ]
      }
    }`, AdvancedPricingRules2JSONString())
	require.JSONEq(t, `{
      "billing_mode": {},
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "segments": [
            {
              "priority": 10,
              "input_min": 0,
              "input_max": 100,
              "input_price": 1.2
            },
            {
              "priority": 20,
              "input_min": 101,
              "input_max": 200,
              "input_price": 2.4
            }
          ]
        }
      }
    }`, AdvancedPricingConfig2JSONString())
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

func TestUpdateAdvancedPricingConfigByJSONStringUpdatesBothRuntimeMaps(t *testing.T) {
	originalModes := advancedPricingModeMap.ReadAll()
	originalRules := advancedPricingRulesMap.ReadAll()
	advancedPricingModeMap.Clear()
	advancedPricingRulesMap.Clear()
	t.Cleanup(func() {
		advancedPricingModeMap.Clear()
		advancedPricingModeMap.AddAll(originalModes)
		advancedPricingRulesMap.Clear()
		advancedPricingRulesMap.AddAll(originalRules)
	})

	err := UpdateAdvancedPricingConfigByJSONString(`{
      "billing_mode": {
        "gpt-5": "advanced"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "segments": [
            {
              "priority": 10,
              "input_min": 0,
              "input_max": 100,
              "input_price": 1.2
            }
          ]
        }
      }
    }`)
	require.NoError(t, err)

	mode, ok := advancedPricingModeMap.Get("gpt-5")
	require.True(t, ok)
	require.Equal(t, BillingModeAdvanced, mode)

	ruleSet, ok := advancedPricingRulesMap.Get("gpt-5")
	require.True(t, ok)
	require.Equal(t, RuleTypeTextSegment, ruleSet.RuleType)
	require.Len(t, ruleSet.Segments, 1)
	require.Equal(t, 1.2, *ruleSet.Segments[0].InputPrice)
}

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
