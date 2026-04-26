package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const (
	modelMonitorNotifyMaxDetails = 10
)

var (
	modelMonitorTaskOnce    sync.Once
	modelMonitorTaskRunning atomic.Bool

	notifyModelMonitorAdminsByEmail = service.NotifyAdminUsersByEmail
)

type modelMonitorRunOptions struct {
	Manual       bool
	Model        string
	ChannelID    int
	EndpointType string
	Stream       bool
}

type modelMonitorPair struct {
	Model  string
	Target modelMonitorChannelTarget
}

type modelMonitorFailureNotificationItem struct {
	Model               string
	ChannelID           int
	ChannelName         string
	Status              string
	Error               string
	ConsecutiveFailures int
}

func StartModelMonitorTask() {
	modelMonitorTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			common.SysLog("model monitor task started")
			for {
				setting := currentModelMonitorSetting()
				if !modelMonitorEnabled(setting) {
					time.Sleep(time.Minute)
					continue
				}
				if _, err := runModelMonitorOnce(modelMonitorRunOptions{}); err != nil {
					common.SysLog(fmt.Sprintf("model monitor task failed: %v", err))
				}
				time.Sleep(modelMonitorIntervalDuration(setting))
			}
		}()
	})
}

func modelMonitorIntervalDuration(setting *operation_setting.ModelMonitorSetting) time.Duration {
	minutes := modelMonitorIntervalMinutes(setting)
	if minutes <= 0 {
		minutes = operation_setting.GetModelMonitorSetting().IntervalMinutes
	}
	interval := time.Duration(minutes) * time.Minute
	if interval < time.Second {
		return time.Duration(operation_setting.GetModelMonitorSetting().IntervalMinutes) * time.Minute
	}
	return interval
}

func runModelMonitorOnce(opts modelMonitorRunOptions) (modelMonitorStateResponse, error) {
	if !modelMonitorTaskRunning.CompareAndSwap(false, true) {
		return modelMonitorStateResponse{}, fmt.Errorf("model monitor is already running")
	}
	defer modelMonitorTaskRunning.Store(false)
	return runModelMonitorOnceLocked(opts)
}

func startModelMonitorRunAsync(opts modelMonitorRunOptions) error {
	if !modelMonitorTaskRunning.CompareAndSwap(false, true) {
		return fmt.Errorf("model monitor is already running")
	}
	go func() {
		defer modelMonitorTaskRunning.Store(false)
		if _, err := runModelMonitorOnceLocked(opts); err != nil {
			common.SysLog(fmt.Sprintf("model monitor async run failed: %v", err))
		}
	}()
	return nil
}

func runModelMonitorOnceLocked(opts modelMonitorRunOptions) (modelMonitorStateResponse, error) {
	setting := currentModelMonitorSetting()
	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		return modelMonitorStateResponse{}, err
	}
	targets := buildModelMonitorTargets(channels, modelMonitorFilterAdapter{setting: setting})
	targets = filterModelMonitorTargets(targets, opts)

	records, err := loadModelMonitorStatusRecords(modelNamesFromTargets(targets))
	if err != nil {
		return modelMonitorStateResponse{}, err
	}
	statusMap := buildModelMonitorStatusMap(records)
	pairs := flattenModelMonitorTargets(targets)

	batchSize := modelMonitorBatchSize(setting)
	threshold := modelMonitorFailureThreshold(setting)
	notifications := make([]modelMonitorFailureNotificationItem, 0)
	processed := 0
	for start := 0; start < len(pairs); start += batchSize {
		end := start + batchSize
		if end > len(pairs) {
			end = len(pairs)
		}
		for _, pair := range pairs[start:end] {
			notice, shouldNotify := testAndPersistModelMonitorPair(pair, setting, opts, statusMap, threshold)
			if shouldNotify {
				notifications = append(notifications, notice)
			}
			processed++
			if common.RequestInterval > 0 {
				time.Sleep(common.RequestInterval)
			}
		}
	}

	if len(notifications) > 0 {
		notifyModelMonitorFailures(threshold, notifications)
	}

	state := withModelMonitorRunning(buildModelMonitorStateFromTargets(setting, targets, statusMap))
	if processed > 0 || common.DebugEnabled {
		common.SysLog(fmt.Sprintf(
			"model monitor run done: models=%d targets=%d success=%d failed=%d timeout=%d",
			len(state.Items),
			processed,
			state.Summary.SuccessCount,
			state.Summary.FailedCount,
			state.Summary.TimeoutCount,
		))
	}
	return state, nil
}

