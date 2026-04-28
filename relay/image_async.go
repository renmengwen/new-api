package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

var errAsyncImageTaskIDMissing = errors.New("async image response missing task id")

func isGPTProtoAsyncImageRequest(request *dto.ImageRequest) bool {
	return request != nil &&
		request.EnableSyncMode != nil &&
		!*request.EnableSyncMode &&
		strings.HasPrefix(request.Model, "gpt-image-2")
}

func shouldUseGPTProtoAsyncImageRequest(info *relaycommon.RelayInfo, request *dto.ImageRequest) bool {
	return isGPTProtoAsyncImageRequest(request) && isGPTProtoAsyncImageChannel(info)
}

func isGPTProtoAsyncImageChannel(info *relaycommon.RelayInfo) bool {
	if info == nil || info.ChannelMeta == nil {
		return false
	}
	baseURL := strings.TrimSpace(info.ChannelBaseUrl)
	if baseURL == "" {
		return false
	}

	lowerBaseURL := strings.ToLower(baseURL)
	if strings.Contains(lowerBaseURL, "/api/v3/openai/") || strings.HasSuffix(lowerBaseURL, "/api/v3/openai") {
		return true
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return strings.Contains(lowerBaseURL, "gptproto")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		host = strings.ToLower(strings.Split(strings.Trim(parsed.Path, "/"), "/")[0])
	}
	return strings.Contains(host, "gptproto")
}

func prepareGPTProtoAsyncImageSubmitRoute(info *relaycommon.RelayInfo, request *dto.ImageRequest) {
	if info == nil || request == nil {
		return
	}
	if info.ChannelMeta == nil {
		info.ChannelMeta = &relaycommon.ChannelMeta{}
	}
	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(request.Model)
	}
	if modelName == "" {
		return
	}
	requestPath := "/api/v3/openai/" + modelName + "/text-to-image"
	baseURL := normalizeGPTProtoImageSubmitBaseURL(info.ChannelBaseUrl)
	if info.ChannelType == constant.ChannelTypeCustom {
		info.ChannelBaseUrl = baseURL + requestPath
	} else {
		info.ChannelBaseUrl = baseURL
	}
	info.RequestURLPath = requestPath
}

func normalizeGPTProtoImageSubmitBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if idx := strings.Index(baseURL, "/api/v3/"); idx >= 0 {
		return baseURL[:idx]
	}
	return baseURL
}

func extractGPTProtoAsyncTaskID(responseBody []byte) (string, error) {
	type asyncImageTaskData struct {
		ID   string          `json:"id"`
		URLs json.RawMessage `json:"urls"`
	}
	var payload struct {
		ID   string          `json:"id"`
		Data json.RawMessage `json:"data"`
	}
	if err := common.Unmarshal(responseBody, &payload); err != nil {
		return "", err
	}

	var candidates []string
	if len(payload.Data) > 0 {
		var data asyncImageTaskData
		if err := common.Unmarshal(payload.Data, &data); err == nil {
			candidates = append(candidates, data.ID)
			candidates = append(candidates, extractPredictionIDsFromURLs(data.URLs)...)
		} else {
			var dataList []asyncImageTaskData
			if listErr := common.Unmarshal(payload.Data, &dataList); listErr != nil {
				return "", err
			}
			for _, item := range dataList {
				candidates = append(candidates, item.ID)
				candidates = append(candidates, extractPredictionIDsFromURLs(item.URLs)...)
			}
		}
	}
	candidates = append(candidates, payload.ID)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) != "" {
			return candidate, nil
		}
	}
	return "", errAsyncImageTaskIDMissing
}

func extractPredictionIDsFromURLs(rawURLs json.RawMessage) []string {
	if len(rawURLs) == 0 {
		return nil
	}
	var candidates []string
	var urlsObject struct {
		Get string `json:"get"`
	}
	if err := common.Unmarshal(rawURLs, &urlsObject); err == nil {
		candidates = append(candidates, extractPredictionIDFromURL(urlsObject.Get))
	}
	var urlsList []struct {
		Get string `json:"get"`
	}
	if err := common.Unmarshal(rawURLs, &urlsList); err == nil {
		for _, item := range urlsList {
			candidates = append(candidates, extractPredictionIDFromURL(item.Get))
		}
	}
	return candidates
}

func extractPredictionIDFromURL(rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "predictions" && parts[i+1] != "" {
			return parts[i+1]
		}
	}
	return ""
}

func asyncImageChargedQuota(info *relaycommon.RelayInfo) int {
	if info == nil {
		return 0
	}
	if info.FinalPreConsumedQuota > 0 {
		return info.FinalPreConsumedQuota
	}
	if info.PriceData.QuotaToPreConsume > 0 {
		return info.PriceData.QuotaToPreConsume
	}
	return info.PriceData.Quota
}

