package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeModelMonitorSetting struct {
	disabled map[string]bool
}

func (s fakeModelMonitorSetting) ModelEnabled(modelName string) bool {
	return !s.disabled[modelName]
}

func TestBuildModelMonitorTargetsGroupsByModelAndKeepsChannelPairs(t *testing.T) {
	channels := []*model.Channel{
		{Id: 2, Name: "channel-b", Type: 1, Status: common.ChannelStatusEnabled, Models: "claude-opus-4-7,gpt-4o"},
		{Id: 1, Name: "channel-a", Type: 1, Status: common.ChannelStatusEnabled, Models: " claude-opus-4-7 , gpt-4o-mini "},
		{Id: 3, Name: "disabled", Type: 1, Status: common.ChannelStatusAutoDisabled, Models: "claude-opus-4-7"},
		{Id: 4, Name: "skipped", Type: 1, Status: common.ChannelStatusEnabled, Models: "disabled-model"},
		{Id: 5, Name: "unsupported", Type: constant.ChannelTypeMidjourney, Status: common.ChannelStatusEnabled, Models: "mj-model"},
	}

	targets := buildModelMonitorTargets(channels, fakeModelMonitorSetting{
		disabled: map[string]bool{"disabled-model": true},
	})

	require.Len(t, targets, 4)
	require.Equal(t, "claude-opus-4-7", targets[0].Model)
	require.Equal(t, []int{1, 2}, []int{targets[0].Channels[0].Channel.Id, targets[0].Channels[1].Channel.Id})
	require.Equal(t, "disabled-model", targets[1].Model)
	require.Equal(t, []int{4}, []int{targets[1].Channels[0].Channel.Id})
	require.Equal(t, "gpt-4o", targets[2].Model)
	require.Equal(t, []int{2}, []int{targets[2].Channels[0].Channel.Id})
	require.Equal(t, "gpt-4o-mini", targets[3].Model)
	require.Equal(t, []int{1}, []int{targets[3].Channels[0].Channel.Id})
}

func TestBuildModelMonitorTargetsSkipsUnsupportedVolcEngineSeedanceModels(t *testing.T) {
	channels := []*model.Channel{
		{
			Id:     1,
			Name:   "volcengine",
			Type:   constant.ChannelTypeVolcEngine,
			Status: common.ChannelStatusEnabled,
			Models: "doubao-seedance-2-0-260128,doubao-seedream-3-0-t2i",
		},
	}

	targets := buildModelMonitorTargets(channels, fakeModelMonitorSetting{})

	require.Len(t, targets, 1)
	require.Equal(t, "doubao-seedream-3-0-t2i", targets[0].Model)
	require.Equal(t, []int{1}, []int{targets[0].Channels[0].Channel.Id})
}

func TestTestModelMonitorRequiresReadPermissionBeforeReturningState(t *testing.T) {
	db := setupAdminPermissionTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.ModelMonitorStatus{}))

	operator := model.User{
		Username: "model_monitor_test_only_admin",
		Password: "hashed-password",
		Role:     common.RoleAdminUser,
		Status:   common.UserStatusEnabled,
		UserType: model.UserTypeAdmin,
		Group:    "default",
	}
	require.NoError(t, db.Create(&operator).Error)
	profile := model.PermissionProfile{
		ProfileName: "Model Monitor Test Only",
		ProfileType: model.UserTypeAdmin,
		Status:      model.CommonStatusEnabled,
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: service.ResourceModelMonitorManagement,
		ActionKey:   service.ActionTest,
		Allowed:     true,
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:    operator.Id,
		ProfileId: profile.Id,
		Status:    model.CommonStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Name:   "secret-channel",
		Type:   constant.ChannelTypeOpenAI,
		Key:    "test-key",
		Status: common.ChannelStatusEnabled,
		Models: "secret-model",
		Group:  "default",
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	payload := []byte(`{"model":"not-configured"}`)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/model_monitor/test", io.NopCloser(bytes.NewReader(payload)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", operator.Id)
	ctx.Set("role", operator.Role)

	TestModelMonitor(ctx)

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "permission denied", response.Message)
	require.NotContains(t, recorder.Body.String(), "secret-channel")
	require.NotContains(t, recorder.Body.String(), "secret-model")
}

