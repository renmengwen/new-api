package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func GetUsageLogExportModelLabel(log *model.Log) string {
	modelName := strings.TrimSpace(log.ModelName)
	other := parseUsageLogOther(log.Other)
	if !usageLogOtherBool(other, "is_model_mapped") {
		return modelName
	}
	upstreamModelName := usageLogOtherString(other, "upstream_model_name")
	if upstreamModelName == "" || upstreamModelName == modelName {
		return modelName
	}
	if modelName == "" {
		return upstreamModelName
	}
	return modelName + " -> " + upstreamModelName
}

func GetUsageLogExportUseTimeLabel(log *model.Log) string {
	if log.Type != model.LogTypeConsume && log.Type != model.LogTypeError {
		return ""
	}
	parts := []string{fmt.Sprintf("%d s", log.UseTime)}
	other := parseUsageLogOther(log.Other)
	if log.IsStream {
		if frt, ok := usageLogOtherFloat(other, "frt"); ok && frt > 0 {
			parts = append(parts, fmt.Sprintf("首字 %.1f s", frt/1000))
		}
		parts = append(parts, "流")
		return strings.Join(parts, " / ")
	}
	parts = append(parts, "非流")
	return strings.Join(parts, " / ")
}

func GetUsageLogExportCostLabel(log *model.Log, requestedQuotaDisplayType string) string {
	quotaDisplayType := normalizeUsageLogQuotaDisplayType(requestedQuotaDisplayType)
	quotaText := formatUsageLogQuota(log.Quota, quotaDisplayType, 6)
	other := parseUsageLogOther(log.Other)
	if usageLogOtherString(other, "billing_source") != "subscription" {
		return quotaText
	}
	if quotaDisplayType == operation_setting.QuotaDisplayTypeTokens {
		return "由订阅抵扣（等价额度：" + quotaText + "）"
	}
	return "由订阅抵扣（等价金额：" + quotaText + "）"
}

func normalizeUsageLogQuotaDisplayType(requestedQuotaDisplayType string) string {
	quotaDisplayType := strings.ToUpper(strings.TrimSpace(requestedQuotaDisplayType))
	switch quotaDisplayType {
	case operation_setting.QuotaDisplayTypeUSD,
		operation_setting.QuotaDisplayTypeCNY,
		operation_setting.QuotaDisplayTypeTokens,
		operation_setting.QuotaDisplayTypeCustom:
		return quotaDisplayType
	default:
		return operation_setting.GetQuotaDisplayType()
	}
}

func formatUsageLogQuota(quota int, quotaDisplayType string, digits int) string {
	if quotaDisplayType == operation_setting.QuotaDisplayTypeTokens || common.QuotaPerUnit <= 0 {
		return strconv.Itoa(quota)
	}

	amount := float64(quota) / common.QuotaPerUnit
	symbol := "$"
	switch quotaDisplayType {
	case operation_setting.QuotaDisplayTypeCNY:
		amount = amount * operation_setting.USDExchangeRate
		symbol = "¥"
	case operation_setting.QuotaDisplayTypeCustom:
		generalSetting := operation_setting.GetGeneralSetting()
		if generalSetting.CustomCurrencyExchangeRate > 0 {
			amount = amount * generalSetting.CustomCurrencyExchangeRate
		}
		if generalSetting.CustomCurrencySymbol != "" {
			symbol = generalSetting.CustomCurrencySymbol
		}
	}

	fixedAmount := strconv.FormatFloat(amount, 'f', digits, 64)
	if amount > 0 {
		minimumAmount := math.Pow10(-digits)
		if parsedAmount, err := strconv.ParseFloat(fixedAmount, 64); err == nil && parsedAmount == 0 {
			fixedAmount = strconv.FormatFloat(minimumAmount, 'f', digits, 64)
		}
	}
	return symbol + fixedAmount
}

func parseUsageLogOther(otherJSON string) map[string]any {
	if strings.TrimSpace(otherJSON) == "" {
		return nil
	}
	var other map[string]any
	if err := common.UnmarshalJsonStr(otherJSON, &other); err != nil {
		return nil
	}
	return other
}

func usageLogOtherString(other map[string]any, key string) string {
	if other == nil {
		return ""
	}
	value, ok := other[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func usageLogOtherBool(other map[string]any, key string) bool {
	if other == nil {
		return false
	}
	value, ok := other[key]
	if !ok {
		return false
	}
	switch typedValue := value.(type) {
	case bool:
		return typedValue
	case string:
		return strings.EqualFold(strings.TrimSpace(typedValue), "true")
	default:
		return false
	}
}

func usageLogOtherFloat(other map[string]any, key string) (float64, bool) {
	if other == nil {
		return 0, false
	}
	value, ok := other[key]
	if !ok {
		return 0, false
	}
	switch typedValue := value.(type) {
	case float64:
		return typedValue, true
	case float32:
		return float64(typedValue), true
	case int:
		return float64(typedValue), true
	case int64:
		return float64(typedValue), true
	case json.Number:
		parsedValue, err := typedValue.Float64()
		return parsedValue, err == nil
	case string:
		parsedValue, err := strconv.ParseFloat(strings.TrimSpace(typedValue), 64)
		return parsedValue, err == nil
	default:
		return 0, false
	}
}
