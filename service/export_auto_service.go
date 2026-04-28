package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	SmartExportModeSync  = "sync"
	SmartExportModeAsync = "async"

	SmartExportReasonAlwaysSync       = "always_sync"
	SmartExportReasonWithinThreshold  = "within_threshold"
	SmartExportReasonExceedsThreshold = "exceeds_threshold"
)

const (
	SmartExportJobTypeUsageLogs            = "usage_logs"
	SmartExportJobTypeAdminAuditLogs       = "admin_audit_logs"
	SmartExportJobTypeQuotaLedger          = "quota_ledger"
	SmartExportJobTypeQuotaCostSummary     = "quota_cost_summary"
	SmartExportJobTypeAdminAnalyticsModels = "operations_analytics_models"
	SmartExportJobTypeAdminAnalyticsUsers  = "operations_analytics_users"
	SmartExportJobTypeAdminAnalyticsDaily  = "operations_analytics_daily"
)

const (
	SmartExportUsageLogsThreshold         = 2000
	SmartExportUsageLogsLongTextThreshold = 800
	SmartExportAdminAuditThreshold        = 3000
	SmartExportQuotaLedgerThreshold       = 3000
	SmartExportQuotaCostSummaryThreshold  = 5000
	SmartExportAdminAnalyticsThreshold    = 5000
	smartExportSyncLimit                  = 2000
)

type SmartExportDecision struct {
	JobType    string `json:"job_type"`
	Mode       string `json:"mode"`
	Reason     string `json:"reason"`
	Threshold  int    `json:"threshold"`
	ProbedRows int    `json:"probed_rows"`
}

type smartExportProbeFunc func(limit int) (int, error)

func DecideUsageLogSmartExport(requesterUserID int, requesterRole int, req dto.UsageLogExportRequest, selectedColumnKeys []string, isAdmin bool) (SmartExportDecision, error) {
	columnKeys := resolveUsageLogSmartExportColumnKeys(req.ColumnKeys, selectedColumnKeys, isAdmin)

	threshold := SmartExportUsageLogsThreshold
	if hasUsageLogLongTextColumns(columnKeys) {
		threshold = SmartExportUsageLogsLongTextThreshold
	}

	channel, _ := strconv.Atoi(strings.TrimSpace(req.Channel))
	return decideSmartExportWithProbe(SmartExportJobTypeUsageLogs, threshold, req.Limit, func(limit int) (int, error) {
		if !isAdmin {
			items, err := model.GetUserLogsForExport(
				requesterUserID,
				req.Type,
				req.StartTimestamp,
				req.EndTimestamp,
				req.ModelName,
				req.TokenName,
				limit,
				req.Group,
				req.RequestID,
			)
			if err != nil {
				return 0, err
			}
			return len(items), nil
		}

		items, err := ListScopedUsageLogsForExport(
			requesterUserID,
			requesterRole,
			req.Type,
			req.StartTimestamp,
			req.EndTimestamp,
			req.ModelName,
			req.Username,
			req.TokenName,
			limit,
			channel,
			req.Group,
			req.RequestID,
		)
		if err != nil {
			return 0, err
		}
		return len(items), nil
	})
}

func DecideAdminAuditLogSmartExport(requesterUserID int, requesterRole int, req dto.AdminAuditExportRequest) (SmartExportDecision, error) {
	return decideSmartExportWithProbe(SmartExportJobTypeAdminAuditLogs, SmartExportAdminAuditThreshold, req.Limit, func(limit int) (int, error) {
		query, err := buildAdminAuditLogsQuery(requesterUserID, requesterRole, req.ActionModule, req.OperatorUserID)
		if err != nil {
			return 0, err
		}
		items, err := fetchAdminAuditLogs(query, limit, 0)
		if err != nil {
			return 0, err
		}
		return len(items), nil
	})
}

