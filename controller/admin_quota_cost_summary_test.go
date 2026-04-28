package controller

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNormalizeAdminQuotaCostSummaryQueryDefaultsToLast7Days(t *testing.T) {
	query, err := dto.NormalizeAdminQuotaCostSummaryQuery(dto.AdminQuotaCostSummaryQuery{}, 1714320000)
	require.NoError(t, err)
	require.Equal(t, int64(1713715200), query.StartTimestamp)
	require.Equal(t, int64(1714320000), query.EndTimestamp)
	require.Equal(t, "date", query.SortBy)
	require.Equal(t, "desc", query.SortOrder)
}

func TestNormalizeAdminQuotaCostSummaryQueryRejectsRangeOver90Days(t *testing.T) {
	_, err := dto.NormalizeAdminQuotaCostSummaryQuery(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714320000 - 91*24*60*60,
		EndTimestamp:   1714320000,
	}, 1714320000)
	require.Error(t, err)
	require.Contains(t, err.Error(), "date range cannot exceed 90 days")
}

func TestNormalizeAdminQuotaCostSummaryQueryNormalizesSort(t *testing.T) {
	query, err := dto.NormalizeAdminQuotaCostSummaryQuery(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
		SortBy:         "paid_usd",
		SortOrder:      "ASC",
	}, 1714320000)
	require.NoError(t, err)
	require.Equal(t, "paid_usd", query.SortBy)
	require.Equal(t, "asc", query.SortOrder)
}

func seedQuotaCostSummaryLog(t *testing.T, db *gorm.DB, log model.Log) model.Log {
	t.Helper()
	require.NoError(t, db.Create(&log).Error)
	return log
}

func seedQuotaCostSummaryVendorModel(t *testing.T, db *gorm.DB, vendorName string, modelName string) {
	t.Helper()
	vendor := model.Vendor{Name: vendorName, Status: model.CommonStatusEnabled}
	require.NoError(t, vendor.Insert())
	require.NoError(t, db.Create(&model.Model{
		ModelName: modelName,
		VendorID:  vendor.Id,
		Status:    model.CommonStatusEnabled,
	}).Error)
}

func setQuotaCostSummaryQuotaPerUnit(t *testing.T, quotaPerUnit float64) {
	t.Helper()
	previous := common.QuotaPerUnit
	common.QuotaPerUnit = quotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = previous
	})
}

func requireQuotaCostSummaryItem(t *testing.T, items []dto.AdminQuotaCostSummaryItem, date string, modelName string, vendorName string) dto.AdminQuotaCostSummaryItem {
	t.Helper()
	for _, item := range items {
		if item.Date == date && item.ModelName == modelName && item.VendorName == vendorName {
			return item
		}
	}
	require.Failf(t, "missing quota cost summary item", "date=%s model=%s vendor=%s", date, modelName, vendorName)
	return dto.AdminQuotaCostSummaryItem{}
}

func seedQuotaCostSummaryRows(t *testing.T, db *gorm.DB, total int, modelPrefix string) model.User {
	t.Helper()
	user := seedQuotaUser(t, db, modelPrefix+"_user", 0)
	logs := make([]model.Log, 0, total)
	for index := 0; index < total; index++ {
		logs = append(logs, model.Log{
			UserId:           user.Id,
			Username:         user.Username,
			Type:             model.LogTypeConsume,
			CreatedAt:        1714237200 + int64(index),
			ModelName:        fmt.Sprintf("%s-%04d", modelPrefix, index),
			TokenName:        "token-" + modelPrefix,
			PromptTokens:     1,
			CompletionTokens: 1,
			Quota:            1,
			ChannelId:        7,
			Group:            "default",
		})
	}
	require.NoError(t, db.CreateInBatches(logs, 500).Error)
	return user
}

