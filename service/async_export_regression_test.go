package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func setupAsyncExportRegressionTestDB(t *testing.T) {
	t.Helper()

	setupSmartExportTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.AsyncExportJob{}))

	originalExportDir := common.ExportDir
	exportDir := t.TempDir()
	common.ExportDir = new(string)
	*common.ExportDir = exportDir

	t.Cleanup(func() {
		common.ExportDir = originalExportDir
	})
}

func TestAsyncExportJobsPreserveUnlimitedLimitInPayload(t *testing.T) {
	setupAsyncExportRegressionTestDB(t)

	usageJob, err := CreateAdminUsageLogExportJob(0, common.RoleRootUser, dto.UsageLogExportRequest{
		Type:      model.LogTypeConsume,
		ModelName: "usage-unlimited",
		Limit:     0,
	})
	require.NoError(t, err)

	var usagePayload asyncUsageLogExportPayload
	require.NoError(t, DecodeAsyncExportPayload(usageJob, &usagePayload))
	require.Zero(t, usagePayload.Limit)

	auditJob, err := CreateAdminAuditExportJob(0, common.RoleRootUser, dto.AdminAuditExportRequest{
		Limit: 0,
	})
	require.NoError(t, err)

	var auditPayload asyncAdminAuditExportPayload
	require.NoError(t, DecodeAsyncExportPayload(auditJob, &auditPayload))
	require.Zero(t, auditPayload.Limit)

	quotaJob, err := CreateQuotaLedgerExportJob(0, common.RoleRootUser, dto.AdminQuotaLedgerExportRequest{
		Limit: 0,
	})
	require.NoError(t, err)

	var quotaPayload asyncQuotaLedgerExportPayload
	require.NoError(t, DecodeAsyncExportPayload(quotaJob, &quotaPayload))
	require.Zero(t, quotaPayload.Limit)

	analyticsJob, err := CreateAdminAnalyticsExportJob(0, common.RoleRootUser, dto.AdminAnalyticsExportRequest{
		View:           dto.AdminAnalyticsViewModels,
		DatePreset:     dto.AdminAnalyticsDatePresetCustom,
		StartTimestamp: 1,
		EndTimestamp:   1_000_000,
		Limit:          0,
	})
	require.NoError(t, err)

	var analyticsPayload asyncAdminAnalyticsExportPayload
	require.NoError(t, DecodeAsyncExportPayload(analyticsJob, &analyticsPayload))
	require.Zero(t, analyticsPayload.Limit)
}

func TestExecuteUsageLogAsyncExportJobExportsAllRowsWhenLimitUnset(t *testing.T) {
	setupAsyncExportRegressionTestDB(t)
	seedUsageLogs(t, 2501, "usage-async-unlimited")

	job, err := CreateAdminUsageLogExportJob(0, common.RoleRootUser, dto.UsageLogExportRequest{
		Type:      model.LogTypeConsume,
		ModelName: "usage-async-unlimited",
		Limit:     0,
	})
	require.NoError(t, err)

	require.NoError(t, executeUsageLogAsyncExportJob(job))

	var persisted model.AsyncExportJob
	require.NoError(t, model.DB.First(&persisted, job.Id).Error)
	require.Equal(t, model.AsyncExportStatusSucceeded, persisted.Status)
	require.EqualValues(t, 2501, persisted.RowCount)
	require.NotEmpty(t, persisted.FilePath)

	workbook, err := excelize.OpenFile(persisted.FilePath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, workbook.Close())
	})

	require.Equal(t, "使用日志", workbook.GetSheetName(0))
	rows, err := workbook.GetRows("使用日志")
	require.NoError(t, err)
	require.Len(t, rows, 2502)
	require.Equal(t, []string{"时间", "渠道", "用户", "令牌", "分组", "类型", "模型", "用时/首字", "输入", "输出", "花费", "重试", "IP", "详情"}, rows[0])
}

