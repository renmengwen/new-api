package service

import (
	"strconv"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

const (
	asyncQuotaLedgerFilePrefix = "额度流水"
	asyncQuotaLedgerSheetName  = "额度流水"
)

type asyncQuotaLedgerExportPayload struct {
	Request dto.AdminQuotaLedgerExportRequest `json:"request"`
	Limit   int                               `json:"limit"`
}

var asyncQuotaLedgerExportHeaders = []string{"ID", "账户", "操作人", "类型", "方向", "额度", "变更前", "变更后", "模型名称", "来源", "原因", "备注", "时间"}

func init() {
	RegisterAsyncExportExecutor(SmartExportJobTypeQuotaLedger, executeQuotaLedgerExportJob)
}

func CreateQuotaLedgerExportJob(requesterUserID int, requesterRole int, req dto.AdminQuotaLedgerExportRequest) (*model.AsyncExportJob, error) {
	if _, err := buildQuotaLedgerListQuery(requesterUserID, requesterRole, req.UserID, req.OperatorUserID, req.EntryType); err != nil {
		return nil, err
	}

	payload := asyncQuotaLedgerExportPayload{
		Request: req,
		Limit:   normalizeAsyncExportLimit(req.Limit),
	}
	return CreateAsyncExportJob(SmartExportJobTypeQuotaLedger, requesterUserID, requesterRole, payload)
}

func executeQuotaLedgerExportJob(job *model.AsyncExportJob) error {
	var payload asyncQuotaLedgerExportPayload
	if err := DecodeAsyncExportPayload(job, &payload); err != nil {
		return err
	}
	if payload.Limit <= 0 {
		payload.Limit = normalizeAsyncExportLimit(payload.Request.Limit)
	}

	return writeAsyncExportJobFile(job, asyncQuotaLedgerFilePrefix, asyncQuotaLedgerSheetName, asyncQuotaLedgerExportHeaders, func(page int, pageSize int) (AsyncExportPage, error) {
		offset := (page - 1) * pageSize
		if asyncExportLimitReached(offset, payload.Limit) {
			return AsyncExportPage{Done: true}, nil
		}

		query, err := buildQuotaLedgerListQuery(job.RequesterUserId, job.RequesterRole, payload.Request.UserID, payload.Request.OperatorUserID, payload.Request.EntryType)
		if err != nil {
			return AsyncExportPage{}, err
		}
		items, err := fetchQuotaLedgerItems(query, pageSize, offset)
		if err != nil {
			return AsyncExportPage{}, err
		}

		items = trimAsyncExportItemsToLimit(items, offset, payload.Limit)
		if len(items) == 0 {
			return AsyncExportPage{Done: true}, nil
		}

		return AsyncExportPage{
			Rows: buildAsyncQuotaLedgerExportRows(items),
			Done: isAsyncExportPageDone(len(items), pageSize, offset, payload.Limit, 0),
		}, nil
	})
}

func buildAsyncQuotaLedgerExportRows(items []QuotaLedgerListItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			strconv.Itoa(item.Id),
			formatAsyncQuotaLedgerAccount(item),
			formatAsyncQuotaLedgerOperator(item),
			GetQuotaEntryTypeLabel(item.EntryType),
			GetQuotaDirectionLabel(item.Direction),
			FormatQuotaUSD(item.Amount),
			FormatQuotaUSD(item.BalanceBefore),
			FormatQuotaUSD(item.BalanceAfter),
			item.ModelName,
			GetQuotaSourceTypeLabel(item.SourceType),
			GetQuotaReasonLabel(item.Reason),
			item.Remark,
			formatAsyncExportTimestamp(item.CreatedAtTs),
		})
	}
	return rows
}

func formatAsyncQuotaLedgerAccount(item QuotaLedgerListItem) string {
	if item.AccountUsername == "" {
		return strconv.Itoa(item.AccountId)
	}
	return item.AccountUsername
}

func formatAsyncQuotaLedgerOperator(item QuotaLedgerListItem) string {
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
