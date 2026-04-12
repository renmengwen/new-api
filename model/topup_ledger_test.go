package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func setupTopUpLedgerTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&User{}, &TopUp{}, &QuotaAccount{}, &QuotaLedger{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM quota_ledgers")
		DB.Exec("DELETE FROM quota_accounts")
		DB.Exec("DELETE FROM top_ups")
		DB.Exec("DELETE FROM users")
	})
}

func seedTopUpLedgerUser(t *testing.T, quota int, email string) User {
	t.Helper()
	user := User{
		Username:    fmt.Sprintf("topup_user_%d", time.Now().UnixNano()),
		Password:    "hashed-password",
		DisplayName: "topup user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Email:       email,
		Quota:       quota,
	}
	require.NoError(t, DB.Create(&user).Error)
	return user
}

func seedPendingTopUp(t *testing.T, userId int, tradeNo string, amount int64, money float64, method string) TopUp {
	t.Helper()
	topUp := TopUp{
		UserId:        userId,
		Amount:        amount,
		Money:         money,
		TradeNo:       tradeNo,
		PaymentMethod: method,
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(&topUp).Error)
	return topUp
}

func getTopUpLedgerEntries(t *testing.T, userId int) []QuotaLedger {
	t.Helper()
	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, userId)
	require.NoError(t, err)

	var ledgers []QuotaLedger
	require.NoError(t, DB.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	return ledgers
}

func TestRechargeCreatesRechargeLedgerAndUpdatesStripeCustomer(t *testing.T) {
	setupTopUpLedgerTables(t)

	user := seedTopUpLedgerUser(t, 100, "")
	tradeNo := fmt.Sprintf("stripe_%d", time.Now().UnixNano())
	seedPendingTopUp(t, user.Id, tradeNo, 0, 2, "stripe")

	require.NoError(t, Recharge(tradeNo, "cus_test_123"))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	expectedQuota := 100 + int(decimal.NewFromFloat(2).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	require.Equal(t, expectedQuota, reloaded.Quota)
	require.Equal(t, "cus_test_123", reloaded.StripeCustomer)

	ledgers := getTopUpLedgerEntries(t, user.Id)
	require.Len(t, ledgers, 1)
	require.Equal(t, LedgerEntryRecharge, ledgers[0].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, expectedQuota-100, ledgers[0].Amount)
	require.Equal(t, "topup_stripe", ledgers[0].SourceType)
}

func TestRechargeCreemCreatesRechargeLedgerAndUpdatesEmail(t *testing.T) {
	setupTopUpLedgerTables(t)

	user := seedTopUpLedgerUser(t, 50, "")
	tradeNo := fmt.Sprintf("creem_%d", time.Now().UnixNano())
	seedPendingTopUp(t, user.Id, tradeNo, 300, 9.9, "creem")

	require.NoError(t, RechargeCreem(tradeNo, "buyer@example.com", "Buyer"))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	require.Equal(t, 350, reloaded.Quota)
	require.Equal(t, "buyer@example.com", reloaded.Email)

	ledgers := getTopUpLedgerEntries(t, user.Id)
	require.Len(t, ledgers, 1)
	require.Equal(t, LedgerEntryRecharge, ledgers[0].EntryType)
	require.Equal(t, 300, ledgers[0].Amount)
	require.Equal(t, "topup_creem", ledgers[0].SourceType)
}

func TestRechargeEpayCreatesRechargeLedger(t *testing.T) {
	setupTopUpLedgerTables(t)

	user := seedTopUpLedgerUser(t, 80, "epay@example.com")
	tradeNo := fmt.Sprintf("epay_%d", time.Now().UnixNano())
	seedPendingTopUp(t, user.Id, tradeNo, 2, 2, "epay")

	require.NoError(t, RechargeEpay(tradeNo))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	expectedIncrease := int(decimal.NewFromInt(2).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	require.Equal(t, 80+expectedIncrease, reloaded.Quota)

	ledgers := getTopUpLedgerEntries(t, user.Id)
	require.Len(t, ledgers, 1)
	require.Equal(t, LedgerEntryRecharge, ledgers[0].EntryType)
	require.Equal(t, expectedIncrease, ledgers[0].Amount)
	require.Equal(t, "topup_epay", ledgers[0].SourceType)
}

func TestRechargeEpayReturnsReadableErrorForEmptyTradeNo(t *testing.T) {
	require.EqualError(t, RechargeEpay(""), "充值订单号不能为空")
}
