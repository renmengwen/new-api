package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const claudeCacheCreation1hMultiplier = 6 / 3.75

func HandleGroupRatio(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) types.GroupRatioInfo {
	groupRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.0,
		GroupSpecialRatio: -1,
	}

	autoGroup, exists := ctx.Get("auto_group")
	if exists {
		logger.LogDebug(ctx, fmt.Sprintf("final group: %s", autoGroup))
		relayInfo.UsingGroup = autoGroup.(string)
	}

	userGroupRatio, ok := ratio_setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup)
	if ok {
		groupRatioInfo.GroupSpecialRatio = userGroupRatio
		groupRatioInfo.GroupRatio = userGroupRatio
		groupRatioInfo.HasSpecialRatio = true
	} else {
		groupRatioInfo.GroupRatio = ratio_setting.GetGroupRatio(relayInfo.UsingGroup)
	}

	return groupRatioInfo
}

func ModelPriceHelper(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) (types.PriceData, error) {
	groupRatioInfo := HandleGroupRatio(c, info)
	mode, hasExplicitMode := ratio_setting.GetExplicitBillingMode(info.OriginModelName)
	if !hasExplicitMode {
		mode = ratio_setting.GetEffectiveBillingMode(info.OriginModelName)
	}

	if mode == ratio_setting.BillingModeAdvanced {
		advancedPriceData, ok, err := ratio_setting.ResolveAdvancedPriceData(info.OriginModelName, buildAdvancedPricingRuntimeContext(c, info, promptTokens, meta))
		if err != nil {
			return types.PriceData{}, err
		}
		if ok {
			priceData := finalizeAdvancedPriceData(info.OriginModelName, promptTokens, meta, groupRatioInfo, advancedPriceData)
			priceData = attachAdvancedTextRuntimeContext(info, priceData)
			return finalizeModelPriceData(info, priceData), nil
		}
		return types.PriceData{}, advancedPricingNoMatchError(info.OriginModelName)
	}

	var (
		priceData types.PriceData
		err       error
	)

	switch mode {
	case ratio_setting.BillingModePerToken:
		priceData, err = buildPerTokenPriceData(info, promptTokens, meta, groupRatioInfo, hasExplicitMode)
	case ratio_setting.BillingModePerRequest:
		priceData, err = buildPerRequestPriceData(info, meta, groupRatioInfo)
	default:
		err = fmt.Errorf("unsupported billing mode: %s", mode)
	}
	if err != nil {
		return types.PriceData{}, err
	}

	return finalizeModelPriceData(info, priceData), nil
}

func RefreshTextPriceDataForSettlement(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, completionTokens int) (types.PriceData, bool, error) {
	if info == nil {
		return types.PriceData{}, false, nil
	}

	staleAdvancedTextPricing := info.PriceData.BillingMode == types.BillingModeAdvanced &&
		info.PriceData.AdvancedRuleType == types.AdvancedRuleTypeTextSegment
	if staleAdvancedTextPricing && hasReusableAdvancedTextSettlementPriceData(info.PriceData) {
		return info.PriceData, true, nil
	}
	currentConfiguredAdvancedTextPricing := false
	if ratio_setting.GetEffectiveBillingMode(info.OriginModelName) == ratio_setting.BillingModeAdvanced {
		if ruleSet, ok := ratio_setting.GetAdvancedPricingRuleSet(info.OriginModelName); ok && ruleSet.RuleType == ratio_setting.RuleTypeTextSegment {
			currentConfiguredAdvancedTextPricing = true
		}
	}
	if !staleAdvancedTextPricing && !currentConfiguredAdvancedTextPricing {
		return types.PriceData{}, false, nil
	}

	originalGroupRatioInfo := info.PriceData.GroupRatioInfo
	runtimeRelayInfo := *info
	priceData, err := ModelPriceHelper(c, &runtimeRelayInfo, promptTokens, &types.TokenCountMeta{
		MaxTokens: completionTokens,
	})
	if err != nil {
		return types.PriceData{}, false, err
	}
	priceData.GroupRatioInfo = originalGroupRatioInfo
	priceData.OtherRatios = info.PriceData.OtherRatios
	return priceData, true, nil
}

func hasReusableAdvancedTextSettlementPriceData(priceData types.PriceData) bool {
	return priceData.AdvancedRuleSnapshot != nil && priceData.AdvancedPricingContext != nil
}

