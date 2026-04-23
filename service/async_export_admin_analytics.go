package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

type asyncAdminAnalyticsExportPayload struct {
	JobType          string                  `json:"job_type"`
	Query            dto.AdminAnalyticsQuery `json:"query"`
	QuotaDisplayType string                  `json:"quota_display_type"`
	Limit            int                     `json:"limit"`
}

var asyncAdminAnalyticsModelExportHeaders = []string{"Model", "Call Count", "Prompt Tokens", "Completion Tokens", "Total Cost", "Avg Use Time", "Success Rate"}
var asyncAdminAnalyticsUserExportHeaders = []string{"User ID", "Username", "Call Count", "Model Count", "Total Tokens", "Total Cost", "Last Called At"}
var asyncAdminAnalyticsDailyExportHeaders = []string{"Bucket Day", "Call Count", "Total Cost", "Active Users", "Active Models"}

func init() {
	RegisterAsyncExportExecutor(SmartExportJobTypeAdminAnalyticsModels, executeAdminAnalyticsExportJob)
	RegisterAsyncExportExecutor(SmartExportJobTypeAdminAnalyticsUsers, executeAdminAnalyticsExportJob)
	RegisterAsyncExportExecutor(SmartExportJobTypeAdminAnalyticsDaily, executeAdminAnalyticsExportJob)
}

func CreateAdminAnalyticsExportJob(requesterUserID int, requesterRole int, req dto.AdminAnalyticsExportRequest) (*model.AsyncExportJob, error) {
	query, err := dto.NormalizeAdminAnalyticsQuery(req.ToQuery(), common.GetTimestamp())
	if err != nil {
		return nil, err
	}

	jobType, _, _, err := resolveAdminAnalyticsExportTarget(strings.ToLower(strings.TrimSpace(req.View)))
	if err != nil {
		return nil, err
	}

	payload := asyncAdminAnalyticsExportPayload{
		JobType:          jobType,
		Query:            query,
		QuotaDisplayType: strings.TrimSpace(req.QuotaDisplayType),
		Limit:            normalizeAsyncExportLimit(req.Limit),
	}
	return CreateAsyncExportJob(jobType, requesterUserID, requesterRole, payload)
}

func executeAdminAnalyticsExportJob(job *model.AsyncExportJob) error {
	var payload asyncAdminAnalyticsExportPayload
	if err := DecodeAsyncExportPayload(job, &payload); err != nil {
		return err
	}
	if payload.JobType == "" {
		payload.JobType = job.JobType
	}

	jobType, filePrefix, sheetName, err := resolveAdminAnalyticsExportTarget(payload.JobType)
	if err != nil {
		return err
	}

	switch jobType {
	case SmartExportJobTypeAdminAnalyticsModels:
		return writeAsyncExportJobFile(job, filePrefix, sheetName, asyncAdminAnalyticsModelExportHeaders, func(page int, pageSize int) (AsyncExportPage, error) {
			return fetchAdminAnalyticsModelExportPage(job.RequesterUserId, job.RequesterRole, payload, page, pageSize)
		})
	case SmartExportJobTypeAdminAnalyticsUsers:
		return writeAsyncExportJobFile(job, filePrefix, sheetName, asyncAdminAnalyticsUserExportHeaders, func(page int, pageSize int) (AsyncExportPage, error) {
			return fetchAdminAnalyticsUserExportPage(job.RequesterUserId, job.RequesterRole, payload, page, pageSize)
		})
	case SmartExportJobTypeAdminAnalyticsDaily:
		return writeAsyncExportJobFile(job, filePrefix, sheetName, asyncAdminAnalyticsDailyExportHeaders, func(page int, pageSize int) (AsyncExportPage, error) {
			return fetchAdminAnalyticsDailyExportPage(job.RequesterUserId, job.RequesterRole, payload, page)
		})
	default:
		return errors.New("invalid analytics view")
	}
}

func resolveAdminAnalyticsExportTarget(view string) (jobType string, filePrefix string, sheetName string, err error) {
	switch strings.ToLower(strings.TrimSpace(view)) {
	case dto.AdminAnalyticsViewModels, SmartExportJobTypeAdminAnalyticsModels:
		return SmartExportJobTypeAdminAnalyticsModels, "operations_analytics_models", "models", nil
	case dto.AdminAnalyticsViewUsers, SmartExportJobTypeAdminAnalyticsUsers:
		return SmartExportJobTypeAdminAnalyticsUsers, "operations_analytics_users", "users", nil
	case dto.AdminAnalyticsViewDaily, SmartExportJobTypeAdminAnalyticsDaily:
		return SmartExportJobTypeAdminAnalyticsDaily, "operations_analytics_daily", "daily", nil
	default:
		return "", "", "", errors.New("invalid analytics view")
	}
}