func DecideQuotaLedgerSmartExport(requesterUserID int, requesterRole int, req dto.AdminQuotaLedgerExportRequest) (SmartExportDecision, error) {
	return decideSmartExportWithProbe(SmartExportJobTypeQuotaLedger, SmartExportQuotaLedgerThreshold, req.Limit, func(limit int) (int, error) {
		query, err := buildQuotaLedgerListQuery(requesterUserID, requesterRole, req.UserID, req.OperatorUserID, req.EntryType)
		if err != nil {
			return 0, err
		}
		return probeQuotaLedgerRows(query, limit)
	})
}

func DecideQuotaCostSummarySmartExport(requesterUserID int, requesterRole int, req dto.AdminQuotaCostSummaryExportRequest) (SmartExportDecision, error) {
	query, err := dto.NormalizeAdminQuotaCostSummaryQuery(req.AdminQuotaCostSummaryQuery, common.GetTimestamp())
	if err != nil {
		return SmartExportDecision{}, err
	}

	return decideSmartExportWithProbe(SmartExportJobTypeQuotaCostSummary, SmartExportQuotaCostSummaryThreshold, req.Limit, func(limit int) (int, error) {
		items, err := ListQuotaCostSummaryForExport(query, requesterUserID, requesterRole, limit)
		if err != nil {
			return 0, err
		}
		return len(items), nil
	})
}

func DecideAdminAnalyticsSmartExport(requesterUserID int, requesterRole int, req dto.AdminAnalyticsExportRequest) (SmartExportDecision, error) {
	query, err := dto.NormalizeAdminAnalyticsQuery(req.ToQuery(), common.GetTimestamp())
	if err != nil {
		return SmartExportDecision{}, err
	}

	switch strings.ToLower(strings.TrimSpace(req.View)) {
	case dto.AdminAnalyticsViewModels:
		return decideSmartExportWithProbe(SmartExportJobTypeAdminAnalyticsModels, SmartExportAdminAnalyticsThreshold, req.Limit, func(limit int) (int, error) {
			return probeOperationsAnalyticsModelRows(query, requesterUserID, requesterRole, limit)
		})
	case dto.AdminAnalyticsViewUsers:
		return decideSmartExportWithProbe(SmartExportJobTypeAdminAnalyticsUsers, SmartExportAdminAnalyticsThreshold, req.Limit, func(limit int) (int, error) {
			return probeOperationsAnalyticsUserRows(query, requesterUserID, requesterRole, limit)
		})
	case dto.AdminAnalyticsViewDaily:
		return SmartExportDecision{
			JobType:    SmartExportJobTypeAdminAnalyticsDaily,
			Mode:       SmartExportModeSync,
			Reason:     SmartExportReasonAlwaysSync,
			Threshold:  0,
			ProbedRows: 0,
		}, nil
	default:
		return SmartExportDecision{}, errors.New("invalid analytics view")
	}
}

func decideSmartExportWithProbe(jobType string, threshold int, requestedLimit int, probe smartExportProbeFunc) (SmartExportDecision, error) {
	syncThreshold := threshold
	if syncThreshold <= 0 || syncThreshold > smartExportSyncLimit {
		syncThreshold = smartExportSyncLimit
	}

	probeLimit := syncThreshold + 1
	if requestedLimit > 0 {
		if requestedLimit > syncThreshold {
			return SmartExportDecision{
				JobType:    jobType,
				Mode:       SmartExportModeAsync,
				Reason:     SmartExportReasonExceedsThreshold,
				Threshold:  syncThreshold,
				ProbedRows: requestedLimit,
			}, nil
		}
		probeLimit = requestedLimit
	}

	probedRows, err := probe(probeLimit)
	if err != nil {
		return SmartExportDecision{}, err
	}

	decision := SmartExportDecision{
		JobType:    jobType,
		Mode:       SmartExportModeSync,
		Reason:     SmartExportReasonWithinThreshold,
		Threshold:  syncThreshold,
		ProbedRows: probedRows,
	}
	if probedRows > syncThreshold {
		decision.Mode = SmartExportModeAsync
		decision.Reason = SmartExportReasonExceedsThreshold
	}
	return decision, nil
}

