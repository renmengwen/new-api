package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAdvancedPricingOptionTest(t *testing.T) *gorm.DB {
	t.Helper()

	originalDB := DB
	originalOptionMap := common.OptionMap
	originalConfigValue := ""
	originalModeValue := ""
	originalRulesValue := ""
	if originalOptionMap != nil {
		originalConfigValue = originalOptionMap["AdvancedPricingConfig"]
		originalModeValue = originalOptionMap["AdvancedPricingMode"]
		originalRulesValue = originalOptionMap["AdvancedPricingRules"]
	}
	originalRuntimeConfig := ratio_setting.AdvancedPricingConfig2JSONString()

	tempDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tempDB.AutoMigrate(&Option{}))

	DB = tempDB
	common.OptionMap = make(map[string]string)
	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(`{}`))
	syncAdvancedPricingOptionViews()

	t.Cleanup(func() {
		DB = originalDB
		common.OptionMap = originalOptionMap
		if common.OptionMap != nil {
			common.OptionMap["AdvancedPricingConfig"] = originalConfigValue
			common.OptionMap["AdvancedPricingMode"] = originalModeValue
			common.OptionMap["AdvancedPricingRules"] = originalRulesValue
		}
		require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(originalRuntimeConfig))
	})

	return tempDB
}

func mustGetOptionValue(t *testing.T, key string) string {
	t.Helper()

	var option Option
	require.NoError(t, DB.Where("key = ?", key).First(&option).Error)
	return option.Value
}

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

