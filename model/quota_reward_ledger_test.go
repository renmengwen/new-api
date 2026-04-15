package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupRewardLedgerTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&User{}, &Log{}, &QuotaAccount{}, &QuotaLedger{}, &Checkin{}, &UserOAuthBinding{}))
	t.Cleanup(func() {
		DB.Exec("DELETE FROM user_oauth_bindings")
		DB.Exec("DELETE FROM quota_ledgers")
		DB.Exec("DELETE FROM quota_accounts")
		DB.Exec("DELETE FROM checkins")
		DB.Exec("DELETE FROM logs")
		DB.Exec("DELETE FROM users")
	})
}

func seedRewardLedgerUser(t *testing.T, quota int, affQuota int) User {
	t.Helper()
	user := User{
		Username:    fmt.Sprintf("reward_user_%d", time.Now().UnixNano()),
		Password:    "hashed-password",
		DisplayName: "reward user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       quota,
		AffQuota:    affQuota,
	}
	require.NoError(t, DB.Create(&user).Error)
	_, err := InitQuotaAccount(QuotaOwnerTypeUser, user.Id, quota)
	require.NoError(t, err)
	return user
}

func listRewardQuotaLedgers(t *testing.T, userId int) []QuotaLedger {
	t.Helper()
	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, userId)
	require.NoError(t, err)

	var ledgers []QuotaLedger
	require.NoError(t, DB.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	return ledgers
}

func TestUserCheckinCreatesRewardLedger(t *testing.T) {
	setupRewardLedgerTables(t)

	user := seedRewardLedgerUser(t, 120, 0)

	setting := operation_setting.GetCheckinSetting()
	previous := *setting
	*setting = operation_setting.CheckinSetting{
		Enabled:  true,
		MinQuota: 35,
		MaxQuota: 35,
	}
	t.Cleanup(func() {
		*setting = previous
	})

	checkin, err := UserCheckin(user.Id)
	require.NoError(t, err)
	require.Equal(t, 35, checkin.QuotaAwarded)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	require.Equal(t, 155, reloaded.Quota)

	ledgers := listRewardQuotaLedgers(t, user.Id)
	require.Len(t, ledgers, 1)
	require.Equal(t, LedgerEntryReward, ledgers[0].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, 35, ledgers[0].Amount)
	require.Equal(t, 120, ledgers[0].BalanceBefore)
	require.Equal(t, 155, ledgers[0].BalanceAfter)
	require.Equal(t, "checkin_reward", ledgers[0].SourceType)
	require.Equal(t, checkin.Id, ledgers[0].SourceId)
}

func TestUserCheckinReconcilesBalanceDriftBeforeRewardLedger(t *testing.T) {
	setupRewardLedgerTables(t)

	user := seedRewardLedgerUser(t, 120, 0)
	require.NoError(t, DB.Model(&QuotaAccount{}).
		Where("owner_type = ? AND owner_id = ?", QuotaOwnerTypeUser, user.Id).
		Updates(map[string]any{
			"balance":            60,
			"total_adjusted_in":  0,
			"total_adjusted_out": 0,
		}).Error)

	setting := operation_setting.GetCheckinSetting()
	previous := *setting
	*setting = operation_setting.CheckinSetting{
		Enabled:  true,
		MinQuota: 35,
		MaxQuota: 35,
	}
	t.Cleanup(func() {
		*setting = previous
	})

	checkin, err := UserCheckin(user.Id)
	require.NoError(t, err)
	require.Equal(t, 35, checkin.QuotaAwarded)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	require.Equal(t, 155, reloaded.Quota)

	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 155, account.Balance)
	require.Equal(t, 60, account.TotalAdjustedIn)

	ledgers := listRewardQuotaLedgers(t, user.Id)
	require.Len(t, ledgers, 2)
	require.Equal(t, LedgerEntryAdjust, ledgers[0].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, 60, ledgers[0].Amount)
	require.Equal(t, 60, ledgers[0].BalanceBefore)
	require.Equal(t, 120, ledgers[0].BalanceAfter)
	require.Equal(t, "quota_reconcile", ledgers[0].SourceType)
	require.Equal(t, user.Id, ledgers[0].SourceId)
	require.Equal(t, "sync_with_user_quota", ledgers[0].Reason)
	require.Equal(t, LedgerEntryReward, ledgers[1].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[1].Direction)
	require.Equal(t, 35, ledgers[1].Amount)
	require.Equal(t, 120, ledgers[1].BalanceBefore)
	require.Equal(t, 155, ledgers[1].BalanceAfter)
	require.Equal(t, "checkin_reward", ledgers[1].SourceType)
	require.Equal(t, checkin.Id, ledgers[1].SourceId)
}

func TestUserQuotaReconcileQueriesUseRowLocksOnLockCapableDB(t *testing.T) {
	setupRewardLedgerTables(t)

	previousSQLite := common.UsingSQLite
	previousMySQL := common.UsingMySQL
	previousPostgreSQL := common.UsingPostgreSQL
	common.UsingSQLite = false
	common.UsingMySQL = true
	common.UsingPostgreSQL = false
	t.Cleanup(func() {
		common.UsingSQLite = previousSQLite
		common.UsingMySQL = previousMySQL
		common.UsingPostgreSQL = previousPostgreSQL
	})

	tx := DB.Session(&gorm.Session{DryRun: true})
	userQuery := userQuotaReconcileUserForUpdateQuery(tx, 123)
	accountQuery := userQuotaReconcileAccountForUpdateQuery(tx, QuotaOwnerTypeUser, 123)

	userClause, ok := userQuery.Statement.Clauses["FOR"]
	require.True(t, ok)
	_, ok = userClause.Expression.(clause.Locking)
	require.True(t, ok)

	accountClause, ok := accountQuery.Statement.Clauses["FOR"]
	require.True(t, ok)
	_, ok = accountClause.Expression.(clause.Locking)
	require.True(t, ok)
}

