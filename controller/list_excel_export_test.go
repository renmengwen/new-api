package controller

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type auditExportFixture struct {
	LatestMatching   model.AdminAuditLog
	ModuleMismatch   model.AdminAuditLog
	OperatorMismatch model.AdminAuditLog
}

type quotaLedgerExportFixture struct {
	LatestMatching    model.QuotaLedger
	EntryTypeMismatch model.QuotaLedger
	UserMismatch      model.QuotaLedger
	OperatorMismatch  model.QuotaLedger
}

type usageLogExportFixture struct {
	LatestMatching  model.Log
	OldestExported  model.Log
	CappedOut       model.Log
	BeforeStart     model.Log
	AfterEnd        model.Log
	TypeMismatch    model.Log
	UserMismatch    model.Log
	TokenMismatch   model.Log
	ModelMismatch   model.Log
	ChannelMismatch model.Log
	GroupMismatch   model.Log
	RequestMismatch model.Log
}

type userUsageLogExportFixture struct {
	LatestOwnMatching model.Log
	OldestOwnMatching model.Log
	OtherUserMatching model.Log
	OwnTokenMismatch  model.Log
}

func TestExportAdminUsageLogsModelHelperCapsAndFilters(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedAdminUsageLogsForExport(t, db)

	logs, err := model.GetAllLogsForExport(
		model.LogTypeConsume,
		fixture.OldestExported.CreatedAt-50,
		fixture.LatestMatching.CreatedAt,
		fixture.LatestMatching.ModelName,
		fixture.LatestMatching.Username,
		fixture.LatestMatching.TokenName,
		5000,
		fixture.LatestMatching.ChannelId,
		fixture.LatestMatching.Group,
		fixture.LatestMatching.RequestId,
	)
	require.NoError(t, err)
	require.Len(t, logs, 2000)
	require.Equal(t, fixture.LatestMatching.Content, logs[0].Content)
	require.Equal(t, fixture.OldestExported.Content, logs[len(logs)-1].Content)
}

func TestExportAdminUsageLogsModelHelperSkipsCountQuery(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedAdminUsageLogsForExport(t, db)
	queryLogger := attachCountQueryLogger(t, db)

	logs, err := model.GetAllLogsForExport(
		model.LogTypeConsume,
		fixture.OldestExported.CreatedAt-50,
		fixture.LatestMatching.CreatedAt,
		fixture.LatestMatching.ModelName,
		fixture.LatestMatching.Username,
		fixture.LatestMatching.TokenName,
		5000,
		fixture.LatestMatching.ChannelId,
		fixture.LatestMatching.Group,
		fixture.LatestMatching.RequestId,
	)
	require.NoError(t, err)
	require.Len(t, logs, 2000)
	require.Zero(t, queryLogger.CountQueries())
}

func TestExportAdminUsageLogsUsesFiltersColumnKeysAndCap(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedAdminUsageLogsForExport(t, db)

	adminUser := testListExportUser(9001, "root_exporter", "Root Exporter", common.RoleRootUser, model.UserTypeRoot)
	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/export", map[string]any{
		"type":               model.LogTypeConsume,
		"start_timestamp":    fixture.OldestExported.CreatedAt - 50,
		"end_timestamp":      fixture.LatestMatching.CreatedAt,
		"username":           fixture.LatestMatching.Username,
		"token_name":         fixture.LatestMatching.TokenName,
		"model_name":         fixture.LatestMatching.ModelName,
		"channel":            strconv.Itoa(fixture.LatestMatching.ChannelId),
		"group":              fixture.LatestMatching.Group,
		"request_id":         fixture.LatestMatching.RequestId,
		"column_keys":        []string{"model", "use_time", "cost", "username", "time"},
		"quota_display_type": "USD",
		"limit":              5000,
	}, adminUser.Id, common.RoleRootUser)
	ctx.Set("username", adminUser.Username)

	ExportAllLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "使用日志_")

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	require.Equal(t, "使用日志", workbook.GetSheetName(0))

	rows, err := workbook.GetRows("使用日志")
	require.NoError(t, err)
	require.Len(t, rows, 2001)
	require.Equal(t, []string{"模型", "用时/首字", "花费", "用户", "时间"}, rows[0])

	dataRows := rows[1:]
	exportedModels := sheetColumnValues(dataRows, 0)
	require.Equal(t, "gpt-4o -> gpt-4.1", dataRows[0][0])
	require.Equal(t, "2079 s / 首字 0.8 s / 流", dataRows[0][1])
	require.Equal(t, "由订阅抵扣（等价金额：$0.004298）", dataRows[0][2])
	require.Equal(t, fixture.LatestMatching.Username, dataRows[0][3])
	require.Equal(t, formatExportTimestamp(fixture.LatestMatching.CreatedAt), dataRows[0][4])
	require.Equal(t, "gpt-4o -> gpt-4.1", dataRows[len(dataRows)-1][0])
	require.NotContains(t, exportedModels, fixture.CappedOut.Content)
	require.NotContains(t, exportedModels, fixture.BeforeStart.Content)
	require.NotContains(t, exportedModels, fixture.AfterEnd.Content)
	require.NotContains(t, exportedModels, fixture.TypeMismatch.Content)
	require.NotContains(t, exportedModels, fixture.UserMismatch.Content)
	require.NotContains(t, exportedModels, fixture.TokenMismatch.Content)
	require.NotContains(t, exportedModels, fixture.ModelMismatch.Content)
	require.NotContains(t, exportedModels, fixture.ChannelMismatch.Content)
	require.NotContains(t, exportedModels, fixture.GroupMismatch.Content)
	require.NotContains(t, exportedModels, fixture.RequestMismatch.Content)
}

func TestExportAdminUsageLogsKeepsNewestFirst(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	logs := []model.Log{
		{
			UserId:    3001,
			Username:  "admin_order_user",
			CreatedAt: 1810000101,
			Type:      model.LogTypeConsume,
			Content:   "oldest",
			TokenName: "admin-order-token",
			ModelName: "gpt-4o-mini",
			Group:     "ops",
			RequestId: "admin-order-req",
		},
		{
			UserId:    3001,
			Username:  "admin_order_user",
			CreatedAt: 1810000102,
			Type:      model.LogTypeConsume,
			Content:   "middle",
			TokenName: "admin-order-token",
			ModelName: "gpt-4o-mini",
			Group:     "ops",
			RequestId: "admin-order-req",
		},
		{
			UserId:    3001,
			Username:  "admin_order_user",
			CreatedAt: 1810000103,
			Type:      model.LogTypeConsume,
			Content:   "latest",
			TokenName: "admin-order-token",
			ModelName: "gpt-4o-mini",
			Group:     "ops",
			RequestId: "admin-order-req",
		},
	}
	require.NoError(t, db.Create(&logs).Error)

	adminUser := testListExportUser(9001, "root_exporter", "Root Exporter", common.RoleRootUser, model.UserTypeRoot)
	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/export", map[string]any{
		"type":        model.LogTypeConsume,
		"username":    "admin_order_user",
		"token_name":  "admin-order-token",
		"model_name":  "gpt-4o-mini",
		"group":       "ops",
		"request_id":  "admin-order-req",
		"column_keys": []string{"details"},
		"limit":       10,
	}, adminUser.Id, common.RoleRootUser)
	ctx.Set("username", adminUser.Username)

	ExportAllLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("使用日志")
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.Equal(t, []string{"详情"}, rows[0])
	require.Equal(t, "latest", rows[1][0])
	require.Equal(t, "middle", rows[2][0])
	require.Equal(t, "oldest", rows[3][0])
}