func TestListQuotaCostSummaryAggregatesByDateModelAndVendor(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 500000)
	user := seedQuotaUser(t, db, "summary_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "OpenAI", "gpt-test")
	seedQuotaCostSummaryVendorModel(t, db, "Anthropic", "claude-test")

	otherA, err := common.Marshal(map[string]any{
		"model_ratio":      2.0,
		"group_ratio":      1.0,
		"completion_ratio": 3.0,
		"cache_tokens":     20,
		"cache_ratio":      0.5,
	})
	require.NoError(t, err)
	otherB, err := common.Marshal(map[string]any{
		"model_ratio":      2.0,
		"group_ratio":      1.0,
		"completion_ratio": 3.0,
		"cache_tokens":     10,
		"cache_ratio":      0.5,
	})
	require.NoError(t, err)

	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "gpt-test", TokenName: "token-a",
		PromptTokens: 100, CompletionTokens: 50, Quota: 1200, ChannelId: 7,
		Group: "default", Other: string(otherA),
	})
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714240800, ModelName: "gpt-test", TokenName: "token-a",
		PromptTokens: 80, CompletionTokens: 20, Quota: 800, ChannelId: 7,
		Group: "default", Other: string(otherB),
	})
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714239000, ModelName: "claude-test", TokenName: "token-b",
		PromptTokens: 10, CompletionTokens: 5, Quota: 500, ChannelId: 8,
		Group: "default", Other: string(otherA),
	})
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714323600, ModelName: "gpt-test", TokenName: "token-a",
		PromptTokens: 7, CompletionTokens: 3, Quota: 100, ChannelId: 7,
		Group: "default", Other: string(otherA),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714406400,
		SortBy:         "date",
		SortOrder:      "desc",
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, items, 3)

	gptApr28 := requireQuotaCostSummaryItem(t, items, "2024-04-28", "gpt-test", "OpenAI")
	require.EqualValues(t, 2, gptApr28.CallCount)
	require.EqualValues(t, 180, gptApr28.InputTokens)
	require.EqualValues(t, 70, gptApr28.OutputTokens)
	require.EqualValues(t, 30, gptApr28.CacheReadTokens)
	require.InDelta(t, 0.004, gptApr28.PaidUSD, 0.000001)

	claudeApr28 := requireQuotaCostSummaryItem(t, items, "2024-04-28", "claude-test", "Anthropic")
	require.EqualValues(t, 1, claudeApr28.CallCount)
	require.EqualValues(t, 10, claudeApr28.InputTokens)

	gptApr29 := requireQuotaCostSummaryItem(t, items, "2024-04-29", "gpt-test", "OpenAI")
	require.EqualValues(t, 1, gptApr29.CallCount)
	require.EqualValues(t, 7, gptApr29.InputTokens)
}