func TestAggregateModelMonitorItemStatusRules(t *testing.T) {
	tests := []struct {
		name     string
		details  []modelMonitorChannelDetail
		expected string
	}{
		{
			name:     "no channels",
			details:  nil,
			expected: modelMonitorStatusNoChannels,
		},
		{
			name: "all skipped",
			details: []modelMonitorChannelDetail{
				{Status: modelMonitorChannelStatusSkipped},
				{Status: modelMonitorChannelStatusSkipped},
			},
			expected: modelMonitorStatusSkipped,
		},
		{
			name: "success and failure is partial",
			details: []modelMonitorChannelDetail{
				{Status: modelMonitorChannelStatusSuccess},
				{Status: modelMonitorChannelStatusFailure},
			},
			expected: modelMonitorStatusPartial,
		},
		{
			name: "success and timeout is partial",
			details: []modelMonitorChannelDetail{
				{Status: modelMonitorChannelStatusSuccess},
				{Status: modelMonitorChannelStatusTimeout},
			},
			expected: modelMonitorStatusPartial,
		},
		{
			name: "success and unavailable is partial",
			details: []modelMonitorChannelDetail{
				{Status: modelMonitorChannelStatusSuccess},
				{Status: modelMonitorChannelStatusUnavailable},
			},
			expected: modelMonitorStatusPartial,
		},
		{
			name: "all success is healthy",
			details: []modelMonitorChannelDetail{
				{Status: modelMonitorChannelStatusSuccess},
				{Status: modelMonitorChannelStatusSuccess},
			},
			expected: modelMonitorStatusHealthy,
		},
		{
			name: "all failure is unavailable",
			details: []modelMonitorChannelDetail{
				{Status: modelMonitorChannelStatusFailure},
				{Status: modelMonitorChannelStatusTimeout},
			},
			expected: modelMonitorStatusUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := aggregateModelMonitorItem(tt.name, tt.details)
			require.Equal(t, tt.expected, item.Status)
		})
	}
}

func TestBuildModelMonitorStateMarksDisabledModelsSkipped(t *testing.T) {
	disabled := false
	targets := []modelMonitorTarget{
		{
			Model: "disabled-model",
			Channels: []modelMonitorChannelTarget{
				{Channel: &model.Channel{Id: 1, Name: "channel-a", Type: 1, Status: common.ChannelStatusEnabled}},
				{Channel: &model.Channel{Id: 2, Name: "channel-b", Type: 1, Status: common.ChannelStatusEnabled}},
			},
		},
	}
	state := buildModelMonitorStateFromTargets(&operation_setting.ModelMonitorSetting{
		ModelOverrides: map[string]operation_setting.ModelMonitorModelOverride{
			"disabled-model": {Enabled: &disabled},
		},
	}, targets, map[modelMonitorStatusKey]modelMonitorStatusRecord{})

	require.Len(t, state.Items, 1)
	require.False(t, state.Items[0].Enabled)
	require.Equal(t, modelMonitorStatusSkipped, state.Items[0].Status)
	require.Equal(t, 2, state.Items[0].ChannelCount)
	require.Equal(t, 2, state.Items[0].SkippedCount)
	require.Equal(t, 1, state.Summary.SkippedModels)
}

func TestBuildModelMonitorStateLeavesUntestedTargetsUnknown(t *testing.T) {
	targets := []modelMonitorTarget{
		{
			Model: "new-model",
			Channels: []modelMonitorChannelTarget{
				{Channel: &model.Channel{Id: 1, Name: "channel-a", Type: 1, Status: common.ChannelStatusEnabled}},
				{Channel: &model.Channel{Id: 2, Name: "channel-b", Type: 1, Status: common.ChannelStatusEnabled}},
			},
		},
	}

	state := buildModelMonitorStateFromTargets(&operation_setting.ModelMonitorSetting{}, targets, map[modelMonitorStatusKey]modelMonitorStatusRecord{})

	require.Len(t, state.Items, 1)
	require.True(t, state.Items[0].Enabled)
	require.Equal(t, "unknown", state.Items[0].Status)
	require.Equal(t, 0, state.Items[0].FailedCount)
	require.Equal(t, "unknown", state.Items[0].Channels[0].Status)
	require.Equal(t, "unknown", state.Items[0].Channels[1].Status)
	require.Equal(t, 0, state.Summary.UnavailableModels)
	require.Equal(t, 0, state.Summary.FailedChannels)
}
