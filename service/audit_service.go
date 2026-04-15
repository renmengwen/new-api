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
	model.AdminAuditLog
	OperatorUsername    string `json:"operator_username" gorm:"column:operator_username"`
	OperatorDisplayName string `json:"operator_display_name" gorm:"column:operator_display_name"`
	TargetUsername      string `json:"target_username" gorm:"column:target_username"`
	TargetDisplayName   string `json:"target_display_name" gorm:"column:target_display_name"`
}

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

func ListAdminAuditLogs(pageInfo *common.PageInfo, actionModule string, operatorUserId int) ([]AdminAuditLogListItem, int64, error) {
	var logs []AdminAuditLogListItem
	var total int64

	query := model.DB.Model(&model.AdminAuditLog{}).
		Select("admin_audit_logs.*, operator_users.username AS operator_username, operator_users.display_name AS operator_display_name, target_users.username AS target_username, target_users.display_name AS target_display_name").
		Joins("LEFT JOIN users AS operator_users ON operator_users.id = admin_audit_logs.operator_user_id").
		Joins("LEFT JOIN users AS target_users ON admin_audit_logs.target_type = ? AND target_users.id = admin_audit_logs.target_id", "user")
	if actionModule != "" {
		query = query.Where("action_module = ?", actionModule)
	}
	if operatorUserId > 0 {
		query = query.Where("operator_user_id = ?", operatorUserId)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
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
