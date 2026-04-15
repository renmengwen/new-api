package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupRedemptionLedgerTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&User{}, &Log{}, &Redemption{}, &QuotaAccount{}, &QuotaLedger{}))
	DB.Exec("DELETE FROM quota_ledgers")
	DB.Exec("DELETE FROM quota_accounts")
	DB.Exec("DELETE FROM redemptions")
	DB.Exec("DELETE FROM logs")
	DB.Exec("DELETE FROM users")
	t.Cleanup(func() {
		DB.Exec("DELETE FROM quota_ledgers")
		DB.Exec("DELETE FROM quota_accounts")
		DB.Exec("DELETE FROM redemptions")
		DB.Exec("DELETE FROM logs")
		DB.Exec("DELETE FROM users")
	})
}

func seedRedemptionLedgerUserWithoutAccount(t *testing.T, quota int) User {
	t.Helper()
	user := User{
		Username:    fmt.Sprintf("redeem_user_%d", time.Now().UnixNano()),
		Password:    "hashed-password",
		DisplayName: "redeem user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       quota,
	}
	require.NoError(t, DB.Create(&user).Error)
	return user
}

func TestRedeemCreatesRechargeLedgerEntry(t *testing.T) {
	setupRedemptionLedgerTables(t)

	user := seedRedemptionLedgerUserWithoutAccount(t, 100)
	redemption := Redemption{
		Key:         fmt.Sprintf("redeem_%d", time.Now().UnixNano()),
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "wallet-ledger-redemption",
		Quota:       60,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(&redemption).Error)

	quota, err := Redeem(redemption.Key, user.Id)
	require.NoError(t, err)
	require.Equal(t, redemption.Quota, quota)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	require.Equal(t, 160, reloadedUser.Quota)

	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 160, account.Balance)
	require.Equal(t, 160, account.TotalRecharged)

	var reloadedRedemption Redemption
	require.NoError(t, DB.First(&reloadedRedemption, redemption.Id).Error)
	require.Equal(t, common.RedemptionCodeStatusUsed, reloadedRedemption.Status)
	require.Equal(t, user.Id, reloadedRedemption.UsedUserId)

	var ledgers []QuotaLedger
	require.NoError(t, DB.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 1)
	require.Equal(t, LedgerEntryRecharge, ledgers[0].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, redemption.Quota, ledgers[0].Amount)
	require.Equal(t, 100, ledgers[0].BalanceBefore)
	require.Equal(t, 160, ledgers[0].BalanceAfter)
	require.Equal(t, "redemption_recharge", ledgers[0].SourceType)
	require.Equal(t, redemption.Id, ledgers[0].SourceId)
	require.Equal(t, "redemption", ledgers[0].Reason)
}

func TestRedeemLocksUserAndQuotaAccountReads(t *testing.T) {
	setupRedemptionLedgerTables(t)

	tx := DB.Session(&gorm.Session{DryRun: true})

	userQuery := redemptionUserForUpdateQuery(tx, 1).First(&User{})
	userLocking, ok := userQuery.Statement.Clauses["FOR"].Expression.(clause.Locking)
	require.True(t, ok)
	require.Equal(t, "UPDATE", userLocking.Strength)

	accountQuery := redemptionQuotaAccountForUpdateQuery(tx, QuotaOwnerTypeUser, 1).First(&QuotaAccount{})
	accountLocking, ok := accountQuery.Statement.Clauses["FOR"].Expression.(clause.Locking)
	require.True(t, ok)
	require.Equal(t, "UPDATE", accountLocking.Strength)
}

func TestRedeemReconcilesQuotaAccountBeforeRecharge(t *testing.T) {
	setupRedemptionLedgerTables(t)

	user := seedRewardLedgerUser(t, 100, 0)
	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&QuotaAccount{}).Where("id = ?", account.Id).Update("balance", 80).Error)
	account, err = GetQuotaAccountByOwner(QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 80, account.Balance)

	redemption := Redemption{
		Key:         fmt.Sprintf("redeem_reconcile_%d", time.Now().UnixNano()),
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "wallet-ledger-redemption-reconcile",
		Quota:       60,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(&redemption).Error)

	quota, err := Redeem(redemption.Key, user.Id)
	require.NoError(t, err)
	require.Equal(t, redemption.Quota, quota)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	require.Equal(t, 160, reloadedUser.Quota)

	reloadedAccount, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 160, reloadedAccount.Balance)
	require.Equal(t, 160, reloadedAccount.TotalRecharged)
	require.Equal(t, 20, reloadedAccount.TotalAdjustedIn)

	var ledgers []QuotaLedger
	require.NoError(t, DB.Where("account_id = ?", reloadedAccount.Id).Order("id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 2)

	require.Equal(t, LedgerEntryAdjust, ledgers[0].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, 20, ledgers[0].Amount)
	require.Equal(t, 80, ledgers[0].BalanceBefore)
	require.Equal(t, 100, ledgers[0].BalanceAfter)
	require.Equal(t, "quota_reconcile", ledgers[0].SourceType)
	require.Equal(t, user.Id, ledgers[0].SourceId)
	require.Equal(t, "sync_with_user_quota", ledgers[0].Reason)

	require.Equal(t, LedgerEntryRecharge, ledgers[1].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[1].Direction)
	require.Equal(t, redemption.Quota, ledgers[1].Amount)
	require.Equal(t, 100, ledgers[1].BalanceBefore)
	require.Equal(t, 160, ledgers[1].BalanceAfter)
	require.Equal(t, "redemption_recharge", ledgers[1].SourceType)
	require.Equal(t, redemption.Id, ledgers[1].SourceId)
	require.Equal(t, "redemption", ledgers[1].Reason)
}
