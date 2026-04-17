package ratio_setting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

type BillingMode string

const (
	BillingModePerToken   BillingMode = "per_token"
	BillingModePerRequest BillingMode = "per_request"
	BillingModeAdvanced   BillingMode = "advanced"
)

type AdvancedRuleType string

const (
	RuleTypeTextSegment AdvancedRuleType = "text_segment"
	RuleTypeMediaTask   AdvancedRuleType = "media_task"
)

type AdvancedPricingConfig struct {
	ModelModes map[string]BillingMode            `json:"billing_mode"`
	ModelRules map[string]AdvancedPricingRuleSet `json:"rules"`
}

type AdvancedPricingRuleSet struct {
	RuleType AdvancedRuleType    `json:"rule_type"`
	Segments []AdvancedPriceRule `json:"segments"`
}

type AdvancedPriceRule struct {
	Priority      *int     `json:"priority,omitempty"`
	InputMin      *int     `json:"input_min,omitempty"`
	InputMax      *int     `json:"input_max,omitempty"`
	InputPrice    *float64 `json:"input_price,omitempty"`
	OutputPrice   *float64 `json:"output_price,omitempty"`
	InferenceMode string   `json:"inference_mode,omitempty"`
	InputVideo    *bool    `json:"input_video,omitempty"`
	Resolution    string   `json:"resolution,omitempty"`
	UnitPrice     *float64 `json:"unit_price,omitempty"`
	MinTokens     *int     `json:"min_tokens,omitempty"`
}

var advancedPricingModeMap = types.NewRWMap[string, BillingMode]()
var advancedPricingRulesMap = types.NewRWMap[string, AdvancedPricingRuleSet]()

func AdvancedPricingMode2JSONString() string {
	return advancedPricingModeMap.MarshalJSONString()
}

func AdvancedPricingRules2JSONString() string {
	return advancedPricingRulesMap.MarshalJSONString()
}

