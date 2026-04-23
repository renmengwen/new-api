package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSmartExportUsageLogsReturnsSyncWithinDefaultThreshold(t *testing.T) {
	setupSmartExportTestDB(t)
	seedUsageLogs(t, SmartExportUsageLogsThreshold, "usage-sync")

	decision, err := DecideUsageLogSmartExport(0, common.RoleRootUser, dto.UsageLogExportRequest{
		Type:       model.LogTypeConsume,
		ModelName:  "usage-sync",
		ColumnKeys: []string{"time", "model"},
	}, nil, true)
	require.NoError(t, err)
	require.Equal(t, SmartExportModeSync, decision.Mode)
	require.Equal(t, SmartExportReasonWithinThreshold, decision.Reason)
	require.Equal(t, SmartExportUsageLogsThreshold, decision.Threshold)
	require.Equal(t, SmartExportUsageLogsThreshold, decision.ProbedRows)
}

func TestSmartExportUsageLogsReturnsAsyncWithinLongTextThreshold(t *testing.T) {
	setupSmartExportTestDB(t)
	seedUsageLogs(t, SmartExportUsageLogsLongTextThreshold+1, "usage-long-text")

	decision, err := DecideUsageLogSmartExport(0, common.RoleRootUser, dto.UsageLogExportRequest{
		Type:      model.LogTypeConsume,
		ModelName: "usage-long-text",
	}, []string{"time", "details"}, true)
	require.NoError(t, err)
	require.Equal(t, SmartExportModeAsync, decision.Mode)
	require.Equal(t, SmartExportReasonExceedsThreshold, decision.Reason)
	require.Equal(t, SmartExportUsageLogsLongTextThreshold, decision.Threshold)
	require.Equal(t, SmartExportUsageLogsLongTextThreshold+1, decision.ProbedRows)
}

func TestSmartExportUsageLogsUsesSelfExportDatasetForEndUsers(t *testing.T) {
	setupSmartExportTestDB(t)
	const userID = 501
	seedUsageLogsForUser(t, SmartExportUsageLogsLongTextThreshold+1, userID, "usage-self")

	decision, err := DecideUsageLogSmartExport(userID, common.RoleCommonUser, dto.UsageLogExportRequest{
		Type:      model.LogTypeConsume,
		ModelName: "usage-self",
	}, []string{"time", "details"}, false)
	require.NoError(t, err)
	require.Equal(t, SmartExportModeAsync, decision.Mode)
	require.Equal(t, SmartExportReasonExceedsThreshold, decision.Reason)
	require.Equal(t, SmartExportUsageLogsLongTextThreshold, decision.Threshold)
	require.Equal(t, SmartExportUsageLogsLongTextThreshold+1, decision.ProbedRows)
}

func TestSmartExportUsageLogsFallsBackToDefaultColumnsWhenSelectedColumnKeysMissing(t *testing.T) {
	setupSmartExportTestDB(t)
	seedUsageLogs(t, SmartExportUsageLogsLongTextThreshold+1, "usage-default-columns")

	decision, err := DecideUsageLogSmartExport(0, common.RoleRootUser, dto.UsageLogExportRequest{
		Type:       model.LogTypeConsume,
		ModelName:  "usage-default-columns",
		ColumnKeys: []string{},
	}, nil, true)
	require.NoError(t, err)
	require.Equal(t, SmartExportModeAsync, decision.Mode)
	require.Equal(t, SmartExportReasonExceedsThreshold, decision.Reason)
	require.Equal(t, SmartExportUsageLogsLongTextThreshold, decision.Threshold)
	require.Equal(t, SmartExportUsageLogsLongTextThreshold+1, decision.ProbedRows)
}

func TestSmartExportAdminAuditReturnsSyncWithinRequestedLimit(t *testing.T) {
	setupSmartExportTestDB(t)
	seedAdminAuditLogs(t, SmartExportAdminAuditThreshold, "quota")

	decision, err := DecideAdminAuditLogSmartExport(0, common.RoleRootUser, dto.AdminAuditExportRequest{
		ActionModule: "quota",
		Limit:        10,
	})
	require.NoError(t, err)
	require.Equal(t, SmartExportModeSync, decision.Mode)
	require.Equal(t, SmartExportReasonWithinThreshold, decision.Reason)
	require.Equal(t, smartExportSyncLimit, decision.Threshold)
	require.Equal(t, 10, decision.ProbedRows)
}

