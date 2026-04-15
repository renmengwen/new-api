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

func ExportQuotaLedger(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceQuotaManagement, service.ActionLedgerRead) {
		return
	}

	var req dto.AdminQuotaLedgerExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	items, _, err := service.ListQuotaLedgerForExport(
		c.GetInt("id"),
		c.GetInt("role"),
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
		Headers:        []string{"ID", "账户", "操作人", "类型", "方向", "额度", "变更前", "变更后", "来源", "原因", "备注", "时间"},
		Rows:           buildQuotaLedgerExportRows(items),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	streamExcelFile(c, fileName, content)
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
			item.EntryType,
			item.Direction,
			strconv.Itoa(item.Amount),
			strconv.Itoa(item.BalanceBefore),
			strconv.Itoa(item.BalanceAfter),
			item.SourceType,
			item.Reason,
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
