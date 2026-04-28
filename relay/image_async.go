package relay

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func isGPTProtoAsyncImageRequest(request *dto.ImageRequest) bool {
	return request != nil &&
		request.EnableSyncMode != nil &&
		!*request.EnableSyncMode &&
		strings.HasPrefix(request.Model, "gpt-image-2")
}

func extractGPTProtoAsyncTaskID(responseBody []byte) (string, error) {
	var payload struct {
		ID   string `json:"id"`
		Data struct {
			ID   string `json:"id"`
			URLs struct {
				Get string `json:"get"`
			} `json:"urls"`
		} `json:"data"`
	}
	if err := common.Unmarshal(responseBody, &payload); err != nil {
		return "", err
	}
	for _, candidate := range []string{payload.Data.ID, payload.ID, extractPredictionIDFromURL(payload.Data.URLs.Get)} {
		if strings.TrimSpace(candidate) != "" {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("gptproto async image response missing task id")
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

func buildImageTask(taskID string, upstreamTaskID string, responseBody []byte, info *relaycommon.RelayInfo) *model.Task {
	task := model.InitTask(constant.TaskPlatformGPTProtoImage, info)
	task.TaskID = taskID
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

func handleGPTProtoAsyncImageResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, request *dto.ImageRequest) *types.NewAPIError {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	upstreamTaskID, err := extractGPTProtoAsyncTaskID(responseBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	info.Action = constant.TaskTypeImageGeneration
	publicTaskID := model.GenerateTaskID()
	task := buildImageTask(publicTaskID, upstreamTaskID, responseBody, info)
	if insertErr := task.Insert(); insertErr != nil {
		logger.LogError(c, "insert gptproto image task error: "+insertErr.Error())
	} else {
		chargedQuota := asyncImageChargedQuota(info)
		info.PriceData.Quota = chargedQuota
		if settleErr := service.SettleBilling(c, info, chargedQuota); settleErr != nil {
			logger.LogError(c, "settle gptproto image task billing error: "+settleErr.Error())
		}
		service.LogTaskConsumption(c, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      publicTaskID,
		"task_id": publicTaskID,
		"object":  "image.generation.task",
		"created": task.SubmitTime,
		"model":   request.Model,
		"status":  task.Status,
	})
	return nil
}