func TestTransferAffQuotaToQuotaCreatesCommissionLedger(t *testing.T) {
	setupRewardLedgerTables(t)

	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
	})

	user := seedRewardLedgerUser(t, 100, 80)

	require.NoError(t, user.TransferAffQuotaToQuota(80))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	require.Equal(t, 180, reloaded.Quota)
	require.Equal(t, 0, reloaded.AffQuota)

	ledgers := listRewardQuotaLedgers(t, user.Id)
	require.Len(t, ledgers, 1)
	require.Equal(t, LedgerEntryCommission, ledgers[0].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, 80, ledgers[0].Amount)
	require.Equal(t, 100, ledgers[0].BalanceBefore)
	require.Equal(t, 180, ledgers[0].BalanceAfter)
	require.Equal(t, "aff_quota_transfer", ledgers[0].SourceType)
	require.Equal(t, user.Id, ledgers[0].SourceId)
}

func TestInsertCreatesOpeningAndInviteRewardLedgers(t *testing.T) {
	setupRewardLedgerTables(t)

	previousNewUserQuota := common.QuotaForNewUser
	previousInviteeQuota := common.QuotaForInvitee
	previousInviterQuota := common.QuotaForInviter
	common.QuotaForNewUser = 100
	common.QuotaForInvitee = 30
	common.QuotaForInviter = 40
	t.Cleanup(func() {
		common.QuotaForNewUser = previousNewUserQuota
		common.QuotaForInvitee = previousInviteeQuota
		common.QuotaForInviter = previousInviterQuota
	})

	inviter := seedRewardLedgerUser(t, 0, 0)
	newUser := &User{
		Username:    fmt.Sprintf("register_user_%d", time.Now().UnixNano()),
		Password:    "register-user-password",
		DisplayName: "register user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}

	require.NoError(t, newUser.Insert(inviter.Id))

	var reloadedNewUser User
	require.NoError(t, DB.First(&reloadedNewUser, newUser.Id).Error)
	require.Equal(t, 130, reloadedNewUser.Quota)

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	require.Equal(t, 40, reloadedInviter.AffQuota)

	ledgers := listRewardQuotaLedgers(t, reloadedNewUser.Id)
	require.Len(t, ledgers, 2)
	require.Equal(t, LedgerEntryOpening, ledgers[0].EntryType)
	require.Equal(t, 100, ledgers[0].Amount)
	require.Equal(t, 0, ledgers[0].BalanceBefore)
	require.Equal(t, 100, ledgers[0].BalanceAfter)
	require.Equal(t, "user_register", ledgers[0].SourceType)
	require.Equal(t, LedgerEntryReward, ledgers[1].EntryType)
	require.Equal(t, 30, ledgers[1].Amount)
	require.Equal(t, 100, ledgers[1].BalanceBefore)
	require.Equal(t, 130, ledgers[1].BalanceAfter)
	require.Equal(t, "invitee_register", ledgers[1].SourceType)
}

func TestFinalizeOAuthUserCreationCreatesOpeningAndInviteRewardLedgers(t *testing.T) {
	setupRewardLedgerTables(t)

	previousNewUserQuota := common.QuotaForNewUser
	previousInviteeQuota := common.QuotaForInvitee
	previousInviterQuota := common.QuotaForInviter
	common.QuotaForNewUser = 60
	common.QuotaForInvitee = 25
	common.QuotaForInviter = 45
	t.Cleanup(func() {
		common.QuotaForNewUser = previousNewUserQuota
		common.QuotaForInvitee = previousInviteeQuota
		common.QuotaForInviter = previousInviterQuota
	})

	inviter := seedRewardLedgerUser(t, 0, 0)
	user := &User{
		Username:    fmt.Sprintf("oauth_user_%d", time.Now().UnixNano()),
		DisplayName: "oauth user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}

	tx := DB.Begin()
	require.NoError(t, tx.Error)
	require.NoError(t, user.InsertWithTx(tx, inviter.Id, "user_register"))
	require.NoError(t, tx.Commit().Error)

	user.FinalizeOAuthUserCreation(inviter.Id)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	require.Equal(t, 85, reloadedUser.Quota)

	var reloadedInviter User
	require.NoError(t, DB.First(&reloadedInviter, inviter.Id).Error)
	require.Equal(t, 45, reloadedInviter.AffQuota)

	ledgers := listRewardQuotaLedgers(t, reloadedUser.Id)
	require.Len(t, ledgers, 2)
	require.Equal(t, LedgerEntryOpening, ledgers[0].EntryType)
	require.Equal(t, "user_register", ledgers[0].SourceType)
	require.Equal(t, 0, ledgers[0].BalanceBefore)
	require.Equal(t, 60, ledgers[0].BalanceAfter)
	require.Equal(t, LedgerEntryReward, ledgers[1].EntryType)
	require.Equal(t, "invitee_register", ledgers[1].SourceType)
	require.Equal(t, 60, ledgers[0].Amount)
	require.Equal(t, 25, ledgers[1].Amount)
	require.Equal(t, 60, ledgers[1].BalanceBefore)
	require.Equal(t, 85, ledgers[1].BalanceAfter)
}