func TestListQuotaCostSummaryScalesLegacyModelRatioByQuotaPerUnit(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "ratio_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "OpenAI", "ratio-model")

	other, err := common.Marshal(map[string]any{
		"model_ratio": 2.0,
		"group_ratio": 1.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "ratio-model",
		PromptTokens: 100, Quota: 1000000, Group: "default", Other: string(other),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.InDelta(t, 2.0, items[0].InputUnitPriceUSD, 0.000001)
	require.InDelta(t, 0.0002, items[0].InputCostUSD, 0.000001)
}

func TestListQuotaCostSummaryPreservesExplicitZeroUserGroupRatio(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "zero_group_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "OpenAI", "zero-group-model")

	other, err := common.Marshal(map[string]any{
		"model_ratio":      2.0,
		"group_ratio":      1.5,
		"user_group_ratio": 0.0,
		"completion_ratio": 3.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "zero-group-model",
		PromptTokens: 100, CompletionTokens: 50, Quota: 123000, Group: "default", Other: string(other),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Zero(t, items[0].InputCostUSD)
	require.Zero(t, items[0].OutputCostUSD)
	require.Zero(t, items[0].TotalCostUSD)
	require.InDelta(t, 0.123, items[0].PaidUSD, 0.000001)
}

func TestListQuotaCostSummaryReturnsEffectiveUnitPricesWithGroupRatio(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "effective_unit_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "OpenAI", "effective-unit-model")

	other, err := common.Marshal(map[string]any{
		"model_ratio":           2.0,
		"group_ratio":           2.0,
		"user_group_ratio":      -1.0,
		"completion_ratio":      3.0,
		"cache_tokens":          40,
		"cache_ratio":           0.5,
		"cache_creation_tokens": 60,
		"cache_creation_ratio":  2.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "effective-unit-model",
		PromptTokens: 200, CompletionTokens: 50, Quota: 1000, Group: "default", Other: string(other),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.InDelta(t, 4.0, items[0].InputUnitPriceUSD, 0.000001)
	require.InDelta(t, 12.0, items[0].OutputUnitPriceUSD, 0.000001)
	require.InDelta(t, 2.0, items[0].CacheReadUnitPrice, 0.000001)
	require.InDelta(t, 8.0, items[0].CacheCreateUnitPrice, 0.000001)
	require.InDelta(t, 0.0004, items[0].InputCostUSD, 0.000001)
	require.InDelta(t, 0.0006, items[0].OutputCostUSD, 0.000001)
	require.InDelta(t, 0.00056, items[0].CacheCostUSD, 0.000001)
}

func TestListQuotaCostSummaryWeightsZeroPriceRowsInUnitPrices(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "mixed_free_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "OpenAI", "mixed-free-model")

	freeOther, err := common.Marshal(map[string]any{
		"model_ratio":      2.0,
		"group_ratio":      1.0,
		"user_group_ratio": 0.0,
	})
	require.NoError(t, err)
	chargedOther, err := common.Marshal(map[string]any{
		"model_ratio":      2.0,
		"group_ratio":      1.0,
		"user_group_ratio": -1.0,
	})
	require.NoError(t, err)

	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "mixed-free-model",
		PromptTokens: 100, Quota: 0, Group: "default", Other: string(freeOther),
	})
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237300, ModelName: "mixed-free-model",
		PromptTokens: 100, Quota: 200, Group: "default", Other: string(chargedOther),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.EqualValues(t, 200, items[0].InputTokens)
	require.InDelta(t, 1.0, items[0].InputUnitPriceUSD, 0.000001)
	require.InDelta(t, 0.0002, items[0].InputCostUSD, 0.000001)
	require.InDelta(t, 0.0002, items[0].TotalCostUSD, 0.000001)
}

func TestListQuotaCostSummaryUsesSplitCacheCreationRatios(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "cache_create_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "Anthropic", "cache-create-model")

	other, err := common.Marshal(map[string]any{
		"model_ratio":              1.0,
		"group_ratio":              1.0,
		"cache_creation_tokens":    100,
		"cache_creation_ratio":     2.0,
		"cache_creation_tokens_5m": 40,
		"cache_creation_ratio_5m":  4.0,
		"cache_creation_tokens_1h": 30,
		"cache_creation_ratio_1h":  6.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "cache-create-model",
		PromptTokens: 100, Quota: 460, Group: "default", Other: string(other),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.EqualValues(t, 100, items[0].CacheCreateTokens)
	require.InDelta(t, 4.0, items[0].CacheCreateUnitPrice, 0.000001)
	require.InDelta(t, 0.0004, items[0].CacheCostUSD, 0.000001)
	require.InDelta(t, 0.0004, items[0].TotalCostUSD, 0.000001)
}

func TestListQuotaCostSummaryTreatsModelPriceAsFixedCost(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "fixed_price_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "TaskVendor", "fixed-price-model")

	other, err := common.Marshal(map[string]any{
		"model_price": 0.25,
		"group_ratio": 2.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "fixed-price-model",
		Quota: 500000, Group: "default", Other: string(other),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Zero(t, items[0].InputUnitPriceUSD)
	require.InDelta(t, 0.5, items[0].InputCostUSD, 0.000001)
	require.InDelta(t, 0.5, items[0].TotalCostUSD, 0.000001)
	require.InDelta(t, 0.5, items[0].PaidUSD, 0.000001)
}

func TestListQuotaCostSummaryDoesNotTreatAdvancedModelPriceAsFixedCost(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "advanced_price_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "AdvancedVendor", "advanced-price-model")

	other, err := common.Marshal(map[string]any{
		"billing_mode": "advanced",
		"model_price":  0.25,
		"group_ratio":  2.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "advanced-price-model",
		Quota: 500000, Group: "default", Other: string(other),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Zero(t, items[0].InputUnitPriceUSD)
	require.Zero(t, items[0].InputCostUSD)
	require.Zero(t, items[0].TotalCostUSD)
	require.InDelta(t, 0.5, items[0].PaidUSD, 0.000001)
}

func TestListQuotaCostSummaryUsesAdvancedTokenPricingFromRatios(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "advanced_token_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "AdvancedVendor", "advanced-token-model")

	other, err := common.Marshal(map[string]any{
		"billing_mode":     "advanced",
		"model_price":      0.25,
		"model_ratio":      1.0,
		"completion_ratio": 2.0,
		"group_ratio":      2.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "advanced-token-model",
		PromptTokens: 100, CompletionTokens: 50, Quota: 400, Group: "default", Other: string(other),
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.InDelta(t, 2.0, items[0].InputUnitPriceUSD, 0.000001)
	require.InDelta(t, 4.0, items[0].OutputUnitPriceUSD, 0.000001)
	require.InDelta(t, 0.0002, items[0].InputCostUSD, 0.000001)
	require.InDelta(t, 0.0002, items[0].OutputCostUSD, 0.000001)
	require.InDelta(t, 0.0004, items[0].TotalCostUSD, 0.000001)
	require.InDelta(t, 0.0004, items[0].PaidUSD, 0.000001)
}

func TestListQuotaCostSummaryRejectsInvalidRawQuery(t *testing.T) {
	setupAdminQuotaTestDB(t)

	_, _, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714320000,
		EndTimestamp:   1714233600,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.Error(t, err)
	require.Contains(t, err.Error(), "end_timestamp must be greater than or equal to start_timestamp")

	_, err = service.ListQuotaCostSummaryForExport(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714320000 - 91*24*60*60,
		EndTimestamp:   1714320000,
	}, 999, common.RoleRootUser, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "date range cannot exceed 90 days")
}

func TestListQuotaCostSummaryUsesDeterministicTieBreakers(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "tie_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "VendorC", "gamma-model")
	seedQuotaCostSummaryVendorModel(t, db, "VendorA", "alpha-model")
	seedQuotaCostSummaryVendorModel(t, db, "VendorB", "beta-model")

	for _, modelName := range []string{"gamma-model", "alpha-model", "beta-model"} {
		seedQuotaCostSummaryLog(t, db, model.Log{
			UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
			CreatedAt: 1714237200, ModelName: modelName,
			Quota: 1000, Group: "default",
		})
	}

	for range 10 {
		items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
			StartTimestamp: 1714233600,
			EndTimestamp:   1714320000,
			SortBy:         "paid_usd",
			SortOrder:      "asc",
		}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
		require.NoError(t, err)
		require.EqualValues(t, 3, total)
		require.Len(t, items, 3)
		require.Equal(t, []string{"alpha-model", "beta-model", "gamma-model"}, []string{
			items[0].ModelName,
			items[1].ModelName,
			items[2].ModelName,
		})
	}
}

func TestListQuotaCostSummaryFiltersByVendorAndMinimums(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "vendor_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "OpenAI", "openai-model")
	seedQuotaCostSummaryVendorModel(t, db, "Anthropic", "claude-model")
	seedQuotaCostSummaryVendorModel(t, db, "Google", "gemini-model")

	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "openai-model", PromptTokens: 100,
		CompletionTokens: 50, Quota: 1000000, Group: "default",
	})
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237300, ModelName: "openai-model", PromptTokens: 100,
		CompletionTokens: 50, Quota: 1000000, Group: "default",
	})
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "claude-model", PromptTokens: 100,
		CompletionTokens: 50, Quota: 50000, Group: "default",
	})
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237300, ModelName: "claude-model", PromptTokens: 100,
		CompletionTokens: 50, Quota: 50000, Group: "default",
	})
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "gemini-model", PromptTokens: 100,
		CompletionTokens: 50, Quota: 2000000, Group: "default",
	})

	items, total, err := service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
		Vendor:         "OpenAI",
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "openai-model", items[0].ModelName)
	require.Equal(t, "OpenAI", items[0].VendorName)

	items, total, err = service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
		MinCallCount:   2,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	requireQuotaCostSummaryItem(t, items, "2024-04-28", "openai-model", "OpenAI")
	requireQuotaCostSummaryItem(t, items, "2024-04-28", "claude-model", "Anthropic")

	items, total, err = service.ListQuotaCostSummary(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
		MinPaidUSD:     1.5,
	}, &common.PageInfo{Page: 1, PageSize: 10}, 999, common.RoleRootUser)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	requireQuotaCostSummaryItem(t, items, "2024-04-28", "openai-model", "OpenAI")
	requireQuotaCostSummaryItem(t, items, "2024-04-28", "gemini-model", "Google")
}

