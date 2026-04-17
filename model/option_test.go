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
