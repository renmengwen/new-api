package helper

import (
	"fmt"

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
	modelPrice, usePrice := ratio_setting.GetModelPrice(info.OriginModelName, false)
	groupRatioInfo := HandleGroupRatio(c, info)

	if ratio_setting.GetEffectiveBillingMode(info.OriginModelName) == ratio_setting.BillingModeAdvanced {
		advancedPriceData, ok, err := ratio_setting.ResolveAdvancedPriceData(info.OriginModelName, ratio_setting.AdvancedPricingRuntimeContext{
			PromptTokens: promptTokens,
			Meta:         meta,
			Request:      info.Request,
		})
		if err != nil {
			return types.PriceData{}, err
		}
		if ok {
			advancedPriceData.GroupRatioInfo = groupRatioInfo
			advancedPriceData.UsePrice = false
			advancedPriceData.ImageRatio, _ = ratio_setting.GetImageRatio(info.OriginModelName)
			advancedPriceData.AudioRatio = ratio_setting.GetAudioRatio(info.OriginModelName)
			advancedPriceData.AudioCompletionRatio = ratio_setting.GetAudioCompletionRatio(info.OriginModelName)
			advancedPriceData.CacheCreation5mRatio = advancedPriceData.CacheCreationRatio
			advancedPriceData.CacheCreation1hRatio = advancedPriceData.CacheCreationRatio * claudeCacheCreation1hMultiplier
			advancedPriceData.QuotaToPreConsume = int(float64(getPreConsumedTokens(promptTokens, meta)) * advancedPriceData.ModelRatio * groupRatioInfo.GroupRatio)
			applyFreeModelPreConsume(&advancedPriceData)

			if common.DebugEnabled {
				println(fmt.Sprintf("model_price_helper result: %s", advancedPriceData.ToSetting()))
			}
			info.PriceData = advancedPriceData
			return advancedPriceData, nil
		}
	}

	var preConsumedQuota int
	var modelRatio float64
	var completionRatio float64
	var cacheRatio float64
	var imageRatio float64
	var cacheCreationRatio float64
	var cacheCreationRatio5m float64
	var cacheCreationRatio1h float64
	var audioRatio float64
	var audioCompletionRatio float64

	if !usePrice {
		preConsumedTokens := getPreConsumedTokens(promptTokens, meta)
		var success bool
		var matchName string
		modelRatio, success, matchName = ratio_setting.GetModelRatio(info.OriginModelName)
		if !success {
			acceptUnsetRatio := false
			if info.UserSetting.AcceptUnsetRatioModel {
				acceptUnsetRatio = true
			}
			if !acceptUnsetRatio {
				return types.PriceData{}, fmt.Errorf("模型 %s 倍率或价格未配置，请联系管理员设置或开始自用模式；Model %s ratio or price not set, please set or start self-use mode", matchName, matchName)
			}
		}
		completionRatio = ratio_setting.GetCompletionRatio(info.OriginModelName)
		cacheRatio, _ = ratio_setting.GetCacheRatio(info.OriginModelName)
		cacheCreationRatio, _ = ratio_setting.GetCreateCacheRatio(info.OriginModelName)
		cacheCreationRatio5m = cacheCreationRatio
		cacheCreationRatio1h = cacheCreationRatio * claudeCacheCreation1hMultiplier
		imageRatio, _ = ratio_setting.GetImageRatio(info.OriginModelName)
		audioRatio = ratio_setting.GetAudioRatio(info.OriginModelName)
		audioCompletionRatio = ratio_setting.GetAudioCompletionRatio(info.OriginModelName)
		preConsumedQuota = int(float64(preConsumedTokens) * modelRatio * groupRatioInfo.GroupRatio)
	} else {
		if meta != nil && meta.ImagePriceRatio != 0 {
			modelPrice = modelPrice * meta.ImagePriceRatio
		}
		preConsumedQuota = int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
	}

	priceData := types.PriceData{
		ModelPrice:           modelPrice,
		ModelRatio:           modelRatio,
		CompletionRatio:      completionRatio,
		CacheRatio:           cacheRatio,
		CacheCreationRatio:   cacheCreationRatio,
		CacheCreation5mRatio: cacheCreationRatio5m,
		CacheCreation1hRatio: cacheCreationRatio1h,
		ImageRatio:           imageRatio,
		AudioRatio:           audioRatio,
		AudioCompletionRatio: audioCompletionRatio,
		BillingMode:          fixedTextBillingMode(usePrice),
		GroupRatioInfo:       groupRatioInfo,
		UsePrice:             usePrice,
		QuotaToPreConsume:    preConsumedQuota,
	}
	applyFreeModelPreConsume(&priceData)

	if common.DebugEnabled {
		println(fmt.Sprintf("model_price_helper result: %s", priceData.ToSetting()))
	}
	info.PriceData = priceData
	return priceData, nil
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
			acceptUnsetRatio := false
			if info.UserSetting.AcceptUnsetRatioModel {
				acceptUnsetRatio = true
			}
			if !ratioSuccess && !acceptUnsetRatio {
				return types.PriceData{}, fmt.Errorf("模型 %s 倍率或价格未配置，请联系管理员设置或开始自用模式；Model %s ratio or price not set, please set or start self-use mode", matchName, matchName)
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

	priceData := types.PriceData{
		FreeModel:      freeModel,
		ModelPrice:     modelPrice,
		BillingMode:    types.BillingModePerRequest,
		Quota:          quota,
		GroupRatioInfo: groupRatioInfo,
	}
	return priceData, nil
}

func ContainPriceOrRatio(modelName string) bool {
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

func fixedTextBillingMode(usePrice bool) types.BillingMode {
	if usePrice {
		return types.BillingModePerRequest
	}
	return types.BillingModePerToken
}
