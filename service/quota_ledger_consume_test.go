package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func getQuotaAccountBalance(t *testing.T, userId int) int {
	t.Helper()
	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userId)
	require.NoError(t, err)
	return account.Balance
}

func listUserQuotaLedgers(t *testing.T, userId int) []model.QuotaLedger {
	t.Helper()
	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userId)
	require.NoError(t, err)

	var ledgers []model.QuotaLedger
	require.NoError(t, model.DB.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	return ledgers
}

func TestWalletFundingPreConsumeCreatesConsumeLedger(t *testing.T) {
	truncate(t)

	const userID = 1001
	seedUser(t, userID, 1000)

	funding := &WalletFunding{userId: userID}
	require.NoError(t, funding.PreConsume(300))

	require.Equal(t, 700, getUserQuota(t, userID))
	require.Equal(t, 700, getQuotaAccountBalance(t, userID))

	ledgers := listUserQuotaLedgers(t, userID)
	require.Len(t, ledgers, 1)
	require.Equal(t, model.LedgerEntryConsume, ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionOut, ledgers[0].Direction)
	require.Equal(t, 300, ledgers[0].Amount)
	require.Equal(t, 1000, ledgers[0].BalanceBefore)
	require.Equal(t, 700, ledgers[0].BalanceAfter)
	require.Equal(t, "wallet_preconsume", ledgers[0].SourceType)
}

func TestWalletFundingRefundCreatesRefundLedger(t *testing.T) {
	truncate(t)

	const userID = 1002
	seedUser(t, userID, 900)

	funding := &WalletFunding{userId: userID}
	require.NoError(t, funding.PreConsume(240))
	require.NoError(t, funding.Refund())

	require.Equal(t, 900, getUserQuota(t, userID))
	require.Equal(t, 900, getQuotaAccountBalance(t, userID))

	ledgers := listUserQuotaLedgers(t, userID)
	require.Len(t, ledgers, 2)
	require.Equal(t, model.LedgerEntryConsume, ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionOut, ledgers[0].Direction)
	require.Equal(t, model.LedgerEntryRefund, ledgers[1].EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledgers[1].Direction)
	require.Equal(t, 240, ledgers[1].Amount)
	require.Equal(t, 660, ledgers[1].BalanceBefore)
	require.Equal(t, 900, ledgers[1].BalanceAfter)
	require.Equal(t, "wallet_refund", ledgers[1].SourceType)
}

func TestPostConsumeQuotaCreatesConsumeAndRefundLedgers(t *testing.T) {
	truncate(t)

	const userID = 1003
	seedUser(t, userID, 1000)

	relayInfo := &relaycommon.RelayInfo{
		UserId:       userID,
		IsPlayground: true,
	}

	require.NoError(t, PostConsumeQuota(relayInfo, 180, 0, false))
	require.NoError(t, PostConsumeQuota(relayInfo, -50, 0, false))

	require.Equal(t, 870, getUserQuota(t, userID))
	require.Equal(t, 870, getQuotaAccountBalance(t, userID))

	ledgers := listUserQuotaLedgers(t, userID)
	require.Len(t, ledgers, 2)
	require.Equal(t, model.LedgerEntryConsume, ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionOut, ledgers[0].Direction)
	require.Equal(t, 1000, ledgers[0].BalanceBefore)
	require.Equal(t, 820, ledgers[0].BalanceAfter)
	require.Equal(t, model.LedgerEntryRefund, ledgers[1].EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledgers[1].Direction)
	require.Equal(t, 820, ledgers[1].BalanceBefore)
	require.Equal(t, 870, ledgers[1].BalanceAfter)
	require.Equal(t, "post_consume_quota", ledgers[1].SourceType)
}

func TestRefundTaskQuotaWalletCreatesRefundLedger(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 1004, 1004, 1004
	const initQuota, preConsumed = 1000, 180
	const tokenRemain = 500

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-ledger-refund-task", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	RefundTaskQuota(ctx, task, "ledger refund")

	require.Equal(t, 1180, getUserQuota(t, userID))
	require.Equal(t, 1180, getQuotaAccountBalance(t, userID))

	ledgers := listUserQuotaLedgers(t, userID)
	require.Len(t, ledgers, 1)
	require.Equal(t, model.LedgerEntryRefund, ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, 180, ledgers[0].Amount)
}

func TestRecalculateTaskQuotaWalletCreatesConsumeAndRefundLedgers(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	const userID, tokenID, channelID = 1005, 1005, 1005
	const initQuota, preConsumed = 1000, 200
	const tokenRemain = 500

	seedUser(t, userID, initQuota)
	seedToken(t, tokenID, userID, "sk-ledger-recalc", tokenRemain)
	seedChannel(t, channelID)

	task := makeTask(userID, channelID, preConsumed, tokenID, BillingSourceWallet, 0)
	RecalculateTaskQuota(ctx, task, 260, "ledger recalc consume")
	RecalculateTaskQuota(ctx, task, 140, "ledger recalc refund")

	require.Equal(t, 1060, getUserQuota(t, userID))
	require.Equal(t, 1060, getQuotaAccountBalance(t, userID))

	ledgers := listUserQuotaLedgers(t, userID)
	require.Len(t, ledgers, 2)
	require.Equal(t, model.LedgerEntryConsume, ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionOut, ledgers[0].Direction)
	require.Equal(t, 60, ledgers[0].Amount)
	require.Equal(t, model.LedgerEntryRefund, ledgers[1].EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledgers[1].Direction)
	require.Equal(t, 120, ledgers[1].Amount)
}

func TestApplyUserQuotaLedgerEntryRefreshesQuotaRedisCache(t *testing.T) {
	truncate(t)

	const userID = 1006
	seedUser(t, userID, 100)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	common.RedisEnabled = true
	common.RDB = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() {
		_ = common.RDB.Close()
		common.RDB = nil
		common.RedisEnabled = false
	}()

	require.NoError(t, common.RedisHSetObj("user:1006", &model.UserBase{
		Id:       userID,
		Group:    "default",
		Quota:    100,
		Status:   common.UserStatusEnabled,
		Username: "test_user",
	}, time.Duration(common.RedisKeyCacheSeconds())*time.Second))

	require.NoError(t, ApplyUserQuotaLedgerEntry(UserQuotaLedgerEntryInput{
		UserId:     userID,
		Delta:      -30,
		EntryType:  model.LedgerEntryConsume,
		SourceType: "cache_refresh_test",
	}))

	quota, err := model.GetUserQuota(userID, false)
	require.NoError(t, err)
	require.Equal(t, 70, quota)
}