func finalizeAdvancedPriceData(modelName string, promptTokens int, meta *types.TokenCountMeta, groupRatioInfo types.GroupRatioInfo, priceData types.PriceData) types.PriceData {
	priceData.GroupRatioInfo = groupRatioInfo
	priceData.ImageRatio, _ = ratio_setting.GetImageRatio(modelName)
	priceData.AudioRatio = ratio_setting.GetAudioRatio(modelName)
	priceData.AudioCompletionRatio = ratio_setting.GetAudioCompletionRatio(modelName)
	priceData.CacheCreation5mRatio = priceData.CacheCreationRatio
	priceData.CacheCreation1hRatio = priceData.CacheCreationRatio * claudeCacheCreation1hMultiplier
	if priceData.UsePrice {
		priceData.QuotaToPreConsume = int(priceData.ModelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
	} else {
		priceData.QuotaToPreConsume = int(float64(getPreConsumedTokens(promptTokens, meta)) * priceData.ModelRatio * groupRatioInfo.GroupRatio)
	}
	applyFreeModelPreConsume(&priceData)
	return priceData
}

func buildPerTokenPriceData(info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta, groupRatioInfo types.GroupRatioInfo, strict bool) (types.PriceData, error) {
	preConsumedTokens := getPreConsumedTokens(promptTokens, meta)
	modelRatio, _, err := resolvePerTokenModelRatio(info, strict)
	if err != nil {
		return types.PriceData{}, err
	}

	completionRatio := ratio_setting.GetCompletionRatio(info.OriginModelName)
	cacheRatio, _ := ratio_setting.GetCacheRatio(info.OriginModelName)
	cacheCreationRatio, _ := ratio_setting.GetCreateCacheRatio(info.OriginModelName)
	imageRatio, _ := ratio_setting.GetImageRatio(info.OriginModelName)
	audioRatio := ratio_setting.GetAudioRatio(info.OriginModelName)
	audioCompletionRatio := ratio_setting.GetAudioCompletionRatio(info.OriginModelName)

	priceData := types.PriceData{
		ModelRatio:           modelRatio,
		CompletionRatio:      completionRatio,
		CacheRatio:           cacheRatio,
		CacheCreationRatio:   cacheCreationRatio,
		CacheCreation5mRatio: cacheCreationRatio,
		CacheCreation1hRatio: cacheCreationRatio * claudeCacheCreation1hMultiplier,
		ImageRatio:           imageRatio,
		AudioRatio:           audioRatio,
		AudioCompletionRatio: audioCompletionRatio,
		BillingMode:          types.BillingModePerToken,
		GroupRatioInfo:       groupRatioInfo,
		UsePrice:             false,
		QuotaToPreConsume:    int(float64(preConsumedTokens) * modelRatio * groupRatioInfo.GroupRatio),
	}
	applyFreeModelPreConsume(&priceData)

	return priceData, nil
}

func resolvePerTokenModelRatio(info *relaycommon.RelayInfo, strict bool) (float64, string, error) {
	if strict {
		modelRatio, ok, matchName := getConfiguredModelRatio(info.OriginModelName)
		if !ok {
			return 0, matchName, fmt.Errorf("model %s requires model_ratio for billing_mode=per_token", matchName)
		}
		return modelRatio, matchName, nil
	}

	modelRatio, success, matchName := ratio_setting.GetModelRatio(info.OriginModelName)
	if success {
		return modelRatio, matchName, nil
	}
	if info.UserSetting.AcceptUnsetRatioModel {
		return modelRatio, matchName, nil
	}
	return 0, matchName, fmt.Errorf("model %s ratio or price not set, please set it or enable self-use mode", matchName)
}

func buildPerRequestPriceData(info *relaycommon.RelayInfo, meta *types.TokenCountMeta, groupRatioInfo types.GroupRatioInfo) (types.PriceData, error) {
	modelPrice, ok := ratio_setting.GetModelPrice(info.OriginModelName, false)
	if !ok {
		matchName := ratio_setting.FormatMatchingModelName(info.OriginModelName)
		return types.PriceData{}, fmt.Errorf("model %s requires model_price for billing_mode=per_request", matchName)
	}

	if meta != nil && meta.ImagePriceRatio != 0 {
		modelPrice = modelPrice * meta.ImagePriceRatio
	}

	priceData := types.PriceData{
		ModelPrice:        modelPrice,
		BillingMode:       types.BillingModePerRequest,
		GroupRatioInfo:    groupRatioInfo,
		UsePrice:          true,
		QuotaToPreConsume: int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio),
	}
	applyFreeModelPreConsume(&priceData)
	return priceData, nil
}