func TestRequeueStaleAsyncExportJobsResetsTimedOutRunningJobs(t *testing.T) {
	db := setupAsyncExportServiceTestDB(t)
	now := common.GetTimestamp()

	staleJob := model.AsyncExportJob{
		JobType:         SmartExportJobTypeUsageLogs,
		Status:          model.AsyncExportStatusRunning,
		RequesterUserId: 1,
		RequesterRole:   common.RoleRootUser,
		PayloadJSON:     `{"limit":0}`,
		CreatedAtTs:     now - 100,
		StartedAtTs:     now - asyncExportRunningStaleTimeoutSeconds - 1,
	}
	require.NoError(t, db.Create(&staleJob).Error)

	freshJob := model.AsyncExportJob{
		JobType:         SmartExportJobTypeUsageLogs,
		Status:          model.AsyncExportStatusRunning,
		RequesterUserId: 2,
		RequesterRole:   common.RoleRootUser,
		PayloadJSON:     `{"limit":0}`,
		CreatedAtTs:     now - 100,
		StartedAtTs:     now,
	}
	require.NoError(t, db.Create(&freshJob).Error)

	requeued, err := RequeueStaleAsyncExportJobs(now)
	require.NoError(t, err)
	require.EqualValues(t, 1, requeued)

	var persistedStale model.AsyncExportJob
	require.NoError(t, db.First(&persistedStale, staleJob.Id).Error)
	require.Equal(t, model.AsyncExportStatusQueued, persistedStale.Status)
	require.Zero(t, persistedStale.StartedAtTs)
	require.Zero(t, persistedStale.CompletedAtTs)

	var persistedFresh model.AsyncExportJob
	require.NoError(t, db.First(&persistedFresh, freshJob.Id).Error)
	require.Equal(t, model.AsyncExportStatusRunning, persistedFresh.Status)
	require.Equal(t, freshJob.StartedAtTs, persistedFresh.StartedAtTs)
}

func TestAsyncExportLabelsUseReadableChinese(t *testing.T) {
	require.Equal(t, "使用日志", asyncUsageLogsFilePrefix)
	require.Equal(t, "使用日志", asyncUsageLogsSheetName)
	require.Equal(t, "时间", asyncUsageLogExportColumns["time"].Header)
	require.Equal(t, "渠道", asyncUsageLogExportColumns["channel"].Header)
	require.Equal(t, "用户", asyncUsageLogExportColumns["username"].Header)
	require.Equal(t, "令牌", asyncUsageLogExportColumns["token"].Header)
	require.Equal(t, "分组", asyncUsageLogExportColumns["group"].Header)
	require.Equal(t, "类型", asyncUsageLogExportColumns["type"].Header)
	require.Equal(t, "模型", asyncUsageLogExportColumns["model"].Header)
	require.Equal(t, "用时/首字", asyncUsageLogExportColumns["use_time"].Header)
	require.Equal(t, "输入", asyncUsageLogExportColumns["prompt"].Header)
	require.Equal(t, "输出", asyncUsageLogExportColumns["completion"].Header)
	require.Equal(t, "花费", asyncUsageLogExportColumns["cost"].Header)
	require.Equal(t, "重试", asyncUsageLogExportColumns["retry"].Header)
	require.Equal(t, "详情", asyncUsageLogExportColumns["details"].Header)
	require.Equal(t, "充值", formatAsyncUsageLogType(model.LogTypeTopup))
	require.Equal(t, "消费", formatAsyncUsageLogType(model.LogTypeConsume))
	require.Equal(t, "管理", formatAsyncUsageLogType(model.LogTypeManage))
	require.Equal(t, "系统", formatAsyncUsageLogType(model.LogTypeSystem))
	require.Equal(t, "错误", formatAsyncUsageLogType(model.LogTypeError))
	require.Equal(t, "退款", formatAsyncUsageLogType(model.LogTypeRefund))
	require.Equal(t, "未知", formatAsyncUsageLogType(model.LogTypeUnknown))

	require.Equal(t, "审计日志", asyncAdminAuditFilePrefix)
	require.Equal(t, "审计日志", asyncAdminAuditSheetName)
	require.Equal(t, []string{"ID", "操作人", "动作模块", "动作类型", "目标", "IP", "时间"}, asyncAdminAuditExportHeaders)

	require.Equal(t, "额度流水", asyncQuotaLedgerFilePrefix)
	require.Equal(t, "额度流水", asyncQuotaLedgerSheetName)
	require.Equal(t, []string{"ID", "账户", "操作人", "类型", "方向", "额度", "变更前", "变更后", "模型名称", "来源", "原因", "备注", "时间"}, asyncQuotaLedgerExportHeaders)
}
