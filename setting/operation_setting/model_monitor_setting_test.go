package operation_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/require"
)

func TestModelMonitorSettingNormalizeDefaults(t *testing.T) {
	setting := ModelMonitorSetting{}

	setting.Normalize()

	require.False(t, setting.Enabled)
	require.GreaterOrEqual(t, setting.IntervalMinutes, 1)
	require.GreaterOrEqual(t, setting.BatchSize, 1)
	require.GreaterOrEqual(t, setting.DefaultTimeoutSeconds, 1)
	require.GreaterOrEqual(t, setting.FailureThreshold, 1)
	require.NotNil(t, setting.ModelOverrides)
	require.NotNil(t, setting.NotificationDisabledUserIds)
}

func TestModelMonitorSettingModelEnabledAndTimeoutOverrides(t *testing.T) {
	disabled := false
	enabled := true
	setting := ModelMonitorSetting{
		Enabled:               true,
		IntervalMinutes:       5,
		BatchSize:             10,
		DefaultTimeoutSeconds: 30,
		FailureThreshold:      3,
		ExcludedModelPatterns: []string{"legacy-*", "exact-skip"},
		ModelOverrides: map[string]ModelMonitorModelOverride{
			"gpt-4o":      {Enabled: &disabled, TimeoutSeconds: 12},
			"legacy-chat": {Enabled: &enabled, TimeoutSeconds: 60},
			"o3-*":        {Enabled: &enabled, TimeoutSeconds: 45},
		},
	}

	setting.Normalize()

	require.False(t, setting.ModelEnabled("legacy-chat"))
	require.False(t, setting.ModelEnabled("exact-skip"))
	require.False(t, setting.ModelEnabled("gpt-4o"))
	require.True(t, setting.ModelEnabled("o3-mini"))
	require.True(t, setting.ModelEnabled("other-model"))

	require.Equal(t, 12, setting.TimeoutSecondsForModel("gpt-4o"))
	require.Equal(t, 45, setting.TimeoutSecondsForModel("o3-mini"))
	require.Equal(t, 30, setting.TimeoutSecondsForModel("other-model"))
}

func TestModelMonitorSettingRegisteredInGlobalConfig(t *testing.T) {
	registered := config.GlobalConfig.Get("model_monitor_setting")

	require.NotNil(t, registered)
	require.IsType(t, &ModelMonitorSetting{}, registered)
	require.NotSame(t, GetModelMonitorSetting(), registered)
}

func TestUpdateModelMonitorSettingFromMapReplacesOverrides(t *testing.T) {
	original := modelMonitorSetting.Clone()
	defer func() {
		modelMonitorSettingMu.Lock()
		modelMonitorSetting = original
		modelMonitorSettingMu.Unlock()
	}()

	disabled := false
	firstOverrides, err := common.Marshal(map[string]ModelMonitorModelOverride{
		"gpt-4o": {Enabled: &disabled},
	})
	require.NoError(t, err)
	require.NoError(t, UpdateModelMonitorSettingFromMap(map[string]string{
		"model_overrides": string(firstOverrides),
	}))
	require.Contains(t, GetModelMonitorSetting().ModelOverrides, "gpt-4o")

	emptyOverrides, err := common.Marshal(map[string]ModelMonitorModelOverride{})
	require.NoError(t, err)
	require.NoError(t, UpdateModelMonitorSettingFromMap(map[string]string{
		"model_overrides": string(emptyOverrides),
	}))
	require.NotContains(t, GetModelMonitorSetting().ModelOverrides, "gpt-4o")
}

func TestUpdateModelMonitorSettingFromMapStoresNotificationDisabledUserIds(t *testing.T) {
	original := modelMonitorSetting.Clone()
	defer func() {
		modelMonitorSettingMu.Lock()
		modelMonitorSetting = original
		modelMonitorSettingMu.Unlock()
	}()

	disabledIds, err := common.Marshal([]int{3, 9})
	require.NoError(t, err)
	require.NoError(t, UpdateModelMonitorSettingFromMap(map[string]string{
		"notification_disabled_user_ids": string(disabledIds),
	}))

	current := GetModelMonitorSetting()
	require.Equal(t, []int{3, 9}, current.NotificationDisabledUserIds)
	require.True(t, current.NotificationDisabledForUser(3))
	require.False(t, current.NotificationDisabledForUser(4))

	clone := current.Clone()
	clone.NotificationDisabledUserIds[0] = 99
	require.Equal(t, []int{3, 9}, current.NotificationDisabledUserIds)
}

func TestUpdateModelMonitorSettingFromMapKeepsOverridesOnInvalidReplacement(t *testing.T) {
	original := modelMonitorSetting.Clone()
	defer func() {
		modelMonitorSettingMu.Lock()
		modelMonitorSetting = original
		modelMonitorSettingMu.Unlock()
	}()

	disabled := false
	firstOverrides, err := common.Marshal(map[string]ModelMonitorModelOverride{
		"gpt-4o": {Enabled: &disabled, TimeoutSeconds: 12},
	})
	require.NoError(t, err)
	require.NoError(t, UpdateModelMonitorSettingFromMap(map[string]string{
		"model_overrides": string(firstOverrides),
	}))

	err = UpdateModelMonitorSettingFromMap(map[string]string{
		"model_overrides": `{"gpt-4o": "invalid"}`,
	})

	require.Error(t, err)
	current := GetModelMonitorSetting()
	require.Contains(t, current.ModelOverrides, "gpt-4o")
	require.Equal(t, 12, current.ModelOverrides["gpt-4o"].TimeoutSeconds)
	require.NotNil(t, current.ModelOverrides["gpt-4o"].Enabled)
	require.False(t, *current.ModelOverrides["gpt-4o"].Enabled)
}

func TestModelMonitorSettingPatternsMatchProviderPrefixedModels(t *testing.T) {
	disabled := false
	setting := ModelMonitorSetting{
		DefaultTimeoutSeconds: 30,
		ExcludedModelPatterns: []string{
			"*image*",
		},
		ModelOverrides: map[string]ModelMonitorModelOverride{
			"*opus*": {Enabled: &disabled, TimeoutSeconds: 45},
		},
	}

	setting.Normalize()

	require.False(t, setting.ModelEnabled("openai/gpt-image-1"))
	require.False(t, setting.ModelEnabled("anthropic/claude-opus-4-7"))
	require.Equal(t, 45, setting.TimeoutSecondsForModel("anthropic/claude-opus-4-7"))
}