func finalizeModelPriceData(info *relaycommon.RelayInfo, priceData types.PriceData) types.PriceData {
	if common.DebugEnabled {
		println(fmt.Sprintf("model_price_helper result: %s", priceData.ToSetting()))
	}
	info.PriceData = priceData
	return priceData
}

func getConfiguredModelRatio(modelName string) (float64, bool, string) {
	matchName := ratio_setting.FormatMatchingModelName(modelName)
	modelRatioMap := ratio_setting.GetModelRatioCopy()

	if modelRatio, ok := modelRatioMap[matchName]; ok {
		return modelRatio, true, matchName
	}
	if strings.HasSuffix(matchName, ratio_setting.CompactModelSuffix) {
		if modelRatio, ok := modelRatioMap[ratio_setting.CompactWildcardModelKey]; ok {
			return modelRatio, true, matchName
		}
	}
	return 0, false, matchName
}

func ModelPriceHelperPerCall(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, error) {
	groupRatioInfo := HandleGroupRatio(c, info)
	if priceData, ok, err := resolveExplicitPerCallPriceData(c, info, groupRatioInfo); ok || err != nil {
		return priceData, err
	}

	modelPrice, success := ratio_setting.GetModelPrice(info.OriginModelName, true)
	if !success {
		defaultPrice, ok := ratio_setting.GetDefaultModelPriceMap()[info.OriginModelName]
		if ok {
			modelPrice = defaultPrice
		} else {
			_, ratioSuccess, matchName := ratio_setting.GetModelRatio(info.OriginModelName)
			if !ratioSuccess && !info.UserSetting.AcceptUnsetRatioModel {
				return types.PriceData{}, fmt.Errorf("model %s ratio or price not set, please set it or enable self-use mode", matchName)
			}
			modelPrice = float64(common.PreConsumedQuota) / common.QuotaPerUnit
		}
	}

	quota := int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 || modelPrice == 0 {
			quota = 0
			freeModel = true
		}
	}

	return types.PriceData{
		FreeModel:      freeModel,
		ModelPrice:     modelPrice,
		BillingMode:    types.BillingModePerRequest,
		Quota:          quota,
		GroupRatioInfo: groupRatioInfo,
	}, nil
}

func resolveExplicitPerCallPriceData(c *gin.Context, info *relaycommon.RelayInfo, groupRatioInfo types.GroupRatioInfo) (types.PriceData, bool, error) {
	mode, hasExplicitMode := ratio_setting.GetExplicitBillingMode(info.OriginModelName)
	if !hasExplicitMode {
		return types.PriceData{}, false, nil
	}

	switch mode {
	case ratio_setting.BillingModePerToken:
		priceData, err := buildPerTokenPriceData(info, 0, nil, groupRatioInfo, true)
		if err != nil {
			return types.PriceData{}, true, err
		}
		return finalizePerCallPriceData(priceData), true, nil
	case ratio_setting.BillingModePerRequest:
		priceData, err := buildPerRequestPriceData(info, nil, groupRatioInfo)
		if err != nil {
			return types.PriceData{}, true, err
		}
		return finalizePerCallPriceData(priceData), true, nil
	case ratio_setting.BillingModeAdvanced:
		runtimeCtx := buildAdvancedPricingRuntimeContext(c, info, 0, nil)
		runtimeCtx.Task = buildAdvancedPricingTaskContext(c, info)
		priceData, ok, err := ratio_setting.ResolveAdvancedPriceData(info.OriginModelName, runtimeCtx)
		if err != nil {
			return types.PriceData{}, true, err
		}
		if ok {
			priceData = finalizeAdvancedPriceData(info.OriginModelName, 0, nil, groupRatioInfo, priceData)
			priceData = attachAdvancedTextRuntimeContext(info, priceData)
			return finalizePerCallPriceData(priceData), true, nil
		}
		return types.PriceData{}, true, advancedPricingNoMatchError(info.OriginModelName)
	default:
		return types.PriceData{}, true, fmt.Errorf("unsupported billing mode: %s", mode)
	}
}

