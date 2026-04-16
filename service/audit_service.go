package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type AuditLogInput struct {
	OperatorUserId   int
	OperatorUserType string
	ActionModule     string
	ActionType       string
	ActionDesc       string
	TargetType       string
	TargetId         int
	BeforeJSON       string
	AfterJSON        string
	IP               string
}

type AdminAuditLogListItem struct {
	Id                  int    `json:"id" gorm:"column:id"`
	OperatorUserId      int    `json:"operator_user_id" gorm:"column:operator_user_id"`
	OperatorUserType    string `json:"operator_user_type" gorm:"column:operator_user_type"`
	ActionModule        string `json:"action_module" gorm:"column:action_module"`
	ActionType          string `json:"action_type" gorm:"column:action_type"`
	TargetType          string `json:"target_type" gorm:"column:target_type"`
	TargetId            int    `json:"target_id" gorm:"column:target_id"`
	IP                  string `json:"ip" gorm:"column:ip"`
	CreatedAt           int64  `json:"created_at" gorm:"column:created_at"`
	OperatorUsername    string `json:"operator_username" gorm:"column:operator_username"`
	OperatorDisplayName string `json:"operator_display_name" gorm:"column:operator_display_name"`
	TargetUsername      string `json:"target_username" gorm:"column:target_username"`
	TargetDisplayName   string `json:"target_display_name" gorm:"column:target_display_name"`
}

const exportQueryLimitMax = 2000

func CreateAdminAuditLog(input AuditLogInput) error {
	return createAdminAuditLogWithDB(model.DB, input)
}

func CreateAdminAuditLogTx(tx *gorm.DB, input AuditLogInput) error {
	return createAdminAuditLogWithDB(tx, input)
}

func createAdminAuditLogWithDB(db *gorm.DB, input AuditLogInput) error {
	log := &model.AdminAuditLog{
		OperatorUserId:   input.OperatorUserId,
		OperatorUserType: input.OperatorUserType,
		ActionModule:     input.ActionModule,
		ActionType:       input.ActionType,
		ActionDesc:       input.ActionDesc,
		TargetType:       input.TargetType,
		TargetId:         input.TargetId,
		BeforeJSON:       input.BeforeJSON,
		AfterJSON:        input.AfterJSON,
		Ip:               input.IP,
		CreatedAtTs:      common.GetTimestamp(),
	}
	return db.Create(log).Error
}

func ListAdminAuditLogs(pageInfo *common.PageInfo, requesterUserId int, requesterRole int, actionModule string, operatorUserId int) ([]AdminAuditLogListItem, int64, error) {
	var total int64

	query, err := buildAdminAuditLogsQuery(requesterUserId, requesterRole, actionModule, operatorUserId)
	if err != nil {
		return nil, 0, err
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	logs, err := fetchAdminAuditLogs(query, pageInfo.GetPageSize(), pageInfo.GetStartIdx())
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func ListAdminAuditLogsForExport(requesterUserId int, requesterRole int, actionModule string, operatorUserId int, limit int) ([]AdminAuditLogListItem, int64, error) {
	query, err := buildAdminAuditLogsQuery(requesterUserId, requesterRole, actionModule, operatorUserId)
	if err != nil {
		return nil, 0, err
	}
	logs, err := fetchAdminAuditLogs(query, clampExportQueryLimit(limit), 0)
	if err != nil {
		return nil, 0, err
	}
	return logs, 0, nil
}

func clampExportQueryLimit(limit int) int {
	if limit <= 0 || limit > exportQueryLimitMax {
		return exportQueryLimitMax
	}
	return limit
}

func buildAdminAuditLogsQuery(requesterUserId int, requesterRole int, actionModule string, operatorUserId int) (*gorm.DB, error) {
	query := model.DB.Model(&model.AdminAuditLog{}).
		Select("admin_audit_logs.id, admin_audit_logs.operator_user_id, admin_audit_logs.operator_user_type, admin_audit_logs.action_module, admin_audit_logs.action_type, admin_audit_logs.target_type, admin_audit_logs.target_id, admin_audit_logs.ip, admin_audit_logs.created_at, operator_users.username AS operator_username, operator_users.display_name AS operator_display_name, target_users.username AS target_username, target_users.display_name AS target_display_name").
		Joins("LEFT JOIN users AS operator_users ON operator_users.id = admin_audit_logs.operator_user_id").
		Joins("LEFT JOIN users AS target_users ON admin_audit_logs.target_type = ? AND target_users.id = admin_audit_logs.target_id", "user")
	operator, err := ResolveOperatorUser(requesterUserId, requesterRole)
	if err != nil {
		return nil, err
	}
	query, err = applyAdminAuditScope(query, operator)
	if err != nil {
		return nil, err
	}
	if actionModule != "" {
		query = query.Where("action_module = ?", actionModule)
	}
	if operatorUserId > 0 {
		query = query.Where("admin_audit_logs.operator_user_id = ?", operatorUserId)
	}
	return query, nil
}

func applyAdminAuditScope(query *gorm.DB, operator *model.User) (*gorm.DB, error) {
	if operator == nil || operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot || operator.GetUserType() == model.UserTypeAdmin {
		return query, nil
	}

	scopeType, scopeUserIDs, err := resolveDataScopeForResource(operator, ResourceAuditManagement)
	if err != nil {
		return nil, err
	}

	switch scopeType {
	case model.ScopeTypeAll:
		return query, nil
	case model.ScopeTypeSelf:
		return query.Where("admin_audit_logs.operator_user_id = ?", operator.Id), nil
	case model.ScopeTypeAssigned:
		if len(scopeUserIDs) == 0 {
			return query.Where("1 = 0"), nil
		}
		return query.Where("admin_audit_logs.operator_user_id IN ?", scopeUserIDs), nil
	default:
		if operator.GetUserType() != model.UserTypeAgent {
			return query.Where("1 = 0"), nil
		}
		managedUserSubQuery := ApplyManagedEndUserScope(
			model.DB.Model(&model.User{}).Select("users.id"),
			operator,
			ResourceAuditManagement,
		)
		return query.Where(
			"(admin_audit_logs.operator_user_id = ? OR admin_audit_logs.operator_user_id IN (?))",
			operator.Id,
			managedUserSubQuery,
		), nil
	}
}

func fetchAdminAuditLogs(query *gorm.DB, limit int, offset int) ([]AdminAuditLogListItem, error) {
	var logs []AdminAuditLogListItem
	err := query.Order("admin_audit_logs.id desc").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, err
}

func UserTypeFromRole(role int) string {
	switch role {
	case common.RoleRootUser:
		return model.UserTypeRoot
	case common.RoleAdminUser:
		return model.UserTypeAdmin
	default:
		return model.UserTypeEndUser
	}
}
