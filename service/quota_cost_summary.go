package service

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	quotaCostSummaryUnknownVendor = "未知供应商"
	quotaCostSummaryLogBatchSize  = 1000
)

type quotaCostSummaryAccumulator struct {
	item                    dto.AdminQuotaCostSummaryItem
	inputUnitWeighted       float64
	outputUnitWeighted      float64
	cacheReadWeighted       float64
	cacheCreateWeighted     float64
	inputUnitWeightTokens   int64
	outputUnitWeightTokens  int64
	cacheReadWeightTokens   int64
	cacheCreateWeightTokens int64
}

type quotaCostSummaryOther struct {
	ModelRatio            float64 `json:"model_ratio"`
	ModelPrice            float64 `json:"model_price"`
	BillingMode           string  `json:"billing_mode"`
	GroupRatio            float64 `json:"group_ratio"`
	UserGroupRatio        float64 `json:"user_group_ratio"`
	CompletionRatio       float64 `json:"completion_ratio"`
	CacheTokens           int64   `json:"cache_tokens"`
	CacheRatio            float64 `json:"cache_ratio"`
	CacheCreationTokens   int64   `json:"cache_creation_tokens"`
	CacheCreationRatio    float64 `json:"cache_creation_ratio"`
	CacheCreationTokens5m int64   `json:"cache_creation_tokens_5m"`
	CacheCreationRatio5m  float64 `json:"cache_creation_ratio_5m"`
	CacheCreationTokens1h int64   `json:"cache_creation_tokens_1h"`
	CacheCreationRatio1h  float64 `json:"cache_creation_ratio_1h"`
}

func ListQuotaCostSummary(query dto.AdminQuotaCostSummaryQuery, pageInfo *common.PageInfo, requesterUserID int, requesterRole int) ([]dto.AdminQuotaCostSummaryItem, int64, error) {
	query, err := normalizeQuotaCostSummaryQuery(query)
	if err != nil {
		return nil, 0, err
	}
	items, err := buildQuotaCostSummaryItems(query, requesterUserID, requesterRole)
	if err != nil {
		return nil, 0, err
	}
	items = filterQuotaCostSummaryItems(items, query)
	sortQuotaCostSummaryItems(items, query.SortBy, query.SortOrder)
	total := int64(len(items))
	pageInfo = normalizeQuotaCostSummaryPageInfo(pageInfo)
	start := pageInfo.GetStartIdx()
	if start >= len(items) {
		return []dto.AdminQuotaCostSummaryItem{}, total, nil
	}
	end := start + pageInfo.GetPageSize()
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], total, nil
}

func ListQuotaCostSummaryForExport(query dto.AdminQuotaCostSummaryQuery, requesterUserID int, requesterRole int, limit int) ([]dto.AdminQuotaCostSummaryItem, error) {
	query, err := normalizeQuotaCostSummaryQuery(query)
	if err != nil {
		return nil, err
	}
	items, err := buildQuotaCostSummaryItems(query, requesterUserID, requesterRole)
	if err != nil {
		return nil, err
	}
	items = filterQuotaCostSummaryItems(items, query)
	sortQuotaCostSummaryItems(items, query.SortBy, query.SortOrder)
	if limit > 0 && len(items) > limit {
		return items[:limit], nil
	}
	return items, nil
}

func buildQuotaCostSummaryItems(query dto.AdminQuotaCostSummaryQuery, requesterUserID int, requesterRole int) ([]dto.AdminQuotaCostSummaryItem, error) {
	modelVendorMap, modelFilter, err := resolveQuotaCostSummaryModelVendors(query.Vendor)
	if err != nil {
		return nil, err
	}
	if query.Vendor != "" && len(modelFilter) == 0 {
		return []dto.AdminQuotaCostSummaryItem{}, nil
	}

	logQuery, err := buildQuotaCostSummaryLogQuery(query, modelFilter, requesterUserID, requesterRole)
	if err != nil {
		return nil, err
	}

	accumulators, err := aggregateQuotaCostSummaryLogQuery(logQuery, modelVendorMap)
	if err != nil {
		return nil, err
	}
	return quotaCostSummaryItemsFromAccumulators(accumulators), nil
}