func advancedPricingNoMatchError(modelName string) error {
	matchName := ratio_setting.FormatMatchingModelName(modelName)
	return fmt.Errorf("model %s advanced pricing did not match any active rule", matchName)
}

func resolveLegacyPerCallPriceData(info *relaycommon.RelayInfo, groupRatioInfo types.GroupRatioInfo) (types.PriceData, error) {
	mode := ratio_setting.GetLegacyBillingMode(info.OriginModelName)
	switch mode {
	case ratio_setting.BillingModePerToken:
		priceData, err := buildPerTokenPriceData(info, 0, nil, groupRatioInfo, false)
		if err != nil {
			return types.PriceData{}, err
		}
		return finalizePerCallPriceData(priceData), nil
	case ratio_setting.BillingModePerRequest:
		priceData, err := buildPerRequestPriceData(info, nil, groupRatioInfo)
		if err != nil {
			return types.PriceData{}, err
		}
		return finalizePerCallPriceData(priceData), nil
	default:
		return types.PriceData{}, fmt.Errorf("unsupported billing mode: %s", mode)
	}
}

func finalizePerCallPriceData(priceData types.PriceData) types.PriceData {
	priceData.Quota = priceData.QuotaToPreConsume
	return priceData
}

func buildAdvancedPricingRuntimeContext(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) ratio_setting.AdvancedPricingRuntimeContext {
	runtimeCtx := ratio_setting.AdvancedPricingRuntimeContext{
		PromptTokens: promptTokens,
		Meta:         meta,
	}
	if info == nil {
		return runtimeCtx
	}

	runtimeCtx.Request = info.Request
	runtimeCtx.RequestURLPath = info.RequestURLPath
	if strings.TrimSpace(info.InputAudioFormat) != "" {
		runtimeCtx.InputModalities = append(runtimeCtx.InputModalities, "audio")
	}
	if strings.TrimSpace(info.OutputAudioFormat) != "" {
		runtimeCtx.OutputModalities = append(runtimeCtx.OutputModalities, "audio")
	}
	runtimeCtx.ToolUsageType, runtimeCtx.ToolUsageCount = resolveAdvancedTextToolUsage(c, info)
	return runtimeCtx
}

func resolveAdvancedTextToolUsage(c *gin.Context, info *relaycommon.RelayInfo) (string, int) {
	if info != nil && info.ResponsesUsageInfo != nil && info.ResponsesUsageInfo.BuiltInTools != nil {
		if toolUsageType, toolUsageCount := resolveAdvancedResponsesToolUsage(info.ResponsesUsageInfo.BuiltInTools); toolUsageType != "" {
			return toolUsageType, toolUsageCount
		}
	}
	if info != nil && strings.HasSuffix(strings.TrimSpace(info.OriginModelName), "search-preview") {
		return ratio_setting.NormalizeAdvancedPricingTextToolUsageType("grounding"), 1
	}
	if c != nil {
		if count := c.GetInt("claude_web_search_requests"); count > 0 {
			return ratio_setting.NormalizeAdvancedPricingTextToolUsageType("web_search"), count
		}
	}
	return "", 0
}

func resolveAdvancedResponsesToolUsage(builtInTools map[string]*relaycommon.BuildInToolInfo) (string, int) {
	if webSearchTool, exists := builtInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool != nil && webSearchTool.CallCount > 0 {
		return ratio_setting.NormalizeAdvancedPricingTextToolUsageType("web_search"), webSearchTool.CallCount
	}

	for toolName, tool := range builtInTools {
		if normalizedToolUsageType, callCount, ok := normalizeAdvancedResponsesToolUsage(toolName, tool); ok && normalizedToolUsageType == "google_maps" {
			return normalizedToolUsageType, callCount
		}
	}
	for toolName, tool := range builtInTools {
		if normalizedToolUsageType, callCount, ok := normalizeAdvancedResponsesToolUsage(toolName, tool); ok && normalizedToolUsageType == "google_search" {
			return normalizedToolUsageType, callCount
		}
	}
	return "", 0
}

