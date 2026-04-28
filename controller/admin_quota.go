package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type adjustUserQuotaRequest struct {
	TargetUserId int    `json:"target_user_id"`
	Delta        int    `json:"delta"`
	Reason       string `json:"reason"`
	Remark       string `json:"remark"`
}

type adjustUserQuotaBatchRequest struct {
	TargetUserIds []int  `json:"target_user_ids"`
	Delta         int    `json:"delta"`
	Reason        string `json:"reason"`
	Remark        string `json:"remark"`
}

func GetUserQuotaSummary(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionReadSummary) {
		return
	}
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	data, err := service.GetScopedUserQuotaSummary(userId, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func AdjustUserQuota(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionAdjust) {
		return
	}
	var req adjustUserQuotaRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	result, err := service.AdjustUserQuota(service.AdjustUserQuotaRequest{
		OperatorUserId:   c.GetInt("id"),
		OperatorUserType: service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")),
		OperatorRole:     c.GetInt("role"),
		TargetUserId:     req.TargetUserId,
		Delta:            req.Delta,
		Reason:           req.Reason,
		Remark:           req.Remark,
		IP:               c.ClientIP(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetQuotaLedger(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionLedgerRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	userId, _ := strconv.Atoi(c.Query("user_id"))
	operatorUserId, _ := strconv.Atoi(c.Query("operator_user_id"))
	entryType := c.Query("entry_type")

	items, total, err := service.ListQuotaLedger(pageInfo, c.GetInt("id"), c.GetInt("role"), userId, operatorUserId, entryType)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetQuotaCostSummary(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionLedgerRead) {
		return
	}

	query, err := dto.ParseAdminQuotaCostSummaryQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo := common.GetPageQuery(c)
	items, total, err := service.ListQuotaCostSummary(query, pageInfo, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func ExportQuotaLedger(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionLedgerRead) {
		return
	}

	var req dto.AdminQuotaLedgerExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	exportQuotaLedgerByRequest(c, c.GetInt("id"), c.GetInt("role"), req)
}

func ExportQuotaLedgerAuto(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionLedgerRead) {
		return
	}

	var req dto.AdminQuotaLedgerExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	decision, err := service.DecideQuotaLedgerSmartExport(c.GetInt("id"), c.GetInt("role"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if decision.Mode == service.SmartExportModeAsync {
		job, err := service.CreateQuotaLedgerExportJob(c.GetInt("id"), c.GetInt("role"), req)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		respondAsyncExportJobCreated(c, decision, job)
		return
	}

	exportQuotaLedgerByRequest(c, c.GetInt("id"), c.GetInt("role"), req)
}

func ExportQuotaCostSummaryAuto(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionLedgerRead) {
		return
	}

	var req dto.AdminQuotaCostSummaryExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	query, err := dto.NormalizeAdminQuotaCostSummaryQuery(req.AdminQuotaCostSummaryQuery, common.GetTimestamp())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	normalizedReq := req
	normalizedReq.AdminQuotaCostSummaryQuery = query

	decision, err := service.DecideQuotaCostSummarySmartExport(c.GetInt("id"), c.GetInt("role"), normalizedReq)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if decision.Mode == service.SmartExportModeAsync {
		job, err := service.CreateQuotaCostSummaryExportJob(c.GetInt("id"), c.GetInt("role"), normalizedReq)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		respondAsyncExportJobCreated(c, decision, job)
		return
	}

	exportQuotaCostSummaryByRequest(c, c.GetInt("id"), c.GetInt("role"), query, req.Limit)
}

func exportQuotaLedgerByRequest(c *gin.Context, requesterUserID int, requesterRole int, req dto.AdminQuotaLedgerExportRequest) {
	items, _, err := service.ListQuotaLedgerForExport(
		requesterUserID,
		requesterRole,
		req.UserID,
		req.OperatorUserID,
		req.EntryType,
		normalizeExportLimit(req.Limit),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	fileName, content, err := service.BuildExcelFile(service.ExcelFileSpec{
		FileNamePrefix: "额度流水",
		SheetName:      "额度流水",
		Headers:        []string{"ID", "账户", "操作人", "类型", "方向", "额度", "变更前", "变更后", "模型名称", "来源", "原因", "备注", "时间"},
		Rows:           buildQuotaLedgerExportRows(items),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	streamExcelFile(c, fileName, content)
}

func exportQuotaCostSummaryByRequest(c *gin.Context, requesterUserID int, requesterRole int, query dto.AdminQuotaCostSummaryQuery, limit int) {
	items, err := service.ListQuotaCostSummaryForExport(query, requesterUserID, requesterRole, normalizeQuotaCostSummaryExportLimit(limit))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	fileName, content, err := service.BuildExcelFile(service.ExcelFileSpec{
		FileNamePrefix: "额度成本汇总",
		SheetName:      "成本汇总",
		Headers:        service.QuotaCostSummaryExportHeaders(),
		Rows:           service.BuildQuotaCostSummaryExportRows(items),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	streamExcelFile(c, fileName, content)
}

func normalizeQuotaCostSummaryExportLimit(limit int) int {
	if limit <= 0 || limit > service.SmartExportQuotaCostSummaryThreshold {
		return service.SmartExportQuotaCostSummaryThreshold
	}
	return limit
}

func CreateQuotaLedgerExportJob(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionLedgerRead) {
		return
	}

	var req dto.AdminQuotaLedgerExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	job, err := service.CreateQuotaLedgerExportJob(c.GetInt("id"), c.GetInt("role"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, service.BuildAsyncExportJobResponse(job))
}

func CreateQuotaCostSummaryExportJob(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionLedgerRead) {
		return
	}

	var req dto.AdminQuotaCostSummaryExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	job, err := service.CreateQuotaCostSummaryExportJob(c.GetInt("id"), c.GetInt("role"), req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, service.BuildAsyncExportJobResponse(job))
}

func AdjustUserQuotaBatch(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionAdjustBatch) {
		return
	}
	var req adjustUserQuotaBatchRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	result, err := service.AdjustUserQuotaBatch(service.AdjustUserQuotaBatchRequest{
		OperatorUserId:   c.GetInt("id"),
		OperatorUserType: service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")),
		OperatorRole:     c.GetInt("role"),
		TargetUserIds:    req.TargetUserIds,
		Delta:            req.Delta,
		Reason:           req.Reason,
		Remark:           req.Remark,
		IP:               c.ClientIP(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func buildQuotaLedgerExportRows(items []service.QuotaLedgerListItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			strconv.Itoa(item.Id),
			formatQuotaLedgerAccount(item),
			formatQuotaLedgerOperator(item),
			service.GetQuotaEntryTypeLabel(item.EntryType),
			service.GetQuotaDirectionLabel(item.Direction),
			service.FormatQuotaUSD(item.Amount),
			service.FormatQuotaUSD(item.BalanceBefore),
			service.FormatQuotaUSD(item.BalanceAfter),
			item.ModelName,
			service.GetQuotaSourceTypeLabel(item.SourceType),
			service.GetQuotaReasonLabel(item.Reason),
			item.Remark,
			formatExportTimestamp(item.CreatedAtTs),
		})
	}
	return rows
}

func formatQuotaLedgerAccount(item service.QuotaLedgerListItem) string {
	if item.AccountUsername == "" {
		return strconv.Itoa(item.AccountId)
	}
	return item.AccountUsername
}

func formatQuotaLedgerOperator(item service.QuotaLedgerListItem) string {
	if item.OperatorUsername == "" {
		if item.OperatorUserId > 0 {
			return strconv.Itoa(item.OperatorUserId)
		}
		return item.OperatorUserType
	}
	if item.OperatorUserId > 0 {
		return item.OperatorUsername + " [ID:" + strconv.Itoa(item.OperatorUserId) + "]"
	}
	return item.OperatorUsername
}
