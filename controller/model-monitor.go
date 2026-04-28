package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

const (
	modelMonitorConfigName = "model_monitor_setting"

	modelMonitorStatusHealthy     = "healthy"
	modelMonitorStatusPartial     = "partial"
	modelMonitorStatusUnavailable = "unavailable"
	modelMonitorStatusSkipped     = "skipped"
	modelMonitorStatusUnknown     = "unknown"
	modelMonitorStatusNoChannels  = "no_channels"

	modelMonitorChannelStatusSuccess     = model.ModelMonitorStatusSuccess
	modelMonitorChannelStatusFailure     = model.ModelMonitorStatusFailed
	modelMonitorChannelStatusTimeout     = model.ModelMonitorStatusTimeout
	modelMonitorChannelStatusSkipped     = model.ModelMonitorStatusSkipped
	modelMonitorChannelStatusUnavailable = "unavailable"
	modelMonitorChannelStatusUnknown     = "unknown"
)

type modelMonitorModelFilter interface {
	ModelEnabled(modelName string) bool
}

type modelMonitorTimeoutProvider interface {
	TimeoutSecondsForModel(modelName string) int
}

type modelMonitorTarget struct {
	Model    string
	Channels []modelMonitorChannelTarget
}

type modelMonitorChannelTarget struct {
	Channel *model.Channel
}

type modelMonitorStateResponse = dto.ModelMonitorResponse
type modelMonitorItem = dto.ModelMonitorItem
type modelMonitorChannelDetail = dto.ModelMonitorChannelItem
type modelMonitorStatusRecord = model.ModelMonitorStatus

type modelMonitorStatusKey struct {
	modelName string
	channelID int
}

type modelMonitorTestRequest struct {
	Model        string `json:"model"`
	ChannelID    int    `json:"channel_id"`
	EndpointType string `json:"endpoint_type"`
	Stream       bool   `json:"stream"`
}

func currentModelMonitorSetting() *operation_setting.ModelMonitorSetting {
	return operation_setting.GetModelMonitorSetting()
}

func modelMonitorSettingModelEnabled(setting any, modelName string) bool {
	if setting == nil {
		return true
	}
	if filter, ok := setting.(modelMonitorModelFilter); ok {
		return filter.ModelEnabled(modelName)
	}
	return true
}

func modelMonitorTimeoutSeconds(setting any, modelName string) int {
	if provider, ok := setting.(modelMonitorTimeoutProvider); ok {
		if seconds := provider.TimeoutSecondsForModel(modelName); seconds > 0 {
			return seconds
		}
	}
	return operation_setting.GetModelMonitorSetting().TimeoutSecondsForModel(modelName)
}

func modelMonitorEnabled(setting *operation_setting.ModelMonitorSetting) bool {
	return setting != nil && setting.Enabled
}

func modelMonitorIntervalMinutes(setting *operation_setting.ModelMonitorSetting) int {
	if setting == nil || setting.IntervalMinutes < 1 {
		return operation_setting.GetModelMonitorSetting().IntervalMinutes
	}
	return setting.IntervalMinutes
}

func modelMonitorBatchSize(setting *operation_setting.ModelMonitorSetting) int {
	if setting == nil || setting.BatchSize < 1 {
		return operation_setting.GetModelMonitorSetting().BatchSize
	}
	return setting.BatchSize
}

func modelMonitorFailureThreshold(setting *operation_setting.ModelMonitorSetting) int {
	if setting == nil || setting.FailureThreshold < 1 {
		return operation_setting.GetModelMonitorSetting().FailureThreshold
	}
	return setting.FailureThreshold
}

func isUnsupportedModelMonitorTarget(channel *model.Channel, modelName string) bool {
	if channel == nil {
		return true
	}
	if isUnsupportedChannelTestType(channel.Type) {
		return true
	}
	resolvedModelName := resolveChannelTestModelName(channel, strings.TrimSpace(modelName))
	return channel.Type == constant.ChannelTypeVolcEngine && isSeedanceChannelTestModel(resolvedModelName)
}