func filterModelMonitorTargets(targets []modelMonitorTarget, opts modelMonitorRunOptions) []modelMonitorTarget {
	modelNameFilter := strings.TrimSpace(opts.Model)
	filtered := make([]modelMonitorTarget, 0, len(targets))
	for _, target := range targets {
		if modelNameFilter != "" && target.Model != modelNameFilter {
			continue
		}
		channels := make([]modelMonitorChannelTarget, 0, len(target.Channels))
		for _, channelTarget := range target.Channels {
			if opts.ChannelID > 0 && channelTarget.Channel.Id != opts.ChannelID {
				continue
			}
			channels = append(channels, channelTarget)
		}
		if len(channels) == 0 {
			continue
		}
		target.Channels = channels
		filtered = append(filtered, target)
	}
	return filtered
}

func flattenModelMonitorTargets(targets []modelMonitorTarget) []modelMonitorPair {
	pairs := make([]modelMonitorPair, 0)
	for _, target := range targets {
		for _, channelTarget := range target.Channels {
			pairs = append(pairs, modelMonitorPair{
				Model:  target.Model,
				Target: channelTarget,
			})
		}
	}
	return pairs
}

func shouldRunModelMonitorTarget(setting *operation_setting.ModelMonitorSetting, modelName string, manual bool) bool {
	if setting == nil {
		return false
	}
	if !manual && !setting.Enabled {
		return false
	}
	return setting.ModelEnabled(modelName)
}

func testAndPersistModelMonitorPair(
	pair modelMonitorPair,
	setting *operation_setting.ModelMonitorSetting,
	opts modelMonitorRunOptions,
	statusMap map[modelMonitorStatusKey]modelMonitorStatusRecord,
	threshold int,
) (modelMonitorFailureNotificationItem, bool) {
	key := modelMonitorRecordKey(pair.Model, pair.Target.Channel.Id)
	previous := statusMap[key]
	if !shouldRunModelMonitorTarget(setting, pair.Model, opts.Manual) {
		record := &model.ModelMonitorStatus{
			ModelName:      pair.Model,
			ChannelId:      pair.Target.Channel.Id,
			ChannelName:    pair.Target.Channel.Name,
			ChannelType:    pair.Target.Channel.Type,
			Status:         modelMonitorChannelStatusSkipped,
			ResponseTimeMs: 0,
			ErrorMessage:   "",
			TestedAt:       common.GetTimestamp(),
		}
		if err := model.UpsertModelMonitorStatus(record); err != nil {
			common.SysLog(fmt.Sprintf(
				"failed to persist skipped model monitor status: model=%s channel_id=%d err=%v",
				pair.Model,
				pair.Target.Channel.Id,
				err,
			))
			record.ConsecutiveFailures = 0
		}
		statusMap[key] = *record
		return modelMonitorFailureNotificationItem{}, false
	}

	detail := testModelMonitorChannel(pair.Target.Channel, pair.Model, opts.EndpointType, opts.Stream, modelMonitorTimeoutSeconds(setting, pair.Model))

	record := &model.ModelMonitorStatus{
		ModelName:           pair.Model,
		ChannelId:           detail.ChannelId,
		ChannelName:         detail.ChannelName,
		ChannelType:         detail.ChannelType,
		Status:              detail.Status,
		ResponseTimeMs:      detail.ResponseTimeMs,
		ErrorMessage:        detail.ErrorMessage,
		TestedAt:            detail.TestedAt,
		ConsecutiveFailures: detail.ConsecutiveFailures,
	}
	if err := model.UpsertModelMonitorStatus(record); err != nil {
		common.SysLog(fmt.Sprintf(
			"failed to persist model monitor status: model=%s channel_id=%d err=%v",
			pair.Model,
			pair.Target.Channel.Id,
			err,
		))
		if isModelMonitorFailureStatus(detail.Status) {
			detail.ConsecutiveFailures = previous.ConsecutiveFailures + 1
		} else {
			detail.ConsecutiveFailures = 0
		}
		record.ConsecutiveFailures = detail.ConsecutiveFailures
	} else {
		detail.ConsecutiveFailures = record.ConsecutiveFailures
	}
	statusMap[key] = *record

	if threshold <= 0 || !isModelMonitorFailureStatus(detail.Status) ||
		detail.ConsecutiveFailures < threshold || previous.ConsecutiveFailures >= threshold {
		return modelMonitorFailureNotificationItem{}, false
	}
	return modelMonitorFailureNotificationItem{
		Model:               pair.Model,
		ChannelID:           detail.ChannelId,
		ChannelName:         detail.ChannelName,
		Status:              detail.Status,
		Error:               detail.ErrorMessage,
		ConsecutiveFailures: detail.ConsecutiveFailures,
	}, true
}

