package controller

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/require"
)

type exportAutoResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    exportAutoResponseData `json:"data"`
}

type exportAutoResponseData struct {
	Mode     string                      `json:"mode"`
	Decision service.SmartExportDecision `json:"decision"`
	Job      dto.AsyncExportJobResponse  `json:"job"`
}

func TestExportAdminUsageLogsAutoReturnsSyncWorkbookWithinRequestedLimit(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedAdminUsageLogsForExport(t, db)
	seedListExportUsers(t, db, testListExportUser(9001, "root_exporter", "Root Exporter", common.RoleRootUser, model.UserTypeRoot))

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/export-auto", map[string]any{
		"type":        model.LogTypeConsume,
		"username":    fixture.LatestMatching.Username,
		"token_name":  fixture.LatestMatching.TokenName,
		"model_name":  fixture.LatestMatching.ModelName,
		"channel":     strconv.Itoa(fixture.LatestMatching.ChannelId),
		"group":       fixture.LatestMatching.Group,
		"request_id":  fixture.LatestMatching.RequestId,
		"column_keys": []string{"time", "model"},
		"limit":       10,
	}, 9001, common.RoleRootUser)
	ctx.Set("username", "root_exporter")

	ExportAllLogsAuto(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows(workbook.GetSheetName(0))
	require.NoError(t, err)
	require.Len(t, rows, 11)
}

func TestExportSelfUsageLogsAutoReturnsAsyncJobForDefaultLongTextColumns(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedListExportUsers(t, db, testListExportUser(7001, "self_exporter", "Self Exporter", common.RoleCommonUser, model.UserTypeEndUser))
	logs := make([]model.Log, 0, service.SmartExportUsageLogsLongTextThreshold+1)
	for index := 0; index < service.SmartExportUsageLogsLongTextThreshold+1; index++ {
		logs = append(logs, model.Log{
			UserId:           7001,
			Username:         "self_exporter",
			CreatedAt:        int64(1811000000 + index),
			Type:             model.LogTypeConsume,
			Content:          "self-long-text",
			TokenName:        "self-export-token",
			ModelName:        "gpt-4o-mini",
			Quota:            1,
			PromptTokens:     1,
			CompletionTokens: 1,
			Group:            "personal",
			RequestId:        "self-export-auto",
		})
	}
	require.NoError(t, db.CreateInBatches(logs, 200).Error)

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/log/self/export-auto", map[string]any{
		"type":       model.LogTypeConsume,
		"token_name": "self-export-token",
		"model_name": "gpt-4o-mini",
		"group":      "personal",
		"request_id": "self-export-auto",
		"limit":      service.SmartExportUsageLogsLongTextThreshold + 1,
	}, 7001, common.RoleCommonUser)
	ctx.Set("username", "self_exporter")

	ExportUserLogsAuto(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))

	var response exportAutoResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, service.SmartExportModeAsync, response.Data.Mode)
	require.Equal(t, service.SmartExportJobTypeUsageLogs, response.Data.Job.JobType)
	require.Equal(t, service.SmartExportUsageLogsLongTextThreshold, response.Data.Decision.Threshold)
}

func TestExportAdminAuditLogsAutoReturnsSyncWorkbookWithinRequestedLimit(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedAuditLogs(t, db, service.SmartExportAdminAuditThreshold+10, "quota", 9001)

	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/admin/audit-logs/export-auto", map[string]any{
		"action_module":    "quota",
		"operator_user_id": 9001,
		"limit":            10,
	}, 9001, common.RoleRootUser)
	ctx.Set("username", "root_exporter")

	ExportAdminAuditLogsAuto(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows(workbook.GetSheetName(0))
	require.NoError(t, err)
	require.Len(t, rows, 11)
}

func TestExportAdminAuditLogsAutoReturnsAsyncJobWhenRequestedLimitExceedsSyncCap(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedAuditLogs(t, db, normalizeExportLimit(0)+50, "quota", 9001)

	requestedLimit := normalizeExportLimit(0) + 50
	ctx, recorder := newListExcelExportContextWithOperator(t, http.MethodPost, "/api/admin/audit-logs/export-auto", map[string]any{
		"action_module":    "quota",
		"operator_user_id": 9001,
		"limit":            requestedLimit,
	}, 9001, common.RoleRootUser)
	ctx.Set("username", "root_exporter")

	ExportAdminAuditLogsAuto(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))

	var response exportAutoResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, service.SmartExportModeAsync, response.Data.Mode)
	require.Equal(t, service.SmartExportJobTypeAdminAuditLogs, response.Data.Job.JobType)
}
