package service

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

func ResolveOperatorUser(operatorUserId int, operatorRole int) (*model.User, error) {
	if operatorUserId > 0 {
		user, err := model.GetUserById(operatorUserId, false)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	return &model.User{
		Id:       operatorUserId,
		Role:     operatorRole,
		UserType: UserTypeFromRole(operatorRole),
		Status:   common.UserStatusEnabled,
	}, nil
}

func ResolveOperatorUserType(operatorUserId int, operatorRole int) string {
	user, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return UserTypeFromRole(operatorRole)
	}
	return user.GetUserType()
}

func ApplyManagedEndUserScope(query *gorm.DB, operator *model.User) *gorm.DB {
	query = query.Where(
		"(users.user_type = ? OR (COALESCE(users.user_type, '') = '' AND users.role < ?))",
		model.UserTypeEndUser,
		common.RoleAdminUser,
	)
	if operator.GetUserType() != model.UserTypeAgent {
		return query
	}

	ownedSubQuery := model.DB.Model(&model.AgentUserRelation{}).
		Select("end_user_id").
		Where("agent_user_id = ? AND status = ?", operator.Id, model.CommonStatusEnabled)

	return query.Where("(users.parent_agent_id = ? OR users.id IN (?))", operator.Id, ownedSubQuery)
}

func GetManagedEndUser(targetUserId int, operatorUserId int, operatorRole int) (*model.User, error) {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, err
	}

	var user model.User
	query := model.DB.Model(&model.User{}).Where("users.id = ?", targetUserId)
	query = ApplyManagedEndUserScope(query, operator)
	if err := query.First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
