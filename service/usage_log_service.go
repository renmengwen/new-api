package service

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

func ListScopedUsageLogs(
	pageInfo *common.PageInfo,
	requesterUserId int,
	requesterRole int,
	logType int,
	startTimestamp int64,
	endTimestamp int64,
	modelName string,
	username string,
	tokenName string,
	channel int,
	group string,
	requestId string,
) ([]*model.Log, int64, error) {
	query, err := buildScopedUsageLogQuery(
		requesterUserId,
		requesterRole,
		logType,
		startTimestamp,
		endTimestamp,
		modelName,
		username,
		tokenName,
		channel,
		group,
		requestId,
	)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if err := query.Model(&model.Log{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	items, err := model.FindAllLogsByQuery(query, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func ListScopedUsageLogsForExport(
	requesterUserId int,
	requesterRole int,
	logType int,
	startTimestamp int64,
	endTimestamp int64,
	modelName string,
	username string,
	tokenName string,
	limit int,
	channel int,
	group string,
	requestId string,
) ([]*model.Log, error) {
	query, err := buildScopedUsageLogQuery(
		requesterUserId,
		requesterRole,
		logType,
		startTimestamp,
		endTimestamp,
		modelName,
		username,
		tokenName,
		channel,
		group,
		requestId,
	)
	if err != nil {
		return nil, err
	}

	return model.FindAllLogsByQuery(query, 0, limit)
}

func GetScopedUsageLogStat(
	requesterUserId int,
	requesterRole int,
	startTimestamp int64,
	endTimestamp int64,
	modelName string,
	username string,
	tokenName string,
	channel int,
	group string,
) (model.Stat, error) {
	quotaQuery, err := buildScopedUsageLogQuery(
		requesterUserId,
		requesterRole,
		model.LogTypeUnknown,
		startTimestamp,
		endTimestamp,
		modelName,
		username,
		tokenName,
		channel,
		group,
		"",
	)
	if err != nil {
		return model.Stat{}, err
	}
	rpmTpmQuery, err := buildScopedUsageLogQuery(
		requesterUserId,
		requesterRole,
		model.LogTypeUnknown,
		startTimestamp,
		endTimestamp,
		modelName,
		username,
		tokenName,
		channel,
		group,
		"",
	)
	if err != nil {
		return model.Stat{}, err
	}

	var stat model.Stat
	if err := quotaQuery.
		Model(&model.Log{}).
		Select("sum(quota) quota").
		Where("logs.type = ?", model.LogTypeConsume).
		Scan(&stat).Error; err != nil {
		return model.Stat{}, err
	}
	if err := rpmTpmQuery.
		Model(&model.Log{}).
		Select("count(*) rpm, sum(prompt_tokens) + sum(completion_tokens) tpm").
		Where("logs.type = ?", model.LogTypeConsume).
		Where("logs.created_at >= ?", time.Now().Add(-60*time.Second).Unix()).
		Scan(&stat).Error; err != nil {
		return model.Stat{}, err
	}
	return stat, nil
}

func buildScopedUsageLogQuery(
	requesterUserId int,
	requesterRole int,
	logType int,
	startTimestamp int64,
	endTimestamp int64,
	modelName string,
	username string,
	tokenName string,
	channel int,
	group string,
	requestId string,
) (*gorm.DB, error) {
	query := model.BuildAllLogsQuery(
		logType,
		startTimestamp,
		endTimestamp,
		modelName,
		username,
		tokenName,
		channel,
		group,
		requestId,
	)
	return applyUsageLogScope(query, requesterUserId, requesterRole)
}

func applyUsageLogScope(query *gorm.DB, requesterUserId int, requesterRole int) (*gorm.DB, error) {
	operator, err := ResolveOperatorUser(requesterUserId, requesterRole)
	if err != nil {
		return nil, err
	}
	if operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot || operator.GetUserType() == model.UserTypeAdmin {
		return query, nil
	}
	if operator.GetUserType() != model.UserTypeAgent {
		return query.Where("1 = 0"), nil
	}

	managedUsers := ApplyManagedEndUserScope(
		model.DB.Model(&model.User{}).Select("users.id"),
		operator,
		ResourceQuotaManagement,
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
