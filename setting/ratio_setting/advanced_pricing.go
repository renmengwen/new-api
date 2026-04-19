package ratio_setting

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

type BillingMode = types.BillingMode

const (
	BillingModePerToken   = types.BillingModePerToken
	BillingModePerRequest = types.BillingModePerRequest
	BillingModeAdvanced   = types.BillingModeAdvanced
)

type AdvancedRuleType = types.AdvancedRuleType

const (
	RuleTypeTextSegment = types.AdvancedRuleTypeTextSegment
	RuleTypeMediaTask   = types.AdvancedRuleTypeMediaTask
)

type AdvancedPricingConfig struct {
	ModelModes map[string]BillingMode            `json:"billing_mode"`
	ModelRules map[string]AdvancedPricingRuleSet `json:"rules"`
}

type AdvancedPricingRuleSet struct {
	RuleType     AdvancedRuleType    `json:"rule_type"`
	DisplayName  string              `json:"display_name,omitempty"`
	SegmentBasis string              `json:"segment_basis,omitempty"`
	BillingUnit  string              `json:"billing_unit,omitempty"`
	DefaultPrice *float64            `json:"default_price,omitempty"`
	TaskType     string              `json:"task_type,omitempty"`
	Note         string              `json:"note,omitempty"`
	Segments     []AdvancedPriceRule `json:"segments"`
}

type advancedPricingRuleSetPayload struct {
	DisplayName  string              `json:"display_name"`
	SegmentBasis string              `json:"segment_basis"`
	BillingUnit  string              `json:"billing_unit"`
	DefaultPrice any                 `json:"default_price"`
	TaskType     string              `json:"task_type"`
	RuleType     AdvancedRuleType    `json:"rule_type"`
	Segments     []AdvancedPriceRule `json:"segments"`
	SegmentsText string              `json:"segments_text"`
	Note         string              `json:"note"`
	UnitPrice    any                 `json:"unit_price"`
}

