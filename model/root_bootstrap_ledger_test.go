package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func setupRootBootstrapLedgerTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&User{}, &QuotaAccount{}, &QuotaLedger{}))
	DB.Exec("DELETE FROM quota_ledgers")
	DB.Exec("DELETE FROM quota_accounts")
	DB.Exec("DELETE FROM users")
	t.Cleanup(func() {
		DB.Exec("DELETE FROM quota_ledgers")
		DB.Exec("DELETE FROM quota_accounts")
		DB.Exec("DELETE FROM users")
	})
}

func TestCreateRootAccountCreatesOpeningLedgerEntry(t *testing.T) {
	setupRootBootstrapLedgerTables(t)

	require.NoError(t, createRootAccountIfNeed())

	var rootUser User
	require.NoError(t, DB.Where("role = ?", common.RoleRootUser).First(&rootUser).Error)
	require.Equal(t, UserTypeRoot, rootUser.GetUserType())
	require.Equal(t, 100000000, rootUser.Quota)

	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, rootUser.Id)
	require.NoError(t, err)
	require.Equal(t, rootUser.Quota, account.Balance)

	var ledgers []QuotaLedger
	require.NoError(t, DB.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 1)
	require.Equal(t, "opening", ledgers[0].EntryType)
	require.Equal(t, LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, rootUser.Quota, ledgers[0].Amount)
	require.Equal(t, 0, ledgers[0].BalanceBefore)
	require.Equal(t, rootUser.Quota, ledgers[0].BalanceAfter)
	require.Equal(t, "root_bootstrap", ledgers[0].SourceType)
	require.Equal(t, rootUser.Id, ledgers[0].SourceId)
}
