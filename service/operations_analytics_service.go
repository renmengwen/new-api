package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type operationsAnalyticsSummaryRow struct {
	TotalCalls   int64 `gorm:"column:total_calls"`
	TotalTokens  int64 `gorm:"column:total_tokens"`
	TotalCost    int64 `gorm:"column:total_cost"`
	ActiveUsers  int64 `gorm:"column:active_users"`
	ActiveModels int64 `gorm:"column:active_models"`
}

type operationsAnalyticsDailyAggregateRow struct {
	BucketDayID  int64 `gorm:"column:bucket_day_id"`
	CallCount    int64 `gorm:"column:call_count"`
	TotalCost    int64 `gorm:"column:total_cost"`
	ActiveUsers  int64 `gorm:"column:active_users"`
	ActiveModels int64 `gorm:"column:active_models"`
}

const operationsAnalyticsEmptyModelName = "(empty)"

func GetOperationsAnalyticsSummary(query dto.AdminAnalyticsQuery, requesterUserId int, requesterRole int) (dto.AdminAnalyticsSummaryResponse, error) {
	baseQuery, err := buildOperationsAnalyticsBaseQuery(query, requesterUserId, requesterRole)
	if err != nil {
		return dto.AdminAnalyticsSummaryResponse{}, err
	}

	current, err := scanOperationsAnalyticsSummary(baseQuery)
	if err != nil {
		return dto.AdminAnalyticsSummaryResponse{}, err
	}

	response := dto.AdminAnalyticsSummaryResponse{
		TotalCalls:   current.TotalCalls,
		TotalTokens:  current.TotalTokens,
		TotalCost:    current.TotalCost,
		ActiveUsers:  current.ActiveUsers,
		ActiveModels: current.ActiveModels,
	}

	if query.DatePreset == dto.AdminAnalyticsDatePresetLast7Days {
		wow, err := buildOperationsAnalyticsWow(requesterUserId, requesterRole, query.EndTimestamp)
		if err != nil {
			return dto.AdminAnalyticsSummaryResponse{}, err
		}
		response.Wow = wow
	}

	return response, nil
}

func GetOperationsAnalyticsModels(query dto.AdminAnalyticsQuery, pageInfo *common.PageInfo, requesterUserId int, requesterRole int) ([]dto.AdminAnalyticsModelItem, int64, error) {
	baseQuery, err := buildOperationsAnalyticsBaseQuery(query, requesterUserId, requesterRole)
	if err != nil {
		return nil, 0, err
	}

	pageInfo = normalizeOperationsAnalyticsPageInfo(pageInfo)
	modelBucketExpr := operationsAnalyticsModelBucketExpr()

	totalQuery := baseQuery.Session(&gorm.Session{}).
		Select(modelBucketExpr + " AS model_name").
		Group(modelBucketExpr)

	var total int64
	if err := model.LOG_DB.Table("(?) AS analytics_model_groups", totalQuery).Count(&total).Error; err != nil {
		return nil, 0, err
	}

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
	err = itemsQuery.Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error
	return items, total, err
}

