package gptproto_image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const ChannelName = "GPTProto Image"

type TaskAdaptor struct {
	taskcommon.BaseBilling
}

func (a *TaskAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *TaskAdaptor) ValidateRequestAndSetAction(_ *gin.Context, _ *relaycommon.RelayInfo) *dto.TaskError {
	return service.TaskErrorWrapperLocal(fmt.Errorf("gptproto image submit uses /v1/images/generations"), "not_supported", http.StatusBadRequest)
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return "", fmt.Errorf("gptproto image submit uses /v1/images/generations")
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, _ *http.Request, _ *relaycommon.RelayInfo) error {
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(_ *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
	return bytes.NewBuffer(nil), nil
}

func (a *TaskAdaptor) DoRequest(_ *gin.Context, _ *relaycommon.RelayInfo, _ io.Reader) (*http.Response, error) {
	return nil, fmt.Errorf("gptproto image submit uses /v1/images/generations")
}

func (a *TaskAdaptor) DoResponse(_ *gin.Context, _ *http.Response, _ *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("gptproto image submit uses /v1/images/generations"), "not_supported", http.StatusBadRequest)
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/api/v3/predictions/%s/result", normalizeBaseURL(baseUrl), taskID)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerAuthorization(key))

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

func bearerAuthorization(key string) string {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(strings.ToLower(key), "bearer ") {
		return key
	}
	return "Bearer " + key
}

func normalizeBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if idx := strings.Index(baseURL, "/api/v3/"); idx >= 0 {
		return baseURL[:idx]
	}
	return baseURL
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var payload gptProtoTaskResponse
	if err := common.Unmarshal(respBody, &payload); err != nil {
		return nil, err
	}

	data := payload.Data
	reason := parseErrorReason(data.Error)
	taskInfo := &relaycommon.TaskInfo{
		TaskID:           data.ID,
		Reason:           reason,
		PromptTokens:     data.Usage.InputTokens,
		CompletionTokens: data.Usage.OutputTokens,
		TotalTokens:      data.Usage.TotalTokens,
	}
	if taskInfo.TotalTokens <= 0 && (taskInfo.PromptTokens > 0 || taskInfo.CompletionTokens > 0) {
		taskInfo.TotalTokens = taskInfo.PromptTokens + taskInfo.CompletionTokens
	}

	switch strings.ToLower(strings.TrimSpace(data.Status)) {
	case "starting", "queued", "pending":
		taskInfo.Status = model.TaskStatusQueued
		taskInfo.Progress = taskcommon.ProgressQueued
	case "processing", "running", "in_progress":
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = taskcommon.ProgressInProgress
	case "completed", "complete", "succeeded", "success":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = taskcommon.ProgressComplete
		taskInfo.Url = firstOutputURL(data)
	case "failed", "failure", "canceled", "cancelled":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = taskcommon.ProgressComplete
		if taskInfo.Reason == "" {
			taskInfo.Reason = data.Status
		}
	default:
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = taskcommon.ProgressInProgress
	}
	return taskInfo, nil
}

func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskInfo *relaycommon.TaskInfo) int {
	if task == nil || taskInfo == nil || task.PrivateData.BillingContext == nil {
		return 0
	}
	billingContext := task.PrivateData.BillingContext
	if billingContext.BillingMode != types.BillingModeAdvanced {
		return 0
	}
	ruleType := billingContext.AdvancedRuleType
	if ruleType == "" && billingContext.AdvancedRuleSnapshot != nil {
		ruleType = billingContext.AdvancedRuleSnapshot.RuleType
	}
	if ruleType != types.AdvancedRuleTypeTextSegment {
		return 0
	}
	billingUnit := ""
	priceSnapshot := types.AdvancedRulePriceSnapshot{}
	if billingContext.AdvancedRuleSnapshot != nil {
		billingUnit = strings.TrimSpace(billingContext.AdvancedRuleSnapshot.BillingUnit)
		priceSnapshot = billingContext.AdvancedRuleSnapshot.PriceSnapshot
	}
	if billingUnit != "" && billingUnit != types.AdvancedBillingUnitPerMillionTokens {
		return 0
	}
	if priceSnapshot.InputPrice == nil || priceSnapshot.OutputPrice == nil {
		return 0
	}

	promptTokens := taskInfo.PromptTokens
	completionTokens := taskInfo.CompletionTokens
	if promptTokens <= 0 && taskInfo.TotalTokens > 0 && completionTokens > 0 && completionTokens <= taskInfo.TotalTokens {
		promptTokens = taskInfo.TotalTokens - completionTokens
	}
	if completionTokens <= 0 && taskInfo.TotalTokens > 0 && promptTokens > 0 && promptTokens <= taskInfo.TotalTokens {
		completionTokens = taskInfo.TotalTokens - promptTokens
	}
	if promptTokens < 0 || completionTokens < 0 || (promptTokens == 0 && completionTokens == 0) {
		return 0
	}

	groupRatio := billingContext.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}

	actualQuota := decimal.NewFromFloat(*priceSnapshot.InputPrice).
		Mul(decimal.NewFromInt(int64(promptTokens))).
		Add(decimal.NewFromFloat(*priceSnapshot.OutputPrice).Mul(decimal.NewFromInt(int64(completionTokens)))).
		Div(decimal.NewFromInt(1000000)).
		Mul(decimal.NewFromFloat(groupRatio)).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		IntPart()
	return int(actualQuota)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"gpt-image-2", "gpt-image-2-plus"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

type gptProtoTaskResponse struct {
	Data gptProtoTaskData `json:"data"`
}

type gptProtoTaskData struct {
	ID       string          `json:"id"`
	Status   string          `json:"status"`
	URL      string          `json:"url"`
	ImageURL string          `json:"image_url"`
	Output   any             `json:"output"`
	Outputs  []any           `json:"outputs"`
	Error    json.RawMessage `json:"error"`
	Usage    gptProtoUsage   `json:"usage"`
}

type gptProtoUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func parseErrorReason(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var message string
	if err := common.Unmarshal(raw, &message); err == nil {
		return message
	}
	var obj struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := common.Unmarshal(raw, &obj); err == nil {
		if obj.Message != "" {
			return obj.Message
		}
		return obj.Code
	}
	return ""
}

func firstOutputURL(data gptProtoTaskData) string {
	if data.ImageURL != "" {
		return data.ImageURL
	}
	if data.URL != "" {
		return data.URL
	}
	if url := outputURL(data.Output); url != "" {
		return url
	}
	for _, item := range data.Outputs {
		if url := outputURL(item); url != "" {
			return url
		}
	}
	return ""
}

func outputURL(item any) string {
	switch v := item.(type) {
	case string:
		return v
	case map[string]any:
		for _, key := range []string{"url", "image_url", "image"} {
			if s, ok := v[key].(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
