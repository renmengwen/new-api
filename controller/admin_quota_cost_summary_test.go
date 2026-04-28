package controller

import (
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
