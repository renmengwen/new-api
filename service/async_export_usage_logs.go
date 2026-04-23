package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

const (
	asyncExportWriterPageSize   = 1000
	asyncUsageLogsFilePrefix    = "使用日志"
	asyncUsageLogsSheetName     = "使用日志"
	asyncUsageLogAdminViewScope = "admin"
	asyncUsageLogSelfViewScope  = "self"
)

type asyncUsageLogExportPayload struct {
	Scope      string                    `json:"scope"`
	Request    dto.UsageLogExportRequest `json:"request"`
	ColumnKeys []string                  `json:"column_keys"`
	Limit      int                       `json:"limit"`
}

type asyncUsageLogExportColumn struct {
	Header string
	Value  func(*model.Log, string) string
}

var asyncUsageLogExportColumns = map[string]asyncUsageLogExportColumn{
	"time": {
		Header: "时间",
		Value: func(log *model.Log, _ string) string {
			return formatAsyncExportTimestamp(log.CreatedAt)
		},
	},
	"channel": {
		Header: "渠道",
		Value: func(log *model.Log, _ string) string {
			return formatAsyncUsageLogChannel(log)
		},
	},
	"username": {
		Header: "用户",
		Value: func(log *model.Log, _ string) string {
			return log.Username
		},
	},
	"token": {
		Header: "令牌",
		Value: func(log *model.Log, _ string) string {
			return log.TokenName
		},
	},
	"group": {
		Header: "分组",
		Value: func(log *model.Log, _ string) string {
			return log.Group
		},
	},
	"type": {
		Header: "类型",
		Value: func(log *model.Log, _ string) string {
			return formatAsyncUsageLogType(log.Type)
		},
	},
	"model": {
		Header: "模型",
		Value: func(log *model.Log, _ string) string {
			return GetUsageLogExportModelLabel(log)
		},
	},
	"use_time": {
		Header: "用时/首字",
		Value: func(log *model.Log, _ string) string {
			return GetUsageLogExportUseTimeLabel(log)
		},
	},
	"prompt": {
		Header: "输入",
		Value: func(log *model.Log, _ string) string {
			return strconv.Itoa(log.PromptTokens)
		},
	},
	"completion": {
		Header: "输出",
		Value: func(log *model.Log, _ string) string {
			return strconv.Itoa(log.CompletionTokens)
		},
	},
	"cost": {
		Header: "花费",
		Value: func(log *model.Log, quotaDisplayType string) string {
			return GetUsageLogExportCostLabel(log, quotaDisplayType)
		},
	},
	"retry": {
		Header: "重试",
		Value: func(log *model.Log, _ string) string {
			return formatAsyncUsageLogRetry(log.Other)
		},
	},
	"ip": {
		Header: "IP",
		Value: func(log *model.Log, _ string) string {
			return log.Ip
		},
	},
	"details": {
		Header: "详情",
		Value: func(log *model.Log, _ string) string {
			return log.Content
		},
	},
}