func prepareGPTProtoAsyncImagePriceDataForSettlement(info *relaycommon.RelayInfo) int {
	chargedQuota := asyncImageChargedQuota(info)
	if info == nil {
		return chargedQuota
	}

	groupRatio := info.PriceData.GroupRatioInfo.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}

	info.PriceData.Quota = chargedQuota
	info.PriceData.QuotaToPreConsume = chargedQuota
	info.PriceData.ModelPrice = float64(chargedQuota) / (common.QuotaPerUnit * groupRatio)
	info.PriceData.ModelRatio = 0
	info.PriceData.CompletionRatio = 0
	info.PriceData.CacheRatio = 0
	info.PriceData.CacheCreationRatio = 0
	info.PriceData.CacheCreation5mRatio = 0
	info.PriceData.CacheCreation1hRatio = 0
	info.PriceData.ImageRatio = 0
	info.PriceData.AudioRatio = 0
	info.PriceData.AudioCompletionRatio = 0
	info.PriceData.OtherRatios = nil
	info.PriceData.BillingMode = types.BillingModePerRequest
	info.PriceData.AdvancedRuleType = ""
	info.PriceData.AdvancedRuleSnapshot = nil
	info.PriceData.AdvancedPricingContext = nil
	info.PriceData.UsePrice = true
	return chargedQuota
}

func ensureGPTProtoAsyncImageTaskRelayInfo(info *relaycommon.RelayInfo) {
	if info != nil && info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
}

func buildImageTask(taskID string, upstreamTaskID string, responseBody []byte, info *relaycommon.RelayInfo, request *dto.ImageRequest) *model.Task {
	task := model.InitTask(constant.TaskPlatformGPTProtoImage, info)
	task.TaskID = taskID
	if request != nil {
		task.Properties.ResponseFormat = strings.TrimSpace(request.ResponseFormat)
	}
	task.PrivateData.RequestId = info.RequestId
	task.PrivateData.UpstreamTaskID = upstreamTaskID
	task.PrivateData.BillingSource = info.BillingSource
	task.PrivateData.SubscriptionId = info.SubscriptionId
	task.PrivateData.TokenId = info.TokenId
	task.PrivateData.BillingContext = &model.TaskBillingContext{
		ModelPrice:             info.PriceData.ModelPrice,
		GroupRatio:             info.PriceData.GroupRatioInfo.GroupRatio,
		ModelRatio:             info.PriceData.ModelRatio,
		GroupRatioCaptured:     info.PriceData.BillingMode == types.BillingModePerToken,
		ModelRatioCaptured:     info.PriceData.BillingMode == types.BillingModePerToken,
		OtherRatios:            info.PriceData.OtherRatios,
		OriginModelName:        info.OriginModelName,
		BillingMode:            info.PriceData.BillingMode,
		AdvancedRuleType:       info.PriceData.AdvancedRuleType,
		AdvancedRuleSnapshot:   info.PriceData.AdvancedRuleSnapshot,
		AdvancedPricingContext: info.PriceData.AdvancedPricingContext,
	}
	task.Quota = asyncImageChargedQuota(info)
	task.Data = responseBody
	task.Action = constant.TaskTypeImageGeneration
	task.Status = model.TaskStatusSubmitted
	task.Progress = "10%"
	task.SubmitTime = time.Now().Unix()
	return task
}

func handleGPTProtoAsyncImageResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, request *dto.ImageRequest) (handled bool, apiErr *types.NewAPIError) {
	defer func() {
		if err := recover(); err != nil {
			logger.LogError(c, fmt.Sprintf("async image response panic: %v\n%s", err, string(debug.Stack())))
			service.CloseResponseBodyGracefully(resp)
			handled = true
			apiErr = types.NewOpenAIError(fmt.Errorf("async image response handler failed"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
	}()
	if info == nil {
		service.CloseResponseBodyGracefully(resp)
		return true, types.NewOpenAIError(fmt.Errorf("missing relay info for async image response"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if request == nil {
		service.CloseResponseBodyGracefully(resp)
		return true, types.NewOpenAIError(fmt.Errorf("missing image request for async image response"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if resp == nil || resp.Body == nil {
		return true, types.NewOpenAIError(fmt.Errorf("empty upstream image response"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		service.CloseResponseBodyGracefully(resp)
		return true, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	upstreamTaskID, err := extractGPTProtoAsyncTaskID(responseBody)
	if err != nil {
		if errors.Is(err, errAsyncImageTaskIDMissing) {
			resp.Body = io.NopCloser(bytes.NewReader(responseBody))
			return false, nil
		}
		service.CloseResponseBodyGracefully(resp)
		return true, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	ensureGPTProtoAsyncImageTaskRelayInfo(info)
	info.Action = constant.TaskTypeImageGeneration
	chargedQuota := prepareGPTProtoAsyncImagePriceDataForSettlement(info)
	publicTaskID := model.GenerateTaskID()
	task := buildImageTask(publicTaskID, upstreamTaskID, responseBody, info, request)
	if insertErr := task.Insert(); insertErr != nil {
		logger.LogError(c, "insert gptproto image task error: "+insertErr.Error())
		return true, types.NewOpenAIError(insertErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if settleErr := service.SettleBilling(c, info, chargedQuota); settleErr != nil {
		logger.LogError(c, "settle gptproto image task billing error: "+settleErr.Error())
	}
	service.LogTaskConsumption(c, info)

	c.JSON(http.StatusOK, gin.H{
		"id":      publicTaskID,
		"task_id": publicTaskID,
		"object":  "image.generation.task",
		"created": task.SubmitTime,
		"model":   request.Model,
		"status":  task.Status,
	})
	return true, nil
}