func buildModelMonitorTargets(channels []*model.Channel, _ modelMonitorModelFilter) []modelMonitorTarget {
	grouped := make(map[string][]modelMonitorChannelTarget)
	for _, channel := range channels {
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		seenInChannel := make(map[string]struct{})
		for _, rawModelName := range channel.GetModels() {
			modelName := strings.TrimSpace(rawModelName)
			if modelName == "" {
				continue
			}
			if isUnsupportedModelMonitorTarget(channel, modelName) {
				continue
			}
			if _, exists := seenInChannel[modelName]; exists {
				continue
			}
			seenInChannel[modelName] = struct{}{}
			grouped[modelName] = append(grouped[modelName], modelMonitorChannelTarget{Channel: channel})
		}
	}

	modelNames := make([]string, 0, len(grouped))
	for modelName := range grouped {
		modelNames = append(modelNames, modelName)
	}
	sort.Strings(modelNames)

	targets := make([]modelMonitorTarget, 0, len(modelNames))
	for _, modelName := range modelNames {
		channelsForModel := grouped[modelName]
		sort.Slice(channelsForModel, func(i, j int) bool {
			return channelsForModel[i].Channel.Id < channelsForModel[j].Channel.Id
		})
		targets = append(targets, modelMonitorTarget{
			Model:    modelName,
			Channels: channelsForModel,
		})
	}
	return targets
}

func aggregateModelMonitorItem(modelName string, details []modelMonitorChannelDetail) modelMonitorItem {
	item := modelMonitorItem{
		ModelName:    modelName,
		Status:       aggregateModelMonitorStatus(details),
		ChannelCount: len(details),
		Channels:     details,
	}
	for _, detail := range details {
		if detail.TestedAt > item.TestedAt {
			item.TestedAt = detail.TestedAt
		}
		if detail.ConsecutiveFailures > item.ConsecutiveFailures {
			item.ConsecutiveFailures = detail.ConsecutiveFailures
		}
		switch detail.Status {
		case modelMonitorChannelStatusSuccess:
			item.SuccessCount++
		case modelMonitorChannelStatusFailure, modelMonitorChannelStatusTimeout, modelMonitorChannelStatusUnavailable:
			item.FailedCount++
		case modelMonitorChannelStatusSkipped:
			item.SkippedCount++
		}
	}
	return item
}

func aggregateModelMonitorStatus(details []modelMonitorChannelDetail) string {
	if len(details) == 0 {
		return modelMonitorStatusNoChannels
	}

	successCount := 0
	failureCount := 0
	skippedCount := 0
	unknownCount := 0
	for _, detail := range details {
		switch detail.Status {
		case modelMonitorChannelStatusSuccess:
			successCount++
		case modelMonitorChannelStatusFailure, modelMonitorChannelStatusTimeout, modelMonitorChannelStatusUnavailable:
			failureCount++
		case modelMonitorChannelStatusSkipped:
			skippedCount++
		case modelMonitorChannelStatusUnknown:
			unknownCount++
		}
	}
	if skippedCount == len(details) {
		return modelMonitorStatusSkipped
	}
	if successCount == len(details) {
		return modelMonitorStatusHealthy
	}
	if successCount > 0 && failureCount > 0 {
		return modelMonitorStatusPartial
	}
	if failureCount > 0 {
		return modelMonitorStatusUnavailable
	}
	if unknownCount > 0 {
		return modelMonitorStatusUnknown
	}
	return modelMonitorStatusUnknown
}

func summarizeModelMonitorItems(items []modelMonitorItem) dto.ModelMonitorSummary {
	summary := dto.ModelMonitorSummary{TotalModels: len(items)}
	for _, item := range items {
		if item.Enabled {
			summary.EnabledModels++
		} else {
			summary.DisabledModels++
		}
		if item.TestedAt > summary.LastTestedAt {
			summary.LastTestedAt = item.TestedAt
		}
		switch item.Status {
		case modelMonitorStatusHealthy:
			summary.HealthyModels++
		case modelMonitorStatusPartial:
			summary.PartialModels++
		case modelMonitorStatusUnavailable, modelMonitorStatusNoChannels:
			summary.UnavailableModels++
		case modelMonitorStatusSkipped:
			summary.SkippedModels++
		}
		for _, detail := range item.Channels {
			summary.TotalChannels++
			if detail.TestedAt > summary.LastTestedAt {
				summary.LastTestedAt = detail.TestedAt
			}
			switch detail.Status {
			case modelMonitorChannelStatusSuccess:
				summary.SuccessCount++
			case modelMonitorChannelStatusFailure:
				summary.FailedCount++
				summary.FailedChannels++
			case modelMonitorChannelStatusTimeout:
				summary.TimeoutCount++
				summary.FailedChannels++
			case modelMonitorChannelStatusSkipped:
				summary.SkippedCount++
			case modelMonitorChannelStatusUnavailable:
				summary.FailedChannels++
			}
		}
	}
	return summary
}

