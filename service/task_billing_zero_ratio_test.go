package service

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
