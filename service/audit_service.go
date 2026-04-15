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

func ListAdminAuditLogs(pageInfo *common.PageInfo, actionModule string, operatorUserId int) ([]model.AdminAuditLog, int64, error) {
	var logs []model.AdminAuditLog
	var total int64

	query := model.DB.Model(&model.AdminAuditLog{})
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