func TestUpdateOptionAdvancedPricingModeSyncsConfig(t *testing.T) {
	setupAdvancedPricingOptionTest(t)

	initialConfig := `{
      "billing_mode": {
        "gpt-5": "per_token"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "display_name": "Persisted text",
          "segment_basis": "character",
          "billing_unit": "1M chars",
          "default_price": 6.6,
          "note": "persisted note",
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
	require.NoError(t, UpdateOption("AdvancedPricingConfig", initialConfig))

	staleRuntimeConfig := `{
      "billing_mode": {
        "legacy-model": "per_request"
      },
      "rules": {
        "legacy-model": {
          "rule_type": "media_task",
          "segments": [
            {
              "priority": 10,
              "unit_price": 9.9
            }
          ]
        }
      }
    }`
	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(staleRuntimeConfig))
	syncAdvancedPricingOptionViews()

	err := UpdateOption("AdvancedPricingMode", `{"gpt-5":"advanced"}`)
	require.NoError(t, err)

	expectedConfig := `{
      "billing_mode": {
        "gpt-5": "advanced"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "display_name": "Persisted text",
          "segment_basis": "character",
          "billing_unit": "1M chars",
          "default_price": 6.6,
          "note": "persisted note",
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
	require.JSONEq(t, expectedConfig, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, expectedConfig, mustGetOptionValue(t, "AdvancedPricingConfig"))
	require.JSONEq(t, `{"gpt-5":"advanced"}`, mustGetOptionValue(t, "AdvancedPricingMode"))
	require.JSONEq(t, `{
      "gpt-5": {
        "rule_type": "text_segment",
        "display_name": "Persisted text",
        "segment_basis": "character",
        "billing_unit": "1M chars",
        "default_price": 6.6,
        "note": "persisted note",
        "segments": [
          {
            "priority": 10,
            "input_min": 0,
            "input_max": 100,
            "input_price": 1.2
          }
        ]
      }
    }`, mustGetOptionValue(t, "AdvancedPricingRules"))
}

func TestUpdateOptionAdvancedPricingRulesSyncsConfig(t *testing.T) {
	setupAdvancedPricingOptionTest(t)

	initialConfig := `{
      "billing_mode": {
        "gpt-5": "advanced"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "media_task",
          "display_name": "Persisted media",
          "task_type": "image_generation",
          "billing_unit": "task",
          "note": "persisted media note",
          "segments": [
            {
              "priority": 10,
              "unit_price": 2.5
            }
          ]
        }
      }
    }`
	require.NoError(t, UpdateOption("AdvancedPricingConfig", initialConfig))

	staleRuntimeConfig := `{
      "billing_mode": {
        "legacy-model": "per_request"
      },
      "rules": {
        "legacy-model": {
          "rule_type": "text_segment",
          "segments": [
            {
              "priority": 10,
              "input_min": 0,
              "input_max": 10,
              "input_price": 0.5
            }
          ]
        }
      }
    }`
	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(staleRuntimeConfig))
	syncAdvancedPricingOptionViews()

	rulesValue := `{
      "gpt-5": {
        "rule_type": "text_segment",
        "display_name": "Updated text",
        "segment_basis": "token",
        "billing_unit": "1K tokens",
        "default_price": 3.3,
        "note": "updated note",
        "segments": [
          {
            "priority": 10,
            "input_min": 0,
            "input_max": 100,
            "input_price": 1.2
          }
        ]
      }
    }`
	err := UpdateOption("AdvancedPricingRules", rulesValue)
	require.NoError(t, err)

	expectedConfig := `{
      "billing_mode": {
        "gpt-5": "advanced"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "display_name": "Updated text",
          "segment_basis": "token",
          "billing_unit": "1K tokens",
          "default_price": 3.3,
          "note": "updated note",
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
	require.JSONEq(t, expectedConfig, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, expectedConfig, mustGetOptionValue(t, "AdvancedPricingConfig"))
	require.JSONEq(t, `{"gpt-5":"advanced"}`, mustGetOptionValue(t, "AdvancedPricingMode"))
	require.JSONEq(t, rulesValue, mustGetOptionValue(t, "AdvancedPricingRules"))
}

func TestLoadOptionsFromDatabaseKeepsAdvancedPricingViewsConsistentAfterLegacyUpdates(t *testing.T) {
	setupAdvancedPricingOptionTest(t)

	staleConfig := `{"billing_mode":{"legacy-model":"per_request"},"rules":{"legacy-model":{"rule_type":"text_segment","segments":[{"priority":5,"input_min":0,"input_max":10,"input_price":0.5}]}}}`
	require.NoError(t, UpdateOption("AdvancedPricingConfig", staleConfig))

	modeValue := `{"gpt-5":"advanced"}`
	rulesValue := `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`
	require.NoError(t, UpdateOption("AdvancedPricingMode", modeValue))
	require.NoError(t, UpdateOption("AdvancedPricingRules", rulesValue))

	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(`{}`))
	common.OptionMap = make(map[string]string)

	loadOptionsFromDatabase()

	expectedConfig := `{"billing_mode":{"gpt-5":"advanced"},"rules":{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}}`
	require.JSONEq(t, expectedConfig, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, modeValue, common.OptionMap["AdvancedPricingMode"])
	require.JSONEq(t, rulesValue, common.OptionMap["AdvancedPricingRules"])
	require.JSONEq(t, expectedConfig, ratio_setting.AdvancedPricingConfig2JSONString())
	require.JSONEq(t, modeValue, ratio_setting.AdvancedPricingMode2JSONString())
	require.JSONEq(t, rulesValue, ratio_setting.AdvancedPricingRules2JSONString())
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

func TestLoadOptionsFromDatabaseFallsBackToLegacyRowsWhenAdvancedPricingConfigIsEmpty(t *testing.T) {
	setupAdvancedPricingOptionTest(t)

	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingConfig",
		Value: "",
	}).Error)
	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingMode",
		Value: `{"gpt-5":"advanced"}`,
	}).Error)
	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingRules",
		Value: `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`,
	}).Error)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(`{}`))
	common.OptionMap = make(map[string]string)

	loadOptionsFromDatabase()

	expectedConfig := `{"billing_mode":{"gpt-5":"advanced"},"rules":{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}}`
	require.JSONEq(t, expectedConfig, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, `{"gpt-5":"advanced"}`, common.OptionMap["AdvancedPricingMode"])
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, common.OptionMap["AdvancedPricingRules"])
	require.JSONEq(t, expectedConfig, ratio_setting.AdvancedPricingConfig2JSONString())
	require.JSONEq(t, `{"gpt-5":"advanced"}`, ratio_setting.AdvancedPricingMode2JSONString())
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, ratio_setting.AdvancedPricingRules2JSONString())
}

func TestLoadOptionsFromDatabaseFallsBackToLegacyRowsWhenAdvancedPricingConfigIsInvalid(t *testing.T) {
	setupAdvancedPricingOptionTest(t)

	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingConfig",
		Value: `{"billing_mode":`,
	}).Error)
	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingMode",
		Value: `{"gpt-5":"advanced"}`,
	}).Error)
	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingRules",
		Value: `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`,
	}).Error)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(`{}`))
	common.OptionMap = make(map[string]string)

	loadOptionsFromDatabase()

	expectedConfig := `{"billing_mode":{"gpt-5":"advanced"},"rules":{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}}`
	require.JSONEq(t, expectedConfig, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, `{"gpt-5":"advanced"}`, common.OptionMap["AdvancedPricingMode"])
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, common.OptionMap["AdvancedPricingRules"])
	require.JSONEq(t, expectedConfig, ratio_setting.AdvancedPricingConfig2JSONString())
	require.JSONEq(t, `{"gpt-5":"advanced"}`, ratio_setting.AdvancedPricingMode2JSONString())
	require.JSONEq(t, `{"gpt-5":{"rule_type":"text_segment","segments":[{"priority":10,"input_min":0,"input_max":100,"input_price":1.2}]}}`, ratio_setting.AdvancedPricingRules2JSONString())
}

func TestLoadOptionsFromDatabaseDoesNotReviveMissingLegacySiblingFromRuntimeState(t *testing.T) {
	setupAdvancedPricingOptionTest(t)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(`{
      "billing_mode": {
        "legacy-model": "per_request"
      },
      "rules": {
        "legacy-model": {
          "rule_type": "text_segment",
          "segments": [
            {
              "priority": 10,
              "input_min": 0,
              "input_max": 10,
              "input_price": 0.5
            }
          ]
        }
      }
    }`))
	syncAdvancedPricingOptionViews()

	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingMode",
		Value: `{"gpt-5":"advanced"}`,
	}).Error)

	common.OptionMap = make(map[string]string)
	loadOptionsFromDatabase()

	require.JSONEq(t, `{"gpt-5":"advanced"}`, common.OptionMap["AdvancedPricingMode"])
	require.JSONEq(t, `{}`, common.OptionMap["AdvancedPricingRules"])
	require.JSONEq(t, `{"billing_mode":{"gpt-5":"advanced"},"rules":{}}`, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, `{"gpt-5":"advanced"}`, ratio_setting.AdvancedPricingMode2JSONString())
	require.JSONEq(t, `{}`, ratio_setting.AdvancedPricingRules2JSONString())
	require.JSONEq(t, `{"billing_mode":{"gpt-5":"advanced"},"rules":{}}`, ratio_setting.AdvancedPricingConfig2JSONString())
}

func TestLoadOptionsFromDatabaseNormalizesLegacyAdvancedPricingRulesShell(t *testing.T) {
	setupAdvancedPricingOptionTest(t)

	require.NoError(t, DB.Create(&Option{
		Key:   "AdvancedPricingMode",
		Value: `{"gpt-5":"advanced"}`,
	}).Error)
	require.NoError(t, DB.Create(&Option{
		Key: "AdvancedPricingRules",
		Value: `{
      "gpt-5": {
        "rule_type": "text_segment",
        "display_name": "Text shell",
        "segment_basis": "character",
        "billing_unit": "1M chars",
        "default_price": "9.9",
        "note": "preserved note",
        "segments_text": "0-100: 1.2\n101-200: 2.4"
      }
    }`,
	}).Error)

	require.NoError(t, ratio_setting.UpdateAdvancedPricingConfigByJSONString(`{}`))
	common.OptionMap = make(map[string]string)

	loadOptionsFromDatabase()

	expectedRules := `{
      "gpt-5": {
        "rule_type": "text_segment",
        "display_name": "Text shell",
        "segment_basis": "character",
        "billing_unit": "1M chars",
        "default_price": 9.9,
        "note": "preserved note",
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
    }`
	expectedConfig := `{
      "billing_mode": {
        "gpt-5": "advanced"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "display_name": "Text shell",
          "segment_basis": "character",
          "billing_unit": "1M chars",
          "default_price": 9.9,
          "note": "preserved note",
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
    }`

	require.JSONEq(t, `{"gpt-5":"advanced"}`, common.OptionMap["AdvancedPricingMode"])
	require.JSONEq(t, expectedRules, common.OptionMap["AdvancedPricingRules"])
	require.JSONEq(t, expectedConfig, common.OptionMap["AdvancedPricingConfig"])
	require.JSONEq(t, expectedRules, ratio_setting.AdvancedPricingRules2JSONString())
	require.JSONEq(t, expectedConfig, ratio_setting.AdvancedPricingConfig2JSONString())
}