func GetOperationsAnalyticsUsers(query dto.AdminAnalyticsQuery, pageInfo *common.PageInfo, requesterUserId int, requesterRole int) ([]dto.AdminAnalyticsUserItem, int64, error) {
	baseQuery, err := buildOperationsAnalyticsBaseQuery(query, requesterUserId, requesterRole)
	if err != nil {
		return nil, 0, err
	}

	pageInfo = normalizeOperationsAnalyticsPageInfo(pageInfo)

	var total int64
	if err := baseQuery.Session(&gorm.Session{}).Distinct("logs.user_id").Count(&total).Error; err != nil {
		return nil, 0, err
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
	err = itemsQuery.Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error
	return items, total, err
}

func GetOperationsAnalyticsDaily(query dto.AdminAnalyticsQuery, requesterUserId int, requesterRole int) ([]dto.AdminAnalyticsDailyItem, error) {
	baseQuery, err := buildOperationsAnalyticsBaseQuery(query, requesterUserId, requesterRole)
	if err != nil {
		return nil, err
	}

	localOffsetSeconds := operationsAnalyticsLocalOffsetSeconds(query.EndTimestamp)
	bucketIDExpr := OperationsAnalyticsDailyBucketIDExpr(localOffsetSeconds)

	var rows []operationsAnalyticsDailyAggregateRow
	if err := baseQuery.Select(
		bucketIDExpr + " AS bucket_day_id, " +
			"COUNT(*) AS call_count, " +
			"COALESCE(SUM(logs.quota), 0) AS total_cost, " +
			"COUNT(DISTINCT logs.user_id) AS active_users, " +
			"COUNT(DISTINCT NULLIF(logs.model_name, '')) AS active_models",
	).
		Group(bucketIDExpr).
		Order("bucket_day_id asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	return buildOperationsAnalyticsDailySeries(query.StartTimestamp, query.EndTimestamp, localOffsetSeconds, rows), nil
}

func buildOperationsAnalyticsBaseQuery(query dto.AdminAnalyticsQuery, requesterUserId int, requesterRole int) (*gorm.DB, error) {
	operator, err := ResolveOperatorUser(requesterUserId, requesterRole)
	if err != nil {
		return nil, err
	}

	baseQuery := model.LOG_DB.Model(&model.Log{}).
		Joins("LEFT JOIN users analytics_users ON analytics_users.id = logs.user_id").
		Where("logs.type IN ?", []int{model.LogTypeConsume, model.LogTypeError}).
		Where("logs.created_at >= ? AND logs.created_at <= ?", query.StartTimestamp, query.EndTimestamp)

	baseQuery, err = applyOperationsAnalyticsLogFilters(baseQuery, query)
	if err != nil {
		return nil, err
	}
	return ApplyOperationsAnalyticsScope(baseQuery, operator)
}

func applyOperationsAnalyticsLogFilters(query *gorm.DB, analyticsQuery dto.AdminAnalyticsQuery) (*gorm.DB, error) {
	modelKeyword := strings.TrimSpace(analyticsQuery.ModelKeyword)
	if modelKeyword != "" {
		query = query.Where("LOWER(logs.model_name) LIKE ? ESCAPE '!'", buildOperationsAnalyticsLikePattern(modelKeyword))
	}

	usernameKeyword := strings.TrimSpace(analyticsQuery.UsernameKeyword)
	if usernameKeyword != "" {
		userIDs, err := findOperationsAnalyticsUserIDsByUsernameKeyword(usernameKeyword)
		if err != nil {
			return nil, err
		}
		if len(userIDs) == 0 {
			return query.Where("1 = 0"), nil
		}
		query = query.Where("logs.user_id IN ?", userIDs)
	}

	return query, nil
}

func applyOperationsAnalyticsModelSort(query *gorm.DB, sortBy string, sortOrder string) *gorm.DB {
	switch normalizeOperationsAnalyticsSortBy(sortBy) {
	case "model_name":
		return query.Order("model_name " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("call_count DESC").
			Order("total_cost DESC")
	case "prompt_tokens":
		return query.Order("prompt_tokens " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("call_count DESC").
			Order("model_name ASC")
	case "completion_tokens":
		return query.Order("completion_tokens " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("call_count DESC").
			Order("model_name ASC")
	case "total_cost":
		return query.Order("total_cost " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("call_count DESC").
			Order("model_name ASC")
	case "avg_use_time", "avg_response_time":
		return query.Order("avg_use_time " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("call_count DESC").
			Order("model_name ASC")
	case "success_rate":
		return query.Order("success_rate " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("call_count DESC").
			Order("model_name ASC")
	case "call_count", "total_calls":
		return query.Order("call_count " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("total_cost DESC").
			Order("model_name ASC")
	default:
		return query.Order("call_count DESC").
			Order("total_cost DESC").
			Order("model_name ASC")
	}
}

func applyOperationsAnalyticsUserSort(query *gorm.DB, sortBy string, sortOrder string) *gorm.DB {
	switch normalizeOperationsAnalyticsSortBy(sortBy) {
	case "user_id":
		return query.Order("user_id " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("last_called_at DESC")
	case "username":
		return query.Order("username " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("user_id ASC")
	case "call_count", "total_calls":
		return query.Order("call_count " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("last_called_at DESC").
			Order("user_id ASC")
	case "model_count":
		return query.Order("model_count " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("last_called_at DESC").
			Order("user_id ASC")
	case "total_cost":
		return query.Order("total_cost " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("last_called_at DESC").
			Order("user_id ASC")
	case "last_called_at":
		return query.Order("last_called_at " + normalizeOperationsAnalyticsSortOrder(sortOrder)).
			Order("user_id ASC")
	default:
		return query.Order("last_called_at DESC").
			Order("user_id ASC")
	}
}

func normalizeOperationsAnalyticsSortBy(sortBy string) string {
	return strings.ToLower(strings.TrimSpace(sortBy))
}

func normalizeOperationsAnalyticsSortOrder(sortOrder string) string {
	if strings.EqualFold(strings.TrimSpace(sortOrder), "asc") {
		return "ASC"
	}
	return "DESC"
}

func buildOperationsAnalyticsDailySeries(startTimestamp int64, endTimestamp int64, localOffsetSeconds int, rows []operationsAnalyticsDailyAggregateRow) []dto.AdminAnalyticsDailyItem {
	seriesStartBucketID := operationsAnalyticsBucketID(startTimestamp, localOffsetSeconds)
	seriesEndBucketID := operationsAnalyticsBucketID(endTimestamp, localOffsetSeconds)
	dayCount := int(seriesEndBucketID-seriesStartBucketID) + 1
	if dayCount < 1 {
		return []dto.AdminAnalyticsDailyItem{}
	}

	items := make([]dto.AdminAnalyticsDailyItem, 0, dayCount)
	rowMap := make(map[int64]operationsAnalyticsDailyAggregateRow, len(rows))
	for _, row := range rows {
		rowMap[row.BucketDayID] = row
	}

	for index := 0; index < dayCount; index++ {
		currentBucketID := seriesStartBucketID + int64(index)
		bucketDay := operationsAnalyticsBucketDayLabel(currentBucketID, localOffsetSeconds)
		item := dto.AdminAnalyticsDailyItem{
			BucketDay: bucketDay,
		}
		if row, ok := rowMap[currentBucketID]; ok {
			item.CallCount = row.CallCount
			item.TotalCost = row.TotalCost
			item.ActiveUsers = row.ActiveUsers
			item.ActiveModels = row.ActiveModels
		}
		items = append(items, dto.AdminAnalyticsDailyItem{
			BucketDay:    item.BucketDay,
			CallCount:    item.CallCount,
			TotalCost:    item.TotalCost,
			ActiveUsers:  item.ActiveUsers,
			ActiveModels: item.ActiveModels,
		})
	}

	return items
}

func operationsAnalyticsLocalOffsetSeconds(referenceTimestamp int64) int {
	return dto.AdminAnalyticsTimezoneOffsetSeconds
}

func operationsAnalyticsBucketID(timestamp int64, localOffsetSeconds int) int64 {
	return (timestamp + int64(localOffsetSeconds)) / dayDurationSeconds
}

func operationsAnalyticsBucketDayLabel(bucketID int64, localOffsetSeconds int) string {
	bucketStart := bucketID*dayDurationSeconds - int64(localOffsetSeconds)
	return time.Unix(bucketStart, 0).
		In(time.FixedZone("operations_analytics", localOffsetSeconds)).
		Format("2006-01-02")
}

func scanOperationsAnalyticsSummary(query *gorm.DB) (operationsAnalyticsSummaryRow, error) {
	row := operationsAnalyticsSummaryRow{}
	err := query.Select(
		"COUNT(*) AS total_calls, " +
			operationsAnalyticsTotalTokensExpr() + " AS total_tokens, " +
			"COALESCE(SUM(logs.quota), 0) AS total_cost, " +
			"COUNT(DISTINCT logs.user_id) AS active_users, " +
			"COUNT(DISTINCT NULLIF(logs.model_name, '')) AS active_models",
	).Scan(&row).Error
	return row, err
}

func operationsAnalyticsTotalTokensExpr() string {
	return "COALESCE(SUM(logs.prompt_tokens), 0) + COALESCE(SUM(logs.completion_tokens), 0)"
}

func normalizeOperationsAnalyticsPageInfo(pageInfo *common.PageInfo) *common.PageInfo {
	if pageInfo == nil {
		return &common.PageInfo{
			Page:     1,
			PageSize: common.ItemsPerPage,
		}
	}
	if pageInfo.Page < 1 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize <= 0 {
		pageInfo.PageSize = common.ItemsPerPage
	}
	return pageInfo
}

func operationsAnalyticsModelBucketExpr() string {
	return "COALESCE(NULLIF(logs.model_name, ''), '" + operationsAnalyticsEmptyModelName + "')"
}

func OperationsAnalyticsDailyBucketIDExpr(localOffsetSeconds int) string {
	adjustedTimestampExpr := "(logs.created_at + " + strconv.Itoa(localOffsetSeconds) + ")"
	alignedTimestampExpr := "(" + adjustedTimestampExpr + " - (" + adjustedTimestampExpr + " % " + strconv.FormatInt(dayDurationSeconds, 10) + "))"

	switch {
	case common.UsingPostgreSQL:
		return "CAST(" + alignedTimestampExpr + " / " + strconv.FormatInt(dayDurationSeconds, 10) + " AS BIGINT)"
	case common.UsingSQLite:
		return "CAST(" + alignedTimestampExpr + " / " + strconv.FormatInt(dayDurationSeconds, 10) + " AS INTEGER)"
	default:
		return "CAST(" + alignedTimestampExpr + " / " + strconv.FormatInt(dayDurationSeconds, 10) + " AS SIGNED)"
	}
}

func buildOperationsAnalyticsLikePattern(keyword string) string {
	pattern := strings.ToLower(strings.TrimSpace(keyword))
	pattern = strings.ReplaceAll(pattern, "!", "!!")
	pattern = strings.ReplaceAll(pattern, "%", "!%")
	pattern = strings.ReplaceAll(pattern, "_", "!_")
	return "%" + pattern + "%"
}

func findOperationsAnalyticsUserIDsByUsernameKeyword(usernameKeyword string) ([]int, error) {
	userIDs := make([]int, 0)
	err := model.DB.Model(&model.User{}).
		Where("LOWER(users.username) LIKE ? ESCAPE '!'", buildOperationsAnalyticsLikePattern(usernameKeyword)).
		Pluck("users.id", &userIDs).Error
	return userIDs, err
}

func buildOperationsAnalyticsWow(requesterUserId int, requesterRole int, endTimestamp int64) (map[string]dto.AdminAnalyticsWowValue, error) {
	currentWeekStart, currentWeekEnd, previousWeekStart, previousWeekEnd := BuildOperationsAnalyticsNaturalWeekRanges(endTimestamp)

	currentQuery, err := buildOperationsAnalyticsBaseQuery(dto.AdminAnalyticsQuery{
		DatePreset:     dto.AdminAnalyticsDatePresetLast7Days,
		StartTimestamp: currentWeekStart,
		EndTimestamp:   currentWeekEnd,
	}, requesterUserId, requesterRole)
	if err != nil {
		return nil, err
	}
	previousQuery, err := buildOperationsAnalyticsBaseQuery(dto.AdminAnalyticsQuery{
		DatePreset:     dto.AdminAnalyticsDatePresetCustom,
		StartTimestamp: previousWeekStart,
		EndTimestamp:   previousWeekEnd,
	}, requesterUserId, requesterRole)
	if err != nil {
		return nil, err
	}

	current, err := scanOperationsAnalyticsSummary(currentQuery)
	if err != nil {
		return nil, err
	}
	previous, err := scanOperationsAnalyticsSummary(previousQuery)
	if err != nil {
		return nil, err
	}

	return map[string]dto.AdminAnalyticsWowValue{
		"total_calls":   buildOperationsAnalyticsWowValue(current.TotalCalls, previous.TotalCalls),
		"total_tokens":  buildOperationsAnalyticsWowValue(current.TotalTokens, previous.TotalTokens),
		"total_cost":    buildOperationsAnalyticsWowValue(current.TotalCost, previous.TotalCost),
		"active_users":  buildOperationsAnalyticsWowValue(current.ActiveUsers, previous.ActiveUsers),
		"active_models": buildOperationsAnalyticsWowValue(current.ActiveModels, previous.ActiveModels),
	}, nil
}

func buildOperationsAnalyticsWowValue(current int64, previous int64) dto.AdminAnalyticsWowValue {
	delta := current - previous
	changeRatio := float64(0)
	switch {
	case previous > 0:
		changeRatio = float64(delta) / float64(previous)
	case current > 0:
		changeRatio = 1
	}

	trend := "flat"
	if delta > 0 {
		trend = "up"
	} else if delta < 0 {
		trend = "down"
	}

	return dto.AdminAnalyticsWowValue{
		Current:     current,
		Previous:    previous,
		Delta:       delta,
		ChangeRatio: changeRatio,
		Trend:       trend,
	}
}

const dayDurationSeconds = int64(24 * 60 * 60)