func normalizeAdvancedResponsesToolUsage(toolName string, tool *relaycommon.BuildInToolInfo) (string, int, bool) {
	if tool == nil || tool.CallCount <= 0 {
		return "", 0, false
	}
	candidateName := strings.TrimSpace(tool.ToolName)
	if candidateName == "" {
		candidateName = toolName
	}
	normalizedToolUsageType := ratio_setting.NormalizeAdvancedPricingTextToolUsageType(candidateName)
	if normalizedToolUsageType == "" {
		return "", 0, false
	}
	return normalizedToolUsageType, tool.CallCount, true
}

func attachAdvancedTextRuntimeContext(info *relaycommon.RelayInfo, priceData types.PriceData) types.PriceData {
	if info == nil || priceData.BillingMode != types.BillingModeAdvanced || priceData.AdvancedRuleType != types.AdvancedRuleTypeTextSegment {
		return priceData
	}

	if priceData.AdvancedPricingContext == nil {
		priceData.AdvancedPricingContext = &types.AdvancedPricingContextSnapshot{}
	}
	if priceData.AdvancedRuleSnapshot != nil && strings.TrimSpace(priceData.AdvancedPricingContext.BillingUnit) == "" {
		priceData.AdvancedPricingContext.BillingUnit = strings.TrimSpace(priceData.AdvancedRuleSnapshot.BillingUnit)
	}
	if priceData.AdvancedPricingContext.ImageCount == nil {
		if imageCount, ok := resolveAdvancedTextImageCount(info, priceData); ok {
			priceData.AdvancedPricingContext.ImageCount = common.GetPointer(imageCount)
		}
	}
	return priceData
}

func resolveAdvancedTextImageCount(info *relaycommon.RelayInfo, priceData types.PriceData) (int, bool) {
	if info == nil || info.Request == nil || !usesAdvancedTextImageCount(info, priceData) {
		return 0, false
	}

	switch req := info.Request.(type) {
	case *dto.GeneralOpenAIRequest:
		if req.N != nil && *req.N > 0 {
			return *req.N, true
		}
		return 1, true
	default:
		return 0, false
	}
}

func usesAdvancedTextImageCount(info *relaycommon.RelayInfo, priceData types.PriceData) bool {
	if isAdvancedTextImageGenerationPath(info.RequestURLPath) {
		return true
	}
	if priceData.AdvancedRuleSnapshot != nil {
		if strings.EqualFold(priceData.AdvancedRuleSnapshot.BillingUnit, types.AdvancedBillingUnitPerImage) {
			return true
		}
		if strings.EqualFold(priceData.AdvancedRuleSnapshot.OutputModality, "image") {
			return true
		}
	}
	if priceData.AdvancedPricingContext == nil {
		return false
	}
	if strings.EqualFold(priceData.AdvancedPricingContext.BillingUnit, types.AdvancedBillingUnitPerImage) {
		return true
	}
	for _, modality := range priceData.AdvancedPricingContext.OutputModalities {
		if strings.EqualFold(modality, "image") {
			return true
		}
	}
	return false
}

func isAdvancedTextImageGenerationPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.HasPrefix(trimmed, "/v1/images/generations")
}

func buildAdvancedPricingTaskContext(c *gin.Context, info *relaycommon.RelayInfo) *ratio_setting.AdvancedPricingTaskContext {
	if c == nil {
		return nil
	}

	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	rawAction := resolveRawTaskAction(info, c)
	runtimeCtx := &ratio_setting.AdvancedPricingTaskContext{
		TaskType:           normalizeAdvancedTaskString(resolveCanonicalTaskType(rawAction)),
		RawAction:          strings.TrimSpace(rawAction),
		InferenceMode:      normalizeAdvancedTaskString(firstTaskString(taskReq.Mode, taskMetadataString(taskReq.Metadata, "inference_mode", "inferenceMode"))),
		Resolution:         normalizeAdvancedTaskString(firstTaskString(taskMetadataString(taskReq.Metadata, "resolution", "Resolution"), deriveTaskResolution(taskReq.Size))),
		AspectRatio:        normalizeAdvancedTaskString(firstTaskString(taskMetadataString(taskReq.Metadata, "aspect_ratio", "aspectRatio"), deriveTaskAspectRatio(taskReq.Size))),
		OutputDuration:     resolveTaskOutputDuration(taskReq),
		InputVideoDuration: taskMetadataIntValue(taskReq.Metadata, "input_video_duration", "inputVideoDuration"),
	}

	if value, ok := taskMetadataBool(taskReq.Metadata, "audio", "generate_audio", "generateAudio"); ok {
		runtimeCtx.Audio = &value
	}
	if value, ok := taskMetadataBool(taskReq.Metadata, "input_video", "inputVideo"); ok {
		runtimeCtx.InputVideo = &value
	}
	if value, ok := taskMetadataBool(taskReq.Metadata, "draft"); ok {
		runtimeCtx.Draft = &value
	}

	return runtimeCtx
}