type AdvancedPriceRule struct {
	Priority *int `json:"priority,omitempty"`

	InputMin *int `json:"input_min,omitempty"`
	InputMax *int `json:"input_max,omitempty"`

	OutputMin *int `json:"output_min,omitempty"`
	OutputMax *int `json:"output_max,omitempty"`

	ServiceTier    string `json:"service_tier,omitempty"`
	InputModality  string `json:"input_modality,omitempty"`
	OutputModality string `json:"output_modality,omitempty"`
	BillingUnit    string `json:"billing_unit,omitempty"`
	CacheRead      *bool  `json:"cache_read,omitempty"`
	CacheCreate    *bool  `json:"cache_create,omitempty"`

	InputPrice        *float64 `json:"input_price,omitempty"`
	OutputPrice       *float64 `json:"output_price,omitempty"`
	CacheReadPrice    *float64 `json:"cache_read_price,omitempty"`
	CacheCreatePrice  *float64 `json:"cache_create_price,omitempty"`
	CacheStoragePrice *float64 `json:"cache_storage_price,omitempty"`

	ImageSizeTier    string `json:"image_size_tier,omitempty"`
	ToolUsageType    string `json:"tool_usage_type,omitempty"`
	ToolUsageCount   *int   `json:"tool_usage_count,omitempty"`
	FreeQuota        *int   `json:"free_quota,omitempty"`
	OverageThreshold *int   `json:"overage_threshold,omitempty"`

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

var legacyAdvancedTextShellSegmentPattern = regexp.MustCompile(`^(\d+)\s*-\s*(\d+)\s*:\s*(-?\d+(?:\.\d+)?)$`)

var advancedPricingModeMap = types.NewRWMap[string, BillingMode]()
var advancedPricingRulesMap = types.NewRWMap[string, AdvancedPricingRuleSet]()

type AdvancedPricingRuntimeContext struct {
	PromptTokens     int
	Meta             *types.TokenCountMeta
	Request          dto.Request
	RequestURLPath   string
	Task             *AdvancedPricingTaskContext
	InputModalities  []string
	OutputModalities []string
	ToolUsageType    string
	ToolUsageCount   int
}

type advancedTextRuntimeContext struct {
	inputTokens      int
	outputTokens     int
	serviceTier      string
	inputModalities  []string
	outputModalities []string
	imageSizeTier    string
	imageCount       *int
	toolUsageType    string
	toolUsageCount   int
	cacheRead        *bool
	cacheCreate      *bool
}

type AdvancedPricingTaskContext struct {
	TaskType           string
	RawAction          string
	InferenceMode      string
	Audio              *bool
	InputVideo         *bool
	Resolution         string
	AspectRatio        string
	OutputDuration     int
	InputVideoDuration int
	Draft              *bool
}

type advancedMediaRuntimeContext struct {
	taskType           string
	rawAction          string
	inferenceMode      string
	audio              *bool
	inputVideo         *bool
	resolution         string
	aspectRatio        string
	outputDuration     int
	inputVideoDuration int
	draft              *bool
}

func (ruleSet *AdvancedPricingRuleSet) UnmarshalJSON(data []byte) error {
	var payload advancedPricingRuleSetPayload
	if err := common.Unmarshal(data, &payload); err != nil {
		return err
	}

	defaultPrice, err := parseLegacyAdvancedPricingOptionalFloat("default_price", payload.DefaultPrice)
	if err != nil {
		return err
	}

	if len(payload.Segments) > 0 {
		ruleSet.RuleType = payload.RuleType
		ruleSet.DisplayName = payload.DisplayName
		ruleSet.SegmentBasis = payload.SegmentBasis
		ruleSet.BillingUnit = payload.BillingUnit
		ruleSet.DefaultPrice = defaultPrice
		ruleSet.TaskType = payload.TaskType
		ruleSet.Note = payload.Note
		ruleSet.Segments = payload.Segments
		normalizeAdvancedPricingRuleSetSegments(ruleSet)
		return nil
	}

	normalizedRuleSet, ok, err := normalizeLegacyAdvancedPricingRuleSet(payload, defaultPrice)
	if err != nil {
		return err
	}
	if ok {
		*ruleSet = normalizedRuleSet
		normalizeAdvancedPricingRuleSetSegments(ruleSet)
		return nil
	}

	ruleSet.RuleType = payload.RuleType
	ruleSet.DisplayName = payload.DisplayName
	ruleSet.SegmentBasis = payload.SegmentBasis
	ruleSet.BillingUnit = payload.BillingUnit
	ruleSet.DefaultPrice = defaultPrice
	ruleSet.TaskType = payload.TaskType
	ruleSet.Note = payload.Note
	ruleSet.Segments = payload.Segments
	normalizeAdvancedPricingRuleSetSegments(ruleSet)
	return nil
}

func GetExplicitBillingMode(modelName string) (BillingMode, bool) {
	modelName = FormatMatchingModelName(modelName)
	return getAdvancedPricingModeMapValue(modelName)
}

func GetLegacyBillingMode(modelName string) BillingMode {
	modelName = FormatMatchingModelName(modelName)
	if _, ok := GetModelPrice(modelName, false); ok {
		return BillingModePerRequest
	}
	return BillingModePerToken
}

func GetEffectiveBillingMode(modelName string) BillingMode {
	if mode, ok := GetExplicitBillingMode(modelName); ok {
		return mode
	}
	return GetLegacyBillingMode(modelName)
}

func ResolveAdvancedPriceData(modelName string, ctx AdvancedPricingRuntimeContext) (types.PriceData, bool, error) {
	modelName = FormatMatchingModelName(modelName)
	if GetEffectiveBillingMode(modelName) != BillingModeAdvanced {
		return types.PriceData{}, false, nil
	}

	ruleSet, ok := GetAdvancedPricingRuleSet(modelName)
	if !ok {
		return types.PriceData{}, false, nil
	}

	switch ruleSet.RuleType {
	case RuleTypeTextSegment:
		return resolveAdvancedTextPriceData(modelName, ctx, ruleSet)
	case RuleTypeMediaTask:
		return resolveAdvancedMediaTaskPriceData(ctx, ruleSet)
	default:
		return types.PriceData{}, false, fmt.Errorf("model %s has invalid advanced pricing rule type: %s", modelName, ruleSet.RuleType)
	}
}

func resolveAdvancedTextPriceData(modelName string, ctx AdvancedPricingRuntimeContext, ruleSet AdvancedPricingRuleSet) (types.PriceData, bool, error) {
	runtimeCtx := buildAdvancedTextRuntimeContext(ctx)

	segment, ok := findMatchedTextSegment(ruleSet.Segments, runtimeCtx)
	if !ok {
		return types.PriceData{}, false, nil
	}

	effectiveBillingUnit := resolveAdvancedPricingBillingUnit(ruleSet.BillingUnit)

	modelRatio := 0.0
	if segment.InputPrice != nil {
		modelRatio = *segment.InputPrice / 2
	}

	completionRatio := GetCompletionRatio(modelName)
	if segment.OutputPrice != nil {
		derivedRatio, err := deriveAdvancedRelativeRatio(segment.InputPrice, segment.OutputPrice)
		if err != nil {
			return types.PriceData{}, false, err
		}
		completionRatio = derivedRatio
	}

	cacheRatio, _ := GetCacheRatio(modelName)
	if segment.CacheReadPrice != nil {
		derivedRatio, err := deriveAdvancedRelativeRatio(segment.InputPrice, segment.CacheReadPrice)
		if err != nil {
			return types.PriceData{}, false, err
		}
		cacheRatio = derivedRatio
	}

	cacheCreationRatio, _ := GetCreateCacheRatio(modelName)
	if segment.CacheCreatePrice != nil {
		derivedRatio, err := deriveAdvancedRelativeRatio(segment.InputPrice, segment.CacheCreatePrice)
		if err != nil {
			return types.PriceData{}, false, err
		}
		cacheCreationRatio = derivedRatio
	}

	return types.PriceData{
		ModelRatio:             modelRatio,
		CompletionRatio:        completionRatio,
		CacheRatio:             cacheRatio,
		CacheCreationRatio:     cacheCreationRatio,
		BillingMode:            types.BillingModeAdvanced,
		AdvancedRuleType:       ruleSet.RuleType,
		AdvancedRuleSnapshot:   buildAdvancedRuleSnapshot(ruleSet.RuleType, effectiveBillingUnit, segment, runtimeCtx),
		AdvancedPricingContext: buildAdvancedPricingContextSnapshot(effectiveBillingUnit, segment, runtimeCtx),
	}, true, nil
}

func resolveAdvancedMediaTaskPriceData(ctx AdvancedPricingRuntimeContext, ruleSet AdvancedPricingRuleSet) (types.PriceData, bool, error) {
	runtimeCtx := buildAdvancedMediaRuntimeContext(ctx)
	if !matchAdvancedMediaTaskType(ruleSet.TaskType, runtimeCtx) {
		return types.PriceData{}, false, nil
	}
	segment, ok := findMatchedMediaTaskSegment(ruleSet.Segments, runtimeCtx, ctx.PromptTokens)
	if !ok {
		return types.PriceData{}, false, nil
	}
	if segment.UnitPrice == nil {
		return types.PriceData{}, false, fmt.Errorf("advanced media task segment is missing unit_price")
	}

	return types.PriceData{
		ModelPrice:             *segment.UnitPrice,
		BillingMode:            types.BillingModeAdvanced,
		AdvancedRuleType:       ruleSet.RuleType,
		AdvancedRuleSnapshot:   buildAdvancedMediaRuleSnapshot(ruleSet.RuleType, ruleSet.TaskType, resolveAdvancedPricingBillingUnit(ruleSet.BillingUnit), segment, runtimeCtx, ctx.PromptTokens),
		AdvancedPricingContext: buildAdvancedMediaPricingContextSnapshot(resolveAdvancedPricingBillingUnit(ruleSet.BillingUnit), segment),
		UsePrice:               true,
	}, true, nil
}

func buildAdvancedMediaRuntimeContext(ctx AdvancedPricingRuntimeContext) advancedMediaRuntimeContext {
	runtimeCtx := advancedMediaRuntimeContext{}
	if ctx.Task == nil {
		return runtimeCtx
	}

	runtimeCtx.taskType = normalizeAdvancedPricingComparableString(ctx.Task.TaskType)
	runtimeCtx.rawAction = strings.TrimSpace(ctx.Task.RawAction)
	runtimeCtx.inferenceMode = normalizeAdvancedPricingComparableString(ctx.Task.InferenceMode)
	runtimeCtx.audio = cloneAdvancedBoolPtr(ctx.Task.Audio)
	runtimeCtx.inputVideo = cloneAdvancedBoolPtr(ctx.Task.InputVideo)
	runtimeCtx.resolution = normalizeAdvancedPricingComparableString(ctx.Task.Resolution)
	runtimeCtx.aspectRatio = normalizeAdvancedPricingComparableString(ctx.Task.AspectRatio)
	runtimeCtx.outputDuration = ctx.Task.OutputDuration
	runtimeCtx.inputVideoDuration = ctx.Task.InputVideoDuration
	runtimeCtx.draft = cloneAdvancedBoolPtr(ctx.Task.Draft)
	return runtimeCtx
}

func buildAdvancedTextRuntimeContext(ctx AdvancedPricingRuntimeContext) advancedTextRuntimeContext {
	return advancedTextRuntimeContext{
		inputTokens:      ctx.PromptTokens,
		outputTokens:     getRuntimeOutputTokens(ctx.Meta),
		serviceTier:      extractAdvancedPricingServiceTier(ctx.Request),
		inputModalities:  collectAdvancedTextInputModalities(ctx),
		outputModalities: collectAdvancedTextOutputModalities(ctx),
		imageSizeTier:    extractAdvancedTextImageSizeTier(ctx),
		imageCount:       extractAdvancedTextImageCount(ctx),
		toolUsageType:    NormalizeAdvancedPricingTextToolUsageType(ctx.ToolUsageType),
		toolUsageCount:   ctx.ToolUsageCount,
	}
}

func getRuntimeOutputTokens(meta *types.TokenCountMeta) int {
	if meta == nil {
		return 0
	}
	return meta.MaxTokens
}

func extractAdvancedPricingServiceTier(request dto.Request) string {
	switch req := request.(type) {
	case *dto.OpenAIResponsesRequest:
		return normalizeAdvancedPricingServiceTier(req.ServiceTier)
	case *dto.ClaudeRequest:
		return normalizeAdvancedPricingServiceTier(req.ServiceTier)
	case *dto.GeneralOpenAIRequest:
		return normalizeAdvancedPricingServiceTier(normalizeAdvancedPricingRawString(req.ServiceTier))
	default:
		return ""
	}
}

func normalizeAdvancedPricingServiceTier(value string) string {
	return normalizeAdvancedPricingComparableString(value)
}

func NormalizeAdvancedPricingTextToolUsageType(value string) string {
	normalized := normalizeAdvancedPricingComparableString(value)
	switch normalized {
	case "web_search", "google_search", "grounding":
		return "google_search"
	default:
		return normalized
	}
}

func resolveAdvancedPricingBillingUnit(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return types.AdvancedBillingUnitPerMillionTokens
	}
	return trimmed
}

