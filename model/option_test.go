package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpdateOptionDoesNotMutateRuntimeStateWhenDBWriteFails(t *testing.T) {
	originalDB := DB
	originalOptionMap := common.OptionMap
	originalOptionValue := ""
	if originalOptionMap != nil {
		originalOptionValue = originalOptionMap["AdvancedPricingMode"]
	} else {
		common.OptionMap = make(map[string]string)
	}
	originalRuntimeValue := ratio_setting.AdvancedPricingMode2JSONString()

	tempDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tempDB.AutoMigrate(&Option{}))

	sqlDB, err := tempDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	DB = tempDB
	common.OptionMap["AdvancedPricingMode"] = `{"existing":"per_token"}`
	require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(`{"existing":"per_token"}`))

	t.Cleanup(func() {
		DB = originalDB
		common.OptionMap = originalOptionMap
		if common.OptionMap != nil {
			common.OptionMap["AdvancedPricingMode"] = originalOptionValue
		}
		require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(originalRuntimeValue))
	})

	err = UpdateOption("AdvancedPricingMode", `{"broken":"advanced"}`)
	require.Error(t, err)
	require.Equal(t, `{"existing":"per_token"}`, common.OptionMap["AdvancedPricingMode"])
	require.Equal(t, `{"existing":"per_token"}`, ratio_setting.AdvancedPricingMode2JSONString())
}

func TestUpdateOptionAdvancedPricingConfigSyncsLegacyViews(t *testing.T) {
	originalDB := DB
	originalOptionMap := common.OptionMap
	originalConfigValue := ""
	if originalOptionMap != nil {
		originalConfigValue = originalOptionMap["AdvancedPricingConfig"]
	} else {
		common.OptionMap = make(map[string]string)
	}
	originalModeValue := ratio_setting.AdvancedPricingMode2JSONString()
	originalRulesValue := ratio_setting.AdvancedPricingRules2JSONString()

	tempDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tempDB.AutoMigrate(&Option{}))

	DB = tempDB
	common.OptionMap["AdvancedPricingConfig"] = "{}"

	t.Cleanup(func() {
		DB = originalDB
		common.OptionMap = originalOptionMap
		if common.OptionMap != nil {
			common.OptionMap["AdvancedPricingConfig"] = originalConfigValue
		}
		require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(originalModeValue))
		require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(originalRulesValue))
	})

	value := `{
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
    }`

	err = UpdateOption("AdvancedPricingConfig", value)
	require.NoError(t, err)
	require.JSONEq(t, value, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, `{"gpt-5":"advanced"}`, common.OptionMap["AdvancedPricingMode"])
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, common.OptionMap["AdvancedPricingRules"])
	require.JSONEq(t, `{"gpt-5":"advanced"}`, ratio_setting.AdvancedPricingMode2JSONString())
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, ratio_setting.AdvancedPricingRules2JSONString())

	var configOption Option
	require.NoError(t, DB.Where("key = ?", "AdvancedPricingConfig").First(&configOption).Error)
	require.JSONEq(t, value, configOption.Value)

	var modeOption Option
	require.NoError(t, DB.Where("key = ?", "AdvancedPricingMode").First(&modeOption).Error)
	require.JSONEq(t, `{"gpt-5":"advanced"}`, modeOption.Value)

	var rulesOption Option
	require.NoError(t, DB.Where("key = ?", "AdvancedPricingRules").First(&rulesOption).Error)
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, rulesOption.Value)
}

func TestLoadOptionsFromDatabasePrefersAdvancedPricingConfigOverLegacyRows(t *testing.T) {
	originalDB := DB
	originalOptionMap := common.OptionMap
	originalModeValue := ratio_setting.AdvancedPricingMode2JSONString()
	originalRulesValue := ratio_setting.AdvancedPricingRules2JSONString()

	tempDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tempDB.AutoMigrate(&Option{}))

	DB = tempDB
	common.OptionMap = make(map[string]string)
	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(`{}`))

	t.Cleanup(func() {
		DB = originalDB
		common.OptionMap = originalOptionMap
		require.NoError(t, ratio_setting.UpdateAdvancedPricingModeByJSONString(originalModeValue))
		require.NoError(t, ratio_setting.UpdateAdvancedPricingRulesByJSONString(originalRulesValue))
	})

	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingConfig",
		Value: `{"billing_mode":{"gpt-5":"advanced"},"rules":{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}}`,
	}).Error)
	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingMode",
		Value: `{"legacy-model":"per_request"}`,
	}).Error)
	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingRules",
		Value: `{"legacy-model":{"rule_type":"media_task","segments":[{"priority":5,"unit_price":9.9}]}}`,
	}).Error)

	loadOptionsFromDatabase()

	require.JSONEq(t, `{"billing_mode":{"gpt-5":"advanced"},"rules":{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}}`, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, `{"gpt-5":"advanced"}`, common.OptionMap["AdvancedPricingMode"])
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, common.OptionMap["AdvancedPricingRules"])
	require.JSONEq(t, `{"gpt-5":"advanced"}`, ratio_setting.AdvancedPricingMode2JSONString())
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, ratio_setting.AdvancedPricingRules2JSONString())
}