func resolveUsageLogSmartExportColumnKeys(requestColumnKeys []string, selectedColumnKeys []string, isAdmin bool) []string {
	if selectedColumnKeys != nil {
		return selectedColumnKeys
	}
	if len(requestColumnKeys) > 0 {
		return requestColumnKeys
	}
	return resolveAsyncUsageLogExportColumnKeys(nil, isAdmin)
}

func hasUsageLogLongTextColumns(columnKeys []string) bool {
	for _, rawKey := range columnKeys {
		switch strings.ToLower(strings.TrimSpace(rawKey)) {
		case "details", "content", "other":
			return true
		}
	}
	return false
}

func probeQuotaLedgerRows(query *gorm.DB, limit int) (int, error) {
	var ids []int
	err := query.
		Select("quota_ledgers.id").
		Order("quota_ledgers.id desc").
		Limit(limit).
		Pluck("quota_ledgers.id", &ids).Error
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

func probeOperationsAnalyticsModelRows(query dto.AdminAnalyticsQuery, requesterUserID int, requesterRole int, limit int) (int, error) {
	baseQuery, err := buildOperationsAnalyticsBaseQuery(query, requesterUserID, requesterRole)
	if err != nil {
		return 0, err
	}

	modelBucketExpr := operationsAnalyticsModelBucketExpr()
	itemsQuery := baseQuery.Select(
		modelBucketExpr + " AS model_name, " +
			"COUNT(*) AS call_count, " +
			"COALESCE(SUM(logs.prompt_tokens), 0) AS prompt_tokens, " +
			"COALESCE(SUM(logs.completion_tokens), 0) AS completion_tokens, " +
			"COALESCE(SUM(logs.quota), 0) AS total_cost, " +
			"COALESCE(AVG(logs.use_time), 0) AS avg_use_time, " +
			"COALESCE((SUM(CASE WHEN logs.type = 2 THEN 1 ELSE 0 END) * 1.0) / NULLIF(COUNT(*), 0), 0) AS success_rate",
	).Group(modelBucketExpr)
	itemsQuery = applyOperationsAnalyticsModelSort(itemsQuery, query.SortBy, query.SortOrder)

	var items []dto.AdminAnalyticsModelItem
	if err := itemsQuery.Limit(limit).Scan(&items).Error; err != nil {
		return 0, err
	}
	return len(items), nil
}

func probeOperationsAnalyticsUserRows(query dto.AdminAnalyticsQuery, requesterUserID int, requesterRole int, limit int) (int, error) {
	baseQuery, err := buildOperationsAnalyticsBaseQuery(query, requesterUserID, requesterRole)
	if err != nil {
		return 0, err
	}

	itemsQuery := baseQuery.Select(
		"logs.user_id AS user_id, " +
			"COALESCE(NULLIF(MAX(analytics_users.username), ''), COALESCE(MAX(logs.username), '')) AS username, " +
			"COUNT(*) AS call_count, " +
			"COUNT(DISTINCT NULLIF(logs.model_name, '')) AS model_count, " +
			operationsAnalyticsTotalTokensExpr() + " AS total_tokens, " +
			"COALESCE(SUM(logs.quota), 0) AS total_cost, " +
			"COALESCE(MAX(logs.created_at), 0) AS last_called_at",
	).Group("logs.user_id")
	itemsQuery = applyOperationsAnalyticsUserSort(itemsQuery, query.SortBy, query.SortOrder)

	var items []dto.AdminAnalyticsUserItem
	if err := itemsQuery.Limit(limit).Scan(&items).Error; err != nil {
		return 0, err
	}
	return len(items), nil
}