var asyncAdminUsageLogExportColumnKeys = []string{
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

var asyncSelfUsageLogExportColumnKeys = []string{
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

func init() {
	RegisterAsyncExportExecutor(SmartExportJobTypeUsageLogs, executeUsageLogAsyncExportJob)
}

func CreateAdminUsageLogExportJob(requesterUserID int, requesterRole int, req dto.UsageLogExportRequest) (*model.AsyncExportJob, error) {
	return createUsageLogExportJob(requesterUserID, requesterRole, req, asyncUsageLogAdminViewScope)
}

func CreateSelfUsageLogExportJob(requesterUserID int, requesterRole int, req dto.UsageLogExportRequest) (*model.AsyncExportJob, error) {
	if _, _, err := model.GetUserLogs(
		requesterUserID,
		req.Type,
		req.StartTimestamp,
		req.EndTimestamp,
		req.ModelName,
		req.TokenName,
		0,
		1,
		req.Group,
		req.RequestID,
	); err != nil {
		return nil, err
	}
	return createUsageLogExportJob(requesterUserID, requesterRole, req, asyncUsageLogSelfViewScope)
}

func createUsageLogExportJob(requesterUserID int, requesterRole int, req dto.UsageLogExportRequest, scope string) (*model.AsyncExportJob, error) {
	columnKeys := resolveAsyncUsageLogExportColumnKeys(req.ColumnKeys, scope == asyncUsageLogAdminViewScope)
	if len(columnKeys) == 0 {
		return nil, errors.New("no export columns selected")
	}

	payload := asyncUsageLogExportPayload{
		Scope:      scope,
		Request:    req,
		ColumnKeys: columnKeys,
		Limit:      normalizeAsyncExportLimit(req.Limit),
	}
	return CreateAsyncExportJob(SmartExportJobTypeUsageLogs, requesterUserID, requesterRole, payload)
}

func executeUsageLogAsyncExportJob(job *model.AsyncExportJob) error {
	if job == nil {
		return errors.New("async export job is nil")
	}

	var payload asyncUsageLogExportPayload
	if err := DecodeAsyncExportPayload(job, &payload); err != nil {
		return err
	}
	if payload.Scope == "" {
		payload.Scope = asyncUsageLogAdminViewScope
	}
	if payload.Limit <= 0 {
		payload.Limit = normalizeAsyncExportLimit(payload.Request.Limit)
	}
	if len(payload.ColumnKeys) == 0 {
		payload.ColumnKeys = resolveAsyncUsageLogExportColumnKeys(payload.Request.ColumnKeys, payload.Scope == asyncUsageLogAdminViewScope)
	}
	headers := buildAsyncUsageLogExportHeaders(payload.ColumnKeys)
	if len(headers) == 0 {
		return errors.New("no export columns selected")
	}

	return writeAsyncExportJobFile(job, asyncUsageLogsFilePrefix, asyncUsageLogsSheetName, headers, func(page int, pageSize int) (AsyncExportPage, error) {
		if payload.Scope == asyncUsageLogSelfViewScope {
			return fetchSelfUsageLogExportPage(job.RequesterUserId, payload, page, pageSize)
		}
		return fetchAdminUsageLogExportPage(job.RequesterUserId, job.RequesterRole, payload, page, pageSize)
	})
}

func fetchAdminUsageLogExportPage(requesterUserID int, requesterRole int, payload asyncUsageLogExportPayload, page int, pageSize int) (AsyncExportPage, error) {
	offset := (page - 1) * pageSize
	if asyncExportLimitReached(offset, payload.Limit) {
		return AsyncExportPage{Done: true}, nil
	}

	channel, _ := strconv.Atoi(strings.TrimSpace(payload.Request.Channel))
	pageInfo := &common.PageInfo{
		Page:     page,
		PageSize: pageSize,
	}
	logs, total, err := ListScopedUsageLogs(
		pageInfo,
		requesterUserID,
		requesterRole,
		payload.Request.Type,
		payload.Request.StartTimestamp,
		payload.Request.EndTimestamp,
		payload.Request.ModelName,
		payload.Request.Username,
		payload.Request.TokenName,
		channel,
		payload.Request.Group,
		payload.Request.RequestID,
	)
	if err != nil {
		return AsyncExportPage{}, err
	}

	return buildAsyncUsageLogExportPage(logs, payload.ColumnKeys, payload.Request.QuotaDisplayType, offset, payload.Limit, total), nil
}

func fetchSelfUsageLogExportPage(requesterUserID int, payload asyncUsageLogExportPayload, page int, pageSize int) (AsyncExportPage, error) {
	offset := (page - 1) * pageSize
	if asyncExportLimitReached(offset, payload.Limit) {
		return AsyncExportPage{Done: true}, nil
	}

	logs, total, err := model.GetUserLogs(
		requesterUserID,
		payload.Request.Type,
		payload.Request.StartTimestamp,
		payload.Request.EndTimestamp,
		payload.Request.ModelName,
		payload.Request.TokenName,
		offset,
		pageSize,
		payload.Request.Group,
		payload.Request.RequestID,
	)
	if err != nil {
		return AsyncExportPage{}, err
	}

	return buildAsyncUsageLogExportPage(logs, payload.ColumnKeys, payload.Request.QuotaDisplayType, offset, payload.Limit, total), nil
}

func buildAsyncUsageLogExportPage(logs []*model.Log, columnKeys []string, quotaDisplayType string, offset int, limit int, total int64) AsyncExportPage {
	if len(logs) == 0 {
		return AsyncExportPage{Done: true}
	}
	logs = trimAsyncExportItemsToLimit(logs, offset, limit)
	if len(logs) == 0 {
		return AsyncExportPage{Done: true}
	}

	rows := buildAsyncUsageLogExportRows(logs, columnKeys, quotaDisplayType)
	done := isAsyncExportPageDone(len(logs), asyncExportWriterPageSize, offset, limit, total)
	return AsyncExportPage{
		Rows: rows,
		Done: done,
	}
}

func buildAsyncUsageLogExportHeaders(columnKeys []string) []string {
	headers := make([]string, 0, len(columnKeys))
	for _, key := range columnKeys {
		column, ok := asyncUsageLogExportColumns[key]
		if !ok {
			continue
		}
		headers = append(headers, column.Header)
	}
	return headers
}

func buildAsyncUsageLogExportRows(logs []*model.Log, columnKeys []string, quotaDisplayType string) [][]string {
	rows := make([][]string, 0, len(logs))
	for _, log := range logs {
		row := make([]string, 0, len(columnKeys))
		for _, key := range columnKeys {
			column, ok := asyncUsageLogExportColumns[key]
			if !ok {
				continue
			}
			row = append(row, column.Value(log, quotaDisplayType))
		}
		rows = append(rows, row)
	}
	return rows
}

func resolveAsyncUsageLogExportColumnKeys(requestedColumnKeys []string, isAdmin bool) []string {
	allowedKeys := asyncSelfUsageLogExportColumnKeys
	if isAdmin {
		allowedKeys = asyncAdminUsageLogExportColumnKeys
	}
	if requestedColumnKeys == nil {
		return append([]string(nil), allowedKeys...)
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

func formatAsyncUsageLogChannel(log *model.Log) string {
	if log.ChannelId == 0 {
		return ""
	}
	if log.ChannelName != "" {
		return fmt.Sprintf("%d - %s", log.ChannelId, log.ChannelName)
	}
	return strconv.Itoa(log.ChannelId)
}

func formatAsyncUsageLogType(logType int) string {
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

func formatAsyncUsageLogRetry(otherJSON string) string {
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

func normalizeAsyncExportLimit(limit int) int {
	if limit <= 0 {
		return 0
	}
	return limit
}

func writeAsyncExportJobFile(job *model.AsyncExportJob, fileNamePrefix string, sheetName string, headers []string, fetchPage func(page int, pageSize int) (AsyncExportPage, error)) error {
	if job == nil {
		return errors.New("async export job is nil")
	}

	filePath := BuildAsyncExportArtifactPath(job.Id, fileNamePrefix)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	rowCount, err := WriteAsyncExportXLSX(AsyncExportWriterSpec{
		FilePath:  filePath,
		SheetName: sheetName,
		Headers:   headers,
		PageSize:  asyncExportWriterPageSize,
		FetchPage: fetchPage,
	})
	if err != nil {
		_ = os.Remove(filePath)
		return err
	}

	return FinalizeAsyncExportJob(job.Id, filepath.Base(filePath), filePath, rowCount)
}

func formatAsyncExportTimestamp(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}
