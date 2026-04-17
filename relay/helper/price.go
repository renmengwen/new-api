package helper

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
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
		advancedPriceData, ok, err := ratio_setting.ResolveAdvancedPriceData(info.OriginModelName, ratio_setting.AdvancedPricingRuntimeContext{
			PromptTokens: promptTokens,
			Meta:         meta,
			Request:      info.Request,
		})
		if err != nil {
			return types.PriceData{}, err
		}
		if ok {
			priceData := finalizeAdvancedPriceData(info.OriginModelName, promptTokens, meta, groupRatioInfo, advancedPriceData)
			return finalizeModelPriceData(info, priceData), nil
		}

		mode = ratio_setting.GetLegacyBillingMode(info.OriginModelName)
		hasExplicitMode = false
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

func finalizeAdvancedPriceData(modelName string, promptTokens int, meta *types.TokenCountMeta, groupRatioInfo types.GroupRatioInfo, priceData types.PriceData) types.PriceData {
	priceData.GroupRatioInfo = groupRatioInfo
	priceData.UsePrice = false
	priceData.ImageRatio, _ = ratio_setting.GetImageRatio(modelName)
	priceData.AudioRatio = ratio_setting.GetAudioRatio(modelName)
	priceData.AudioCompletionRatio = ratio_setting.GetAudioCompletionRatio(modelName)
	priceData.CacheCreation5mRatio = priceData.CacheCreationRatio
	priceData.CacheCreation1hRatio = priceData.CacheCreationRatio * claudeCacheCreation1hMultiplier
	priceData.QuotaToPreConsume = int(float64(getPreConsumedTokens(promptTokens, meta)) * priceData.ModelRatio * groupRatioInfo.GroupRatio)
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