func buildQuotaCostSummaryLogQuery(query dto.AdminQuotaCostSummaryQuery, modelFilter []string, requesterUserID int, requesterRole int) (*gorm.DB, error) {
	tx := model.BuildAllLogsQuery(
		model.LogTypeConsume,
		query.StartTimestamp,
		query.EndTimestamp,
		"",
		"",
		query.TokenName,
		query.Channel,
		query.Group,
		"",
	)

	if query.ModelName != "" {
		tx = tx.Where("LOWER(logs.model_name) LIKE ? ESCAPE '!'", quotaCostSummaryLikePattern(query.ModelName))
	}
	if len(modelFilter) > 0 {
		tx = tx.Where("logs.model_name IN ?", modelFilter)
	}
	if query.User != "" {
		if userID, err := strconv.Atoi(query.User); err == nil && userID > 0 {
			tx = tx.Where("(logs.user_id = ? OR LOWER(logs.username) LIKE ? ESCAPE '!')", userID, quotaCostSummaryLikePattern(query.User))
		} else {
			tx = tx.Where("LOWER(logs.username) LIKE ? ESCAPE '!'", quotaCostSummaryLikePattern(query.User))
		}
	}

	return applyUsageLogScope(tx, requesterUserID, requesterRole)
}

func resolveQuotaCostSummaryModelVendors(vendorFilter string) (map[string]string, []string, error) {
	type row struct {
		ModelName  string `gorm:"column:model_name"`
		VendorName string `gorm:"column:vendor_name"`
	}
	query := model.DB.Model(&model.Model{}).
		Select("models.model_name, COALESCE(vendors.name, '') AS vendor_name").
		Joins("LEFT JOIN vendors ON vendors.id = models.vendor_id")
	if strings.TrimSpace(vendorFilter) != "" {
		query = query.Where("LOWER(vendors.name) LIKE ? ESCAPE '!'", quotaCostSummaryLikePattern(vendorFilter))
	}

	var rows []row
	if err := query.Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	modelVendorMap := make(map[string]string, len(rows))
	modelFilter := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.ModelName) == "" {
			continue
		}
		vendorName := strings.TrimSpace(row.VendorName)
		if vendorName == "" {
			vendorName = quotaCostSummaryUnknownVendor
		}
		modelVendorMap[row.ModelName] = vendorName
		if vendorFilter != "" {
			modelFilter = append(modelFilter, row.ModelName)
		}
	}
	return modelVendorMap, modelFilter, nil
}

func ensureQuotaCostSummaryVendorsForLogs(modelVendorMap map[string]string, logs []*model.Log) error {
	missing := make([]string, 0)
	seen := make(map[string]struct{})
	for _, log := range logs {
		modelName := strings.TrimSpace(log.ModelName)
		if modelName == "" {
			continue
		}
		if _, ok := modelVendorMap[modelName]; ok {
			continue
		}
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		missing = append(missing, modelName)
	}
	if len(missing) == 0 {
		return nil
	}

	type row struct {
		ModelName  string `gorm:"column:model_name"`
		VendorName string `gorm:"column:vendor_name"`
	}
	var rows []row
	if err := model.DB.Model(&model.Model{}).
		Select("models.model_name, COALESCE(vendors.name, '') AS vendor_name").
		Joins("LEFT JOIN vendors ON vendors.id = models.vendor_id").
		Where("models.model_name IN ?", missing).
		Find(&rows).Error; err != nil {
		return err
	}
	for _, modelName := range missing {
		modelVendorMap[modelName] = quotaCostSummaryUnknownVendor
	}
	for _, row := range rows {
		vendorName := strings.TrimSpace(row.VendorName)
		if vendorName == "" {
			vendorName = quotaCostSummaryUnknownVendor
		}
		modelVendorMap[row.ModelName] = vendorName
	}
	return nil
}

func fetchQuotaCostSummaryLogBatch(tx *gorm.DB, lastID int) ([]*model.Log, error) {
	var logs []*model.Log
	err := tx.Select("logs.id, logs.user_id, logs.username, logs.created_at, logs.type, logs.token_name, logs.model_name, logs.quota, logs.prompt_tokens, logs.completion_tokens, logs.channel_id, logs.other").
		Where("logs.id > ?", lastID).
		Order("logs.id asc").
		Limit(quotaCostSummaryLogBatchSize).
		Find(&logs).Error
	return logs, err
}

