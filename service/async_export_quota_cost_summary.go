package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

const (
	asyncQuotaCostSummaryFilePrefix = "额度成本汇总"
	asyncQuotaCostSummarySheetName  = "成本汇总"
)

type asyncQuotaCostSummaryExportPayload struct {
	Query dto.AdminQuotaCostSummaryQuery `json:"query"`
	Limit int                            `json:"limit"`
}

func init() {
	RegisterAsyncExportExecutor(SmartExportJobTypeQuotaCostSummary, executeQuotaCostSummaryExportJob)
}

func CreateQuotaCostSummaryExportJob(requesterUserID int, requesterRole int, req dto.AdminQuotaCostSummaryExportRequest) (*model.AsyncExportJob, error) {
	query, err := dto.NormalizeAdminQuotaCostSummaryQuery(req.AdminQuotaCostSummaryQuery, common.GetTimestamp())
	if err != nil {
		return nil, err
	}

	payload := asyncQuotaCostSummaryExportPayload{
		Query: query,
		Limit: normalizeAsyncExportLimit(req.Limit),
	}
	return CreateAsyncExportJob(SmartExportJobTypeQuotaCostSummary, requesterUserID, requesterRole, payload)
}

func executeQuotaCostSummaryExportJob(job *model.AsyncExportJob) error {
	var payload asyncQuotaCostSummaryExportPayload
	if err := DecodeAsyncExportPayload(job, &payload); err != nil {
		return err
	}
	payload.Limit = normalizeAsyncExportLimit(payload.Limit)

	items, err := ListQuotaCostSummaryForExport(payload.Query, job.RequesterUserId, job.RequesterRole, payload.Limit)
	if err != nil {
		return err
	}

	return writeAsyncExportJobFile(job, asyncQuotaCostSummaryFilePrefix, asyncQuotaCostSummarySheetName, QuotaCostSummaryExportHeaders(), func(page int, pageSize int) (AsyncExportPage, error) {
		offset := (page - 1) * pageSize
		if offset >= len(items) {
			return AsyncExportPage{Done: true}, nil
		}

		end := offset + pageSize
		if end > len(items) {
			end = len(items)
		}
		pageItems := items[offset:end]

		return AsyncExportPage{
			Rows: BuildQuotaCostSummaryExportRows(pageItems),
			Done: end >= len(items),
		}, nil
	})
}