func normalizeAdvancedPricingComparableString(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeAdvancedPricingRawString(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var value string
	if err := common.Unmarshal(data, &value); err == nil {
		return strings.TrimSpace(value)
	}
	return strings.Trim(strings.TrimSpace(string(data)), `"`)
}

func collectAdvancedTextInputModalities(ctx AdvancedPricingRuntimeContext) []string {
	modalities := make([]string, 0, 6+len(ctx.InputModalities))
	modalities = append(modalities, ctx.InputModalities...)
	modalities = append(modalities, extractAdvancedPricingRequestInputModalities(ctx.Request)...)
	if ctx.Meta != nil {
		for _, file := range ctx.Meta.Files {
			if file == nil {
				continue
			}
			switch file.FileType {
			case types.FileTypeImage:
				modalities = append(modalities, "image")
			case types.FileTypeAudio:
				modalities = append(modalities, "audio")
			case types.FileTypeVideo:
				modalities = append(modalities, "video")
			case types.FileTypeFile:
				modalities = append(modalities, "file")
			}
		}
	}

	switch req := ctx.Request.(type) {
	case *dto.OpenAIResponsesRequest:
		modalities = append(modalities, extractResponsesRequestInputModalities(req.Input)...)
	}

	if len(modalities) == 0 {
		modalities = append(modalities, "text")
	}
	return normalizeAdvancedPricingModalities(modalities)
}

func extractAdvancedPricingRequestInputModalities(request dto.Request) []string {
	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		return extractGeneralOpenAIRequestInputModalities(req)
	case *dto.OpenAIResponsesRequest:
		return extractResponsesRequestDeclaredInputModalities(req)
	default:
		return nil
	}
}

func extractGeneralOpenAIRequestInputModalities(req *dto.GeneralOpenAIRequest) []string {
	if req == nil {
		return nil
	}

	modalities := make([]string, 0, 6)
	if req.Prompt != nil || req.Input != nil {
		modalities = append(modalities, "text")
	}
	for _, message := range req.Messages {
		if content := strings.TrimSpace(message.StringContent()); content != "" {
			modalities = append(modalities, "text")
		}
		for _, content := range message.ParseContent() {
			switch content.Type {
			case dto.ContentTypeText:
				if strings.TrimSpace(content.Text) != "" {
					modalities = append(modalities, "text")
				}
			case dto.ContentTypeImageURL:
				modalities = append(modalities, "image")
			case dto.ContentTypeInputAudio:
				modalities = append(modalities, "audio")
			case dto.ContentTypeFile:
				modalities = append(modalities, "file")
			case dto.ContentTypeVideoUrl:
				modalities = append(modalities, "video")
			}
		}
	}
	return modalities
}

func extractResponsesRequestDeclaredInputModalities(req *dto.OpenAIResponsesRequest) []string {
	if req == nil {
		return nil
	}

	modalities := make([]string, 0, 6)
	for _, input := range req.ParseInput() {
		switch input.Type {
		case "input_text":
			if strings.TrimSpace(input.Text) != "" {
				modalities = append(modalities, "text")
			}
		case "input_image":
			modalities = append(modalities, "image")
		case "input_file":
			modalities = append(modalities, "file")
		case "input_video":
			modalities = append(modalities, "video")
		}
	}
	if len(req.Instructions) > 0 || len(req.Prompt) > 0 {
		modalities = append(modalities, "text")
	}
	return modalities
}

func collectAdvancedTextOutputModalities(ctx AdvancedPricingRuntimeContext) []string {
	modalities := make([]string, 0, 2+len(ctx.OutputModalities))
	modalities = append(modalities, ctx.OutputModalities...)
	switch req := ctx.Request.(type) {
	case *dto.GeneralOpenAIRequest:
		modalities = append(modalities, extractAdvancedPricingRawStringSlice(req.Modalities)...)
		modalities = append(modalities, extractAdvancedPricingGoogleResponseModalities(req.ExtraBody)...)
		if len(req.Audio) > 0 {
			modalities = append(modalities, "audio")
		}
		if isAdvancedPricingImageGenerationPath(ctx.RequestURLPath) {
			modalities = append(modalities, "image")
		}
	case *dto.GeminiChatRequest:
		modalities = append(modalities, req.GenerationConfig.ResponseModalities...)
	}

	if len(modalities) == 0 {
		modalities = append(modalities, "text")
	}
	return normalizeAdvancedPricingModalities(modalities)
}

func extractAdvancedTextImageSizeTier(ctx AdvancedPricingRuntimeContext) string {
	switch req := ctx.Request.(type) {
	case *dto.GeneralOpenAIRequest:
		if imageSizeTier := extractAdvancedPricingGoogleImageSizeTier(req.ExtraBody); imageSizeTier != "" {
			return imageSizeTier
		}
		if imageSizeTier := deriveAdvancedPricingImageSizeTierFromSize(req.Size); imageSizeTier != "" {
			return imageSizeTier
		}
	case *dto.GeminiChatRequest:
		if imageSizeTier := extractGeminiChatRequestImageSizeTier(req); imageSizeTier != "" {
			return imageSizeTier
		}
	}
	return ""
}

func extractAdvancedTextImageCount(ctx AdvancedPricingRuntimeContext) *int {
	if !isAdvancedPricingImageGenerationPath(ctx.RequestURLPath) {
		return nil
	}
	switch req := ctx.Request.(type) {
	case *dto.GeneralOpenAIRequest:
		if req.N != nil && *req.N > 0 {
			return cloneAdvancedIntPtr(req.N)
		}
		defaultCount := 1
		return &defaultCount
	default:
		return nil
	}
}

type advancedPricingGoogleExtraBody struct {
	Google struct {
		GenerationConfig struct {
			ResponseModalities []string `json:"response_modalities,omitempty"`
		} `json:"generation_config,omitempty"`
		ImageConfig struct {
			ImageSize      string `json:"image_size,omitempty"`
			ImageSizeCamel string `json:"imageSize,omitempty"`
		} `json:"image_config,omitempty"`
	} `json:"google,omitempty"`
}

func extractAdvancedPricingGoogleResponseModalities(data []byte) []string {
	extraBody, ok := parseAdvancedPricingGoogleExtraBody(data)
	if !ok {
		return nil
	}
	return extraBody.Google.GenerationConfig.ResponseModalities
}

func extractAdvancedPricingGoogleImageSizeTier(data []byte) string {
	extraBody, ok := parseAdvancedPricingGoogleExtraBody(data)
	if !ok {
		return ""
	}
	return normalizeAdvancedPricingImageSizeTier(firstNonEmptyString(extraBody.Google.ImageConfig.ImageSize, extraBody.Google.ImageConfig.ImageSizeCamel))
}

func extractGeminiChatRequestImageSizeTier(req *dto.GeminiChatRequest) string {
	if req == nil || len(req.GenerationConfig.ImageConfig) == 0 {
		return ""
	}

	var imageConfig struct {
		ImageSize      string `json:"image_size,omitempty"`
		ImageSizeCamel string `json:"imageSize,omitempty"`
	}
	if err := common.Unmarshal(req.GenerationConfig.ImageConfig, &imageConfig); err != nil {
		return ""
	}
	return normalizeAdvancedPricingImageSizeTier(firstNonEmptyString(imageConfig.ImageSize, imageConfig.ImageSizeCamel))
}

