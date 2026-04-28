package service

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

func GetScopedQuotaData(requesterUserId int, requesterRole int, startTime int64, endTime int64, username string) ([]*model.QuotaData, error) {
	operator, err := ResolveOperatorUser(requesterUserId, requesterRole)
	if err != nil {
		return nil, err
	}

	if operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot || operator.GetUserType() == model.UserTypeAdmin {
		return model.GetAllQuotaDates(startTime, endTime, username)
	}
	if operator.GetUserType() != model.UserTypeAgent {
		return nil, errors.New("permission denied")
	}

	query := model.DB.Table("quota_data").
		Where("created_at >= ? and created_at <= ?", startTime, endTime)
	query = applyQuotaDataAgentScope(query, operator)
	if username != "" {
		var quotaDatas []*model.QuotaData
		err := query.Where("username = ?", username).Find(&quotaDatas).Error
		return quotaDatas, err
	}

	var quotaDatas []*model.QuotaData
	err = query.
		Select("model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at").
		Group("model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func applyQuotaDataAgentScope(query *gorm.DB, operator *model.User) *gorm.DB {
	managedUsers := ApplyManagedEndUserScope(
		model.DB.Model(&model.User{}).Select("users.id"),
		operator,
		ResourceQuotaManagement,
	)
	return query.Where("(user_id = ? OR user_id IN (?))", operator.Id, managedUsers)
}
