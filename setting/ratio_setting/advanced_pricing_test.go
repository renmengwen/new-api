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
            {"priority": 10, "input_min": 0, "input_max": 32, "input_price": 3.2, "output_price": 16}
          ]
        },
        "doubao-seedance-2.0": {
          "rule_type": "media_task",
          "segments": [
            {"priority": 10, "inference_mode": "online", "input_video": false, "resolution": "720p", "unit_price": 28, "min_tokens": 194400}
          ]
        }
      }
    }`

	cfg, err := ParseAdvancedPricingConfig(jsonStr)
	require.NoError(t, err)
	require.Equal(t, BillingModeAdvanced, cfg.ModelModes["doubao-seed-2.0-pro"])
	require.Equal(t, RuleTypeTextSegment, cfg.ModelRules["doubao-seed-2.0-pro"].RuleType)
	require.Equal(t, RuleTypeMediaTask, cfg.ModelRules["doubao-seedance-2.0"].RuleType)
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