func parseAdvancedPricingGoogleExtraBody(data []byte) (advancedPricingGoogleExtraBody, bool) {
	if len(data) == 0 || common.GetJsonType(data) != "object" {
		return advancedPricingGoogleExtraBody{}, false
	}
	var extraBody advancedPricingGoogleExtraBody
	if err := common.Unmarshal(data, &extraBody); err != nil {
		return advancedPricingGoogleExtraBody{}, false
	}
	return extraBody, true
}

func deriveAdvancedPricingImageSizeTierFromSize(size string) string {
	normalizedSize := strings.ToLower(strings.TrimSpace(size))
	switch normalizedSize {
	case "1k", "2k", "4k":
		return normalizedSize
	}

	parts := strings.Split(normalizedSize, "x")
	if len(parts) != 2 {
		return ""
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return ""
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return ""
	}
	longestEdge := width
	if height > longestEdge {
		longestEdge = height
	}
	switch {
	case longestEdge >= 4096:
		return "4k"
	case longestEdge >= 2048:
		return "2k"
	case longestEdge >= 1024:
		return "1k"
	default:
		return ""
	}
}

func normalizeAdvancedPricingImageSizeTier(value string) string {
	return normalizeAdvancedPricingComparableString(value)
}

func isAdvancedPricingImageGenerationPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.HasPrefix(trimmed, "/v1/images/generations")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeAdvancedPricingModalities(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizeAdvancedPricingComparableString(value)
		if normalized == "" {
			continue
		}
		unique[normalized] = struct{}{}
	}
	if len(unique) == 0 {
		return nil
	}

	normalizedValues := make([]string, 0, len(unique))
	for value := range unique {
		normalizedValues = append(normalizedValues, value)
	}
	sort.Strings(normalizedValues)
	return normalizedValues
}

func extractAdvancedPricingRawStringSlice(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	switch common.GetJsonType(data) {
	case "string":
		value := normalizeAdvancedPricingRawString(data)
		if value == "" {
			return nil
		}
		return []string{value}
	case "array":
		var values []string
		if err := common.Unmarshal(data, &values); err == nil {
			return values
		}
	}
	return nil
}

func extractResponsesRequestInputModalities(data []byte) []string {
	if len(data) == 0 || common.GetJsonType(data) != "array" {
		return nil
	}

	var inputs []map[string]any
	if err := common.Unmarshal(data, &inputs); err != nil {
		return nil
	}

	modalities := make([]string, 0, 4)
	for _, input := range inputs {
		content, ok := input["content"].([]any)
		if !ok {
			continue
		}
		for _, contentItem := range content {
			item, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			modality := normalizeAdvancedPricingComparableString(common.Interface2String(item["type"]))
			switch modality {
			case "input_audio":
				modalities = append(modalities, "audio")
			case "input_image":
				modalities = append(modalities, "image")
			case "input_file":
				modalities = append(modalities, "file")
			case "input_video":
				modalities = append(modalities, "video")
			}
		}
	}
	return modalities
}

func normalizeLegacyAdvancedPricingRuleSet(payload advancedPricingRuleSetPayload, defaultPrice *float64) (AdvancedPricingRuleSet, bool, error) {
	ruleType := payload.RuleType
	if ruleType == "" {
		switch {
		case strings.TrimSpace(payload.SegmentsText) != "":
			ruleType = RuleTypeTextSegment
		case hasLegacyAdvancedPricingUnitPrice(payload.UnitPrice):
			ruleType = RuleTypeMediaTask
		default:
			return AdvancedPricingRuleSet{}, false, nil
		}
	}

	switch ruleType {
	case RuleTypeTextSegment:
		return normalizeLegacyAdvancedTextRuleSet(payload, defaultPrice)
	case RuleTypeMediaTask:
		return normalizeLegacyAdvancedMediaRuleSet(payload)
	default:
		return AdvancedPricingRuleSet{}, false, nil
	}
}

func normalizeLegacyAdvancedTextRuleSet(payload advancedPricingRuleSetPayload, defaultPrice *float64) (AdvancedPricingRuleSet, bool, error) {
	rawLines := strings.Split(payload.SegmentsText, "\n")
	segments := make([]AdvancedPriceRule, 0, len(rawLines))
	for index, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		segment, err := parseLegacyAdvancedTextShellSegment(trimmed, index)
		if err != nil {
			return AdvancedPricingRuleSet{}, false, err
		}
		segments = append(segments, segment)
	}
	if len(segments) == 0 {
		return AdvancedPricingRuleSet{}, false, nil
	}

	return AdvancedPricingRuleSet{
		RuleType:     RuleTypeTextSegment,
		DisplayName:  payload.DisplayName,
		SegmentBasis: payload.SegmentBasis,
		BillingUnit:  payload.BillingUnit,
		DefaultPrice: defaultPrice,
		Note:         payload.Note,
		Segments:     segments,
	}, true, nil
}

func parseLegacyAdvancedTextShellSegment(line string, index int) (AdvancedPriceRule, error) {
	matches := legacyAdvancedTextShellSegmentPattern.FindStringSubmatch(line)
	if matches == nil {
		return AdvancedPriceRule{}, fmt.Errorf("invalid advanced pricing segment on line %d", index+1)
	}

	start, err := strconv.Atoi(matches[1])
	if err != nil {
		return AdvancedPriceRule{}, fmt.Errorf("invalid advanced pricing segment on line %d", index+1)
	}
	end, err := strconv.Atoi(matches[2])
	if err != nil {
		return AdvancedPriceRule{}, fmt.Errorf("invalid advanced pricing segment on line %d", index+1)
	}
	price, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return AdvancedPriceRule{}, fmt.Errorf("invalid advanced pricing segment on line %d", index+1)
	}

	priority := (index + 1) * 10
	return AdvancedPriceRule{
		Priority:   &priority,
		InputMin:   &start,
		InputMax:   &end,
		InputPrice: &price,
	}, nil
}

func normalizeLegacyAdvancedMediaRuleSet(payload advancedPricingRuleSetPayload) (AdvancedPricingRuleSet, bool, error) {
	unitPrice, ok, err := parseLegacyAdvancedPricingUnitPrice(payload.UnitPrice)
	if err != nil {
		return AdvancedPricingRuleSet{}, false, err
	}
	if !ok {
		return AdvancedPricingRuleSet{}, false, nil
	}

	priority := 10
	segment := AdvancedPriceRule{
		Priority:  &priority,
		UnitPrice: &unitPrice,
	}
	if remark := strings.TrimSpace(payload.Note); remark != "" {
		segment.Remark = remark
	}

	return AdvancedPricingRuleSet{
		RuleType:    RuleTypeMediaTask,
		DisplayName: payload.DisplayName,
		BillingUnit: payload.BillingUnit,
		TaskType:    payload.TaskType,
		Note:        payload.Note,
		Segments:    []AdvancedPriceRule{segment},
	}, true, nil
}

func hasLegacyAdvancedPricingUnitPrice(value any) bool {
	switch data := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(data) != ""
	default:
		return true
	}
}