func TestGetQuotaCostSummaryReturnsPagedItems(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "summary_controller_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "OpenAI", "controller-model")

	other, err := common.Marshal(map[string]any{
		"model_ratio":      2.0,
		"group_ratio":      1.0,
		"completion_ratio": 3.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "controller-model", TokenName: "token-controller",
		PromptTokens: 100, CompletionTokens: 50, Quota: 1000, ChannelId: 7,
		Group: "default", Other: string(other),
	})

	ctx, recorder := newAdminQuotaContext(t, http.MethodGet, "/api/admin/quota/cost-summary?start_timestamp=1714233600&end_timestamp=1714320000&p=1&page_size=10", nil)

	GetQuotaCostSummary(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Page     int                             `json:"page"`
			PageSize int                             `json:"page_size"`
			Total    int                             `json:"total"`
			Items    []dto.AdminQuotaCostSummaryItem `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Page)
	require.Equal(t, 10, response.Data.PageSize)
	require.Equal(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, "2024-04-28", response.Data.Items[0].Date)
	require.Equal(t, "controller-model", response.Data.Items[0].ModelName)
	require.Equal(t, "OpenAI", response.Data.Items[0].VendorName)
	require.EqualValues(t, 100, response.Data.Items[0].InputTokens)
	require.EqualValues(t, 50, response.Data.Items[0].OutputTokens)
	require.EqualValues(t, 1, response.Data.Items[0].CallCount)
}

func TestExportQuotaCostSummaryWritesExcel(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	user := seedQuotaUser(t, db, "summary_export_user", 0)
	seedQuotaCostSummaryVendorModel(t, db, "OpenAI", "excel-model")

	other, err := common.Marshal(map[string]any{
		"model_ratio":      2.0,
		"group_ratio":      1.0,
		"completion_ratio": 3.0,
	})
	require.NoError(t, err)
	seedQuotaCostSummaryLog(t, db, model.Log{
		UserId: user.Id, Username: user.Username, Type: model.LogTypeConsume,
		CreatedAt: 1714237200, ModelName: "excel-model", TokenName: "token-excel",
		PromptTokens: 100, CompletionTokens: 50, Quota: 1000, ChannelId: 7,
		Group: "default", Other: string(other),
	})

	ctx, recorder := newAdminQuotaContext(t, http.MethodPost, "/api/admin/quota/cost-summary/export-auto", map[string]any{
		"start_timestamp": 1714233600,
		"end_timestamp":   1714320000,
		"model_name":      "excel-model",
		"limit":           10,
	})

	ExportQuotaCostSummaryAuto(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("成本汇总")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, []string{
		"日期",
		"模型名称",
		"供应商名称",
		"结算含税价input",
		"结算含税价output",
		"输入tokens",
		"输出tokens",
		"调用次数",
		"input费用",
		"output费用",
		"缓存创建",
		"缓存读取",
		"缓存创建单价",
		"缓存读取单价",
		"cache的token",
		"cache的金额",
		"总费用（USD）",
		"折扣",
		"实付金额（USD）",
	}, rows[0])
	require.Equal(t, []string{
		"2024-04-28",
		"excel-model",
		"OpenAI",
		"2.000000",
		"6.000000",
		"100",
		"50",
		"1",
		"0.000200",
		"0.000300",
		"0",
		"0",
		"0.000000",
		"0.000000",
		"0",
		"0.000000",
		"0.000500",
		"0.000000",
		"0.001000",
	}, rows[1])
}

func TestExportQuotaCostSummaryAutoKeepsRequestedLimit2001Sync(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	setQuotaCostSummaryQuotaPerUnit(t, 1000000)
	seedQuotaCostSummaryRows(t, db, 2001, "sync-limit-model")

	ctx, recorder := newAdminQuotaContext(t, http.MethodPost, "/api/admin/quota/cost-summary/export-auto", map[string]any{
		"start_timestamp": 1714233600,
		"end_timestamp":   1714320000,
		"limit":           2001,
	})

	ExportQuotaCostSummaryAuto(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("成本汇总")
	require.NoError(t, err)
	require.Len(t, rows, 2002)
}

func TestDecideQuotaCostSummarySmartExportUses5000ThresholdForRequestedLimit5001(t *testing.T) {
	setupAdminQuotaTestDB(t)

	decision, err := service.DecideQuotaCostSummarySmartExport(999, common.RoleRootUser, dto.AdminQuotaCostSummaryExportRequest{
		AdminQuotaCostSummaryQuery: dto.AdminQuotaCostSummaryQuery{
			StartTimestamp: 1714233600,
			EndTimestamp:   1714320000,
		},
		Limit: 5001,
	})

	require.NoError(t, err)
	require.Equal(t, service.SmartExportJobTypeQuotaCostSummary, decision.JobType)
	require.Equal(t, service.SmartExportModeAsync, decision.Mode)
	require.Equal(t, service.SmartExportReasonExceedsThreshold, decision.Reason)
	require.Equal(t, service.SmartExportQuotaCostSummaryThreshold, decision.Threshold)
	require.Equal(t, 5001, decision.ProbedRows)
}

func TestCreateQuotaCostSummaryExportJobCreatesAsyncJob(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AsyncExportJob{}))

	ctx, recorder := newAdminQuotaContext(t, http.MethodPost, "/api/admin/quota/cost-summary/export-jobs", map[string]any{
		"start_timestamp": 1714233600,
		"end_timestamp":   1714320000,
		"model_name":      "async-model",
		"limit":           25,
	})

	CreateQuotaCostSummaryExportJob(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool                       `json:"success"`
		Message string                     `json:"message"`
		Data    dto.AsyncExportJobResponse `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, service.SmartExportJobTypeQuotaCostSummary, response.Data.JobType)
	require.Equal(t, model.AsyncExportStatusQueued, response.Data.Status)

	var job model.AsyncExportJob
	require.NoError(t, db.First(&job, response.Data.Id).Error)
	require.Equal(t, service.SmartExportJobTypeQuotaCostSummary, job.JobType)
	require.Equal(t, 999, job.RequesterUserId)
	require.Equal(t, common.RoleRootUser, job.RequesterRole)

	var payload struct {
		Query dto.AdminQuotaCostSummaryQuery `json:"query"`
		Limit int                            `json:"limit"`
	}
	require.NoError(t, common.UnmarshalJsonStr(job.PayloadJSON, &payload))
	require.Equal(t, int64(1714233600), payload.Query.StartTimestamp)
	require.Equal(t, int64(1714320000), payload.Query.EndTimestamp)
	require.Equal(t, "async-model", payload.Query.ModelName)
	require.Equal(t, "date", payload.Query.SortBy)
	require.Equal(t, "desc", payload.Query.SortOrder)
	require.Equal(t, 25, payload.Limit)
}
