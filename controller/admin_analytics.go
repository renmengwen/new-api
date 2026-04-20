package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetAdminAnalyticsSummary(c *gin.Context) {
	if !requireAdminAnalyticsAccess(c, service.ActionRead) {
		return
	}

	query, err := dto.ParseAdminAnalyticsQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	summary, err := service.GetOperationsAnalyticsSummary(query, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, summary)
}

func GetAdminAnalyticsModels(c *gin.Context) {
	if !requireAdminAnalyticsAccess(c, service.ActionRead) {
		return
	}

	query, err := dto.ParseAdminAnalyticsQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo := common.GetPageQuery(c)
	items, total, err := service.GetOperationsAnalyticsModels(query, pageInfo, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetAdminAnalyticsUsers(c *gin.Context) {
	if !requireAdminAnalyticsAccess(c, service.ActionRead) {
		return
	}

	query, err := dto.ParseAdminAnalyticsQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo := common.GetPageQuery(c)
	items, total, err := service.GetOperationsAnalyticsUsers(query, pageInfo, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetAdminAnalyticsDaily(c *gin.Context) {
	if !requireAdminAnalyticsAccess(c, service.ActionRead) {
		return
	}

	query, err := dto.ParseAdminAnalyticsQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items, err := service.GetOperationsAnalyticsDaily(query, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, dto.AdminAnalyticsDailyResponse{
		Items: items,
	})
}

func ExportAdminAnalytics(c *gin.Context) {
	if !requireAdminAnalyticsAccess(c, service.ActionRead, service.ActionExport) {
		return
	}

	var req dto.AdminAnalyticsExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	query, err := dto.NormalizeAdminAnalyticsQuery(req.ToQuery(), common.GetTimestamp())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	fileName, content, err := buildAdminAnalyticsExportFile(
		strings.ToLower(strings.TrimSpace(req.View)),
		query,
		strings.TrimSpace(req.QuotaDisplayType),
		normalizeExportLimit(req.Limit),
		c.GetInt("id"),
		c.GetInt("role"),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	streamExcelFile(c, fileName, content)
}

func buildAdminAnalyticsExportFile(view string, query dto.AdminAnalyticsQuery, quotaDisplayType string, limit int, requesterUserID int, requesterRole int) (string, []byte, error) {
	switch view {
	case dto.AdminAnalyticsViewModels:
		pageInfo := &common.PageInfo{Page: 1, PageSize: limit}
		items, _, err := service.GetOperationsAnalyticsModels(query, pageInfo, requesterUserID, requesterRole)
		if err != nil {
			return "", nil, err
		}
		return service.BuildExcelFile(service.ExcelFileSpec{
			FileNamePrefix: "operations_analytics_models",
			SheetName:      "models",
			Headers:        []string{"Model", "Call Count", "Prompt Tokens", "Completion Tokens", "Total Cost", "Avg Use Time", "Success Rate"},
			Rows:           buildAdminAnalyticsModelExportRows(items, quotaDisplayType),
		})
	case dto.AdminAnalyticsViewUsers:
		pageInfo := &common.PageInfo{Page: 1, PageSize: limit}
		items, _, err := service.GetOperationsAnalyticsUsers(query, pageInfo, requesterUserID, requesterRole)
		if err != nil {
			return "", nil, err
		}
		return service.BuildExcelFile(service.ExcelFileSpec{
			FileNamePrefix: "operations_analytics_users",
			SheetName:      "users",
			Headers:        []string{"User ID", "Username", "Call Count", "Model Count", "Total Tokens", "Total Cost", "Last Called At"},
			Rows:           buildAdminAnalyticsUserExportRows(items, quotaDisplayType),
		})
	case dto.AdminAnalyticsViewDaily:
		items, err := service.GetOperationsAnalyticsDaily(query, requesterUserID, requesterRole)
		if err != nil {
			return "", nil, err
		}
		return service.BuildExcelFile(service.ExcelFileSpec{
			FileNamePrefix: "operations_analytics_daily",
			SheetName:      "daily",
			Headers:        []string{"Bucket Day", "Call Count", "Total Cost", "Active Users", "Active Models"},
			Rows:           buildAdminAnalyticsDailyExportRows(items, quotaDisplayType),
		})
	default:
		return "", nil, errors.New("invalid analytics view")
	}
}

func buildAdminAnalyticsModelExportRows(items []dto.AdminAnalyticsModelItem, quotaDisplayType string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ModelName,
			strconv.FormatInt(item.CallCount, 10),
			strconv.FormatInt(item.PromptTokens, 10),
			strconv.FormatInt(item.CompletionTokens, 10),
			formatAdminAnalyticsCost(item.TotalCost, quotaDisplayType),
			formatAdminAnalyticsFloat(item.AvgUseTime),
			formatAdminAnalyticsFloat(item.SuccessRate),
		})
	}
	return rows
}

func buildAdminAnalyticsUserExportRows(items []dto.AdminAnalyticsUserItem, quotaDisplayType string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			strconv.Itoa(item.UserID),
			item.Username,
			strconv.FormatInt(item.CallCount, 10),
			strconv.FormatInt(item.ModelCount, 10),
			strconv.FormatInt(item.TotalTokens, 10),
			formatAdminAnalyticsCost(item.TotalCost, quotaDisplayType),
			formatExportTimestamp(item.LastCalledAt),
		})
	}
	return rows
}

func buildAdminAnalyticsDailyExportRows(items []dto.AdminAnalyticsDailyItem, quotaDisplayType string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.BucketDay,
			strconv.FormatInt(item.CallCount, 10),
			formatAdminAnalyticsCost(item.TotalCost, quotaDisplayType),
			strconv.FormatInt(item.ActiveUsers, 10),
			strconv.FormatInt(item.ActiveModels, 10),
		})
	}
	return rows
}

func requireAdminAnalyticsAccess(c *gin.Context, actionKeys ...string) bool {
	operator, err := service.ResolveOperatorUser(c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return false
	}

	if operator.Role < common.RoleAdminUser && operator.GetUserType() != model.UserTypeAgent {
		common.ApiError(c, errors.New("permission denied"))
		return false
	}

	for _, actionKey := range actionKeys {
		if !requireAdminActionPermission(c, service.ResourceAnalyticsManagement, actionKey) {
			return false
		}
	}

	return true
}

func formatAdminAnalyticsCost(totalCost int64, quotaDisplayType string) string {
	return service.FormatAnalyticsExportCostLabel(totalCost, quotaDisplayType)
}

func formatAdminAnalyticsFloat(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', 4, 64)
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")
	if formatted == "" {
		return "0"
	}
	return formatted
}
