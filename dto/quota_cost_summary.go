package dto

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const adminQuotaCostSummaryMaxRangeSeconds = int64(90 * 24 * 60 * 60)

var adminQuotaCostSummarySortFields = map[string]struct{}{
	"date":          {},
	"model_name":    {},
	"vendor_name":   {},
	"call_count":    {},
	"input_tokens":  {},
	"output_tokens": {},
	"paid_usd":      {},
}

type AdminQuotaCostSummaryQuery struct {
	StartTimestamp int64   `form:"start_timestamp" json:"start_timestamp"`
	EndTimestamp   int64   `form:"end_timestamp" json:"end_timestamp"`
	ModelName      string  `form:"model_name" json:"model_name"`
	Vendor         string  `form:"vendor" json:"vendor"`
	User           string  `form:"user" json:"user"`
	TokenName      string  `form:"token_name" json:"token_name"`
	Channel        int     `form:"channel" json:"channel"`
	Group          string  `form:"group" json:"group"`
	MinCallCount   int64   `form:"min_call_count" json:"min_call_count"`
	MinPaidUSD     float64 `form:"min_paid_usd" json:"min_paid_usd"`
	SortBy         string  `form:"sort_by" json:"sort_by"`
	SortOrder      string  `form:"sort_order" json:"sort_order"`
}

type AdminQuotaCostSummaryExportRequest struct {
	AdminQuotaCostSummaryQuery
	Limit int `json:"limit"`
}

type AdminQuotaCostSummaryItem struct {
	Date                 string  `json:"date"`
	ModelName            string  `json:"model_name"`
	VendorName           string  `json:"vendor_name"`
	InputUnitPriceUSD    float64 `json:"input_unit_price_usd"`
	OutputUnitPriceUSD   float64 `json:"output_unit_price_usd"`
	InputTokens          int64   `json:"input_tokens"`
	OutputTokens         int64   `json:"output_tokens"`
	CallCount            int64   `json:"call_count"`
	InputCostUSD         float64 `json:"input_cost_usd"`
	OutputCostUSD        float64 `json:"output_cost_usd"`
	CacheCreateTokens    int64   `json:"cache_create_tokens"`
	CacheReadTokens      int64   `json:"cache_read_tokens"`
	CacheCreateUnitPrice float64 `json:"cache_create_unit_price_usd"`
	CacheReadUnitPrice   float64 `json:"cache_read_unit_price_usd"`
	CacheTokens          int64   `json:"cache_tokens"`
	CacheCostUSD         float64 `json:"cache_cost_usd"`
	TotalCostUSD         float64 `json:"total_cost_usd"`
	DiscountUSD          float64 `json:"discount_usd"`
	PaidUSD              float64 `json:"paid_usd"`
}

func ParseAdminQuotaCostSummaryQuery(c *gin.Context) (AdminQuotaCostSummaryQuery, error) {
	var query AdminQuotaCostSummaryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		return AdminQuotaCostSummaryQuery{}, errors.New("invalid cost summary query")
	}
	return NormalizeAdminQuotaCostSummaryQuery(query, common.GetTimestamp())
}

func NormalizeAdminQuotaCostSummaryQuery(query AdminQuotaCostSummaryQuery, nowTimestamp int64) (AdminQuotaCostSummaryQuery, error) {
	if query.EndTimestamp <= 0 {
		query.EndTimestamp = nowTimestamp
	}
	if query.StartTimestamp <= 0 {
		query.StartTimestamp = query.EndTimestamp - 7*24*60*60
	}
	if query.EndTimestamp < query.StartTimestamp {
		return AdminQuotaCostSummaryQuery{}, errors.New("end_timestamp must be greater than or equal to start_timestamp")
	}
	if query.EndTimestamp-query.StartTimestamp > adminQuotaCostSummaryMaxRangeSeconds {
		return AdminQuotaCostSummaryQuery{}, errors.New("date range cannot exceed 90 days")
	}

	query.ModelName = strings.TrimSpace(query.ModelName)
	query.Vendor = strings.TrimSpace(query.Vendor)
	query.User = strings.TrimSpace(query.User)
	query.TokenName = strings.TrimSpace(query.TokenName)
	query.Group = strings.TrimSpace(query.Group)
	if query.MinCallCount < 0 {
		query.MinCallCount = 0
	}
	if query.MinPaidUSD < 0 {
		query.MinPaidUSD = 0
	}

	query.SortBy = strings.ToLower(strings.TrimSpace(query.SortBy))
	if _, ok := adminQuotaCostSummarySortFields[query.SortBy]; !ok {
		query.SortBy = "date"
	}
	query.SortOrder = strings.ToLower(strings.TrimSpace(query.SortOrder))
	if query.SortOrder != "asc" {
		query.SortOrder = "desc"
	}
	return query, nil
}