func parseLegacyAdvancedPricingUnitPrice(value any) (float64, bool, error) {
	switch data := value.(type) {
	case nil:
		return 0, false, nil
	case float64:
		return data, true, nil
	case string:
		trimmed := strings.TrimSpace(data)
		if trimmed == "" {
			return 0, false, nil
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, false, fmt.Errorf("invalid advanced media task unit_price: %s", trimmed)
		}
		return parsed, true, nil
	default:
		return 0, false, fmt.Errorf("invalid advanced media task unit_price type")
	}
}

func parseLegacyAdvancedPricingOptionalFloat(fieldName string, value any) (*float64, error) {
	switch data := value.(type) {
	case nil:
		return nil, nil
	case float64:
		return &data, nil
	case string:
		trimmed := strings.TrimSpace(data)
		if trimmed == "" {
			return nil, nil
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid advanced pricing %s: %s", fieldName, trimmed)
		}
		return &parsed, nil
	default:
		return nil, fmt.Errorf("invalid advanced pricing %s type", fieldName)
	}
}

func findMatchedTextSegment(segments []AdvancedPriceRule, runtimeCtx advancedTextRuntimeContext) (AdvancedPriceRule, bool) {
	sortedSegments := append([]AdvancedPriceRule(nil), segments...)
	sort.Slice(sortedSegments, func(i, j int) bool {
		return *sortedSegments[i].Priority < *sortedSegments[j].Priority
	})

	var defaultSegment *AdvancedPriceRule
	for _, segment := range sortedSegments {
		if !hasTextCondition(segment) {
			if defaultSegment == nil {
				segmentCopy := segment
				defaultSegment = &segmentCopy
			}
			continue
		}
		if matchAdvancedTextSegment(segment, runtimeCtx) {
			return segment, true
		}
	}
	if defaultSegment != nil {
		return *defaultSegment, true
	}
	return AdvancedPriceRule{}, false
}

func findMatchedMediaTaskSegment(segments []AdvancedPriceRule, runtimeCtx advancedMediaRuntimeContext, promptTokens int) (AdvancedPriceRule, bool) {
	sortedSegments := append([]AdvancedPriceRule(nil), segments...)
	sort.Slice(sortedSegments, func(i, j int) bool {
		return *sortedSegments[i].Priority < *sortedSegments[j].Priority
	})

	for _, segment := range sortedSegments {
		if matchAdvancedMediaTaskSegment(segment, runtimeCtx, promptTokens) {
			return segment, true
		}
	}
	return AdvancedPriceRule{}, false
}

func matchAdvancedTextSegment(segment AdvancedPriceRule, runtimeCtx advancedTextRuntimeContext) bool {
	if hasIntRange(segment.InputMin, segment.InputMax) && !isAdvancedTokenCountInRange(runtimeCtx.inputTokens, segment.InputMin, segment.InputMax) {
		return false
	}
	if hasIntRange(segment.OutputMin, segment.OutputMax) && !isAdvancedTokenCountInRange(runtimeCtx.outputTokens, segment.OutputMin, segment.OutputMax) {
		return false
	}
	if serviceTier := normalizeAdvancedPricingServiceTier(segment.ServiceTier); serviceTier != "" && serviceTier != runtimeCtx.serviceTier {
		return false
	}
	if inputModality := normalizeAdvancedPricingComparableString(segment.InputModality); inputModality != "" && !advancedPricingModalityMatch(runtimeCtx.inputModalities, inputModality) {
		return false
	}
	if outputModality := normalizeAdvancedPricingComparableString(segment.OutputModality); outputModality != "" && !advancedPricingModalityMatch(runtimeCtx.outputModalities, outputModality) {
		return false
	}
	if imageSizeTier := normalizeAdvancedPricingImageSizeTier(segment.ImageSizeTier); imageSizeTier != "" && imageSizeTier != runtimeCtx.imageSizeTier {
		return false
	}
	if toolUsageType := NormalizeAdvancedPricingTextToolUsageType(segment.ToolUsageType); toolUsageType != "" && toolUsageType != runtimeCtx.toolUsageType {
		return false
	}
	if segment.ToolUsageCount != nil && runtimeCtx.toolUsageCount < *segment.ToolUsageCount {
		return false
	}
	if segment.CacheRead != nil {
		if runtimeCtx.cacheRead == nil || *segment.CacheRead != *runtimeCtx.cacheRead {
			return false
		}
	}
	if segment.CacheCreate != nil {
		if runtimeCtx.cacheCreate == nil || *segment.CacheCreate != *runtimeCtx.cacheCreate {
			return false
		}
	}
	return true
}

func advancedPricingModalityMatch(runtimeModalities []string, ruleModality string) bool {
	for _, modality := range runtimeModalities {
		if modality == ruleModality {
			return true
		}
	}
	return false
}

func matchAdvancedMediaTaskType(ruleTaskType string, runtimeCtx advancedMediaRuntimeContext) bool {
	taskType := normalizeAdvancedPricingComparableString(ruleTaskType)
	if taskType == "" {
		return true
	}
	if taskType == runtimeCtx.taskType {
		return true
	}
	return taskType == normalizeAdvancedPricingComparableString(runtimeCtx.rawAction)
}

func matchAdvancedMediaTaskSegment(segment AdvancedPriceRule, runtimeCtx advancedMediaRuntimeContext, promptTokens int) bool {
	if inferenceMode := normalizeAdvancedPricingComparableString(segment.InferenceMode); inferenceMode != "" && inferenceMode != runtimeCtx.inferenceMode {
		return false
	}
	if segment.Audio != nil && !boolPointerEqual(segment.Audio, runtimeCtx.audio) {
		return false
	}
	if segment.InputVideo != nil && !boolPointerEqual(segment.InputVideo, runtimeCtx.inputVideo) {
		return false
	}
	if resolution := normalizeAdvancedPricingComparableString(segment.Resolution); resolution != "" && resolution != runtimeCtx.resolution {
		return false
	}
	if aspectRatio := normalizeAdvancedPricingComparableString(segment.AspectRatio); aspectRatio != "" && aspectRatio != runtimeCtx.aspectRatio {
		return false
	}
	if hasIntRange(segment.OutputDurationMin, segment.OutputDurationMax) && !isAdvancedTokenCountInRange(runtimeCtx.outputDuration, segment.OutputDurationMin, segment.OutputDurationMax) {
		return false
	}
	if hasIntRange(segment.InputVideoDurationMin, segment.InputVideoDurationMax) && !isAdvancedTokenCountInRange(runtimeCtx.inputVideoDuration, segment.InputVideoDurationMin, segment.InputVideoDurationMax) {
		return false
	}
	if segment.Draft != nil && !boolPointerEqual(segment.Draft, runtimeCtx.draft) {
		return false
	}
	return true
}

func isAdvancedTokenCountInRange(value int, minVal, maxVal *int) bool {
	if minVal != nil && value < *minVal {
		return false
	}
	if maxVal != nil && value > *maxVal {
		return false
	}
	return true
}

func deriveAdvancedRelativeRatio(inputPrice *float64, targetPrice *float64) (float64, error) {
	if targetPrice == nil {
		return 0, nil
	}
	if inputPrice == nil {
		return 0, fmt.Errorf("advanced pricing input price is required to derive relative ratio")
	}
	if *inputPrice == 0 {
		if *targetPrice == 0 {
			return 0, nil
		}
		return 0, fmt.Errorf("advanced pricing input price cannot be zero when deriving relative ratio")
	}
	return *targetPrice / *inputPrice, nil
}