func TestSmartExportAdminAuditReturnsAsyncWhenFullExportWouldExceedSyncCap(t *testing.T) {
	setupSmartExportTestDB(t)
	seedAdminAuditLogs(t, SmartExportAdminAuditThreshold+1, "quota-cap")

	decision, err := DecideAdminAuditLogSmartExport(0, common.RoleRootUser, dto.AdminAuditExportRequest{
		ActionModule: "quota-cap",
	})
	require.NoError(t, err)
	require.Equal(t, SmartExportModeAsync, decision.Mode)
	require.Equal(t, SmartExportReasonExceedsThreshold, decision.Reason)
	require.Equal(t, smartExportSyncLimit, decision.Threshold)
	require.Equal(t, smartExportSyncLimit+1, decision.ProbedRows)
}

func TestSmartExportQuotaLedgerReturnsSyncWithinRequestedLimit(t *testing.T) {
	setupSmartExportTestDB(t)
	seedQuotaLedgerItems(t, SmartExportQuotaLedgerThreshold+1)

	decision, err := DecideQuotaLedgerSmartExport(0, common.RoleRootUser, dto.AdminQuotaLedgerExportRequest{
		Limit: 10,
	})
	require.NoError(t, err)
	require.Equal(t, SmartExportModeSync, decision.Mode)
	require.Equal(t, SmartExportReasonWithinThreshold, decision.Reason)
	require.Equal(t, smartExportSyncLimit, decision.Threshold)
	require.Equal(t, 10, decision.ProbedRows)
}

func TestSmartExportAdminAnalyticsModelsReturnsAsyncWhenFullExportWouldExceedSyncCap(t *testing.T) {
	setupSmartExportTestDB(t)
	seedAnalyticsModelLogs(t, SmartExportAdminAnalyticsThreshold+1)

	decision, err := DecideAdminAnalyticsSmartExport(0, common.RoleRootUser, dto.AdminAnalyticsExportRequest{
		View:           dto.AdminAnalyticsViewModels,
		DatePreset:     dto.AdminAnalyticsDatePresetCustom,
		StartTimestamp: 1,
		EndTimestamp:   1_000_000,
	})
	require.NoError(t, err)
	require.Equal(t, SmartExportModeAsync, decision.Mode)
	require.Equal(t, SmartExportReasonExceedsThreshold, decision.Reason)
	require.Equal(t, smartExportSyncLimit, decision.Threshold)
	require.Equal(t, smartExportSyncLimit+1, decision.ProbedRows)
}

func TestSmartExportAdminAnalyticsUsersReturnsSyncWithinRequestedLimit(t *testing.T) {
	setupSmartExportTestDB(t)
	seedAnalyticsUserLogs(t, SmartExportAdminAnalyticsThreshold)

	decision, err := DecideAdminAnalyticsSmartExport(0, common.RoleRootUser, dto.AdminAnalyticsExportRequest{
		View:           dto.AdminAnalyticsViewUsers,
		DatePreset:     dto.AdminAnalyticsDatePresetCustom,
		StartTimestamp: 1,
		EndTimestamp:   1_000_000,
		Limit:          10,
	})
	require.NoError(t, err)
	require.Equal(t, SmartExportModeSync, decision.Mode)
	require.Equal(t, SmartExportReasonWithinThreshold, decision.Reason)
	require.Equal(t, smartExportSyncLimit, decision.Threshold)
	require.Equal(t, 10, decision.ProbedRows)
}

func TestSmartExportAdminAnalyticsDailyAlwaysReturnsSyncWithoutProbe(t *testing.T) {
	setupSmartExportTestDB(t)
	seedAnalyticsModelLogs(t, SmartExportAdminAnalyticsThreshold+10)

	decision, err := DecideAdminAnalyticsSmartExport(0, common.RoleRootUser, dto.AdminAnalyticsExportRequest{
		View:           dto.AdminAnalyticsViewDaily,
		DatePreset:     dto.AdminAnalyticsDatePresetCustom,
		StartTimestamp: 1,
		EndTimestamp:   1_000_000,
	})
	require.NoError(t, err)
	require.Equal(t, SmartExportModeSync, decision.Mode)
	require.Equal(t, SmartExportReasonAlwaysSync, decision.Reason)
	require.Equal(t, 0, decision.Threshold)
	require.Equal(t, 0, decision.ProbedRows)
}

func setupSmartExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalRedisEnabled := common.RedisEnabled
	originalMemoryCacheEnabled := common.MemoryCacheEnabled

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", sanitizeSmartExportTestName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Log{},
		&model.AdminAuditLog{},
		&model.QuotaAccount{},
		&model.QuotaLedger{},
	))

	t.Cleanup(func() {
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		common.RedisEnabled = originalRedisEnabled
		common.MemoryCacheEnabled = originalMemoryCacheEnabled

		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func sanitizeSmartExportTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_", "'", "_")
	return replacer.Replace(name)
}

func seedUsageLogs(t *testing.T, count int, modelName string) {
	t.Helper()

	seedUsageLogsForUser(t, count, 100, modelName)
}

func seedUsageLogsForUser(t *testing.T, count int, userID int, modelName string) {
	t.Helper()

	logs := make([]model.Log, 0, count)
	for index := 0; index < count; index++ {
		logs = append(logs, model.Log{
			UserId:           userID,
			CreatedAt:        int64(index + 1),
			Type:             model.LogTypeConsume,
			Content:          fmt.Sprintf("content-%d", index),
			Username:         "usage-user",
			TokenName:        "usage-token",
			ModelName:        modelName,
			Quota:            1,
			PromptTokens:     1,
			CompletionTokens: 1,
			Group:            "default",
			Other:            fmt.Sprintf("{\"row\":%d}", index),
		})
	}
	require.NoError(t, model.LOG_DB.CreateInBatches(logs, 500).Error)
}

func seedAdminAuditLogs(t *testing.T, count int, actionModule string) {
	t.Helper()

	items := make([]model.AdminAuditLog, 0, count)
	for index := 0; index < count; index++ {
		items = append(items, model.AdminAuditLog{
			OperatorUserId:   10,
			OperatorUserType: model.UserTypeAdmin,
			ActionModule:     actionModule,
			ActionType:       "update",
			TargetType:       "user",
			TargetId:         index + 1,
			Ip:               "127.0.0.1",
			CreatedAtTs:      int64(index + 1),
		})
	}
	require.NoError(t, model.DB.CreateInBatches(items, 500).Error)
}

func seedQuotaLedgerItems(t *testing.T, count int) {
	t.Helper()

	account := model.QuotaAccount{
		OwnerType:   model.QuotaOwnerTypeUser,
		OwnerId:     2001,
		Balance:     count,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: 1,
		UpdatedAtTs: 1,
	}
	require.NoError(t, model.DB.Create(&account).Error)

	ledgers := make([]model.QuotaLedger, 0, count)
	for index := 0; index < count; index++ {
		ledgers = append(ledgers, model.QuotaLedger{
			BizNo:            fmt.Sprintf("ledger-%d", index+1),
			AccountId:        account.Id,
			EntryType:        model.LedgerEntryAdjust,
			Direction:        model.LedgerDirectionIn,
			Amount:           1,
			BalanceBefore:    index,
			BalanceAfter:     index + 1,
			SourceType:       "test",
			SourceId:         index + 1,
			OperatorUserId:   99,
			OperatorUserType: model.UserTypeAdmin,
			Reason:           "seed",
			CreatedAtTs:      int64(index + 1),
		})
	}
	require.NoError(t, model.DB.CreateInBatches(ledgers, 500).Error)
}

func seedAnalyticsModelLogs(t *testing.T, count int) {
	t.Helper()

	logs := make([]model.Log, 0, count)
	for index := 0; index < count; index++ {
		logs = append(logs, model.Log{
			UserId:           3001,
			CreatedAt:        int64(index + 1),
			Type:             model.LogTypeConsume,
			Username:         "analytics-model-user",
			ModelName:        fmt.Sprintf("analytics-model-%05d", index+1),
			Quota:            1,
			PromptTokens:     1,
			CompletionTokens: 1,
		})
	}
	require.NoError(t, model.LOG_DB.CreateInBatches(logs, 500).Error)
}

func seedAnalyticsUserLogs(t *testing.T, count int) {
	t.Helper()

	logs := make([]model.Log, 0, count)
	for index := 0; index < count; index++ {
		logs = append(logs, model.Log{
			UserId:           4000 + index + 1,
			CreatedAt:        int64(index + 1),
			Type:             model.LogTypeConsume,
			Username:         fmt.Sprintf("analytics-user-%05d", index+1),
			ModelName:        "analytics-user-model",
			Quota:            1,
			PromptTokens:     1,
			CompletionTokens: 1,
		})
	}
	require.NoError(t, model.LOG_DB.CreateInBatches(logs, 500).Error)
}