func TestExportSelfUsageLogsModelHelperOnlyReturnsSelfRows(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedUserUsageLogsForExport(t, db)

	logs, err := model.GetUserLogsForExport(
		7001,
		model.LogTypeConsume,
		0,
		0,
		fixture.LatestOwnMatching.ModelName,
		fixture.LatestOwnMatching.TokenName,
		100,
		fixture.LatestOwnMatching.Group,
		fixture.LatestOwnMatching.RequestId,
	)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, fixture.LatestOwnMatching.Content, logs[0].Content)
	require.Equal(t, fixture.OldestOwnMatching.Content, logs[1].Content)
}

func TestExportSelfUsageLogsModelHelperSkipsCountQuery(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedUserUsageLogsForExport(t, db)
	queryLogger := attachCountQueryLogger(t, db)

	logs, err := model.GetUserLogsForExport(
		7001,
		model.LogTypeConsume,
		0,
		0,
		fixture.LatestOwnMatching.ModelName,
		fixture.LatestOwnMatching.TokenName,
		100,
		fixture.LatestOwnMatching.Group,
		fixture.LatestOwnMatching.RequestId,
	)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Zero(t, queryLogger.CountQueries())
}

func TestExportSelfUsageLogsOnlyExportsOwnRows(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedUserUsageLogsForExport(t, db)

	selfUser := testListExportUser(7001, "self_exporter", "Self Exporter", common.RoleCommonUser, model.UserTypeEndUser)
	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/self/export", map[string]any{
		"type":        model.LogTypeConsume,
		"token_name":  fixture.LatestOwnMatching.TokenName,
		"model_name":  fixture.LatestOwnMatching.ModelName,
		"group":       fixture.LatestOwnMatching.Group,
		"request_id":  fixture.LatestOwnMatching.RequestId,
		"column_keys": []string{"details", "token", "time"},
		"limit":       100,
	}, selfUser.Id, common.RoleCommonUser)
	ctx.Set("username", selfUser.Username)

	ExportUserLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("使用日志")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, []string{"详情", "令牌", "时间"}, rows[0])
	require.Equal(t, fixture.LatestOwnMatching.Content, rows[1][0])
	require.Equal(t, fixture.LatestOwnMatching.TokenName, rows[1][1])
	require.Equal(t, formatExportTimestamp(fixture.LatestOwnMatching.CreatedAt), rows[1][2])
	require.Equal(t, fixture.OldestOwnMatching.Content, rows[2][0])
	require.NotEqual(t, fixture.OtherUserMatching.Content, rows[1][0])
	require.NotEqual(t, fixture.OtherUserMatching.Content, rows[2][0])
	require.NotEqual(t, fixture.OwnTokenMismatch.Content, rows[1][0])
	require.NotEqual(t, fixture.OwnTokenMismatch.Content, rows[2][0])
}

func TestExportSelfUsageLogsUsesDefaultColumnsWhenColumnKeysOmitted(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedUserUsageLogsForExport(t, db)

	selfUser := testListExportUser(7001, "self_exporter", "Self Exporter", common.RoleCommonUser, model.UserTypeEndUser)
	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/self/export", map[string]any{
		"type":               model.LogTypeConsume,
		"token_name":         fixture.LatestOwnMatching.TokenName,
		"model_name":         fixture.LatestOwnMatching.ModelName,
		"group":              fixture.LatestOwnMatching.Group,
		"request_id":         fixture.LatestOwnMatching.RequestId,
		"quota_display_type": "USD",
		"limit":              10,
	}, selfUser.Id, common.RoleCommonUser)
	ctx.Set("username", selfUser.Username)

	ExportUserLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("使用日志")
	require.NoError(t, err)
	require.Len(t, rows, 3)

	dataRow := rows[1]
	require.Len(t, dataRow, 11)
	require.Equal(t, formatExportTimestamp(fixture.LatestOwnMatching.CreatedAt), dataRow[0])
	require.Equal(t, fixture.LatestOwnMatching.TokenName, dataRow[1])
	require.Equal(t, fixture.LatestOwnMatching.Group, dataRow[2])
	require.Equal(t, "消费", dataRow[3])
	require.Equal(t, "gpt-4o-mini -> gpt-4.1-mini", dataRow[4])
	require.Equal(t, "8 s / 首字 0.8 s / 流", dataRow[5])
	require.Equal(t, strconv.Itoa(fixture.LatestOwnMatching.PromptTokens), dataRow[6])
	require.Equal(t, strconv.Itoa(fixture.LatestOwnMatching.CompletionTokens), dataRow[7])
	require.Equal(t, "由订阅抵扣（等价金额：$0.000044）", dataRow[8])
	require.Equal(t, fixture.LatestOwnMatching.Ip, dataRow[9])
	require.Equal(t, fixture.LatestOwnMatching.Content, dataRow[10])
}

func TestExportSelfUsageLogsDoesNotFallbackWhenColumnKeysExplicitlyEmpty(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedUserUsageLogsForExport(t, db)

	selfUser := testListExportUser(7001, "self_exporter", "Self Exporter", common.RoleCommonUser, model.UserTypeEndUser)
	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/self/export", map[string]any{
		"type":        model.LogTypeConsume,
		"token_name":  fixture.LatestOwnMatching.TokenName,
		"model_name":  fixture.LatestOwnMatching.ModelName,
		"group":       fixture.LatestOwnMatching.Group,
		"request_id":  fixture.LatestOwnMatching.RequestId,
		"column_keys": []string{},
		"limit":       10,
	}, selfUser.Id, common.RoleCommonUser)
	ctx.Set("username", selfUser.Username)

	ExportUserLogs(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "no export columns selected", response.Message)
	require.NotEqual(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
}

func TestExportSelfUsageLogsDoesNotFallbackWhenColumnKeysAllInvalid(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedUserUsageLogsForExport(t, db)

	selfUser := testListExportUser(7001, "self_exporter", "Self Exporter", common.RoleCommonUser, model.UserTypeEndUser)
	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/self/export", map[string]any{
		"type":        model.LogTypeConsume,
		"token_name":  fixture.LatestOwnMatching.TokenName,
		"model_name":  fixture.LatestOwnMatching.ModelName,
		"group":       fixture.LatestOwnMatching.Group,
		"request_id":  fixture.LatestOwnMatching.RequestId,
		"column_keys": []string{"unknown", "invalid"},
		"limit":       10,
	}, selfUser.Id, common.RoleCommonUser)
	ctx.Set("username", selfUser.Username)

	ExportUserLogs(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "no export columns selected", response.Message)
	require.NotEqual(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
}