func aggregateQuotaCostSummaryLogQuery(tx *gorm.DB, modelVendorMap map[string]string) (map[string]*quotaCostSummaryAccumulator, error) {
	accumulators := make(map[string]*quotaCostSummaryAccumulator)
	lastID := 0
	for {
		logs, err := fetchQuotaCostSummaryLogBatch(tx, lastID)
		if err != nil {
			return nil, err
		}
		if len(logs) == 0 {
			return accumulators, nil
		}
		if err := ensureQuotaCostSummaryVendorsForLogs(modelVendorMap, logs); err != nil {
			return nil, err
		}
		for _, log := range logs {
			applyQuotaCostSummaryLogToAccumulators(accumulators, log, modelVendorMap)
		}
		lastID = logs[len(logs)-1].Id
		if len(logs) < quotaCostSummaryLogBatchSize {
			return accumulators, nil
		}
	}
}

func quotaCostSummaryItemsFromAccumulators(accumulators map[string]*quotaCostSummaryAccumulator) []dto.AdminQuotaCostSummaryItem {
	items := make([]dto.AdminQuotaCostSummaryItem, 0, len(accumulators))
	for _, acc := range accumulators {
		finalizeQuotaCostSummaryAccumulator(acc)
		items = append(items, acc.item)
	}
	return items
}

func applyQuotaCostSummaryLogToAccumulators(accumulators map[string]*quotaCostSummaryAccumulator, log *model.Log, modelVendorMap map[string]string) {
	date := quotaCostSummaryDate(log.CreatedAt)
	modelName := strings.TrimSpace(log.ModelName)
	if modelName == "" {
		modelName = "-"
	}
	vendorName := modelVendorMap[log.ModelName]
	if vendorName == "" {
		vendorName = quotaCostSummaryUnknownVendor
	}
	key := date + "\x00" + modelName + "\x00" + vendorName
	acc := accumulators[key]
	if acc == nil {
		acc = &quotaCostSummaryAccumulator{item: dto.AdminQuotaCostSummaryItem{
			Date:       date,
			ModelName:  modelName,
			VendorName: vendorName,
		}}
		accumulators[key] = acc
	}
	applyQuotaCostSummaryLog(acc, log)
}