func resolveRawTaskAction(info *relaycommon.RelayInfo, c *gin.Context) string {
	if c != nil {
		if taskReq, err := relaycommon.GetTaskRequest(c); err == nil {
			if action := deriveRawTaskActionFromTaskRequest(info, c, taskReq); action != "" {
				return action
			}
		}
		if action := strings.TrimSpace(c.GetString("action")); action != "" {
			return action
		}
	}
	if info != nil && info.TaskRelayInfo != nil {
		if action := strings.TrimSpace(info.Action); action != "" {
			return action
		}
	}
	return ""
}

func resolveCanonicalTaskType(rawAction string) string {
	switch normalizeAdvancedTaskString(rawAction) {
	case normalizeAdvancedTaskString(constant.TaskTypeImageGeneration):
		return constant.TaskTypeImageGeneration
	case normalizeAdvancedTaskString(constant.TaskTypeVideoGeneration),
		normalizeAdvancedTaskString(constant.TaskActionGenerate),
		normalizeAdvancedTaskString(constant.TaskActionTextGenerate),
		normalizeAdvancedTaskString(constant.TaskActionFirstTailGenerate),
		normalizeAdvancedTaskString(constant.TaskActionReferenceGenerate),
		normalizeAdvancedTaskString(constant.TaskActionRemix):
		return constant.TaskTypeVideoGeneration
	default:
		return strings.TrimSpace(rawAction)
	}
}

func deriveRawTaskActionFromTaskRequest(info *relaycommon.RelayInfo, c *gin.Context, taskReq relaycommon.TaskSubmitReq) string {
	if action := normalizeAdvancedTaskString(taskMetadataString(taskReq.Metadata, "action")); action != "" {
		return action
	}

	imageCount := len(taskReq.Images)
	if imageCount == 0 && strings.TrimSpace(taskReq.Image) != "" {
		imageCount = 1
	}
	if multipartInputRefCount := taskMetadataIntValue(taskReq.Metadata, "input_reference_count"); multipartInputRefCount > imageCount {
		imageCount = multipartInputRefCount
	}
	hasImage := imageCount > 0 || strings.TrimSpace(taskReq.InputReference) != ""
	hasReferenceMedia := taskMetadataHasReferenceURLs(taskReq.Metadata, "videos", "video_url", "audios", "audio_url")

	switch resolveTaskChannelType(info, c) {
	case constant.ChannelTypeVidu:
		if !hasImage {
			return constant.TaskActionTextGenerate
		}
		switch {
		case imageCount > 2:
			return constant.TaskActionReferenceGenerate
		case imageCount == 2:
			return constant.TaskActionFirstTailGenerate
		default:
			return constant.TaskActionGenerate
		}
	case constant.ChannelTypeGemini, constant.ChannelTypeVertexAi, constant.ChannelTypeKling:
		if hasImage {
			return constant.TaskActionGenerate
		}
		return constant.TaskActionTextGenerate
	case constant.ChannelTypeJimeng:
		switch {
		case imageCount > 1:
			return constant.TaskActionFirstTailGenerate
		case hasImage:
			return constant.TaskActionGenerate
		default:
			return constant.TaskActionTextGenerate
		}
	case constant.ChannelTypeDoubaoVideo, constant.ChannelTypeVolcEngine:
		if hasImage || hasReferenceMedia {
			return constant.TaskActionGenerate
		}
		return constant.TaskActionTextGenerate
	}

	if hasImage || hasReferenceMedia {
		return constant.TaskActionGenerate
	}
	if taskReq.Prompt != "" || taskReq.Model != "" {
		return constant.TaskActionTextGenerate
	}
	return ""
}

func resolveTaskChannelType(info *relaycommon.RelayInfo, c *gin.Context) int {
	if info != nil && info.ChannelMeta != nil {
		return info.ChannelType
	}
	if c == nil {
		return 0
	}
	return c.GetInt("channel_type")
}