func TestExportAdminAuditLogsUsesFiltersAndCap(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedAuditLogs(t, db, 2050, "quota", 9001)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/audit-logs/export", map[string]any{
		"action_module":    "quota",
		"operator_user_id": 9001,
		"limit":            2000,
	})

	ExportAdminAuditLogs(ctx)

	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "审计日志_")

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("审计日志")
	require.NoError(t, err)
	require.Len(t, rows, 2001)
	require.Equal(t, "操作人", rows[0][1])

	dataRows := rows[1:]
	exportedIDs := sheetColumnValues(dataRows, 0)
	require.Equal(t, strconv.Itoa(fixture.LatestMatching.Id), dataRows[0][0])
	require.Contains(t, dataRows[0][1], "[ID:9001]")
	require.Equal(t, "额度管理", dataRows[0][2])
	require.Equal(t, "额度调整", dataRows[0][3])
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.ModuleMismatch.Id))
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.OperatorMismatch.Id))
	requireStrictlyDescendingIDs(t, exportedIDs)

	for _, row := range dataRows {
		require.Equal(t, "额度管理", row[2])
		require.Contains(t, row[1], "[ID:9001]")
	}
}

func TestExportAdminAuditLogsFormatsSettingLabelsAndColumnWidths(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedListExportUsers(t, db,
		testListExportUser(9001, "admin", "Root User", common.RoleRootUser, model.UserTypeRoot),
	)
	settingLog := model.AdminAuditLog{
		OperatorUserId:   9001,
		OperatorUserType: model.UserTypeRoot,
		ActionModule:     "setting_system",
		ActionType:       "toggle_allow_private_ip",
		ActionDesc:       "系统设置-系统设置-SSRF-允许访问私有 IP",
		TargetType:       "option_key",
		TargetId:         0,
		Ip:               "::1",
		CreatedAtTs:      1810001234,
	}
	require.NoError(t, db.Create(&settingLog).Error)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/audit-logs/export", map[string]any{
		"action_module":    "setting_system",
		"operator_user_id": 9001,
		"limit":            10,
	})

	ExportAdminAuditLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("审计日志")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, []string{"ID", "操作人", "动作模块", "动作类型", "目标", "IP", "时间"}, rows[0])
	require.Equal(t, "系统设置", rows[1][2])
	require.Equal(t, "允许访问私有 IP", rows[1][3])
	require.Equal(t, "配置项", rows[1][4])

	widthB, err := workbook.GetColWidth("审计日志", "B")
	require.NoError(t, err)
	require.Equal(t, 28.0, widthB)

	widthC, err := workbook.GetColWidth("审计日志", "C")
	require.NoError(t, err)
	require.Equal(t, 18.0, widthC)

	widthD, err := workbook.GetColWidth("审计日志", "D")
	require.NoError(t, err)
	require.Equal(t, 24.0, widthD)

	widthE, err := workbook.GetColWidth("审计日志", "E")
	require.NoError(t, err)
	require.Equal(t, 24.0, widthE)

	widthF, err := workbook.GetColWidth("审计日志", "F")
	require.NoError(t, err)
	require.Equal(t, 16.0, widthF)

	widthG, err := workbook.GetColWidth("审计日志", "G")
	require.NoError(t, err)
	require.Equal(t, 22.0, widthG)
}

func TestExportQuotaLedgerUsesEntryTypeFilterAndCap(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedQuotaLedgerRows(t, db, 2088, model.LedgerEntryAdjust)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/quota/ledger/export", map[string]any{
		"user_id":          2001,
		"operator_user_id": 9001,
		"entry_type":       model.LedgerEntryAdjust,
		"limit":            2000,
	})

	ExportQuotaLedger(ctx)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("额度流水")
	require.NoError(t, err)
	require.Len(t, rows, 2001)
	require.Equal(t, "类型", rows[0][3])

	dataRows := rows[1:]
	exportedIDs := sheetColumnValues(dataRows, 0)
	require.Equal(t, strconv.Itoa(fixture.LatestMatching.Id), dataRows[0][0])
	require.Equal(t, "quota_user", dataRows[0][1])
	require.Contains(t, dataRows[0][2], "[ID:9001]")
	require.Equal(t, "调额", dataRows[0][3])
	require.Equal(t, "手动调额", dataRows[0][10])
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.EntryTypeMismatch.Id))
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.UserMismatch.Id))
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.OperatorMismatch.Id))
	requireStrictlyDescendingIDs(t, exportedIDs)

	for _, row := range dataRows {
		require.Equal(t, "quota_user", row[1])
		require.Contains(t, row[2], "[ID:9001]")
		require.Equal(t, "调额", row[3])
		require.Equal(t, "手动调额", row[10])
	}
}

func TestExportQuotaLedgerFormatsQuotaColumnsAsUSD(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedQuotaLedgerRows(t, db, 1, model.LedgerEntryAdjust)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/quota/ledger/export", map[string]any{
		"user_id":          2001,
		"operator_user_id": 9001,
		"entry_type":       model.LedgerEntryAdjust,
		"limit":            10,
	})

	ExportQuotaLedger(ctx)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows(workbook.GetSheetName(0))
	require.NoError(t, err)
	require.Len(t, rows, 2)

	dataRow := rows[1]
	require.Equal(t, strconv.Itoa(fixture.LatestMatching.Id), dataRow[0])
	require.Equal(t, "$0.000002", dataRow[5])
	require.Equal(t, "$0.200000", dataRow[6])
	require.Equal(t, "$0.200002", dataRow[7])
}

func TestExportQuotaLedgerIncludesModelNameColumnForConsumeEntries(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedQuotaLedgerRows(t, db, 1, model.LedgerEntryConsume)

	require.NoError(t, db.Create(&model.Log{
		UserId:    2001,
		Username:  "quota_user",
		CreatedAt: fixture.LatestMatching.CreatedAtTs - 1,
		Type:      model.LogTypeConsume,
		Content:   "consume export match",
		ModelName: "claude-4-sonnet",
		Quota:     fixture.LatestMatching.Amount,
	}).Error)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/quota/ledger/export", map[string]any{
		"user_id":          2001,
		"operator_user_id": 9001,
		"entry_type":       model.LedgerEntryConsume,
		"limit":            10,
	})

	ExportQuotaLedger(ctx)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows(workbook.GetSheetName(0))
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Contains(t, rows[0], "模型名称")

	modelNameColumn := -1
	for index, header := range rows[0] {
		if header == "模型名称" {
			modelNameColumn = index
			break
		}
	}
	require.NotEqual(t, -1, modelNameColumn)
	require.Equal(t, "claude-4-sonnet", rows[1][modelNameColumn])
}

