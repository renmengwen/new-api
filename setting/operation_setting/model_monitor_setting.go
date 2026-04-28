package operation_setting

import (
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

const (
	defaultModelMonitorIntervalMinutes  = 10
	defaultModelMonitorBatchSize        = 50
	defaultModelMonitorTimeoutSeconds   = 30
	defaultModelMonitorFailureThreshold = 3
)

type ModelMonitorModelOverride struct {
	Enabled        *bool `json:"enabled,omitempty"`
	TimeoutSeconds int   `json:"timeout_seconds,omitempty"`
}

type ModelMonitorSetting struct {
	Enabled                     bool                                 `json:"enabled"`
	IntervalMinutes             int                                  `json:"interval_minutes"`
	BatchSize                   int                                  `json:"batch_size"`
	DefaultTimeoutSeconds       int                                  `json:"default_timeout_seconds"`
	FailureThreshold            int                                  `json:"failure_threshold"`
	ExcludedModelPatterns       []string                             `json:"excluded_model_patterns"`
	ModelOverrides              map[string]ModelMonitorModelOverride `json:"model_overrides"`
	NotificationDisabledUserIds []int                                `json:"notification_disabled_user_ids"`
}

var modelMonitorSetting = ModelMonitorSetting{
	Enabled:                     false,
	IntervalMinutes:             defaultModelMonitorIntervalMinutes,
	BatchSize:                   defaultModelMonitorBatchSize,
	DefaultTimeoutSeconds:       defaultModelMonitorTimeoutSeconds,
	FailureThreshold:            defaultModelMonitorFailureThreshold,
	ExcludedModelPatterns:       []string{},
	ModelOverrides:              map[string]ModelMonitorModelOverride{},
	NotificationDisabledUserIds: []int{},
}

var modelMonitorSettingMu sync.RWMutex

func init() {
	config.GlobalConfig.Register("model_monitor_setting", &modelMonitorSetting)
}

func (s *ModelMonitorSetting) Normalize() {
	if s == nil {
		return
	}
	if s.IntervalMinutes < 1 {
		s.IntervalMinutes = defaultModelMonitorIntervalMinutes
	}
	if s.BatchSize < 1 {
		s.BatchSize = defaultModelMonitorBatchSize
	}
	if s.DefaultTimeoutSeconds < 1 {
		s.DefaultTimeoutSeconds = defaultModelMonitorTimeoutSeconds
	}
	if s.FailureThreshold < 1 {
		s.FailureThreshold = defaultModelMonitorFailureThreshold
	}
	if s.ExcludedModelPatterns == nil {
		s.ExcludedModelPatterns = []string{}
	}
	if s.ModelOverrides == nil {
		s.ModelOverrides = map[string]ModelMonitorModelOverride{}
	}
	if s.NotificationDisabledUserIds == nil {
		s.NotificationDisabledUserIds = []int{}
	}
}

func GetModelMonitorSetting() *ModelMonitorSetting {
	modelMonitorSettingMu.Lock()
	defer modelMonitorSettingMu.Unlock()
	modelMonitorSetting.Normalize()
	snapshot := modelMonitorSetting.Clone()
	return &snapshot
}

func UpdateModelMonitorSettingFromMap(configMap map[string]string) error {
	modelMonitorSettingMu.Lock()
	defer modelMonitorSettingMu.Unlock()

	next := modelMonitorSetting.Clone()
	updateMap := make(map[string]string, len(configMap))
	for key, value := range configMap {
		updateMap[key] = value
	}
	if rawOverrides, ok := updateMap["model_overrides"]; ok {
		var overrides map[string]ModelMonitorModelOverride
		if err := common.Unmarshal([]byte(rawOverrides), &overrides); err != nil {
			return err
		}
		if overrides == nil {
			overrides = map[string]ModelMonitorModelOverride{}
		}
		next.ModelOverrides = overrides
		delete(updateMap, "model_overrides")
	}
	if err := config.UpdateConfigFromMap(&next, updateMap); err != nil {
		return err
	}
	next.Normalize()
	modelMonitorSetting = next
	return nil
}

func (s ModelMonitorSetting) Clone() ModelMonitorSetting {
	s.Normalize()
	clone := s
	clone.ExcludedModelPatterns = append([]string(nil), s.ExcludedModelPatterns...)
	clone.NotificationDisabledUserIds = append([]int(nil), s.NotificationDisabledUserIds...)
	clone.ModelOverrides = make(map[string]ModelMonitorModelOverride, len(s.ModelOverrides))
	for modelName, override := range s.ModelOverrides {
		copied := override
		if override.Enabled != nil {
			enabled := *override.Enabled
			copied.Enabled = &enabled
		}
		clone.ModelOverrides[modelName] = copied
	}
	return clone
}

func (s *ModelMonitorSetting) ModelEnabled(modelName string) bool {
	if s == nil {
		return false
	}
	s.Normalize()
	if s.modelExcluded(modelName) {
		return false
	}
	if override, ok := s.modelOverride(modelName); ok && override.Enabled != nil {
		return *override.Enabled
	}
	return strings.TrimSpace(modelName) != ""
}

func (s *ModelMonitorSetting) ModelExcluded(modelName string) bool {
	if s == nil {
		return false
	}
	s.Normalize()
	return s.modelExcluded(modelName)
}

func (s *ModelMonitorSetting) TimeoutSecondsForModel(modelName string) int {
	if s == nil {
		return defaultModelMonitorTimeoutSeconds
	}
	s.Normalize()
	if override, ok := s.modelOverride(modelName); ok && override.TimeoutSeconds > 0 {
		return override.TimeoutSeconds
	}
	return s.DefaultTimeoutSeconds
}

func (s *ModelMonitorSetting) NotificationDisabledForUser(userId int) bool {
	if s == nil || userId <= 0 {
		return false
	}
	s.Normalize()
	for _, disabledUserId := range s.NotificationDisabledUserIds {
		if disabledUserId == userId {
			return true
		}
	}
	return false
}

func (s *ModelMonitorSetting) modelExcluded(modelName string) bool {
	for _, pattern := range s.ExcludedModelPatterns {
		if modelMonitorPatternMatches(pattern, modelName) {
			return true
		}
	}
	return false
}

func (s *ModelMonitorSetting) modelOverride(modelName string) (ModelMonitorModelOverride, bool) {
	if override, ok := s.ModelOverrides[modelName]; ok {
		return override, true
	}

	patterns := make([]string, 0, len(s.ModelOverrides))
	for pattern := range s.ModelOverrides {
		if strings.Contains(pattern, "*") {
			patterns = append(patterns, pattern)
		}
	}
	sort.Strings(patterns)
	for _, pattern := range patterns {
		if modelMonitorPatternMatches(pattern, modelName) {
			return s.ModelOverrides[pattern], true
		}
	}
	return ModelMonitorModelOverride{}, false
}

func modelMonitorPatternMatches(pattern string, modelName string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if pattern == modelName {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return false
	}
	return modelMonitorStarPatternMatches(pattern, modelName)
}

func modelMonitorStarPatternMatches(pattern string, value string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == value
	}

	position := 0
	if parts[0] != "" {
		if !strings.HasPrefix(value, parts[0]) {
			return false
		}
		position = len(parts[0])
	}

	lastIndex := len(parts) - 1
	for i := 1; i < lastIndex; i++ {
		part := parts[i]
		if part == "" {
			continue
		}
		index := strings.Index(value[position:], part)
		if index < 0 {
			return false
		}
		position += index + len(part)
	}

	lastPart := parts[lastIndex]
	if lastPart == "" {
		return true
	}
	return strings.HasSuffix(value[position:], lastPart)
}