func testModelMonitorChannel(channel *model.Channel, modelName string, endpointType string, stream bool, timeoutSeconds int) modelMonitorChannelDetail {
	if timeoutSeconds <= 0 {
		timeoutSeconds = operation_setting.GetModelMonitorSetting().DefaultTimeoutSeconds
	}
	detail := modelMonitorChannelDetail{
		ChannelId:   channel.Id,
		ChannelName: channel.Name,
		ChannelType: channel.Type,
		TestedAt:    common.GetTimestamp(),
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	result := testChannelWithContext(ctx, channel, modelName, endpointType, stream)

	detail.ResponseTimeMs = int(time.Since(start).Milliseconds())
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		detail.Status = modelMonitorChannelStatusTimeout
		detail.ErrorMessage = fmt.Sprintf("model monitor timeout after %ds", timeoutSeconds)
		return detail
	}
	if result.localErr != nil {
		if errors.Is(result.localErr, context.DeadlineExceeded) {
			detail.Status = modelMonitorChannelStatusTimeout
			detail.ErrorMessage = fmt.Sprintf("model monitor timeout after %ds", timeoutSeconds)
			return detail
		}
		detail.Status = modelMonitorChannelStatusFailure
		detail.ErrorMessage = result.localErr.Error()
		return detail
	}
	if result.newAPIError != nil {
		if errors.Is(result.newAPIError, context.DeadlineExceeded) {
			detail.Status = modelMonitorChannelStatusTimeout
			detail.ErrorMessage = fmt.Sprintf("model monitor timeout after %ds", timeoutSeconds)
			return detail
		}
		detail.Status = modelMonitorChannelStatusFailure
		detail.ErrorMessage = result.newAPIError.Error()
		return detail
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		detail.Status = modelMonitorChannelStatusTimeout
		detail.ErrorMessage = fmt.Sprintf("model monitor timeout after %ds", timeoutSeconds)
		return detail
	}
	detail.Status = modelMonitorChannelStatusSuccess
	return detail
}

func isModelMonitorFailureStatus(status string) bool {
	return status == modelMonitorChannelStatusFailure || status == modelMonitorChannelStatusTimeout
}

func notifyModelMonitorFailures(threshold int, items []modelMonitorFailureNotificationItem) {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Model monitor failure threshold reached: threshold=%d, affected_targets=%d", threshold, len(items)))
	displayCount := len(items)
	if displayCount > modelMonitorNotifyMaxDetails {
		displayCount = modelMonitorNotifyMaxDetails
	}
	for _, item := range items[:displayCount] {
		builder.WriteString(fmt.Sprintf(
			"\n- model=%s channel_id=%d channel_name=%s status=%s consecutive_failures=%d",
			item.Model,
			item.ChannelID,
			item.ChannelName,
			item.Status,
			item.ConsecutiveFailures,
		))
		if item.Error != "" {
			builder.WriteString(" error=")
			builder.WriteString(item.Error)
		}
	}
	if len(items) > displayCount {
		builder.WriteString(fmt.Sprintf("\n- omitted_targets=%d", len(items)-displayCount))
	}
	notifyModelMonitorAdminsByEmail(dto.NotifyTypeModelMonitor, "Model monitor failure threshold reached", builder.String())
}
