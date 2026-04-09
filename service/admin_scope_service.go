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

func ApplyManagedEndUserScope(query *gorm.DB, operator *model.User, resourceKeys ...string) *gorm.DB {
	resourceKey := ResourceUserManagement
	if len(resourceKeys) > 0 && resourceKeys[0] != "" {
		resourceKey = resourceKeys[0]
	}

	query = query.Where(
		"(users.user_type = ? OR (COALESCE(users.user_type, '') = '' AND users.role < ?))",
		model.UserTypeEndUser,
		common.RoleAdminUser,
	)
	scopeType, scopeUserIDs, err := resolveDataScopeForResource(operator, resourceKey)
	if err != nil {
		return query.Where("1 = 0")
	}

	switch scopeType {
	case model.ScopeTypeAll:
		return query
	case model.ScopeTypeSelf:
		return query.Where("users.id = ?", operator.Id)
	case model.ScopeTypeAssigned:
		if len(scopeUserIDs) == 0 {
			return query.Where("1 = 0")
		}
		return query.Where("users.id IN ?", scopeUserIDs)
	default:
		if operator.GetUserType() != model.UserTypeAgent {
			return query.Where("1 = 0")
		}
	}

	ownedSubQuery := model.DB.Model(&model.AgentUserRelation{}).
		Select("end_user_id").
		Where("agent_user_id = ? AND status = ?", operator.Id, model.CommonStatusEnabled)

	return query.Where("(users.parent_agent_id = ? OR users.id IN (?))", operator.Id, ownedSubQuery)
}

func GetManagedEndUser(targetUserId int, operatorUserId int, operatorRole int) (*model.User, error) {
	return GetManagedEndUserForResource(targetUserId, operatorUserId, operatorRole, ResourceUserManagement)
}

func GetManagedEndUserForResource(targetUserId int, operatorUserId int, operatorRole int, resourceKey string) (*model.User, error) {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, err
	}

	var user model.User
	query := model.DB.Model(&model.User{}).Where("users.id = ?", targetUserId)
	query = ApplyManagedEndUserScope(query, operator, resourceKey)
	if err := query.First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func resolveDataScopeForResource(operator *model.User, resourceKey string) (string, []int, error) {
	if operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot {
		return model.ScopeTypeAll, nil, nil
	}

	overrides, err := loadUserDataScopeOverrides(operator.Id)
	if err != nil {
		return "", nil, err
	}
	for _, item := range overrides {
		if item.ResourceKey != resourceKey {
			continue
		}
		return item.ScopeType, parseDataScopeValue(item.ScopeValueJSON), nil
	}

	return defaultDataScopeForUserType(operator.GetUserType()), nil, nil
}

func parseDataScopeValue(raw string) []int {
	if raw == "" {
		return nil
	}
	var values []int
	if err := common.UnmarshalJsonStr(raw, &values); err != nil {
		return nil
	}
	return values
}