func UpdateAdvancedPricingModeByJSONString(jsonStr string) error {
	jsonStr = normalizeAdvancedPricingJSON(jsonStr)
	if _, err := parseAdvancedPricingModeMap(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonStringWithCallback(advancedPricingModeMap, jsonStr, InvalidateExposedDataCache)
}

func UpdateAdvancedPricingRulesByJSONString(jsonStr string) error {
	jsonStr = normalizeAdvancedPricingJSON(jsonStr)
	if _, err := parseAdvancedPricingRuleMap(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonStringWithCallback(advancedPricingRulesMap, jsonStr, InvalidateExposedDataCache)
}

func ParseAdvancedPricingConfig(jsonStr string) (*AdvancedPricingConfig, error) {
	jsonStr = normalizeAdvancedPricingJSON(jsonStr)

	cfg := &AdvancedPricingConfig{}
	if err := common.UnmarshalJsonStr(jsonStr, cfg); err != nil {
		return nil, err
	}
	if cfg.ModelModes == nil {
		cfg.ModelModes = make(map[string]BillingMode)
	}
	if cfg.ModelRules == nil {
		cfg.ModelRules = make(map[string]AdvancedPricingRuleSet)
	}
	if err := validateAdvancedPricingModes(cfg.ModelModes); err != nil {
		return nil, err
	}
	if err := validateAdvancedPricingRules(cfg.ModelRules); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeAdvancedPricingJSON(jsonStr string) string {
	if strings.TrimSpace(jsonStr) == "" {
		return "{}"
	}
	return jsonStr
}

func parseAdvancedPricingModeMap(jsonStr string) (map[string]BillingMode, error) {
	cfg, err := ParseAdvancedPricingConfig(fmt.Sprintf(`{"billing_mode":%s}`, jsonStr))
	if err != nil {
		return nil, err
	}
	return cfg.ModelModes, nil
}

func parseAdvancedPricingRuleMap(jsonStr string) (map[string]AdvancedPricingRuleSet, error) {
	cfg, err := ParseAdvancedPricingConfig(fmt.Sprintf(`{"rules":%s}`, jsonStr))
	if err != nil {
		return nil, err
	}
	return cfg.ModelRules, nil
}

func validateAdvancedPricingModes(modes map[string]BillingMode) error {
	for modelName, mode := range modes {
		if strings.TrimSpace(modelName) == "" {
			return fmt.Errorf("advanced pricing model name cannot be empty")
		}
		switch mode {
		case BillingModePerToken, BillingModePerRequest, BillingModeAdvanced:
		default:
			return fmt.Errorf("model %s has invalid billing mode: %s", modelName, mode)
		}
	}
	return nil
}

func validateAdvancedPricingRules(rules map[string]AdvancedPricingRuleSet) error {
	for modelName, ruleSet := range rules {
		if strings.TrimSpace(modelName) == "" {
			return fmt.Errorf("advanced pricing rule model name cannot be empty")
		}
		if err := validateAdvancedPricingRuleSet(modelName, ruleSet); err != nil {
			return err
		}
	}
	return nil
}

func validateAdvancedPricingRuleSet(modelName string, ruleSet AdvancedPricingRuleSet) error {
	if len(ruleSet.Segments) == 0 {
		return fmt.Errorf("model %s requires at least one advanced pricing segment", modelName)
	}

	switch ruleSet.RuleType {
	case RuleTypeTextSegment:
		return validateTextSegmentRules(modelName, ruleSet.Segments)
	case RuleTypeMediaTask:
		return validateMediaTaskRules(modelName, ruleSet.Segments)
	default:
		return fmt.Errorf("model %s has invalid advanced pricing rule type: %s", modelName, ruleSet.RuleType)
	}
}

func validateTextSegmentRules(modelName string, segments []AdvancedPriceRule) error {
	type textInterval struct {
		Min int
		Max int
	}

	intervals := make([]textInterval, 0, len(segments))
	for _, segment := range segments {
		if segment.Priority == nil {
			return fmt.Errorf("model %s text segment is missing priority", modelName)
		}
		if segment.InputMin == nil || segment.InputMax == nil || segment.InputPrice == nil || segment.OutputPrice == nil {
			return fmt.Errorf("model %s text segment is missing required fields", modelName)
		}
		if *segment.InputMin < 0 || *segment.InputMax < 0 {
			return fmt.Errorf("model %s text segment 区间 cannot be negative", modelName)
		}
		if *segment.InputMax <= *segment.InputMin {
			return fmt.Errorf("model %s text segment 区间 is invalid", modelName)
		}
		if *segment.InputPrice < 0 || *segment.OutputPrice < 0 {
			return fmt.Errorf("model %s text segment price cannot be negative", modelName)
		}
		intervals = append(intervals, textInterval{
			Min: *segment.InputMin,
			Max: *segment.InputMax,
		})
	}

	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].Min == intervals[j].Min {
			return intervals[i].Max < intervals[j].Max
		}
		return intervals[i].Min < intervals[j].Min
	})

	for i := 1; i < len(intervals); i++ {
		if intervals[i].Min < intervals[i-1].Max {
			return fmt.Errorf("model %s text segment 区间 overlap", modelName)
		}
	}

	return nil
}

func validateMediaTaskRules(modelName string, segments []AdvancedPriceRule) error {
	for _, segment := range segments {
		if segment.Priority == nil {
			return fmt.Errorf("model %s media task segment is missing priority", modelName)
		}
		if strings.TrimSpace(segment.InferenceMode) == "" || strings.TrimSpace(segment.Resolution) == "" {
			return fmt.Errorf("model %s media task segment is missing required fields", modelName)
		}
		if segment.InputVideo == nil || segment.UnitPrice == nil || segment.MinTokens == nil {
			return fmt.Errorf("model %s media task segment is missing required fields", modelName)
		}
		if *segment.UnitPrice < 0 || *segment.MinTokens < 0 {
			return fmt.Errorf("model %s media task segment config is invalid", modelName)
		}
	}
	return nil
}
