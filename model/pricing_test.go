package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func resetPricingCacheForTest() {
	pricingMap = nil
	vendorsList = nil
	supportedEndpointMap = nil
	lastGetPricingTime = time.Time{}
	modelEnableGroups = make(map[string][]string)
	modelQuotaTypeMap = make(map[string]int)
	modelSupportEndpointTypes = make(map[string][]constant.EndpointType)
}

func TestGetPricingSerializesAdvancedTextSegmentRules(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Ability{}, &Model{}, &Vendor{}))
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM models").Error)
	require.NoError(t, DB.Exec("DELETE FROM vendors").Error)

	previousAdvancedPricing := ratio_setting.AdvancedPricingConfig2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(previousAdvancedPricing))
		resetPricingCacheForTest()
	})

	const modelName = "gpt-tiered-pricing-test"
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     modelName,
		ChannelId: 90101,
		Enabled:   true,
	}).Error)
	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(`{
		"billing_mode": {"gpt-tiered-pricing-test": "advanced"},
		"rules": {
			"gpt-tiered-pricing-test": {
				"rule_type": "text_segment",
				"segment_basis": "input_tokens",
				"billing_unit": "per_million_tokens",
				"segments": [
					{"priority": 10, "input_min": 0, "input_max": 272000, "input_price": 5, "output_price": 30, "cache_read_price": 0.5},
					{"priority": 20, "input_min": 272001, "input_price": 10, "output_price": 45, "cache_read_price": 1}
				]
			}
		}
	}`))

	RefreshPricing()
	pricings := GetPricing()

	var pricing Pricing
	for _, candidate := range pricings {
		if candidate.ModelName == modelName {
			pricing = candidate
			break
		}
	}
	require.Equal(t, modelName, pricing.ModelName)

	payload, err := common.Marshal(pricing)
	require.NoError(t, err)

	var serialized map[string]any
	require.NoError(t, common.Unmarshal(payload, &serialized))
	require.Equal(t, "advanced", serialized["billing_mode"])

	ruleSet, ok := serialized["advanced_rule_set"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "text_segment", ruleSet["rule_type"])
	require.Equal(t, "input_tokens", ruleSet["segment_basis"])

	segments, ok := ruleSet["segments"].([]any)
	require.True(t, ok)
	require.Len(t, segments, 2)

	firstSegment, ok := segments[0].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 5, firstSegment["input_price"])
	require.EqualValues(t, 30, firstSegment["output_price"])
	require.EqualValues(t, 0.5, firstSegment["cache_read_price"])
}