func buildAdvancedRuleSnapshot(ruleType AdvancedRuleType, billingUnit string, segment AdvancedPriceRule, runtimeCtx advancedTextRuntimeContext) *types.AdvancedRuleSnapshot {
	return &types.AdvancedRuleSnapshot{
		RuleType:       ruleType,
		MatchSummary:   buildAdvancedMatchSummary(segment, runtimeCtx),
		ConditionTags:  buildAdvancedConditionTags(segment),
		Priority:       cloneAdvancedIntPtr(segment.Priority),
		BillingUnit:    billingUnit,
		ServiceTier:    normalizeAdvancedPricingServiceTier(segment.ServiceTier),
		InputModality:  normalizeAdvancedPricingComparableString(segment.InputModality),
		OutputModality: normalizeAdvancedPricingComparableString(segment.OutputModality),
		ImageSizeTier:  normalizeAdvancedPricingComparableString(segment.ImageSizeTier),
		ToolUsageType:  normalizeAdvancedPricingComparableString(segment.ToolUsageType),
		CacheRead:      cloneAdvancedBoolPtr(segment.CacheRead),
		CacheCreate:    cloneAdvancedBoolPtr(segment.CacheCreate),
		PriceSnapshot: types.AdvancedRulePriceSnapshot{
			InputPrice:        cloneAdvancedFloatPtr(segment.InputPrice),
			OutputPrice:       cloneAdvancedFloatPtr(segment.OutputPrice),
			CacheReadPrice:    cloneAdvancedFloatPtr(segment.CacheReadPrice),
			CacheCreatePrice:  cloneAdvancedFloatPtr(segment.CacheCreatePrice),
			CacheStoragePrice: cloneAdvancedFloatPtr(segment.CacheStoragePrice),
		},
		ThresholdSnapshot: types.AdvancedRuleThresholdSnapshot{
			InputMin:         cloneAdvancedIntPtr(segment.InputMin),
			InputMax:         cloneAdvancedIntPtr(segment.InputMax),
			OutputMin:        cloneAdvancedIntPtr(segment.OutputMin),
			OutputMax:        cloneAdvancedIntPtr(segment.OutputMax),
			ToolUsageCount:   cloneAdvancedIntPtr(segment.ToolUsageCount),
			FreeQuota:        cloneAdvancedIntPtr(segment.FreeQuota),
			OverageThreshold: cloneAdvancedIntPtr(segment.OverageThreshold),
		},
	}
}

func buildAdvancedMediaRuleSnapshot(ruleType AdvancedRuleType, taskType string, billingUnit string, segment AdvancedPriceRule, runtimeCtx advancedMediaRuntimeContext, promptTokens int) *types.AdvancedRuleSnapshot {
	return &types.AdvancedRuleSnapshot{
		RuleType:      ruleType,
		MatchSummary:  buildAdvancedMediaMatchSummary(segment, runtimeCtx, promptTokens),
		ConditionTags: buildAdvancedMediaConditionTags(segment),
		Priority:      cloneAdvancedIntPtr(segment.Priority),
		TaskType:      strings.TrimSpace(taskType),
		BillingUnit:   billingUnit,
		ImageSizeTier: normalizeAdvancedPricingComparableString(segment.ImageSizeTier),
		ToolUsageType: normalizeAdvancedPricingComparableString(segment.ToolUsageType),
		ThresholdSnapshot: types.AdvancedRuleThresholdSnapshot{
			MinTokens:        cloneAdvancedIntPtr(segment.MinTokens),
			ToolUsageCount:   cloneAdvancedIntPtr(segment.ToolUsageCount),
			FreeQuota:        cloneAdvancedIntPtr(segment.FreeQuota),
			OverageThreshold: cloneAdvancedIntPtr(segment.OverageThreshold),
		},
	}
}

func buildAdvancedPricingContextSnapshot(billingUnit string, segment AdvancedPriceRule, runtimeCtx advancedTextRuntimeContext) *types.AdvancedPricingContextSnapshot {
	var toolUsageCount *int
	if runtimeCtx.toolUsageCount > 0 {
		toolUsageCount = cloneAdvancedIntPtr(&runtimeCtx.toolUsageCount)
	}
	return &types.AdvancedPricingContextSnapshot{
		BillingUnit:      billingUnit,
		InputModalities:  cloneAdvancedStringSlice(runtimeCtx.inputModalities),
		OutputModalities: cloneAdvancedStringSlice(runtimeCtx.outputModalities),
		ImageSizeTier:    runtimeCtx.imageSizeTier,
		ImageCount:       cloneAdvancedIntPtr(runtimeCtx.imageCount),
		ToolUsageType:    runtimeCtx.toolUsageType,
		ToolUsageCount:   toolUsageCount,
		FreeQuota:        cloneAdvancedIntPtr(segment.FreeQuota),
		OverageThreshold: cloneAdvancedIntPtr(segment.OverageThreshold),
	}
}

func buildAdvancedMediaPricingContextSnapshot(billingUnit string, segment AdvancedPriceRule) *types.AdvancedPricingContextSnapshot {
	return &types.AdvancedPricingContextSnapshot{
		BillingUnit:      billingUnit,
		ImageSizeTier:    normalizeAdvancedPricingComparableString(segment.ImageSizeTier),
		ToolUsageType:    normalizeAdvancedPricingComparableString(segment.ToolUsageType),
		ToolUsageCount:   cloneAdvancedIntPtr(segment.ToolUsageCount),
		FreeQuota:        cloneAdvancedIntPtr(segment.FreeQuota),
		OverageThreshold: cloneAdvancedIntPtr(segment.OverageThreshold),
	}
}

func buildAdvancedMatchSummary(segment AdvancedPriceRule, runtimeCtx advancedTextRuntimeContext) string {
	parts := []string{
		fmt.Sprintf("priority=%d", valueFromAdvancedIntPtr(segment.Priority)),
		fmt.Sprintf("input_tokens=%d", runtimeCtx.inputTokens),
		fmt.Sprintf("output_tokens=%d", runtimeCtx.outputTokens),
	}
	if runtimeCtx.serviceTier != "" {
		parts = append(parts, fmt.Sprintf("service_tier=%s", runtimeCtx.serviceTier))
	}
	if len(runtimeCtx.inputModalities) > 0 {
		parts = append(parts, fmt.Sprintf("input_modalities=%s", strings.Join(runtimeCtx.inputModalities, ",")))
	}
	if len(runtimeCtx.outputModalities) > 0 {
		parts = append(parts, fmt.Sprintf("output_modalities=%s", strings.Join(runtimeCtx.outputModalities, ",")))
	}
	if runtimeCtx.imageSizeTier != "" {
		parts = append(parts, fmt.Sprintf("image_size_tier=%s", runtimeCtx.imageSizeTier))
	}
	if runtimeCtx.toolUsageType != "" {
		parts = append(parts, fmt.Sprintf("tool_usage_type=%s", runtimeCtx.toolUsageType))
	}
	if runtimeCtx.toolUsageCount > 0 {
		parts = append(parts, fmt.Sprintf("tool_usage_count=%d", runtimeCtx.toolUsageCount))
	}
	return strings.Join(parts, ", ")
}