func TestExportQuotaLedgerBackfillsModelNameForSplitWalletConsumeRows(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	seedListExportUsers(t, db,
		testListExportUser(9001, "root_operator", "Root Operator", common.RoleRootUser, model.UserTypeRoot),
	)
	user := testListExportUser(9201, "quota_export_split_user", "Quota Export Split User", common.RoleCommonUser, model.UserTypeEndUser)
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.QuotaAccount{
		OwnerType:      model.QuotaOwnerTypeUser,
		OwnerId:        user.Id,
		Balance:        1000000,
		TotalRecharged: 1000000,
		Status:         model.CommonStatusEnabled,
		CreatedAtTs:    1811000400,
		UpdatedAtTs:    1811000400,
	}).Error)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)

	require.NoError(t, db.Create(&[]model.QuotaLedger{
		{
			BizNo:            "ql_export_wallet_preconsume_match",
			AccountId:        account.Id,
			EntryType:        model.LedgerEntryConsume,
			Direction:        model.LedgerDirectionOut,
			Amount:           7240,
			BalanceBefore:    1000000,
			BalanceAfter:     992760,
			SourceType:       "wallet_preconsume",
			SourceId:         user.Id,
			OperatorUserId:   user.Id,
			OperatorUserType: model.UserTypeEndUser,
			Reason:           "钱包预扣",
			CreatedAtTs:      1811000400,
		},
		{
			BizNo:            "ql_export_wallet_settle_match",
			AccountId:        account.Id,
			EntryType:        model.LedgerEntryConsume,
			Direction:        model.LedgerDirectionOut,
			Amount:           716,
			BalanceBefore:    992760,
			BalanceAfter:     992044,
			SourceType:       "wallet_settle",
			SourceId:         user.Id,
			OperatorUserId:   user.Id,
			OperatorUserType: model.UserTypeEndUser,
			Reason:           "钱包结算扣费",
			CreatedAtTs:      1811000404,
		},
	}).Error)

	require.NoError(t, db.Create(&model.Log{
		UserId:    user.Id,
		Username:  user.Username,
		CreatedAt: 1811000404,
		Type:      model.LogTypeConsume,
		Content:   "wallet split consume export log",
		ModelName: "claude-opus-4-6",
		Quota:     7956,
	}).Error)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/quota/ledger/export", map[string]any{
		"user_id":    user.Id,
		"entry_type": model.LedgerEntryConsume,
		"limit":      10,
	})

	ExportQuotaLedger(ctx)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows(workbook.GetSheetName(0))
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, "出账", rows[1][4])
	require.Equal(t, "出账", rows[2][4])
	require.Equal(t, "claude-opus-4-6", rows[1][8])
	require.Equal(t, "claude-opus-4-6", rows[2][8])
	require.Equal(t, "钱包结算", rows[1][9])
	require.Equal(t, "钱包预扣", rows[2][9])
}

func TestExportAdminAuditLogsServiceHelperCapsLimit(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedAuditLogs(t, db, 2050, "quota", 9001)

	items, total, err := service.ListAdminAuditLogsForExport(9001, common.RoleRootUser, "quota", 9001, 5000)
	require.NoError(t, err)
	require.Len(t, items, 2000)
	require.Zero(t, total)
	require.True(t, items[0].Id > items[len(items)-1].Id)

	items, total, err = service.ListAdminAuditLogsForExport(9001, common.RoleRootUser, "quota", 9001, 123)
	require.NoError(t, err)
	require.Len(t, items, 123)
	require.Zero(t, total)
}

func TestExportQuotaLedgerServiceHelperCapsLimit(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedQuotaLedgerRows(t, db, 2088, model.LedgerEntryAdjust)

	items, total, err := service.ListQuotaLedgerForExport(9001, common.RoleRootUser, 2001, 9001, model.LedgerEntryAdjust, 5000)
	require.NoError(t, err)
	require.Len(t, items, 2000)
	require.Zero(t, total)
	require.True(t, items[0].Id > items[len(items)-1].Id)

	items, total, err = service.ListQuotaLedgerForExport(9001, common.RoleRootUser, 2001, 9001, model.LedgerEntryAdjust, 321)
	require.NoError(t, err)
	require.Len(t, items, 321)
	require.Zero(t, total)
}

func TestExportAdminAuditLogsAllowsAgentReadGrantAndOnlyExportsOwnRows(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	agent := testListExportUser(9101, "audit_agent", "Audit Agent", common.RoleAdminUser, model.UserTypeAgent)
	otherOperator := testListExportUser(9102, "other_audit_operator", "Other Audit Operator", common.RoleAdminUser, model.UserTypeAdmin)
	managedUser := testListExportUser(9103, "managed_audit_user", "Managed Audit User", common.RoleCommonUser, model.UserTypeEndUser)
	require.NoError(t, db.Create(&agent).Error)
	require.NoError(t, db.Create(&otherOperator).Error)
	require.NoError(t, db.Create(&managedUser).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   managedUser.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: service.ResourceAuditManagement, Action: service.ActionRead},
	)
	require.NoError(t, db.Create(&[]model.AdminAuditLog{
		{
			OperatorUserId:   agent.Id,
			OperatorUserType: model.UserTypeAgent,
			ActionModule:     "quota",
			ActionType:       "adjust",
			ActionDesc:       "agent_self_row",
			TargetType:       "user",
			TargetId:         agent.Id,
			Ip:               "203.0.113.10",
			CreatedAtTs:      1810000201,
		},
		{
			OperatorUserId:   managedUser.Id,
			OperatorUserType: model.UserTypeEndUser,
			ActionModule:     "quota",
			ActionType:       "adjust",
			ActionDesc:       "managed_user_row",
			TargetType:       "user",
			TargetId:         managedUser.Id,
			Ip:               "203.0.113.12",
			CreatedAtTs:      1810000202,
		},
		{
			OperatorUserId:   otherOperator.Id,
			OperatorUserType: model.UserTypeAdmin,
			ActionModule:     "quota",
			ActionType:       "adjust",
			ActionDesc:       "other_operator_row",
			TargetType:       "user",
			TargetId:         otherOperator.Id,
			Ip:               "203.0.113.11",
			CreatedAtTs:      1810000203,
		},
	}).Error)

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/admin/audit-logs/export", map[string]any{
		"action_module": "quota",
		"limit":         10,
	}, agent.Id, common.RoleAdminUser)

	ExportAdminAuditLogs(ctx)

	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	sheets := workbook.GetSheetList()
	require.NotEmpty(t, sheets)
	rows, err := workbook.GetRows(sheets[0])
	require.NoError(t, err)
	require.Len(t, rows, 3)
	operatorCells := []string{rows[1][1], rows[2][1]}
	require.Contains(t, operatorCells[0]+operatorCells[1], agent.Username)
	require.Contains(t, operatorCells[0]+operatorCells[1], strconv.Itoa(agent.Id))
	require.Contains(t, operatorCells[0]+operatorCells[1], managedUser.Username)
	require.Contains(t, operatorCells[0]+operatorCells[1], strconv.Itoa(managedUser.Id))
}

func TestExportAdminAuditLogsRequiresReadPermission(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	admin := testListExportUser(9104, "audit_admin", "Audit Admin", common.RoleAdminUser, model.UserTypeAdmin)
	require.NoError(t, db.Create(&admin).Error)

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/admin/audit-logs/export", map[string]any{
		"action_module": "quota",
		"limit":         10,
	}, admin.Id, common.RoleAdminUser)

	ExportAdminAuditLogs(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "permission denied", response.Message)
	require.NotEqual(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
}

