package service

import (
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

const (
	asyncAdminAuditFilePrefix = "审计日志"
	asyncAdminAuditSheetName  = "审计日志"
)

type asyncAdminAuditExportPayload struct {
	Request dto.AdminAuditExportRequest `json:"request"`
	Limit   int                         `json:"limit"`
}

var asyncAdminAuditExportHeaders = []string{"ID", "操作人", "动作模块", "动作类型", "目标", "IP", "时间"}

func init() {
	RegisterAsyncExportExecutor(SmartExportJobTypeAdminAuditLogs, executeAdminAuditExportJob)
}

func CreateAdminAuditExportJob(requesterUserID int, requesterRole int, req dto.AdminAuditExportRequest) (*model.AsyncExportJob, error) {
	if _, err := buildAdminAuditLogsQuery(requesterUserID, requesterRole, req.ActionModule, req.OperatorUserID); err != nil {
		return nil, err
	}

	payload := asyncAdminAuditExportPayload{
		Request: req,
		Limit:   normalizeAsyncExportLimit(req.Limit),
	}
	return CreateAsyncExportJob(SmartExportJobTypeAdminAuditLogs, requesterUserID, requesterRole, payload)
}

func executeAdminAuditExportJob(job *model.AsyncExportJob) error {
	var payload asyncAdminAuditExportPayload
	if err := DecodeAsyncExportPayload(job, &payload); err != nil {
		return err
	}
	if payload.Limit <= 0 {
		payload.Limit = normalizeAsyncExportLimit(payload.Request.Limit)
	}

	return writeAsyncExportJobFile(job, asyncAdminAuditFilePrefix, asyncAdminAuditSheetName, asyncAdminAuditExportHeaders, func(page int, pageSize int) (AsyncExportPage, error) {
		offset := (page - 1) * pageSize
		if asyncExportLimitReached(offset, payload.Limit) {
			return AsyncExportPage{Done: true}, nil
		}

		query, err := buildAdminAuditLogsQuery(job.RequesterUserId, job.RequesterRole, payload.Request.ActionModule, payload.Request.OperatorUserID)
		if err != nil {
			return AsyncExportPage{}, err
		}
		items, err := fetchAdminAuditLogs(query, pageSize, offset)
		if err != nil {
			return AsyncExportPage{}, err
		}

		items = trimAsyncExportItemsToLimit(items, offset, payload.Limit)
		if len(items) == 0 {
			return AsyncExportPage{Done: true}, nil
		}

		return AsyncExportPage{
			Rows: buildAsyncAdminAuditExportRows(items),
			Done: isAsyncExportPageDone(len(items), pageSize, offset, payload.Limit, 0),
		}, nil
	})
}

func buildAsyncAdminAuditExportRows(items []AdminAuditLogListItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			strconv.Itoa(item.Id),
			formatAsyncAuditOperator(item),
			GetAuditLogModuleLabel(item.ActionModule),
			GetAuditLogActionLabel(item.ActionType),
			formatAsyncAuditTarget(item),
			item.IP,
			formatAsyncExportTimestamp(item.CreatedAt),
		})
	}
	return rows
}

func formatAsyncAuditOperator(item AdminAuditLogListItem) string {
	name := item.OperatorUsername
	if item.OperatorDisplayName != "" && item.OperatorDisplayName != item.OperatorUsername {
		if name != "" {
			name = fmt.Sprintf("%s (%s)", item.OperatorUsername, item.OperatorDisplayName)
		} else {
			name = item.OperatorDisplayName
		}
	}
	if name == "" {
		name = item.OperatorUserType
	}
	if item.OperatorUserId > 0 {
		return fmt.Sprintf("%s [ID:%d]", name, item.OperatorUserId)
	}
	return name
}

func formatAsyncAuditTarget(item AdminAuditLogListItem) string {
	name := item.TargetUsername
	if item.TargetDisplayName != "" && item.TargetDisplayName != item.TargetUsername {
		if name != "" {
			name = fmt.Sprintf("%s (%s)", item.TargetUsername, item.TargetDisplayName)
		} else {
			name = item.TargetDisplayName
		}
	}
	if name == "" {
		name = GetAuditLogTargetTypeLabel(item.TargetType)
	}
	if item.TargetId > 0 {
		return fmt.Sprintf("%s [ID:%d]", name, item.TargetId)
	}
	return name
}
