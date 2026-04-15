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
	logs, total, err := model.GetAllLogs(logType, startTimestamp, endTimestamp, modelName, username, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), channel, group, requestId)
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
			return log.ModelName
		},
	},
	"use_time": {
		Header: "用时/首字",
		Value: func(log *model.Log) string {
			return formatUsageLogUseTime(log)
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
	var req dto.UsageLogExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	logs, _, err := model.GetAllLogs(
		req.Type,
		req.StartTimestamp,
		req.EndTimestamp,
		req.ModelName,
		req.Username,
		req.TokenName,
		0,
		normalizeExportLimit(req.Limit),
		req.Channel,
		req.Group,
		req.RequestID,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	exportUsageLogs(c, logs, req.ColumnKeys, true)
}

func ExportUserLogs(c *gin.Context) {
	var req dto.UsageLogExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	logs, _, err := model.GetUserLogs(
		c.GetInt("id"),
		req.Type,
		req.StartTimestamp,
		req.EndTimestamp,
		req.ModelName,
		req.TokenName,
		0,
		normalizeExportLimit(req.Limit),
		req.Group,
		req.RequestID,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	exportUsageLogs(c, logs, req.ColumnKeys, false)
}

func exportUsageLogs(c *gin.Context, logs []*model.Log, requestedColumnKeys []string, isAdmin bool) {
	columnKeys := resolveUsageLogExportColumnKeys(requestedColumnKeys, isAdmin)
	headers := make([]string, 0, len(columnKeys))
	rows := make([][]string, 0, len(logs))

	for _, key := range columnKeys {
		headers = append(headers, usageLogExportColumns[key].Header)
	}
	for _, log := range logs {
		row := make([]string, 0, len(columnKeys))
		for _, key := range columnKeys {
			row = append(row, usageLogExportColumns[key].Value(log))
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

func resolveUsageLogExportColumnKeys(requestedColumnKeys []string, isAdmin bool) []string {
	allowedKeys := defaultSelfUsageLogExportColumnKeys
	if isAdmin {
		allowedKeys = defaultAdminUsageLogExportColumnKeys
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
	if len(selected) > 0 {
		return selected
	}
	return allowedKeys
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
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
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