func buildAdvancedMediaMatchSummary(segment AdvancedPriceRule, runtimeCtx advancedMediaRuntimeContext, promptTokens int) string {
	parts := []string{
		fmt.Sprintf("priority=%d", valueFromAdvancedIntPtr(segment.Priority)),
		fmt.Sprintf("prompt_tokens=%d", promptTokens),
		fmt.Sprintf("unit_price=%g", valueFromAdvancedFloatPtr(segment.UnitPrice)),
	}
	if runtimeCtx.taskType != "" {
		parts = append(parts, fmt.Sprintf("task_type=%s", runtimeCtx.taskType))
	}
	if runtimeCtx.rawAction != "" {
		parts = append(parts, fmt.Sprintf("raw_action=%s", runtimeCtx.rawAction))
	}
	if runtimeCtx.inferenceMode != "" {
		parts = append(parts, fmt.Sprintf("inference_mode=%s", runtimeCtx.inferenceMode))
	}
	if runtimeCtx.resolution != "" {
		parts = append(parts, fmt.Sprintf("resolution=%s", runtimeCtx.resolution))
	}
	if runtimeCtx.aspectRatio != "" {
		parts = append(parts, fmt.Sprintf("aspect_ratio=%s", runtimeCtx.aspectRatio))
	}
	if runtimeCtx.outputDuration > 0 {
		parts = append(parts, fmt.Sprintf("output_duration=%d", runtimeCtx.outputDuration))
	}
	if runtimeCtx.inputVideoDuration > 0 {
		parts = append(parts, fmt.Sprintf("input_video_duration=%d", runtimeCtx.inputVideoDuration))
	}
	if runtimeCtx.audio != nil {
		parts = append(parts, fmt.Sprintf("audio=%t", *runtimeCtx.audio))
	}
	if runtimeCtx.inputVideo != nil {
		parts = append(parts, fmt.Sprintf("input_video=%t", *runtimeCtx.inputVideo))
	}
	if runtimeCtx.draft != nil {
		parts = append(parts, fmt.Sprintf("draft=%t", *runtimeCtx.draft))
	}
	return strings.Join(parts, ", ")
}

func buildAdvancedConditionTags(segment AdvancedPriceRule) []string {
	tags := make([]string, 0, 10)
	if hasIntRange(segment.InputMin, segment.InputMax) {
		tags = append(tags, formatAdvancedRangeTag("input", segment.InputMin, segment.InputMax))
	}
	if hasIntRange(segment.OutputMin, segment.OutputMax) {
		tags = append(tags, formatAdvancedRangeTag("output", segment.OutputMin, segment.OutputMax))
	}
	if serviceTier := normalizeAdvancedPricingServiceTier(segment.ServiceTier); serviceTier != "" {
		tags = append(tags, fmt.Sprintf("service_tier:%s", serviceTier))
	}
	if inputModality := normalizeAdvancedPricingComparableString(segment.InputModality); inputModality != "" {
		tags = append(tags, fmt.Sprintf("input_modality:%s", inputModality))
	}
	if outputModality := normalizeAdvancedPricingComparableString(segment.OutputModality); outputModality != "" {
		tags = append(tags, fmt.Sprintf("output_modality:%s", outputModality))
	}
	if imageSizeTier := normalizeAdvancedPricingComparableString(segment.ImageSizeTier); imageSizeTier != "" {
		tags = append(tags, fmt.Sprintf("image_size_tier:%s", imageSizeTier))
	}
	if toolUsageType := normalizeAdvancedPricingComparableString(segment.ToolUsageType); toolUsageType != "" {
		tags = append(tags, fmt.Sprintf("tool_usage_type:%s", toolUsageType))
	}
	if segment.CacheRead != nil {
		tags = append(tags, fmt.Sprintf("cache_read:%t", *segment.CacheRead))
	}
	if segment.CacheCreate != nil {
		tags = append(tags, fmt.Sprintf("cache_create:%t", *segment.CacheCreate))
	}
	return tags
}

func formatAdvancedRangeTag(label string, minVal, maxVal *int) string {
	minText := "-inf"
	if minVal != nil {
		minText = strconv.Itoa(*minVal)
	}
	maxText := "+inf"
	if maxVal != nil {
		maxText = strconv.Itoa(*maxVal)
	}
	return fmt.Sprintf("%s:%s-%s", label, minText, maxText)
}

func buildAdvancedMediaConditionTags(segment AdvancedPriceRule) []string {
	tags := make([]string, 0, 10)
	if inferenceMode := normalizeAdvancedPricingComparableString(segment.InferenceMode); inferenceMode != "" {
		tags = append(tags, fmt.Sprintf("inference_mode:%s", inferenceMode))
	}
	if segment.Audio != nil {
		tags = append(tags, fmt.Sprintf("audio:%t", *segment.Audio))
	}
	if segment.InputVideo != nil {
		tags = append(tags, fmt.Sprintf("input_video:%t", *segment.InputVideo))
	}
	if resolution := normalizeAdvancedPricingComparableString(segment.Resolution); resolution != "" {
		tags = append(tags, fmt.Sprintf("resolution:%s", resolution))
	}
	if aspectRatio := normalizeAdvancedPricingComparableString(segment.AspectRatio); aspectRatio != "" {
		tags = append(tags, fmt.Sprintf("aspect_ratio:%s", aspectRatio))
	}
	if hasIntRange(segment.OutputDurationMin, segment.OutputDurationMax) {
		tags = append(tags, fmt.Sprintf("output_duration:%d-%d", *segment.OutputDurationMin, *segment.OutputDurationMax))
	}
	if hasIntRange(segment.InputVideoDurationMin, segment.InputVideoDurationMax) {
		tags = append(tags, fmt.Sprintf("input_video_duration:%d-%d", *segment.InputVideoDurationMin, *segment.InputVideoDurationMax))
	}
	if segment.Draft != nil {
		tags = append(tags, fmt.Sprintf("draft:%t", *segment.Draft))
	}
	if segment.MinTokens != nil {
		tags = append(tags, fmt.Sprintf("min_tokens:%d", *segment.MinTokens))
	}
	return tags
}

func cloneAdvancedIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func cloneAdvancedFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func cloneAdvancedStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := append([]string(nil), values...)
	return cloned
}

