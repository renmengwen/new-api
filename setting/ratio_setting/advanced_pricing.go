package ratio_setting

import (
	"fmt"
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
	Priority *int `json:"priority,omitempty"`

	InputMin *int `json:"input_min,omitempty"`
	InputMax *int `json:"input_max,omitempty"`

	OutputMin *int `json:"output_min,omitempty"`
	OutputMax *int `json:"output_max,omitempty"`

	ServiceTier string `json:"service_tier,omitempty"`
	CacheRead   *bool  `json:"cache_read,omitempty"`
	CacheCreate *bool  `json:"cache_create,omitempty"`

	InputPrice       *float64 `json:"input_price,omitempty"`
	OutputPrice      *float64 `json:"output_price,omitempty"`
	CacheReadPrice   *float64 `json:"cache_read_price,omitempty"`
	CacheCreatePrice *float64 `json:"cache_create_price,omitempty"`

	InferenceMode string `json:"inference_mode,omitempty"`
	Audio         *bool  `json:"audio,omitempty"`
	InputVideo    *bool  `json:"input_video,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	AspectRatio   string `json:"aspect_ratio,omitempty"`

	OutputDurationMin *int `json:"output_duration_min,omitempty"`
	OutputDurationMax *int `json:"output_duration_max,omitempty"`

	InputVideoDurationMin *int `json:"input_video_duration_min,omitempty"`
	InputVideoDurationMax *int `json:"input_video_duration_max,omitempty"`

	Draft            *bool    `json:"draft,omitempty"`
	DraftCoefficient *float64 `json:"draft_coefficient,omitempty"`
	Remark           string   `json:"remark,omitempty"`
	UnitPrice        *float64 `json:"unit_price,omitempty"`
	MinTokens        *int     `json:"min_tokens,omitempty"`
}

var advancedPricingModeMap = types.NewRWMap[string, BillingMode]()
var advancedPricingRulesMap = types.NewRWMap[string, AdvancedPricingRuleSet]()

func AdvancedPricingMode2JSONString() string {
	return advancedPricingModeMap.MarshalJSONString()
}

func AdvancedPricingRules2JSONString() string {
	return advancedPricingRulesMap.MarshalJSONString()
}

func ValidateAdvancedPricingModeJSONString(jsonStr string) error {
	_, err := parseAdvancedPricingModeMap(normalizeAdvancedPricingJSON(jsonStr))
	return err
}

func ValidateAdvancedPricingRulesJSONString(jsonStr string) error {
	_, err := parseAdvancedPricingRuleMap(normalizeAdvancedPricingJSON(jsonStr))
	return err
}

func UpdateAdvancedPricingModeByJSONString(jsonStr string) error {
	jsonStr = normalizeAdvancedPricingJSON(jsonStr)
	if err := ValidateAdvancedPricingModeJSONString(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonString(advancedPricingModeMap, jsonStr)
}

func UpdateAdvancedPricingRulesByJSONString(jsonStr string) error {
	jsonStr = normalizeAdvancedPricingJSON(jsonStr)
	if err := ValidateAdvancedPricingRulesJSONString(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonString(advancedPricingRulesMap, jsonStr)
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
	if err := validateUniqueSegmentPriorities(modelName, ruleSet.Segments); err != nil {
		return err
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

func validateUniqueSegmentPriorities(modelName string, segments []AdvancedPriceRule) error {
	priorities := make(map[int]struct{}, len(segments))
	for _, segment := range segments {
		if segment.Priority == nil {
			return fmt.Errorf("model %s segment is missing priority", modelName)
		}
		if _, exists := priorities[*segment.Priority]; exists {
			return fmt.Errorf("model %s has duplicate priority: %d", modelName, *segment.Priority)
		}
		priorities[*segment.Priority] = struct{}{}
	}
	return nil
}

func validateTextSegmentRules(modelName string, segments []AdvancedPriceRule) error {
	for _, segment := range segments {
		if err := validateTextRange(modelName, "input", segment.InputMin, segment.InputMax); err != nil {
			return err
		}
		if err := validateTextRange(modelName, "output", segment.OutputMin, segment.OutputMax); err != nil {
			return err
		}
		if !hasTextCondition(segment) {
			return fmt.Errorf("model %s text segment requires at least one condition", modelName)
		}
		if segment.InputPrice == nil {
			return fmt.Errorf("model %s text segment is missing input_price", modelName)
		}
		if *segment.InputPrice < 0 {
			return fmt.Errorf("model %s text segment input_price cannot be negative", modelName)
		}
		if segment.OutputPrice != nil && *segment.OutputPrice < 0 {
			return fmt.Errorf("model %s text segment output_price cannot be negative", modelName)
		}
		if segment.CacheReadPrice != nil && *segment.CacheReadPrice < 0 {
			return fmt.Errorf("model %s text segment cache_read_price cannot be negative", modelName)
		}
		if segment.CacheCreatePrice != nil && *segment.CacheCreatePrice < 0 {
			return fmt.Errorf("model %s text segment cache_create_price cannot be negative", modelName)
		}
	}

	for i := 0; i < len(segments); i++ {
		for j := i + 1; j < len(segments); j++ {
			if textSegmentsOverlap(segments[i], segments[j]) {
				return fmt.Errorf("model %s text segment 区间 overlap", modelName)
			}
		}
	}
	return nil
}

func validateTextRange(modelName, rangeName string, minVal, maxVal *int) error {
	if minVal == nil && maxVal == nil {
		return nil
	}
	if minVal == nil || maxVal == nil {
		return fmt.Errorf("model %s text segment %s 区间 must include both min and max", modelName, rangeName)
	}
	if *minVal < 0 || *maxVal < 0 {
		return fmt.Errorf("model %s text segment %s 区间 cannot be negative", modelName, rangeName)
	}
	if *maxVal < *minVal {
		return fmt.Errorf("model %s text segment %s 区间 is invalid", modelName, rangeName)
	}
	return nil
}

func hasTextCondition(segment AdvancedPriceRule) bool {
	return hasIntRange(segment.InputMin, segment.InputMax) ||
		hasIntRange(segment.OutputMin, segment.OutputMax) ||
		strings.TrimSpace(segment.ServiceTier) != "" ||
		segment.CacheRead != nil ||
		segment.CacheCreate != nil
}

func hasIntRange(minVal, maxVal *int) bool {
	return minVal != nil && maxVal != nil
}

func textSegmentsOverlap(left, right AdvancedPriceRule) bool {
	if strings.TrimSpace(left.ServiceTier) != strings.TrimSpace(right.ServiceTier) {
		return false
	}
	if !boolPointerEqual(left.CacheRead, right.CacheRead) {
		return false
	}
	if !boolPointerEqual(left.CacheCreate, right.CacheCreate) {
		return false
	}
	if !intRangeOverlap(left.InputMin, left.InputMax, right.InputMin, right.InputMax) {
		return false
	}
	if !intRangeOverlap(left.OutputMin, left.OutputMax, right.OutputMin, right.OutputMax) {
		return false
	}
	return true
}

func intRangeOverlap(leftMin, leftMax, rightMin, rightMax *int) bool {
	if !hasIntRange(leftMin, leftMax) || !hasIntRange(rightMin, rightMax) {
		return true
	}
	return *leftMin <= *rightMax && *rightMin <= *leftMax
}

func boolPointerEqual(left, right *bool) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func validateMediaTaskRules(modelName string, segments []AdvancedPriceRule) error {
	for _, segment := range segments {
		if segment.UnitPrice == nil {
			return fmt.Errorf("model %s media task segment is missing unit_price", modelName)
		}
		if *segment.UnitPrice < 0 {
			return fmt.Errorf("model %s media task segment unit_price cannot be negative", modelName)
		}
		if segment.MinTokens != nil && *segment.MinTokens < 0 {
			return fmt.Errorf("model %s media task segment min_tokens cannot be negative", modelName)
		}
		if err := validateMediaRange(modelName, "output_duration", segment.OutputDurationMin, segment.OutputDurationMax); err != nil {
			return err
		}
		if err := validateMediaRange(modelName, "input_video_duration", segment.InputVideoDurationMin, segment.InputVideoDurationMax); err != nil {
			return err
		}
		if segment.DraftCoefficient != nil && *segment.DraftCoefficient < 0 {
			return fmt.Errorf("model %s media task segment draft_coefficient cannot be negative", modelName)
		}
	}
	return nil
}

func validateMediaRange(modelName, rangeName string, minVal, maxVal *int) error {
	if minVal == nil && maxVal == nil {
		return nil
	}
	if minVal == nil || maxVal == nil {
		return fmt.Errorf("model %s media task segment %s 区间 must include both min and max", modelName, rangeName)
	}
	if *minVal < 0 || *maxVal < 0 {
		return fmt.Errorf("model %s media task segment %s 区间 cannot be negative", modelName, rangeName)
	}
	if *maxVal < *minVal {
		return fmt.Errorf("model %s media task segment %s 区间 is invalid", modelName, rangeName)
	}
	return nil
}