func TestExportAdminAuditLogsAllowsReadPermission(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedAuditLogs(t, db, 1, "quota", 9001)

	admin := testListExportUser(9105, "audit_reader", "Audit Reader", common.RoleAdminUser, model.UserTypeAdmin)
	require.NoError(t, db.Create(&admin).Error)
	grantPermissionActions(t, db, admin.Id, "admin",
		permissionGrant{Resource: service.ResourceAuditManagement, Action: service.ActionRead},
	)

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/admin/audit-logs/export", map[string]any{
		"action_module":    "quota",
		"operator_user_id": 9001,
		"limit":            10,
	}, admin.Id, common.RoleAdminUser)

	ExportAdminAuditLogs(ctx)

	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("审计日志")
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func TestExportQuotaLedgerRequiresLedgerReadPermission(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	admin := testListExportUser(9102, "quota_admin", "Quota Admin", common.RoleAdminUser, model.UserTypeAdmin)
	require.NoError(t, db.Create(&admin).Error)

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/admin/quota/ledger/export", map[string]any{
		"limit": 10,
	}, admin.Id, common.RoleAdminUser)

	ExportQuotaLedger(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "permission denied", response.Message)
	require.NotEqual(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
}

func TestExportQuotaLedgerDoesNotCreateAccountForFilteredUserWithoutAccount(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	user := testListExportUser(9301, "no_quota_account_user", "No Quota Account User", common.RoleCommonUser, model.UserTypeEndUser)
	require.NoError(t, db.Create(&user).Error)

	_, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/quota/ledger/export", map[string]any{
		"user_id": user.Id,
		"limit":   10,
	})

	ExportQuotaLedger(ctx)

	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("额度流水")
	require.NoError(t, err)
	require.Len(t, rows, 1)

	_, err = model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestExportQuotaLedgerAgentOnlyExportsSelfAndManagedRows(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	agent := testListExportUser(9201, "export_agent", "Export Agent", common.RoleAdminUser, model.UserTypeAgent)
	ownedUser := testListExportUser(9202, "managed_user", "Managed User", common.RoleCommonUser, model.UserTypeEndUser)
	unownedUser := testListExportUser(9203, "unmanaged_user", "Unmanaged User", common.RoleCommonUser, model.UserTypeEndUser)
	require.NoError(t, db.Create(&[]model.User{agent, ownedUser, unownedUser}).Error)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", ownedUser.Id).Update("parent_agent_id", agent.Id).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   ownedUser.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: service.ResourceQuotaManagement, Action: service.ActionLedgerRead},
	)

	agentAccount := seedListExportQuotaAccount(t, db, agent.Id, 300)
	ownedAccount := seedListExportQuotaAccount(t, db, ownedUser.Id, 200)
	unownedAccount := seedListExportQuotaAccount(t, db, unownedUser.Id, 400)

	selfLedger := seedListExportLedger(t, db, model.QuotaLedger{
		BizNo:            "agent_scope_self",
		AccountId:        agentAccount.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionOut,
		Amount:           20,
		BalanceBefore:    300,
		BalanceAfter:     280,
		SourceType:       "admin_quota_adjust",
		SourceId:         agent.Id,
		OperatorUserId:   agent.Id,
		OperatorUserType: model.UserTypeAgent,
		CreatedAtTs:      1810000101,
	})
	ownedLedger := seedListExportLedger(t, db, model.QuotaLedger{
		BizNo:            "agent_scope_owned",
		AccountId:        ownedAccount.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           30,
		BalanceBefore:    200,
		BalanceAfter:     230,
		SourceType:       "admin_quota_adjust",
		SourceId:         ownedUser.Id,
		OperatorUserId:   agent.Id,
		OperatorUserType: model.UserTypeAgent,
		CreatedAtTs:      1810000102,
	})
	unownedLedger := seedListExportLedger(t, db, model.QuotaLedger{
		BizNo:            "agent_scope_unowned",
		AccountId:        unownedAccount.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           40,
		BalanceBefore:    400,
		BalanceAfter:     440,
		SourceType:       "admin_quota_adjust",
		SourceId:         unownedUser.Id,
		OperatorUserId:   agent.Id,
		OperatorUserType: model.UserTypeAgent,
		CreatedAtTs:      1810000103,
	})

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/admin/quota/ledger/export", map[string]any{
		"entry_type": model.LedgerEntryAdjust,
		"limit":      10,
	}, agent.Id, common.RoleAdminUser)

	ExportQuotaLedger(ctx)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("额度流水")
	require.NoError(t, err)
	require.Len(t, rows, 3)

	dataRows := rows[1:]
	exportedIDs := sheetColumnValues(dataRows, 0)
	require.ElementsMatch(t, []string{strconv.Itoa(selfLedger.Id), strconv.Itoa(ownedLedger.Id)}, exportedIDs)
	require.NotContains(t, exportedIDs, strconv.Itoa(unownedLedger.Id))
	requireStrictlyDescendingIDs(t, exportedIDs)
}

func TestExportAdminUsageLogsScopesAgentRows(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	agent := testListExportUser(9401, "usage_export_agent", "Usage Export Agent", common.RoleCommonUser, model.UserTypeAgent)
	ownedUser := testListExportUser(9402, "usage_export_owned", "Usage Export Owned", common.RoleCommonUser, model.UserTypeEndUser)
	unownedUser := testListExportUser(9403, "usage_export_unowned", "Usage Export Unowned", common.RoleCommonUser, model.UserTypeEndUser)
	ownedUser.ParentAgentId = agent.Id
	require.NoError(t, db.Create(&[]model.User{agent, ownedUser, unownedUser}).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   ownedUser.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	ownedToken := model.Token{
		UserId:      ownedUser.Id,
		Key:         "usage-export-owned-token-key",
		Status:      common.TokenStatusEnabled,
		Name:        "usage-export-owned-token",
		CreatedTime: 1813000000,
		Group:       "default",
	}
	require.NoError(t, db.Create(&ownedToken).Error)

	require.NoError(t, db.Create(&[]model.Log{
		{
			UserId:    agent.Id,
			Username:  agent.Username,
			CreatedAt: 1813000001,
			Type:      model.LogTypeConsume,
			Content:   "agent export log",
			TokenName: "agent-export-token",
			ModelName: "agent-model",
			Group:     "default",
		},
		{
			UserId:    ownedUser.Id,
			Username:  ownedUser.Username,
			CreatedAt: 1813000002,
			Type:      model.LogTypeConsume,
			Content:   "owned export log",
			TokenName: "owned-export-token",
			ModelName: "owned-model",
			Group:     "default",
		},
		{
			UserId:    unownedUser.Id,
			Username:  unownedUser.Username,
			CreatedAt: 1813000003,
			Type:      model.LogTypeConsume,
			Content:   "unowned export log",
			TokenName: "unowned-export-token",
			ModelName: "unowned-model",
			Group:     "default",
		},
		{
			UserId:    unownedUser.Id,
			Username:  unownedUser.Username,
			CreatedAt: 1813000004,
			Type:      model.LogTypeConsume,
			Content:   "borrowed managed token export log",
			TokenId:   ownedToken.Id,
			TokenName: ownedToken.Name,
			ModelName: "borrowed-model",
			Group:     "default",
		},
	}).Error)

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/export", map[string]any{
		"type":        model.LogTypeConsume,
		"column_keys": []string{"username", "details"},
		"limit":       10,
	}, agent.Id, agent.Role)
	ctx.Set("username", agent.Username)

	ExportAllLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows(workbook.GetSheetName(0))
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.Equal(t, []string{"用户", "详情"}, rows[0])
	require.Equal(t, unownedUser.Username, rows[1][0])
	require.Equal(t, "borrowed managed token export log", rows[1][1])
	require.Equal(t, ownedUser.Username, rows[2][0])
	require.Equal(t, "owned export log", rows[2][1])
	require.Equal(t, agent.Username, rows[3][0])
	require.Equal(t, "agent export log", rows[3][1])
}