func valueFromAdvancedFloatPtr(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func cloneAdvancedBoolPtr(v *bool) *bool {
	if v == nil {
		return nil
	}
	cloned := *v
	return &cloned
}

func valueFromAdvancedIntPtr(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func GetAdvancedPricingRuleSet(modelName string) (AdvancedPricingRuleSet, bool) {
	modelName = FormatMatchingModelName(modelName)
	return getAdvancedPricingRuleSetMapValue(modelName)
}

func getAdvancedPricingModeMapValue(modelName string) (BillingMode, bool) {
	if mode, ok := advancedPricingModeMap.Get(modelName); ok {
		return mode, true
	}
	if strings.HasSuffix(modelName, CompactModelSuffix) {
		return advancedPricingModeMap.Get(CompactWildcardModelKey)
	}
	return "", false
}

func getAdvancedPricingRuleSetMapValue(modelName string) (AdvancedPricingRuleSet, bool) {
	if ruleSet, ok := advancedPricingRulesMap.Get(modelName); ok {
		return ruleSet, true
	}
	if strings.HasSuffix(modelName, CompactModelSuffix) {
		return advancedPricingRulesMap.Get(CompactWildcardModelKey)
	}
	return AdvancedPricingRuleSet{}, false
}

func AdvancedPricingMode2JSONString() string {
	return advancedPricingModeMap.MarshalJSONString()
}

func AdvancedPricingRules2JSONString() string {
	return advancedPricingRulesMap.MarshalJSONString()
}

func AdvancedPricingConfig2JSONString() string {
	jsonBytes, err := common.Marshal(AdvancedPricingConfig{
		ModelModes: advancedPricingModeMap.ReadAll(),
		ModelRules: advancedPricingRulesMap.ReadAll(),
	})
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

func ValidateAdvancedPricingConfigJSONString(jsonStr string) error {
	_, err := ParseAdvancedPricingConfig(jsonStr)
	return err
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

func UpdateAdvancedPricingConfigByJSONString(jsonStr string) error {
	cfg, err := ParseAdvancedPricingConfig(jsonStr)
	if err != nil {
		return err
	}
	advancedPricingModeMap.Clear()
	advancedPricingModeMap.AddAll(cfg.ModelModes)
	advancedPricingRulesMap.Clear()
	advancedPricingRulesMap.AddAll(cfg.ModelRules)
	return nil
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
	normalizeAdvancedPricingRuleServiceTiers(cfg.ModelRules)
	if err := validateAdvancedPricingModes(cfg.ModelModes); err != nil {
		return nil, err
	}
	if err := validateAdvancedPricingRules(cfg.ModelRules); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeAdvancedPricingRuleServiceTiers(rules map[string]AdvancedPricingRuleSet) {
	for modelName, ruleSet := range rules {
		normalizeAdvancedPricingRuleSetSegments(&ruleSet)
		rules[modelName] = ruleSet
	}
}

func normalizeAdvancedPricingRuleSetSegments(ruleSet *AdvancedPricingRuleSet) {
	if ruleSet == nil || len(ruleSet.Segments) == 0 {
		return
	}
	normalizedSegments := make([]AdvancedPriceRule, len(ruleSet.Segments))
	for index, segment := range ruleSet.Segments {
		segment.ServiceTier = normalizeAdvancedPricingServiceTier(segment.ServiceTier)
		segment.InputModality = normalizeAdvancedPricingComparableString(segment.InputModality)
		segment.OutputModality = normalizeAdvancedPricingComparableString(segment.OutputModality)
		segment.ImageSizeTier = normalizeAdvancedPricingComparableString(segment.ImageSizeTier)
		segment.ToolUsageType = NormalizeAdvancedPricingTextToolUsageType(segment.ToolUsageType)
		normalizedSegments[index] = segment
	}
	ruleSet.Segments = normalizedSegments
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
	defaultSegmentCount := 0
	for _, segment := range segments {
		if err := validateTextRange(modelName, "input", segment.InputMin, segment.InputMax); err != nil {
			return err
		}
		if err := validateTextRange(modelName, "output", segment.OutputMin, segment.OutputMax); err != nil {
			return err
		}
		if err := validateUnsupportedTextRuntimeFields(modelName, segment); err != nil {
			return err
		}
		if !hasTextCondition(segment) {
			defaultSegmentCount++
		}
		if segment.InputPrice == nil {
			return fmt.Errorf("model %s text segment is missing input_price", modelName)
		}
		if *segment.InputPrice < 0 {
			return fmt.Errorf("model %s text segment input_price cannot be negative", modelName)
		}
		if err := validateTextPriceDependencies(modelName, segment); err != nil {
			return err
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
	if defaultSegmentCount > 1 {
		return fmt.Errorf("model %s text segment allows at most one default segment", modelName)
	}

	for i := 0; i < len(segments); i++ {
		for j := i + 1; j < len(segments); j++ {
			if !hasTextCondition(segments[i]) || !hasTextCondition(segments[j]) {
				continue
			}
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
	if minVal != nil && *minVal < 0 {
		return fmt.Errorf("model %s text segment %s 区间 cannot be negative", modelName, rangeName)
	}
	if maxVal != nil && *maxVal < 0 {
		return fmt.Errorf("model %s text segment %s 区间 cannot be negative", modelName, rangeName)
	}
	if minVal != nil && maxVal != nil && *maxVal < *minVal {
		return fmt.Errorf("model %s text segment %s 区间 is invalid", modelName, rangeName)
	}
	return nil
}

func hasTextCondition(segment AdvancedPriceRule) bool {
	return hasIntRange(segment.InputMin, segment.InputMax) ||
		hasIntRange(segment.OutputMin, segment.OutputMax) ||
		normalizeAdvancedPricingServiceTier(segment.ServiceTier) != "" ||
		normalizeAdvancedPricingComparableString(segment.InputModality) != "" ||
		normalizeAdvancedPricingComparableString(segment.OutputModality) != "" ||
		normalizeAdvancedPricingImageSizeTier(segment.ImageSizeTier) != "" ||
		NormalizeAdvancedPricingTextToolUsageType(segment.ToolUsageType) != "" ||
		segment.ToolUsageCount != nil
}

func validateUnsupportedTextRuntimeFields(modelName string, segment AdvancedPriceRule) error {
	if segment.CacheRead != nil || segment.CacheCreate != nil {
		return fmt.Errorf("model %s text segment cache_read/cache_create conditions are not supported in advanced runtime", modelName)
	}
	if segment.CacheStoragePrice != nil && *segment.CacheStoragePrice < 0 {
		return fmt.Errorf("model %s text segment cache_storage_price cannot be negative", modelName)
	}
	if segment.ToolUsageCount != nil && *segment.ToolUsageCount < 0 {
		return fmt.Errorf("model %s text segment tool_usage_count cannot be negative", modelName)
	}
	if segment.FreeQuota != nil && *segment.FreeQuota < 0 {
		return fmt.Errorf("model %s text segment free_quota cannot be negative", modelName)
	}
	if segment.OverageThreshold != nil && *segment.OverageThreshold < 0 {
		return fmt.Errorf("model %s text segment overage_threshold cannot be negative", modelName)
	}
	return nil
}

func validateTextPriceDependencies(modelName string, segment AdvancedPriceRule) error {
	if segment.InputPrice == nil {
		return nil
	}
	if *segment.InputPrice > 0 {
		return nil
	}
	if hasPositiveAdvancedPrice(segment.OutputPrice) || hasPositiveAdvancedPrice(segment.CacheReadPrice) || hasPositiveAdvancedPrice(segment.CacheCreatePrice) {
		return fmt.Errorf("model %s text segment input_price must be greater than zero when output/cache prices are non-zero", modelName)
	}
	return nil
}

func hasPositiveAdvancedPrice(price *float64) bool {
	return price != nil && *price > 0
}

func hasIntRange(minVal, maxVal *int) bool {
	return minVal != nil || maxVal != nil
}

func textSegmentsOverlap(left, right AdvancedPriceRule) bool {
	if normalizeAdvancedPricingServiceTier(left.ServiceTier) != normalizeAdvancedPricingServiceTier(right.ServiceTier) {
		return false
	}
	if normalizeAdvancedPricingComparableString(left.InputModality) != normalizeAdvancedPricingComparableString(right.InputModality) {
		return false
	}
	if normalizeAdvancedPricingComparableString(left.OutputModality) != normalizeAdvancedPricingComparableString(right.OutputModality) {
		return false
	}
	if normalizeAdvancedPricingImageSizeTier(left.ImageSizeTier) != normalizeAdvancedPricingImageSizeTier(right.ImageSizeTier) {
		return false
	}
	if NormalizeAdvancedPricingTextToolUsageType(left.ToolUsageType) != NormalizeAdvancedPricingTextToolUsageType(right.ToolUsageType) {
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
	if leftMax != nil && rightMin != nil && *leftMax < *rightMin {
		return false
	}
	if rightMax != nil && leftMin != nil && *rightMax < *leftMin {
		return false
	}
	return true
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