func modelNamesFromTargets(targets []modelMonitorTarget) []string {
	modelNames := make([]string, 0, len(targets))
	for _, target := range targets {
		modelNames = append(modelNames, target.Model)
	}
	return modelNames
}

func modelMonitorSettingsDTO(setting *operation_setting.ModelMonitorSetting) dto.ModelMonitorSettingsUpdateRequest {
	if setting == nil {
		setting = operation_setting.GetModelMonitorSetting()
	}
	setting.Normalize()
	overrides := make(map[string]dto.ModelMonitorModelOverrideDTO, len(setting.ModelOverrides))
	for modelName, override := range setting.ModelOverrides {
		overrides[modelName] = dto.ModelMonitorModelOverrideDTO{
			Enabled:        override.Enabled,
			TimeoutSeconds: override.TimeoutSeconds,
		}
	}
	return dto.ModelMonitorSettingsUpdateRequest{
		Enabled:                     setting.Enabled,
		IntervalMinutes:             setting.IntervalMinutes,
		BatchSize:                   setting.BatchSize,
		DefaultTimeoutSeconds:       setting.DefaultTimeoutSeconds,
		FailureThreshold:            setting.FailureThreshold,
		ExcludedModelPatterns:       append([]string(nil), setting.ExcludedModelPatterns...),
		ModelOverrides:              overrides,
		NotificationDisabledUserIds: append([]int(nil), setting.NotificationDisabledUserIds...),
	}
}

func modelMonitorRecordKey(modelName string, channelID int) modelMonitorStatusKey {
	return modelMonitorStatusKey{modelName: strings.TrimSpace(modelName), channelID: channelID}
}

func buildModelMonitorStatusMap(records []modelMonitorStatusRecord) map[modelMonitorStatusKey]modelMonitorStatusRecord {
	statusMap := make(map[modelMonitorStatusKey]modelMonitorStatusRecord, len(records))
	for _, record := range records {
		statusMap[modelMonitorRecordKey(record.ModelName, record.ChannelId)] = record
	}
	return statusMap
}

func loadModelMonitorStatusRecords(modelNames []string) ([]modelMonitorStatusRecord, error) {
	if len(modelNames) == 0 {
		return nil, nil
	}
	records, err := model.ListModelMonitorStatusesByModels(modelNames)
	if err != nil {
		return nil, err
	}
	result := make([]modelMonitorStatusRecord, 0, len(records))
	for _, record := range records {
		if record != nil {
			result = append(result, *record)
		}
	}
	return result, nil
}

func detailFromStatusRecord(target modelMonitorChannelTarget, record modelMonitorStatusRecord, exists bool) modelMonitorChannelDetail {
	channel := target.Channel
	detail := modelMonitorChannelDetail{
		ChannelId:   channel.Id,
		ChannelName: channel.Name,
		ChannelType: channel.Type,
		Status:      modelMonitorChannelStatusUnknown,
	}
	if !exists {
		return detail
	}
	detail.Status = record.Status
	detail.ResponseTimeMs = record.ResponseTimeMs
	detail.ErrorMessage = record.ErrorMessage
	detail.TestedAt = record.TestedAt
	detail.ConsecutiveFailures = record.ConsecutiveFailures
	return detail
}