func TestExportAdminUsageLogsRequiresLedgerReadPermissionForAgent(t *testing.T) {
	db := setupListExcelExportTestDB(t)

	agent := testListExportUser(9411, "usage_export_agent_deny", "Usage Export Agent Deny", common.RoleCommonUser, model.UserTypeAgent)
	require.NoError(t, db.Create(&agent).Error)
	grantPermissionActions(t, db, agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceQuotaManagement, Action: service.ActionReadSummary},
	)
	require.NoError(t, db.Create(&model.Log{
		UserId:    agent.Id,
		Username:  agent.Username,
		CreatedAt: 1813100001,
		Type:      model.LogTypeConsume,
		Content:   "agent export log",
		TokenName: "agent-export-token",
		ModelName: "agent-model",
		Group:     "default",
	}).Error)

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/export", map[string]any{
		"type":        model.LogTypeConsume,
		"column_keys": []string{"username", "details"},
		"limit":       10,
	}, agent.Id, agent.Role)
	ctx.Set("username", agent.Username)

	ExportAllLogs(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "permission denied", response.Message)
	require.NotEqual(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
}

func setupListExcelExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := setupSettingAuditTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Token{},
		&model.Log{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.UserPermissionOverride{},
		&model.UserMenuOverride{},
		&model.UserDataScopeOverride{},
		&model.AgentUserRelation{},
		&model.QuotaAccount{},
		&model.QuotaLedger{},
		&model.Channel{},
	))
	return db
}

func openWorkbookBytes(t *testing.T, content []byte) *excelize.File {
	t.Helper()

	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, workbook.Close())
	})
	return workbook
}

func seedAuditLogs(t *testing.T, db *gorm.DB, total int, actionModule string, operatorUserID int) auditExportFixture {
	t.Helper()

	seedListExportUsers(t, db,
		testListExportUser(operatorUserID, "root_operator", "Root Operator", common.RoleRootUser, model.UserTypeRoot),
		testListExportUser(1001, "target_user", "Target User", common.RoleCommonUser, model.UserTypeEndUser),
		testListExportUser(9002, "other_operator", "Other Operator", common.RoleAdminUser, model.UserTypeAdmin),
	)

	logs := make([]model.AdminAuditLog, 0, total+2)
	for i := 0; i < total; i++ {
		logs = append(logs, model.AdminAuditLog{
			OperatorUserId:   operatorUserID,
			OperatorUserType: model.UserTypeRoot,
			ActionModule:     actionModule,
			ActionType:       "adjust",
			ActionDesc:       "quota_adjust",
			TargetType:       "user",
			TargetId:         1001,
			Ip:               "203.0.113.9",
			CreatedAtTs:      int64(1710000000 + i),
		})
	}
	logs = append(logs,
		model.AdminAuditLog{
			OperatorUserId:   operatorUserID,
			OperatorUserType: model.UserTypeRoot,
			ActionModule:     "setting_misc",
			ActionType:       "module_mismatch",
			ActionDesc:       "other_module",
			TargetType:       "user",
			TargetId:         1001,
			Ip:               "198.51.100.10",
			CreatedAtTs:      1810000001,
		},
		model.AdminAuditLog{
			OperatorUserId:   9002,
			OperatorUserType: model.UserTypeAdmin,
			ActionModule:     actionModule,
			ActionType:       "operator_mismatch",
			ActionDesc:       "other_operator",
			TargetType:       "user",
			TargetId:         1001,
			Ip:               "198.51.100.11",
			CreatedAtTs:      1810000002,
		},
	)

	require.NoError(t, db.CreateInBatches(logs, 200).Error)
	return auditExportFixture{
		LatestMatching:   logs[total-1],
		ModuleMismatch:   logs[total],
		OperatorMismatch: logs[total+1],
	}
}