func fetchAdminAnalyticsModelExportPage(requesterUserID int, requesterRole int, payload asyncAdminAnalyticsExportPayload, page int, pageSize int) (AsyncExportPage, error) {
	offset := (page - 1) * pageSize
	if asyncExportLimitReached(offset, payload.Limit) {
		return AsyncExportPage{Done: true}, nil
	}

	pageInfo := &common.PageInfo{
		Page:     page,
		PageSize: pageSize,
	}
	items, total, err := GetOperationsAnalyticsModels(payload.Query, pageInfo, requesterUserID, requesterRole)
	if err != nil {
		return AsyncExportPage{}, err
	}

	items = trimAsyncExportItemsToLimit(items, offset, payload.Limit)
	if len(items) == 0 {
		return AsyncExportPage{Done: true}, nil
	}

	return AsyncExportPage{
		Rows: buildAsyncAdminAnalyticsModelExportRows(items, payload.QuotaDisplayType),
		Done: isAsyncExportPageDone(len(items), pageSize, offset, payload.Limit, total),
	}, nil
}

func fetchAdminAnalyticsUserExportPage(requesterUserID int, requesterRole int, payload asyncAdminAnalyticsExportPayload, page int, pageSize int) (AsyncExportPage, error) {
	offset := (page - 1) * pageSize
	if asyncExportLimitReached(offset, payload.Limit) {
		return AsyncExportPage{Done: true}, nil
	}

	pageInfo := &common.PageInfo{
		Page:     page,
		PageSize: pageSize,
	}
	items, total, err := GetOperationsAnalyticsUsers(payload.Query, pageInfo, requesterUserID, requesterRole)
	if err != nil {
		return AsyncExportPage{}, err
	}

	items = trimAsyncExportItemsToLimit(items, offset, payload.Limit)
	if len(items) == 0 {
		return AsyncExportPage{Done: true}, nil
	}

	return AsyncExportPage{
		Rows: buildAsyncAdminAnalyticsUserExportRows(items, payload.QuotaDisplayType),
		Done: isAsyncExportPageDone(len(items), pageSize, offset, payload.Limit, total),
	}, nil
}

func fetchAdminAnalyticsDailyExportPage(requesterUserID int, requesterRole int, payload asyncAdminAnalyticsExportPayload, page int) (AsyncExportPage, error) {
	if page > 1 {
		return AsyncExportPage{Done: true}, nil
	}

	items, err := GetOperationsAnalyticsDaily(payload.Query, requesterUserID, requesterRole)
	if err != nil {
		return AsyncExportPage{}, err
	}
	return AsyncExportPage{
		Rows: buildAsyncAdminAnalyticsDailyExportRows(items, payload.QuotaDisplayType),
		Done: true,
	}, nil
}

func buildAsyncAdminAnalyticsModelExportRows(items []dto.AdminAnalyticsModelItem, quotaDisplayType string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ModelName,
			strconv.FormatInt(item.CallCount, 10),
			strconv.FormatInt(item.PromptTokens, 10),
			strconv.FormatInt(item.CompletionTokens, 10),
			FormatAnalyticsExportCostLabel(item.TotalCost, quotaDisplayType),
			formatAsyncAdminAnalyticsFloat(item.AvgUseTime),
			formatAsyncAdminAnalyticsFloat(item.SuccessRate),
		})
	}
	return rows
}

func buildAsyncAdminAnalyticsUserExportRows(items []dto.AdminAnalyticsUserItem, quotaDisplayType string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			strconv.Itoa(item.UserID),
			item.Username,
			strconv.FormatInt(item.CallCount, 10),
			strconv.FormatInt(item.ModelCount, 10),
			strconv.FormatInt(item.TotalTokens, 10),
			FormatAnalyticsExportCostLabel(item.TotalCost, quotaDisplayType),
			formatAsyncExportTimestamp(item.LastCalledAt),
		})
	}
	return rows
}

func buildAsyncAdminAnalyticsDailyExportRows(items []dto.AdminAnalyticsDailyItem, quotaDisplayType string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.BucketDay,
			strconv.FormatInt(item.CallCount, 10),
			FormatAnalyticsExportCostLabel(item.TotalCost, quotaDisplayType),
			strconv.FormatInt(item.ActiveUsers, 10),
			strconv.FormatInt(item.ActiveModels, 10),
		})
	}
	return rows
}

func formatAsyncAdminAnalyticsFloat(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', 4, 64)
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")
	if formatted == "" {
		return "0"
	}
	return formatted
}
