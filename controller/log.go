package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetAllLogs(c *gin.Context) {
	operator, ok := requireUsageLogAdminAccess(c, service.ActionLedgerRead)
	if !ok {
		return
	}
	pageInfo := common.GetPageQuery(c)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := service.ListScopedUsageLogs(
		pageInfo,
		operator.Id,
		operator.Role,
		logType,
		startTimestamp,
		endTimestamp,
		modelName,
		username,
		tokenName,
		channel,
		group,
		requestId,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetUserLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt("id")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := model.GetUserLogs(userId, logType, startTimestamp, endTimestamp, modelName, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

// Deprecated: SearchAllLogs 已废弃，前端未使用该接口。
func SearchAllLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// Deprecated: SearchUserLogs 已废弃，前端未使用该接口。
func SearchUserLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

type usageLogExportColumn struct {
	Header string
	Value  func(*model.Log) string
}

var usageLogExportColumns = map[string]usageLogExportColumn{
	"time": {
		Header: "时间",
		Value: func(log *model.Log) string {
			return formatExportTimestamp(log.CreatedAt)
		},
	},
	"channel": {
		Header: "渠道",
		Value: func(log *model.Log) string {
			return formatUsageLogChannel(log)
		},
	},
	"username": {
		Header: "用户",
		Value: func(log *model.Log) string {
			return log.Username
		},
	},
	"token": {
		Header: "令牌",
		Value: func(log *model.Log) string {
			return log.TokenName
		},
	},
	"group": {
		Header: "分组",
		Value: func(log *model.Log) string {
			return log.Group
		},
	},
	"type": {
		Header: "类型",
		Value: func(log *model.Log) string {
			return formatUsageLogType(log.Type)
		},
	},
	"model": {
		Header: "模型",
		Value: func(log *model.Log) string {
			return service.GetUsageLogExportModelLabel(log)
		},
	},
	"use_time": {
		Header: "用时/首字",
		Value: func(log *model.Log) string {
			return service.GetUsageLogExportUseTimeLabel(log)
		},
	},
	"prompt": {
		Header: "输入",
		Value: func(log *model.Log) string {
			return strconv.Itoa(log.PromptTokens)
		},
	},
	"completion": {
		Header: "输出",
		Value: func(log *model.Log) string {
			return strconv.Itoa(log.CompletionTokens)
		},
	},
	"cost": {
		Header: "花费",
		Value: func(log *model.Log) string {
			return strconv.Itoa(log.Quota)
		},
	},
	"retry": {
		Header: "重试",
		Value: func(log *model.Log) string {
			return formatUsageLogRetry(log.Other)
		},
	},
	"ip": {
		Header: "IP",
		Value: func(log *model.Log) string {
			return log.Ip
		},
	},
	"details": {
		Header: "详情",
		Value: func(log *model.Log) string {
			return log.Content
		},
	},
}

var defaultAdminUsageLogExportColumnKeys = []string{
	"time",
	"channel",
	"username",
	"token",
	"group",
	"type",
	"model",
	"use_time",
	"prompt",
	"completion",
	"cost",
	"retry",
	"ip",
	"details",
}

var defaultSelfUsageLogExportColumnKeys = []string{
	"time",
	"token",
	"group",
	"type",
	"model",
	"use_time",
	"prompt",
	"completion",
	"cost",
	"ip",
	"details",
}

func ExportAllLogs(c *gin.Context) {
	operator, ok := requireUsageLogAdminAccess(c, service.ActionLedgerRead)
	if !ok {
		return
	}
	var req dto.UsageLogExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	exportAdminUsageLogsByRequest(c, operator.Id, operator.Role, req)
}

func ExportAllLogsAuto(c *gin.Context) {
	operator, ok := requireUsageLogAdminAccess(c, service.ActionLedgerRead)
	if !ok {
		return
	}
	var req dto.UsageLogExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	decision, err := service.DecideUsageLogSmartExport(operator.Id, operator.Role, req, resolveUsageLogExportColumnKeys(req.ColumnKeys, true), true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if decision.Mode == service.SmartExportModeAsync {
		job, err := service.CreateAdminUsageLogExportJob(operator.Id, operator.Role, req)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		respondAsyncExportJobCreated(c, decision, job)
		return
	}

	exportAdminUsageLogsByRequest(c, operator.Id, operator.Role, req)
}

func exportAdminUsageLogsByRequest(c *gin.Context, requesterUserID int, requesterRole int, req dto.UsageLogExportRequest) {
	channel, _ := strconv.Atoi(req.Channel)
	logs, err := service.ListScopedUsageLogsForExport(
		requesterUserID,
		requesterRole,
		req.Type,
		req.StartTimestamp,
		req.EndTimestamp,
		req.ModelName,
		req.Username,
		req.TokenName,
		normalizeExportLimit(req.Limit),
		channel,
		req.Group,
		req.RequestID,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	exportUsageLogs(c, logs, req.ColumnKeys, req.QuotaDisplayType, true)
}

func CreateAdminUsageLogExportJob(c *gin.Context) {
	operator, ok := requireUsageLogAdminAccess(c, service.ActionLedgerRead)
	if !ok {
		return
	}

	var req dto.UsageLogExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	job, err := service.CreateAdminUsageLogExportJob(operator.Id, operator.Role, req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, service.BuildAsyncExportJobResponse(job))
}

func ExportUserLogs(c *gin.Context) {
	var req dto.UsageLogExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	exportSelfUsageLogsByRequest(c, c.GetInt("id"), req)
}

func ExportUserLogsAuto(c *gin.Context) {
	var req dto.UsageLogExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	decision, err := service.DecideUsageLogSmartExport(c.GetInt("id"), c.GetInt("role"), req, resolveUsageLogExportColumnKeys(req.ColumnKeys, false), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if decision.Mode == service.SmartExportModeAsync {
		job, err := service.CreateSelfUsageLogExportJob(c.GetInt("id"), c.GetInt("role"), req)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		respondAsyncExportJobCreated(c, decision, job)
		return
	}

	exportSelfUsageLogsByRequest(c, c.GetInt("id"), req)
}

func exportSelfUsageLogsByRequest(c *gin.Context, requesterUserID int, req dto.UsageLogExportRequest) {
	logs, err := model.GetUserLogsForExport(
		requesterUserID,
		req.Type,
		req.StartTimestamp,
		req.EndTimestamp,
		req.ModelName,
		req.TokenName,
		req.Limit,
		req.Group,
		req.RequestID,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	exportUsageLogs(c, logs, req.ColumnKeys, req.QuotaDisplayType, false)
}

func CreateSelfUsageLogExportJob(c *gin.Context) {
	var req dto.UsageLogExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	job, err := service.CreateSelfUsageLogExportJob(c.GetInt("id"), c.GetInt("role"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, service.BuildAsyncExportJobResponse(job))
}

func exportUsageLogs(c *gin.Context, logs []*model.Log, requestedColumnKeys []string, requestedQuotaDisplayType string, isAdmin bool) {
	columnKeys := resolveUsageLogExportColumnKeys(requestedColumnKeys, isAdmin)
	if len(columnKeys) == 0 {
		common.ApiError(c, errors.New("no export columns selected"))
		return
	}
	headers := make([]string, 0, len(columnKeys))
	rows := make([][]string, 0, len(logs))

	for _, key := range columnKeys {
		headers = append(headers, usageLogExportColumns[key].Header)
	}
	for _, log := range logs {
		row := make([]string, 0, len(columnKeys))
		for _, key := range columnKeys {
			row = append(row, resolveUsageLogExportValue(log, key, requestedQuotaDisplayType))
		}
		rows = append(rows, row)
	}

	fileName, content, err := service.BuildExcelFile(service.ExcelFileSpec{
		FileNamePrefix: "使用日志",
		SheetName:      "使用日志",
		Headers:        headers,
		Rows:           rows,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	streamExcelFile(c, fileName, content)
}

func resolveUsageLogExportValue(log *model.Log, key string, requestedQuotaDisplayType string) string {
	if key == "cost" {
		return service.GetUsageLogExportCostLabel(log, requestedQuotaDisplayType)
	}
	return usageLogExportColumns[key].Value(log)
}

func resolveUsageLogExportColumnKeys(requestedColumnKeys []string, isAdmin bool) []string {
	allowedKeys := defaultSelfUsageLogExportColumnKeys
	if isAdmin {
		allowedKeys = defaultAdminUsageLogExportColumnKeys
	}
	if requestedColumnKeys == nil {
		return allowedKeys
	}
	allowedSet := make(map[string]struct{}, len(allowedKeys))
	for _, key := range allowedKeys {
		allowedSet[key] = struct{}{}
	}
	selected := make([]string, 0, len(requestedColumnKeys))
	seen := make(map[string]struct{}, len(requestedColumnKeys))
	for _, rawKey := range requestedColumnKeys {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			continue
		}
		if _, ok := allowedSet[key]; !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		selected = append(selected, key)
	}
	return selected
}

func formatUsageLogChannel(log *model.Log) string {
	if log.ChannelId == 0 {
		return ""
	}
	if log.ChannelName != "" {
		return fmt.Sprintf("%d - %s", log.ChannelId, log.ChannelName)
	}
	return strconv.Itoa(log.ChannelId)
}

func formatUsageLogType(logType int) string {
	switch logType {
	case model.LogTypeTopup:
		return "充值"
	case model.LogTypeConsume:
		return "消费"
	case model.LogTypeManage:
		return "管理"
	case model.LogTypeSystem:
		return "系统"
	case model.LogTypeError:
		return "错误"
	case model.LogTypeRefund:
		return "退款"
	default:
		return "未知"
	}
}

func formatUsageLogUseTime(log *model.Log) string {
	if log.Type != model.LogTypeConsume && log.Type != model.LogTypeError {
		return ""
	}
	return strconv.Itoa(log.UseTime)
}

func formatUsageLogRetry(otherJSON string) string {
	if strings.TrimSpace(otherJSON) == "" {
		return ""
	}
	var other map[string]any
	if err := common.UnmarshalJsonStr(otherJSON, &other); err != nil {
		return ""
	}
	adminInfo, ok := other["admin_info"].(map[string]any)
	if !ok {
		return ""
	}
	useChannels, ok := adminInfo["use_channel"].([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(useChannels))
	for _, useChannel := range useChannels {
		part := strings.TrimSpace(fmt.Sprint(useChannel))
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "->")
}

func GetLogByKey(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(200, gin.H{
			"success": false,
			"message": "无效的令牌",
		})
		return
	}
	logs, err := model.GetLogByTokenId(tokenId)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    logs,
	})
}

func GetLogsStat(c *gin.Context) {
	operator, ok := requireUsageLogAdminAccess(c, service.ActionReadSummary)
	if !ok {
		return
	}
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	_ = logType
	stat, err := service.GetScopedUsageLogStat(
		operator.Id,
		operator.Role,
		startTimestamp,
		endTimestamp,
		modelName,
		username,
		tokenName,
		channel,
		group,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, "")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": stat.Quota,
			"rpm":   stat.Rpm,
			"tpm":   stat.Tpm,
		},
	})
	return
}

func requireUsageLogAdminAccess(c *gin.Context, actionKey string) (*model.User, bool) {
	if err := service.RequirePermissionAction(c.GetInt("id"), c.GetInt("role"), service.ResourceQuotaManagement, actionKey); err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	operator, err := service.ResolveOperatorUser(c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	return operator, true
}

func GetLogsSelfStat(c *gin.Context) {
	username := c.GetString("username")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	quotaNum, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, tokenName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": quotaNum.Quota,
			"rpm":   quotaNum.Rpm,
			"tpm":   quotaNum.Tpm,
			//"token": tokenNum,
		},
	})
	return
}

func DeleteHistoryLogs(c *gin.Context) {
	targetTimestamp, _ := strconv.ParseInt(c.Query("target_timestamp"), 10, 64)
	if targetTimestamp == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "target timestamp is required",
		})
		return
	}
	count, err := model.DeleteOldLog(c.Request.Context(), targetTimestamp, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	createSettingAuditLog(c, settingAuditMetaClearHistoryLogs, 0, "", marshalSettingAuditPayload(map[string]any{
		"target_timestamp": targetTimestamp,
		"deleted_count":    count,
	}))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
	return
}