func firstTaskString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeAdvancedTaskString(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func taskMetadataString(metadata map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(common.Interface2String(value)); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func taskMetadataHasReferenceURLs(metadata map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch data := value.(type) {
		case string:
			if strings.TrimSpace(data) != "" {
				return true
			}
		case []string:
			for _, item := range data {
				if strings.TrimSpace(item) != "" {
					return true
				}
			}
		case []interface{}:
			for _, item := range data {
				if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
					return true
				}
			}
		}
	}
	return false
}

func taskMetadataBool(metadata map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch data := value.(type) {
		case bool:
			return data, true
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(data))
			if err == nil {
				return parsed, true
			}
		case int:
			return data != 0, true
		case int64:
			return data != 0, true
		case float64:
			return data != 0, true
		}
	}
	return false, false
}

func taskMetadataIntValue(metadata map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		switch data := value.(type) {
		case int:
			return data
		case int64:
			return int(data)
		case float64:
			return int(data)
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(data))
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func resolveTaskOutputDuration(taskReq relaycommon.TaskSubmitReq) int {
	if taskReq.Duration > 0 {
		return taskReq.Duration
	}
	if seconds, err := strconv.Atoi(strings.TrimSpace(taskReq.Seconds)); err == nil && seconds > 0 {
		return seconds
	}
	return taskMetadataIntValue(taskReq.Metadata, "duration", "durationSeconds", "duration_seconds", "output_duration")
}

func deriveTaskResolution(size string) string {
	width, height, ok := parseTaskSize(size)
	if !ok {
		return ""
	}
	longEdge := width
	if height > longEdge {
		longEdge = height
	}
	switch {
	case longEdge >= 3840:
		return "4k"
	case longEdge >= 1920:
		return "1080p"
	case longEdge >= 1280:
		return "720p"
	case longEdge >= 854:
		return "480p"
	default:
		return ""
	}
}

func deriveTaskAspectRatio(size string) string {
	width, height, ok := parseTaskSize(size)
	if !ok || width <= 0 || height <= 0 {
		return ""
	}
	divisor := greatestCommonDivisor(width, height)
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor)
}

func parseTaskSize(size string) (int, int, bool) {
	normalized := strings.ToLower(strings.TrimSpace(size))
	parts := strings.Split(normalized, "x")
	if len(parts) != 2 {
		parts = strings.Split(normalized, "*")
	}
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}
	return width, height, true
}

func greatestCommonDivisor(left, right int) int {
	for right != 0 {
		left, right = right, left%right
	}
	if left == 0 {
		return 1
	}
	return left
}

func ContainPriceOrRatio(modelName string) bool {
	if mode, ok := ratio_setting.GetExplicitBillingMode(modelName); ok {
		switch mode {
		case ratio_setting.BillingModeAdvanced:
			_, ok := ratio_setting.GetAdvancedPricingRuleSet(modelName)
			return ok
		case ratio_setting.BillingModePerRequest:
			_, ok := ratio_setting.GetModelPrice(modelName, false)
			return ok
		case ratio_setting.BillingModePerToken:
			_, ok, _ := getConfiguredModelRatio(modelName)
			return ok
		default:
			return false
		}
	}

	_, ok := ratio_setting.GetModelPrice(modelName, false)
	if ok {
		return true
	}
	_, ok, _ = ratio_setting.GetModelRatio(modelName)
	if ok {
		return true
	}
	return false
}

func getPreConsumedTokens(promptTokens int, meta *types.TokenCountMeta) int {
	preConsumedTokens := common.Max(promptTokens, common.PreConsumedQuota)
	if meta != nil && meta.MaxTokens != 0 {
		preConsumedTokens += meta.MaxTokens
	}
	return preConsumedTokens
}

func applyFreeModelPreConsume(priceData *types.PriceData) {
	if operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		return
	}

	if priceData.GroupRatioInfo.GroupRatio == 0 {
		priceData.QuotaToPreConsume = 0
		priceData.FreeModel = true
		return
	}

	if priceData.UsePrice {
		if priceData.ModelPrice == 0 {
			priceData.QuotaToPreConsume = 0
			priceData.FreeModel = true
		}
		return
	}

	if priceData.ModelRatio == 0 {
		priceData.QuotaToPreConsume = 0
		priceData.FreeModel = true
	}
}
