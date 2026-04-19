package types

import "fmt"

type BillingMode string

const (
	BillingModePerToken   BillingMode = "per_token"
	BillingModePerRequest BillingMode = "per_request"
	BillingModeAdvanced   BillingMode = "advanced"
)

type AdvancedRuleType string

const (
	AdvancedRuleTypeTextSegment AdvancedRuleType = "text_segment"
	AdvancedRuleTypeMediaTask   AdvancedRuleType = "media_task"
)

type AdvancedRulePriceSnapshot struct {
	InputPrice        *float64 `json:"input_price,omitempty"`
	OutputPrice       *float64 `json:"output_price,omitempty"`
	CacheReadPrice    *float64 `json:"cache_read_price,omitempty"`
	CacheCreatePrice  *float64 `json:"cache_create_price,omitempty"`
	CacheStoragePrice *float64 `json:"cache_storage_price,omitempty"`
}

type AdvancedRuleThresholdSnapshot struct {
	InputMin         *int `json:"input_min,omitempty"`
	InputMax         *int `json:"input_max,omitempty"`
	OutputMin        *int `json:"output_min,omitempty"`
	OutputMax        *int `json:"output_max,omitempty"`
	MinTokens        *int `json:"min_tokens,omitempty"`
	ToolUsageCount   *int `json:"tool_usage_count,omitempty"`
	FreeQuota        *int `json:"free_quota,omitempty"`
	OverageThreshold *int `json:"overage_threshold,omitempty"`
}

type AdvancedRuleSnapshot struct {
	RuleType          AdvancedRuleType              `json:"rule_type,omitempty"`
	MatchSummary      string                        `json:"match_summary,omitempty"`
	ConditionTags     []string                      `json:"condition_tags,omitempty"`
	Priority          *int                          `json:"priority,omitempty"`
	TaskType          string                        `json:"task_type,omitempty"`
	BillingUnit       string                        `json:"billing_unit,omitempty"`
	ServiceTier       string                        `json:"service_tier,omitempty"`
	InputModality     string                        `json:"input_modality,omitempty"`
	OutputModality    string                        `json:"output_modality,omitempty"`
	ImageSizeTier     string                        `json:"image_size_tier,omitempty"`
	ToolUsageType     string                        `json:"tool_usage_type,omitempty"`
	CacheRead         *bool                         `json:"cache_read,omitempty"`
	CacheCreate       *bool                         `json:"cache_create,omitempty"`
	PriceSnapshot     AdvancedRulePriceSnapshot     `json:"price_snapshot,omitempty"`
	ThresholdSnapshot AdvancedRuleThresholdSnapshot `json:"threshold_snapshot,omitempty"`
}

const (
	AdvancedBillingUnitPerMillionTokens = "per_million_tokens"
	AdvancedBillingUnitPerSecond        = "per_second"
	AdvancedBillingUnitPerMinute        = "per_minute"
	AdvancedBillingUnitPerImage         = "per_image"
	AdvancedBillingUnitPer1000Calls     = "per_1000_calls"
)

type AdvancedPricingContextSnapshot struct {
	BillingUnit      string   `json:"billing_unit,omitempty"`
	InputModalities  []string `json:"input_modalities,omitempty"`
	OutputModalities []string `json:"output_modalities,omitempty"`
	ImageSizeTier    string   `json:"image_size_tier,omitempty"`
	ToolUsageType    string   `json:"tool_usage_type,omitempty"`
	ToolUsageCount   *int     `json:"tool_usage_count,omitempty"`
	FreeQuota        *int     `json:"free_quota,omitempty"`
	OverageThreshold *int     `json:"overage_threshold,omitempty"`
}

type GroupRatioInfo struct {
	GroupRatio        float64
	GroupSpecialRatio float64
	HasSpecialRatio   bool
}

type PriceData struct {
	FreeModel              bool
	ModelPrice             float64
	ModelRatio             float64
	CompletionRatio        float64
	CacheRatio             float64
	CacheCreationRatio     float64
	CacheCreation5mRatio   float64
	CacheCreation1hRatio   float64
	ImageRatio             float64
	AudioRatio             float64
	AudioCompletionRatio   float64
	OtherRatios            map[string]float64
	BillingMode            BillingMode
	AdvancedRuleType       AdvancedRuleType
	AdvancedRuleSnapshot   *AdvancedRuleSnapshot
	AdvancedPricingContext *AdvancedPricingContextSnapshot
	UsePrice               bool
	Quota                  int
	QuotaToPreConsume      int
	GroupRatioInfo         GroupRatioInfo
}

func (p *PriceData) AddOtherRatio(key string, ratio float64) {
	if p.OtherRatios == nil {
		p.OtherRatios = make(map[string]float64)
	}
	if ratio <= 0 {
		return
	}
	p.OtherRatios[key] = ratio
}

func (p *PriceData) ToSetting() string {
	return fmt.Sprintf("ModelPrice: %f, ModelRatio: %f, CompletionRatio: %f, CacheRatio: %f, GroupRatio: %f, UsePrice: %t, CacheCreationRatio: %f, CacheCreation5mRatio: %f, CacheCreation1hRatio: %f, QuotaToPreConsume: %d, ImageRatio: %f, AudioRatio: %f, AudioCompletionRatio: %f", p.ModelPrice, p.ModelRatio, p.CompletionRatio, p.CacheRatio, p.GroupRatioInfo.GroupRatio, p.UsePrice, p.CacheCreationRatio, p.CacheCreation5mRatio, p.CacheCreation1hRatio, p.QuotaToPreConsume, p.ImageRatio, p.AudioRatio, p.AudioCompletionRatio)
}