func applyQuotaCostSummaryLog(acc *quotaCostSummaryAccumulator, log *model.Log) {
	other := parseQuotaCostSummaryOther(log.Other)
	groupRatio := firstPositiveFloat(other.UserGroupRatio, other.GroupRatio, 1)
	advancedBilling := quotaCostSummaryIsAdvancedBilling(other)
	fixedPrice := other.ModelPrice > 0 && !advancedBilling
	inputUnitPrice := quotaCostSummaryInputUnitPrice(other)
	if fixedPrice || advancedBilling {
		inputUnitPrice = 0
	}
	outputUnitPrice := inputUnitPrice * firstPositiveFloat(other.CompletionRatio, 0)
	cacheReadUnitPrice := inputUnitPrice * firstPositiveFloat(other.CacheRatio, 0)
	cacheCreateUnitPrice := inputUnitPrice * firstPositiveFloat(other.CacheCreationRatio, 0)
	cacheCreateUnitPrice5m := inputUnitPrice * firstPositiveFloat(other.CacheCreationRatio5m, 0)
	cacheCreateUnitPrice1h := inputUnitPrice * firstPositiveFloat(other.CacheCreationRatio1h, 0)

	cacheReadTokens := positiveInt64(other.CacheTokens)
	cacheCreateTokens, cacheCreateTokens5m, cacheCreateTokens1h, cacheCreateTotalTokens := quotaCostSummaryCacheCreationTokens(other)

	inputTokens := int64(log.PromptTokens)
	outputTokens := int64(log.CompletionTokens)
	nonCacheInputTokens := inputTokens - cacheReadTokens - cacheCreateTotalTokens
	if nonCacheInputTokens < 0 {
		nonCacheInputTokens = 0
	}

	inputCost := float64(nonCacheInputTokens) / 1000000 * inputUnitPrice * groupRatio
	outputCost := float64(outputTokens) / 1000000 * outputUnitPrice * groupRatio
	cacheReadCost := float64(cacheReadTokens) / 1000000 * cacheReadUnitPrice * groupRatio
	cacheCreateCost := float64(cacheCreateTokens)/1000000*cacheCreateUnitPrice*groupRatio +
		float64(cacheCreateTokens5m)/1000000*cacheCreateUnitPrice5m*groupRatio +
		float64(cacheCreateTokens1h)/1000000*cacheCreateUnitPrice1h*groupRatio
	if fixedPrice {
		inputCost = other.ModelPrice * groupRatio
		outputCost = 0
		cacheReadCost = 0
		cacheCreateCost = 0
	}
	paidUSD := quotaToUSD(log.Quota)

	acc.item.CallCount++
	acc.item.InputTokens += inputTokens
	acc.item.OutputTokens += outputTokens
	acc.item.CacheReadTokens += cacheReadTokens
	acc.item.CacheCreateTokens += cacheCreateTotalTokens
	acc.item.CacheTokens += cacheReadTokens + cacheCreateTotalTokens
	acc.item.InputCostUSD += inputCost
	acc.item.OutputCostUSD += outputCost
	acc.item.CacheCostUSD += cacheReadCost + cacheCreateCost
	acc.item.PaidUSD += paidUSD

	addWeightedUnitPrice(&acc.inputUnitWeighted, &acc.inputUnitWeightTokens, inputUnitPrice, nonCacheInputTokens)
	addWeightedUnitPrice(&acc.outputUnitWeighted, &acc.outputUnitWeightTokens, outputUnitPrice, outputTokens)
	addWeightedUnitPrice(&acc.cacheReadWeighted, &acc.cacheReadWeightTokens, cacheReadUnitPrice, cacheReadTokens)
	addWeightedUnitPrice(&acc.cacheCreateWeighted, &acc.cacheCreateWeightTokens, cacheCreateUnitPrice, cacheCreateTokens)
	addWeightedUnitPrice(&acc.cacheCreateWeighted, &acc.cacheCreateWeightTokens, cacheCreateUnitPrice5m, cacheCreateTokens5m)
	addWeightedUnitPrice(&acc.cacheCreateWeighted, &acc.cacheCreateWeightTokens, cacheCreateUnitPrice1h, cacheCreateTokens1h)
}

func finalizeQuotaCostSummaryAccumulator(acc *quotaCostSummaryAccumulator) {
	acc.item.InputUnitPriceUSD = weightedAverage(acc.inputUnitWeighted, acc.inputUnitWeightTokens)
	acc.item.OutputUnitPriceUSD = weightedAverage(acc.outputUnitWeighted, acc.outputUnitWeightTokens)
	acc.item.CacheReadUnitPrice = weightedAverage(acc.cacheReadWeighted, acc.cacheReadWeightTokens)
	acc.item.CacheCreateUnitPrice = weightedAverage(acc.cacheCreateWeighted, acc.cacheCreateWeightTokens)
	acc.item.TotalCostUSD = acc.item.InputCostUSD + acc.item.OutputCostUSD + acc.item.CacheCostUSD
	acc.item.DiscountUSD = math.Max(acc.item.TotalCostUSD-acc.item.PaidUSD, 0)
}

func parseQuotaCostSummaryOther(otherJSON string) quotaCostSummaryOther {
	var other quotaCostSummaryOther
	if strings.TrimSpace(otherJSON) == "" {
		return other
	}
	_ = common.UnmarshalJsonStr(otherJSON, &other)
	return other
}

func quotaCostSummaryIsAdvancedBilling(other quotaCostSummaryOther) bool {
	return strings.EqualFold(strings.TrimSpace(other.BillingMode), "advanced")
}

func quotaCostSummaryCacheCreationTokens(other quotaCostSummaryOther) (int64, int64, int64, int64) {
	totalCreation := positiveInt64(other.CacheCreationTokens)
	split5m := positiveInt64(other.CacheCreationTokens5m)
	split1h := positiveInt64(other.CacheCreationTokens1h)
	if split5m > 0 || split1h > 0 {
		splitTotal := split5m + split1h
		baseCreation := totalCreation - splitTotal
		if baseCreation < 0 {
			baseCreation = 0
		}
		if totalCreation < splitTotal {
			totalCreation = splitTotal
		}
		return baseCreation, split5m, split1h, totalCreation
	}
	return totalCreation, 0, 0, totalCreation
}