func seedQuotaLedgerRows(t *testing.T, db *gorm.DB, total int, entryType string) quotaLedgerExportFixture {
	t.Helper()

	seedListExportUsers(t, db,
		testListExportUser(9001, "root_operator", "Root Operator", common.RoleRootUser, model.UserTypeRoot),
		testListExportUser(9002, "other_operator", "Other Operator", common.RoleAdminUser, model.UserTypeAdmin),
		testListExportUser(2001, "quota_user", "Quota User", common.RoleCommonUser, model.UserTypeEndUser),
		testListExportUser(2002, "other_user", "Other User", common.RoleCommonUser, model.UserTypeEndUser),
	)

	account := &model.QuotaAccount{
		OwnerType:        model.QuotaOwnerTypeUser,
		OwnerId:          2001,
		Balance:          500000,
		TotalRecharged:   500000,
		TotalAdjustedIn:  0,
		TotalAdjustedOut: 0,
		Status:           model.CommonStatusEnabled,
		CreatedAtTs:      1710000000,
		UpdatedAtTs:      1710000000,
	}
	require.NoError(t, db.Create(account).Error)

	otherAccount := &model.QuotaAccount{
		OwnerType:        model.QuotaOwnerTypeUser,
		OwnerId:          2002,
		Balance:          1000,
		TotalRecharged:   1000,
		TotalAdjustedIn:  0,
		TotalAdjustedOut: 0,
		Status:           model.CommonStatusEnabled,
		CreatedAtTs:      1710000000,
		UpdatedAtTs:      1710000000,
	}
	require.NoError(t, db.Create(otherAccount).Error)

	ledgers := make([]model.QuotaLedger, 0, total+3)
	for i := 0; i < total; i++ {
		before := 100000 + i
		ledgers = append(ledgers, model.QuotaLedger{
			BizNo:            fmt.Sprintf("ql_export_%d", i),
			AccountId:        account.Id,
			TransferOrderId:  0,
			EntryType:        entryType,
			Direction:        model.LedgerDirectionIn,
			Amount:           1,
			BalanceBefore:    before,
			BalanceAfter:     before + 1,
			SourceType:       "admin_quota_adjust",
			SourceId:         2001,
			OperatorUserId:   9001,
			OperatorUserType: model.UserTypeRoot,
			Reason:           "manual_adjust",
			Remark:           "",
			CreatedAtTs:      int64(1710000000 + i),
		})
	}
	ledgers = append(ledgers,
		model.QuotaLedger{
			BizNo:            "ql_export_other_type",
			AccountId:        account.Id,
			TransferOrderId:  0,
			EntryType:        model.LedgerEntryRecharge,
			Direction:        model.LedgerDirectionIn,
			Amount:           10,
			BalanceBefore:    1,
			BalanceAfter:     11,
			SourceType:       "topup",
			SourceId:         2001,
			OperatorUserId:   9001,
			OperatorUserType: model.UserTypeRoot,
			Reason:           "wrong_entry_type",
			CreatedAtTs:      1810000001,
		},
		model.QuotaLedger{
			BizNo:            "ql_export_other_account",
			AccountId:        otherAccount.Id,
			TransferOrderId:  0,
			EntryType:        entryType,
			Direction:        model.LedgerDirectionIn,
			Amount:           5,
			BalanceBefore:    10,
			BalanceAfter:     15,
			SourceType:       "admin_quota_adjust",
			SourceId:         2002,
			OperatorUserId:   9001,
			OperatorUserType: model.UserTypeRoot,
			Reason:           "wrong_user",
			CreatedAtTs:      1810000002,
		},
		model.QuotaLedger{
			BizNo:            "ql_export_other_operator",
			AccountId:        account.Id,
			TransferOrderId:  0,
			EntryType:        entryType,
			Direction:        model.LedgerDirectionIn,
			Amount:           7,
			BalanceBefore:    20,
			BalanceAfter:     27,
			SourceType:       "admin_quota_adjust",
			SourceId:         2001,
			OperatorUserId:   9002,
			OperatorUserType: model.UserTypeAdmin,
			Reason:           "wrong_operator",
			CreatedAtTs:      1810000003,
		},
	)

	require.NoError(t, db.CreateInBatches(ledgers, 200).Error)
	return quotaLedgerExportFixture{
		LatestMatching:    ledgers[total-1],
		EntryTypeMismatch: ledgers[total],
		UserMismatch:      ledgers[total+1],
		OperatorMismatch:  ledgers[total+2],
	}
}

func seedListExportUsers(t *testing.T, db *gorm.DB, users ...model.User) {
	t.Helper()

	require.NoError(t, db.Create(&users).Error)
}

func testListExportUser(id int, username string, displayName string, role int, userType string) model.User {
	return model.User{
		Id:          id,
		Username:    username,
		Password:    "hashed-password",
		DisplayName: displayName,
		Role:        role,
		Status:      common.UserStatusEnabled,
		UserType:    userType,
		Group:       "default",
		AffCode:     fmt.Sprintf("aff_%d", id),
		Quota:       0,
		Email:       fmt.Sprintf("%s@example.com", username),
	}
}

func newListExcelExportContextWithOperator(t *testing.T, method string, target string, body any, operatorID int, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	ctx, recorder := newSettingAuditContext(t, method, target, body)
	ctx.Set("id", operatorID)
	ctx.Set("role", role)
	return ctx, recorder
}

func sheetColumnValues(rows [][]string, column int) []string {
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) <= column {
			values = append(values, "")
			continue
		}
		values = append(values, row[column])
	}
	return values
}

func requireStrictlyDescendingIDs(t *testing.T, ids []string) {
	t.Helper()

	require.NotEmpty(t, ids)

	previousID, err := strconv.Atoi(ids[0])
	require.NoError(t, err)

	for _, rawID := range ids[1:] {
		currentID, err := strconv.Atoi(rawID)
		require.NoError(t, err)
		require.Less(t, currentID, previousID)
		previousID = currentID
	}
}

func seedListExportQuotaAccount(t *testing.T, db *gorm.DB, ownerID int, balance int) *model.QuotaAccount {
	t.Helper()

	account := &model.QuotaAccount{
		OwnerType:        model.QuotaOwnerTypeUser,
		OwnerId:          ownerID,
		Balance:          balance,
		TotalRecharged:   balance,
		TotalAdjustedIn:  0,
		TotalAdjustedOut: 0,
		Status:           model.CommonStatusEnabled,
		CreatedAtTs:      common.GetTimestamp(),
		UpdatedAtTs:      common.GetTimestamp(),
	}
	require.NoError(t, db.Create(account).Error)
	return account
}

func seedListExportLedger(t *testing.T, db *gorm.DB, ledger model.QuotaLedger) model.QuotaLedger {
	t.Helper()

	require.NoError(t, db.Create(&ledger).Error)
	return ledger
}

