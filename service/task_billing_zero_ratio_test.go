package service

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func restoreTaskBillingModelRatioSettings(t *testing.T) {
	t.Helper()

	modelRatioJSON := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(modelRatioJSON))
	})
}

func roundTripTaskPrivateData(t *testing.T, task *model.Task) {
	t.Helper()

	data, err := common.Marshal(task.PrivateData)
	require.NoError(t, err)

	var persisted model.TaskPrivateData
	require.NoError(t, common.Unmarshal(data, &persisted))
	task.PrivateData = persisted
}

func TestRecalculateTaskQuotaByTokensUsesStoredPerTokenBillingContextWithZeroRatio(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 17, 17, 17
	const initQuota, preConsumed = 10000, 2000
	const tokenRemain = 5000

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-task-per-token-zero", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.PrivateData.BillingContext.BillingMode = types.BillingModePerToken
	task.PrivateData.BillingContext.ModelRatio = 0
	task.PrivateData.BillingContext.GroupRatio = 1
	task.PrivateData.BillingContext.ModelRatioCaptured = true
	task.PrivateData.BillingContext.GroupRatioCaptured = true
	roundTripTaskPrivateData(t, task)

	RecalculateTaskQuotaByTokens(ctx, task, 100)

	require.Equal(t, 0, task.Quota)
	assert.Equal(t, initQuota+preConsumed, getUserQuota(t, userID))
	assert.Equal(t, tokenRemain+preConsumed, getTokenRemainQuota(t, tokenID))

	log := getLastLog(t)
	require.NotNil(t, log)
	var other map[string]interface{}
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	assert.Equal(t, string(types.BillingModePerToken), other["billing_mode"])
}

func TestRecalculateTaskQuotaByTokensFallsBackWhenStoredPerTokenSnapshotFieldsAreAbsent(t *testing.T) {
	truncate(t)
	restoreTaskBillingModelRatioSettings(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 18, 18, 18
	const initQuota, preConsumed = 10000, 2000
	const tokenRemain = 5000

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"test-model":4}`))

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-task-per-token-fallback", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.PrivateData.BillingContext.BillingMode = types.BillingModePerToken
	task.PrivateData.BillingContext.ModelRatio = 0
	task.PrivateData.BillingContext.GroupRatio = 1
	roundTripTaskPrivateData(t, task)

	RecalculateTaskQuotaByTokens(ctx, task, 100)

	require.Equal(t, 400, task.Quota)
	assert.Equal(t, initQuota+(preConsumed-400), getUserQuota(t, userID))
	assert.Equal(t, tokenRemain+(preConsumed-400), getTokenRemainQuota(t, tokenID))
}

func TestRecalculateTaskQuotaByTokensFallbackAcceptsZeroConfiguredModelRatio(t *testing.T) {
	truncate(t)
	restoreTaskBillingModelRatioSettings(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 19, 19, 19
	const initQuota, preConsumed = 10000, 2000
	const tokenRemain = 5000

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"test-model":0}`))

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-task-per-token-zero-config", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	task.PrivateData.BillingContext.BillingMode = types.BillingModePerToken
	task.PrivateData.BillingContext.ModelRatio = 0
	task.PrivateData.BillingContext.GroupRatio = 1
	roundTripTaskPrivateData(t, task)

	RecalculateTaskQuotaByTokens(ctx, task, 100)

	require.Equal(t, 0, task.Quota)
	assert.Equal(t, initQuota+preConsumed, getUserQuota(t, userID))
	assert.Equal(t, tokenRemain+preConsumed, getTokenRemainQuota(t, tokenID))
}