func quotaCostSummaryInputUnitPrice(other quotaCostSummaryOther) float64 {
	if other.ModelRatio > 0 && common.QuotaPerUnit > 0 {
		return other.ModelRatio * (1000000 / common.QuotaPerUnit)
	}
	return 0
}

func quotaToUSD(quota int) float64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	return float64(quota) / common.QuotaPerUnit
}

func addWeightedUnitPrice(total *float64, weight *int64, price float64, tokens int64) {
	if price <= 0 || tokens <= 0 {
		return
	}
	*total += price * float64(tokens)
	*weight += tokens
}

func weightedAverage(total float64, weight int64) float64 {
	if weight <= 0 {
		return 0
	}
	return total / float64(weight)
}

func filterQuotaCostSummaryItems(items []dto.AdminQuotaCostSummaryItem, query dto.AdminQuotaCostSummaryQuery) []dto.AdminQuotaCostSummaryItem {
	filtered := make([]dto.AdminQuotaCostSummaryItem, 0, len(items))
	for _, item := range items {
		if query.MinCallCount > 0 && item.CallCount < query.MinCallCount {
			continue
		}
		if query.MinPaidUSD > 0 && item.PaidUSD < query.MinPaidUSD {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func sortQuotaCostSummaryItems(items []dto.AdminQuotaCostSummaryItem, sortBy string, sortOrder string) {
	desc := sortOrder != "asc"
	sort.SliceStable(items, func(i int, j int) bool {
		cmp := compareQuotaCostSummaryItemsWithTieBreakers(items[i], items[j], sortBy)
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareQuotaCostSummaryItemsWithTieBreakers(a dto.AdminQuotaCostSummaryItem, b dto.AdminQuotaCostSummaryItem, sortBy string) int {
	fields := []string{sortBy, "paid_usd", "call_count", "date", "model_name", "vendor_name"}
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		if cmp := compareQuotaCostSummaryItem(a, b, field); cmp != 0 {
			return cmp
		}
	}
	return 0
}

func compareQuotaCostSummaryItem(a dto.AdminQuotaCostSummaryItem, b dto.AdminQuotaCostSummaryItem, sortBy string) int {
	switch sortBy {
	case "model_name":
		return strings.Compare(a.ModelName, b.ModelName)
	case "vendor_name":
		return strings.Compare(a.VendorName, b.VendorName)
	case "call_count":
		return compareInt64(a.CallCount, b.CallCount)
	case "input_tokens":
		return compareInt64(a.InputTokens, b.InputTokens)
	case "output_tokens":
		return compareInt64(a.OutputTokens, b.OutputTokens)
	case "paid_usd":
		return compareFloat64(a.PaidUSD, b.PaidUSD)
	default:
		return strings.Compare(a.Date, b.Date)
	}
}

func compareInt64(a int64, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFloat64(a float64, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func normalizeQuotaCostSummaryQuery(query dto.AdminQuotaCostSummaryQuery) (dto.AdminQuotaCostSummaryQuery, error) {
	return dto.NormalizeAdminQuotaCostSummaryQuery(query, common.GetTimestamp())
}

func normalizeQuotaCostSummaryPageInfo(pageInfo *common.PageInfo) *common.PageInfo {
	if pageInfo == nil {
		return &common.PageInfo{Page: 1, PageSize: common.ItemsPerPage}
	}
	if pageInfo.Page < 1 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize <= 0 {
		pageInfo.PageSize = common.ItemsPerPage
	}
	return pageInfo
}

func quotaCostSummaryDate(timestamp int64) string {
	return time.Unix(timestamp, 0).In(dto.AdminAnalyticsLocation()).Format("2006-01-02")
}

func quotaCostSummaryLikePattern(keyword string) string {
	pattern := strings.ToLower(strings.TrimSpace(keyword))
	pattern = strings.ReplaceAll(pattern, "!", "!!")
	pattern = strings.ReplaceAll(pattern, "%", "!%")
	pattern = strings.ReplaceAll(pattern, "_", "!_")
	return "%" + pattern + "%"
}

func firstPositiveFloat(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func positiveInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func ValidateQuotaCostSummaryExport(query dto.AdminQuotaCostSummaryQuery) error {
	_, err := normalizeQuotaCostSummaryQuery(query)
	return err
}
