package service

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

func ApplyOperationsAnalyticsScope(query *gorm.DB, operator *model.User) (*gorm.DB, error) {
	if operator == nil || operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot || operator.GetUserType() == model.UserTypeAdmin {
		return query, nil
	}

	scopeType, scopeUserIDs, err := resolveDataScopeForResource(operator, ResourceAnalyticsManagement)
	if err != nil {
		return nil, err
	}

	switch scopeType {
	case model.ScopeTypeAll:
		return query, nil
	case model.ScopeTypeSelf:
		return query.Where("logs.user_id = ?", operator.Id), nil
	case model.ScopeTypeAssigned:
		if len(scopeUserIDs) == 0 {
			return query.Where("1 = 0"), nil
		}
		return query.Where("logs.user_id IN ?", scopeUserIDs), nil
	default:
		if operator.GetUserType() != model.UserTypeAgent {
			return query.Where("1 = 0"), nil
		}
		managedUsers := ApplyManagedEndUserScope(
			model.DB.Model(&model.User{}).Select("users.id"),
			operator,
			ResourceAnalyticsManagement,
		)
		ownedTokens := model.DB.Model(&model.Token{}).
			Select("tokens.id").
			Where("tokens.user_id = ? OR tokens.user_id IN (?)", operator.Id, managedUsers)
		return query.Where(
			"(logs.user_id = ? OR logs.user_id IN (?) OR (logs.token_id > 0 AND logs.token_id IN (?)))",
			operator.Id,
			managedUsers,
			ownedTokens,
		), nil
	}
}

func BuildOperationsAnalyticsNaturalWeekRanges(endTimestamp int64) (int64, int64, int64, int64) {
	endTime := time.Unix(endTimestamp, 0).In(dto.AdminAnalyticsLocation())
	weekday := int(endTime.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	currentWeekStartTime := time.Date(
		endTime.Year(),
		endTime.Month(),
		endTime.Day(),
		0,
		0,
		0,
		0,
		endTime.Location(),
	).AddDate(0, 0, -(weekday - 1))
	currentWeekStart := currentWeekStartTime.Unix()
	elapsedSeconds := endTimestamp - currentWeekStart
	previousWeekStart := currentWeekStartTime.AddDate(0, 0, -7).Unix()
	previousWeekEnd := previousWeekStart + elapsedSeconds
	return currentWeekStart, endTimestamp, previousWeekStart, previousWeekEnd
}