func buildModelMonitorStateFromTargets(setting *operation_setting.ModelMonitorSetting, targets []modelMonitorTarget, statusMap map[modelMonitorStatusKey]modelMonitorStatusRecord) modelMonitorStateResponse {
	items := make([]modelMonitorItem, 0, len(targets))
	for _, target := range targets {
		if setting != nil && setting.ModelExcluded(target.Model) {
			continue
		}
		enabled := modelMonitorSettingModelEnabled(setting, target.Model)
		details := make([]modelMonitorChannelDetail, 0, len(target.Channels))
		for _, channelTarget := range target.Channels {
			record, exists := statusMap[modelMonitorRecordKey(target.Model, channelTarget.Channel.Id)]
			detail := detailFromStatusRecord(channelTarget, record, exists)
			if !enabled {
				detail.Status = modelMonitorChannelStatusSkipped
			}
			details = append(details, detail)
		}
		item := aggregateModelMonitorItem(target.Model, details)
		item.Enabled = enabled
		item.TimeoutSeconds = modelMonitorTimeoutSeconds(setting, target.Model)
		items = append(items, item)
	}
	return modelMonitorStateResponse{
		Settings: modelMonitorSettingsDTO(setting),
		Summary:  summarizeModelMonitorItems(items),
		Items:    items,
	}
}

func withModelMonitorRunning(state modelMonitorStateResponse) modelMonitorStateResponse {
	state.Running = modelMonitorTaskRunning.Load()
	return state
}

func loadModelMonitorState() (modelMonitorStateResponse, error) {
	setting := currentModelMonitorSetting()
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		return modelMonitorStateResponse{}, err
	}
	targets := buildModelMonitorTargets(channels, modelMonitorFilterAdapter{setting: setting})
	records, err := loadModelMonitorStatusRecords(modelNamesFromTargets(targets))
	if err != nil {
		return modelMonitorStateResponse{}, err
	}
	return withModelMonitorRunning(buildModelMonitorStateFromTargets(setting, targets, buildModelMonitorStatusMap(records))), nil
}

type modelMonitorFilterAdapter struct {
	setting any
}

func (a modelMonitorFilterAdapter) ModelEnabled(modelName string) bool {
	return modelMonitorSettingModelEnabled(a.setting, modelName)
}

func GetModelMonitor(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceModelMonitorManagement, service.ActionRead) {
		return
	}
	state, err := loadModelMonitorState()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, state)
}

func GetModelMonitorNotificationUsers(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceModelMonitorManagement, service.ActionUpdate) {
		return
	}
	users, err := service.ListModelMonitorNotificationUsers(currentModelMonitorSetting())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, users)
}

func UpdateModelMonitorSetting(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceModelMonitorManagement, service.ActionUpdate) {
		return
	}
	var req map[string]any
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request body",
		})
		return
	}

	payload := normalizeModelMonitorSettingPayload(req)
	if len(payload) == 0 {
		common.ApiErrorMsg(c, "empty model monitor setting payload")
		return
	}

	for key, value := range payload {
		optionKey, err := normalizeModelMonitorOptionKey(key)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		optionValue, err := modelMonitorOptionValue(value)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if err := model.UpdateOption(optionKey, optionValue); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	state, err := loadModelMonitorState()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, state)
}

func normalizeModelMonitorSettingPayload(req map[string]any) map[string]any {
	if raw, ok := req["settings"]; ok {
		if nested, ok := raw.(map[string]any); ok {
			return nested
		}
	}
	if raw, ok := req["setting"]; ok {
		if nested, ok := raw.(map[string]any); ok {
			return nested
		}
	}
	return req
}

func normalizeModelMonitorOptionKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("empty model monitor setting key")
	}
	if strings.Contains(key, ".") {
		if !strings.HasPrefix(key, modelMonitorConfigName+".") {
			return "", fmt.Errorf("invalid model monitor setting key: %s", key)
		}
		return key, nil
	}
	return modelMonitorConfigName + "." + key, nil
}

func modelMonitorOptionValue(value any) (string, error) {
	switch value.(type) {
	case string, int, float64, bool, nil:
		return common.Interface2String(value), nil
	default:
		data, err := common.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func TestModelMonitor(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceModelMonitorManagement, service.ActionTest) {
		return
	}
	if !requireAdminActionPermission(c, service.ResourceModelMonitorManagement, service.ActionRead) {
		return
	}
	req := modelMonitorTestRequest{}
	if c.Request != nil && c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := common.DecodeJson(c.Request.Body, &req); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "invalid request body",
			})
			return
		}
	}

	if err := startModelMonitorRunAsync(modelMonitorRunOptions{
		Manual:       true,
		Model:        req.Model,
		ChannelID:    req.ChannelID,
		EndpointType: req.EndpointType,
		Stream:       req.Stream,
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	state, err := loadModelMonitorState()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, state)
}
