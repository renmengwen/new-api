package dto

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	AdminAnalyticsDatePresetToday       = "today"
	AdminAnalyticsDatePresetLast7Days   = "last7days"
	AdminAnalyticsDatePresetCustom      = "custom"
	AdminAnalyticsViewModels            = "models"
	AdminAnalyticsViewUsers             = "users"
	AdminAnalyticsViewDaily             = "daily"
	AdminAnalyticsTimezoneOffsetSeconds = 8 * 60 * 60
	adminAnalyticsMaxRangeSeconds       = int64(90 * 24 * 60 * 60)
)

var adminAnalyticsLocation = time.FixedZone("UTC+8", AdminAnalyticsTimezoneOffsetSeconds)

type AdminAnalyticsQuery struct {
	DatePreset      string `form:"date_preset" json:"date_preset"`
	StartTimestamp  int64  `form:"start_timestamp" json:"start_timestamp"`
	EndTimestamp    int64  `form:"end_timestamp" json:"end_timestamp"`
	SortBy          string `form:"sort_by" json:"sort_by"`
	SortOrder       string `form:"sort_order" json:"sort_order"`
	ModelKeyword    string `form:"model_keyword" json:"model_keyword"`
	UsernameKeyword string `form:"username_keyword" json:"username_keyword"`
}

type AdminAnalyticsWowValue struct {
	Current     int64   `json:"current"`
	Previous    int64   `json:"previous"`
	Delta       int64   `json:"delta"`
	ChangeRatio float64 `json:"change_ratio"`
	Trend       string  `json:"trend"`
}

type AdminAnalyticsSummaryResponse struct {
	TotalCalls   int64                             `json:"total_calls"`
	TotalTokens  int64                             `json:"total_tokens"`
	TotalCost    int64                             `json:"total_cost"`
	ActiveUsers  int64                             `json:"active_users"`
	ActiveModels int64                             `json:"active_models"`
	Wow          map[string]AdminAnalyticsWowValue `json:"wow,omitempty"`
}

type AdminAnalyticsBreakdownResponse struct {
	Items []map[string]any `json:"items"`
}

type AdminAnalyticsDailyResponse struct {
	Items []AdminAnalyticsDailyItem `json:"items"`
}

type AdminAnalyticsExportResponse struct {
	Status string `json:"status"`
}

type AdminAnalyticsExportRequest struct {
	View             string `json:"view"`
	DatePreset       string `json:"date_preset"`
	StartTimestamp   int64  `json:"start_timestamp"`
	EndTimestamp     int64  `json:"end_timestamp"`
	SortBy           string `json:"sort_by"`
	SortOrder        string `json:"sort_order"`
	ModelKeyword     string `json:"model_keyword"`
	UsernameKeyword  string `json:"username_keyword"`
	QuotaDisplayType string `json:"quota_display_type"`
	Limit            int    `json:"limit"`
}

type AdminAnalyticsModelItem struct {
	ModelName        string  `json:"model_name"`
	CallCount        int64   `json:"call_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalCost        int64   `json:"total_cost"`
	AvgUseTime       float64 `json:"avg_use_time"`
	SuccessRate      float64 `json:"success_rate"`
}

type AdminAnalyticsUserItem struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	CallCount    int64  `json:"call_count"`
	ModelCount   int64  `json:"model_count"`
	TotalTokens  int64  `json:"total_tokens"`
	TotalCost    int64  `json:"total_cost"`
	LastCalledAt int64  `json:"last_called_at"`
}

type AdminAnalyticsDailyItem struct {
	BucketDay    string `json:"bucket_day"`
	CallCount    int64  `json:"call_count"`
	TotalCost    int64  `json:"total_cost"`
	ActiveUsers  int64  `json:"active_users"`
	ActiveModels int64  `json:"active_models"`
}

func ParseAdminAnalyticsQuery(c *gin.Context) (AdminAnalyticsQuery, error) {
	var query AdminAnalyticsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		return AdminAnalyticsQuery{}, errors.New("invalid analytics query")
	}
	return NormalizeAdminAnalyticsQuery(query, common.GetTimestamp())
}

func (req AdminAnalyticsExportRequest) ToQuery() AdminAnalyticsQuery {
	return AdminAnalyticsQuery{
		DatePreset:      req.DatePreset,
		StartTimestamp:  req.StartTimestamp,
		EndTimestamp:    req.EndTimestamp,
		SortBy:          req.SortBy,
		SortOrder:       req.SortOrder,
		ModelKeyword:    req.ModelKeyword,
		UsernameKeyword: req.UsernameKeyword,
	}
}

func NormalizeAdminAnalyticsQuery(query AdminAnalyticsQuery, nowTimestamp int64) (AdminAnalyticsQuery, error) {
	if query.DatePreset == "" {
		query.DatePreset = AdminAnalyticsDatePresetLast7Days
	}

	endTimestamp := query.EndTimestamp
	if endTimestamp <= 0 {
		endTimestamp = nowTimestamp
	}

	switch query.DatePreset {
	case AdminAnalyticsDatePresetToday:
		query.StartTimestamp = startOfAdminAnalyticsDay(endTimestamp)
		query.EndTimestamp = endTimestamp
	case AdminAnalyticsDatePresetLast7Days:
		query.StartTimestamp = startOfAdminAnalyticsDay(endTimestamp - 6*24*60*60)
		query.EndTimestamp = endTimestamp
	case AdminAnalyticsDatePresetCustom:
		if query.StartTimestamp <= 0 || query.EndTimestamp <= 0 {
			return AdminAnalyticsQuery{}, errors.New("custom date range requires start_timestamp and end_timestamp")
		}
	default:
		return AdminAnalyticsQuery{}, errors.New("invalid date_preset")
	}

	if query.EndTimestamp < query.StartTimestamp {
		return AdminAnalyticsQuery{}, errors.New("end_timestamp must be greater than or equal to start_timestamp")
	}
	if query.EndTimestamp-query.StartTimestamp > adminAnalyticsMaxRangeSeconds {
		return AdminAnalyticsQuery{}, errors.New("analytics date range cannot exceed 90 days")
	}
	return query, nil
}

func AdminAnalyticsLocation() *time.Location {
	return adminAnalyticsLocation
}

func startOfAdminAnalyticsDay(timestamp int64) int64 {
	dayTime := time.Unix(timestamp, 0).In(AdminAnalyticsLocation())
	return time.Date(dayTime.Year(), dayTime.Month(), dayTime.Day(), 0, 0, 0, 0, dayTime.Location()).Unix()
}