func seedAdminUsageLogsForExport(t *testing.T, db *gorm.DB) usageLogExportFixture {
	t.Helper()

	require.NoError(t, db.Create(&model.Channel{
		Id:     7,
		Name:   "Main Channel",
		Key:    "channel-key",
		Status: common.ChannelStatusEnabled,
		Group:  "default",
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Id:     8,
		Name:   "Backup Channel",
		Key:    "channel-key-2",
		Status: common.ChannelStatusEnabled,
		Group:  "default",
	}).Error)

	matchedOther := common.MapToJsonStr(map[string]any{
		"is_model_mapped":     true,
		"upstream_model_name": "gpt-4.1",
		"frt":                 800,
		"billing_source":      "subscription",
	})

	logs := make([]model.Log, 0, 2058)
	for i := 0; i < 2050; i++ {
		logs = append(logs, model.Log{
			UserId:           501,
			Username:         "admin_export_user",
			CreatedAt:        int64(1710000000 + i),
			Type:             model.LogTypeConsume,
			Content:          fmt.Sprintf("match_%04d", i),
			TokenName:        "admin-export-token",
			ModelName:        "gpt-4o",
			Quota:            100 + i,
			PromptTokens:     10 + i,
			CompletionTokens: 20 + i,
			UseTime:          30 + i,
			IsStream:         true,
			ChannelId:        7,
			Group:            "ops",
			Ip:               fmt.Sprintf("203.0.113.%d", (i%200)+1),
			RequestId:        "admin-export-request",
			Other:            matchedOther,
		})
	}
	logs = append(logs,
		model.Log{
			UserId:           501,
			Username:         "admin_export_user",
			CreatedAt:        1709999999,
			Type:             model.LogTypeConsume,
			Content:          "before_start",
			TokenName:        "admin-export-token",
			ModelName:        "gpt-4o",
			Quota:            1,
			PromptTokens:     1,
			CompletionTokens: 1,
			UseTime:          1,
			ChannelId:        7,
			Group:            "ops",
			RequestId:        "admin-export-request",
		},
		model.Log{
			UserId:           501,
			Username:         "admin_export_user",
			CreatedAt:        1710002050,
			Type:             model.LogTypeConsume,
			Content:          "after_end",
			TokenName:        "admin-export-token",
			ModelName:        "gpt-4o",
			Quota:            1,
			PromptTokens:     1,
			CompletionTokens: 1,
			UseTime:          1,
			ChannelId:        7,
			Group:            "ops",
			RequestId:        "admin-export-request",
		},
		model.Log{
			UserId:    501,
			Username:  "admin_export_user",
			CreatedAt: 1710002048,
			Type:      model.LogTypeError,
			Content:   "type_mismatch",
			TokenName: "admin-export-token",
			ModelName: "gpt-4o",
			ChannelId: 7,
			Group:     "ops",
			RequestId: "admin-export-request",
		},
		model.Log{
			UserId:    502,
			Username:  "other_user",
			CreatedAt: 1710002048,
			Type:      model.LogTypeConsume,
			Content:   "username_mismatch",
			TokenName: "admin-export-token",
			ModelName: "gpt-4o",
			ChannelId: 7,
			Group:     "ops",
			RequestId: "admin-export-request",
		},
		model.Log{
			UserId:    501,
			Username:  "admin_export_user",
			CreatedAt: 1710002048,
			Type:      model.LogTypeConsume,
			Content:   "token_mismatch",
			TokenName: "other-token",
			ModelName: "gpt-4o",
			ChannelId: 7,
			Group:     "ops",
			RequestId: "admin-export-request",
		},
		model.Log{
			UserId:    501,
			Username:  "admin_export_user",
			CreatedAt: 1710002048,
			Type:      model.LogTypeConsume,
			Content:   "model_mismatch",
			TokenName: "admin-export-token",
			ModelName: "claude-3-5-sonnet",
			ChannelId: 7,
			Group:     "ops",
			RequestId: "admin-export-request",
		},
		model.Log{
			UserId:    501,
			Username:  "admin_export_user",
			CreatedAt: 1710002048,
			Type:      model.LogTypeConsume,
			Content:   "channel_mismatch",
			TokenName: "admin-export-token",
			ModelName: "gpt-4o",
			ChannelId: 8,
			Group:     "ops",
			RequestId: "admin-export-request",
		},
		model.Log{
			UserId:    501,
			Username:  "admin_export_user",
			CreatedAt: 1710002048,
			Type:      model.LogTypeConsume,
			Content:   "group_mismatch",
			TokenName: "admin-export-token",
			ModelName: "gpt-4o",
			ChannelId: 7,
			Group:     "sales",
			RequestId: "admin-export-request",
		},
		model.Log{
			UserId:    501,
			Username:  "admin_export_user",
			CreatedAt: 1710002048,
			Type:      model.LogTypeConsume,
			Content:   "request_mismatch",
			TokenName: "admin-export-token",
			ModelName: "gpt-4o",
			ChannelId: 7,
			Group:     "ops",
			RequestId: "other-request",
		},
	)

	require.NoError(t, db.CreateInBatches(logs, 200).Error)

	return usageLogExportFixture{
		LatestMatching:  logs[2049],
		OldestExported:  logs[50],
		CappedOut:       logs[49],
		BeforeStart:     logs[2050],
		AfterEnd:        logs[2051],
		TypeMismatch:    logs[2052],
		UserMismatch:    logs[2053],
		TokenMismatch:   logs[2054],
		ModelMismatch:   logs[2055],
		ChannelMismatch: logs[2056],
		GroupMismatch:   logs[2057],
		RequestMismatch: logs[2058],
	}
}

func seedUserUsageLogsForExport(t *testing.T, db *gorm.DB) userUsageLogExportFixture {
	t.Helper()

	logs := []model.Log{
		{
			UserId:           7001,
			Username:         "self_exporter",
			CreatedAt:        1810000201,
			Type:             model.LogTypeConsume,
			Content:          "self_oldest",
			TokenName:        "self-export-token",
			ModelName:        "gpt-4o-mini",
			Quota:            12,
			PromptTokens:     120,
			CompletionTokens: 24,
			UseTime:          5,
			Group:            "personal",
			Ip:               "203.0.113.30",
			RequestId:        "self-export-request",
		},
		{
			UserId:           7001,
			Username:         "self_exporter",
			CreatedAt:        1810000202,
			Type:             model.LogTypeConsume,
			Content:          "self_latest",
			TokenName:        "self-export-token",
			ModelName:        "gpt-4o-mini",
			Quota:            22,
			PromptTokens:     220,
			CompletionTokens: 44,
			UseTime:          8,
			IsStream:         true,
			Group:            "personal",
			Ip:               "203.0.113.31",
			RequestId:        "self-export-request",
			Other: common.MapToJsonStr(map[string]any{
				"is_model_mapped":     true,
				"upstream_model_name": "gpt-4.1-mini",
				"frt":                 800,
				"billing_source":      "subscription",
			}),
		},
		{
			UserId:           7002,
			Username:         "other_user",
			CreatedAt:        1810000203,
			Type:             model.LogTypeConsume,
			Content:          "other_user_match",
			TokenName:        "self-export-token",
			ModelName:        "gpt-4o-mini",
			Quota:            32,
			PromptTokens:     320,
			CompletionTokens: 64,
			UseTime:          9,
			Group:            "personal",
			Ip:               "203.0.113.32",
			RequestId:        "self-export-request",
		},
		{
			UserId:           7001,
			Username:         "self_exporter",
			CreatedAt:        1810000204,
			Type:             model.LogTypeConsume,
			Content:          "own_token_mismatch",
			TokenName:        "other-token",
			ModelName:        "gpt-4o-mini",
			Quota:            42,
			PromptTokens:     420,
			CompletionTokens: 84,
			UseTime:          10,
			Group:            "personal",
			Ip:               "203.0.113.33",
			RequestId:        "self-export-request",
		},
	}

	require.NoError(t, db.Create(&logs).Error)
	return userUsageLogExportFixture{
		OldestOwnMatching: logs[0],
		LatestOwnMatching: logs[1],
		OtherUserMatching: logs[2],
		OwnTokenMismatch:  logs[3],
	}
}

type countQueryLogger struct {
	base    gormlogger.Interface
	queries *atomic.Int32
}

func newCountQueryLogger() *countQueryLogger {
	return &countQueryLogger{
		base:    gormlogger.Discard,
		queries: &atomic.Int32{},
	}
}

func (l *countQueryLogger) CountQueries() int32 {
	return l.queries.Load()
}

func (l *countQueryLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &countQueryLogger{
		base:    l.base.LogMode(level),
		queries: l.queries,
	}
}

func (l *countQueryLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.base.Info(ctx, msg, data...)
}

func (l *countQueryLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.base.Warn(ctx, msg, data...)
}

func (l *countQueryLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.base.Error(ctx, msg, data...)
}

func (l *countQueryLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	if strings.Contains(strings.ToLower(sql), "count(") {
		l.queries.Add(1)
	}
	l.base.Trace(ctx, begin, fc, err)
}

func attachCountQueryLogger(t *testing.T, db *gorm.DB) *countQueryLogger {
	t.Helper()

	queryLogger := newCountQueryLogger()
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	tracedDB := db.Session(&gorm.Session{Logger: queryLogger})
	model.DB = tracedDB
	model.LOG_DB = tracedDB
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
	})
	return queryLogger
}
