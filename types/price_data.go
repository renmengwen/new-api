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
	InputPrice       *float64
	OutputPrice      *float64
	CacheReadPrice   *float64
	CacheCreatePrice *float64
}

type AdvancedRuleThresholdSnapshot struct {
	InputMin  *int
	InputMax  *int
	OutputMin *int
	OutputMax *int
}

type AdvancedRuleSnapshot struct {
	RuleType          AdvancedRuleType
	MatchSummary      string
	ConditionTags     []string
	Priority          *int
	ServiceTier       string
	CacheRead         *bool
	CacheCreate       *bool
	PriceSnapshot     AdvancedRulePriceSnapshot
	ThresholdSnapshot AdvancedRuleThresholdSnapshot
}

type GroupRatioInfo struct {
	GroupRatio        float64
	GroupSpecialRatio float64
	HasSpecialRatio   bool
}

type PriceData struct {
	FreeModel            bool
	ModelPrice           float64
	ModelRatio           float64
	CompletionRatio      float64
	CacheRatio           float64
	CacheCreationRatio   float64
	CacheCreation5mRatio float64
	CacheCreation1hRatio float64
	ImageRatio           float64
	AudioRatio           float64
	AudioCompletionRatio float64
	OtherRatios          map[string]float64
	BillingMode          BillingMode
	AdvancedRuleType     AdvancedRuleType
	AdvancedRuleSnapshot *AdvancedRuleSnapshot
	UsePrice             bool
	Quota                int
	QuotaToPreConsume    int
	GroupRatioInfo       GroupRatioInfo
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
