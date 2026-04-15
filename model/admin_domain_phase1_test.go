package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhaseOneAdminModelsAutoMigrate(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(
		&PermissionProfile{},
		&PermissionProfileItem{},
		&UserPermissionBinding{},
		&UserDataScope{},
		&AgentProfile{},
		&AgentUserRelation{},
		&AgentQuotaPolicy{},
		&QuotaAccount{},
		&QuotaTransferOrder{},
		&QuotaLedger{},
		&QuotaAdjustmentBatch{},
		&QuotaAdjustmentBatchItem{},
		&AdminAuditLog{},
	))

	assert.True(t, DB.Migrator().HasTable(&PermissionProfile{}))
	assert.True(t, DB.Migrator().HasTable(&AgentProfile{}))
	assert.True(t, DB.Migrator().HasTable(&QuotaAccount{}))
	assert.True(t, DB.Migrator().HasTable(&AdminAuditLog{}))

	assert.True(t, DB.Migrator().HasColumn(&User{}, "user_type"))
	assert.True(t, DB.Migrator().HasColumn(&User{}, "parent_agent_id"))
	assert.True(t, DB.Migrator().HasColumn(&User{}, "phone"))
	assert.True(t, DB.Migrator().HasColumn(&User{}, "last_active_at"))
}

func TestUserInsertCreatesOpeningLedgerEntry(t *testing.T) {
	previousNewUserQuota := common.QuotaForNewUser
	previousInviteeQuota := common.QuotaForInvitee
	previousInviterQuota := common.QuotaForInviter
	common.QuotaForNewUser = 88
	common.QuotaForInvitee = 0
	common.QuotaForInviter = 0
	t.Cleanup(func() {
		common.QuotaForNewUser = previousNewUserQuota
		common.QuotaForInvitee = previousInviteeQuota
		common.QuotaForInviter = previousInviterQuota
	})

	require.NoError(t, DB.AutoMigrate(&Log{}, &QuotaAccount{}, &QuotaLedger{}))

	user := &User{
		Username:    fmt.Sprintf("phase1_user_%d", time.Now().UnixNano()),
		Password:    "12345678",
		DisplayName: "Phase1 User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}

	require.NoError(t, user.Insert(0))

	t.Cleanup(func() {
		DB.Delete(&QuotaLedger{}, "account_id IN (SELECT id FROM quota_accounts WHERE owner_type = ? AND owner_id = ?)", QuotaOwnerTypeUser, user.Id)
		DB.Delete(&QuotaAccount{}, "owner_type = ? AND owner_id = ?", QuotaOwnerTypeUser, user.Id)
		DB.Delete(&Log{}, "user_id = ?", user.Id)
		DB.Delete(&User{}, "id = ?", user.Id)
	})

	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Id, account.OwnerId)
	assert.Equal(t, QuotaOwnerTypeUser, account.OwnerType)
	assert.Equal(t, common.QuotaForNewUser, account.Balance)

	var ledgers []QuotaLedger
	require.NoError(t, DB.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 1)
	assert.Equal(t, "opening", ledgers[0].EntryType)
	assert.Equal(t, LedgerDirectionIn, ledgers[0].Direction)
	assert.Equal(t, common.QuotaForNewUser, ledgers[0].Amount)
	assert.Equal(t, 0, ledgers[0].BalanceBefore)
	assert.Equal(t, common.QuotaForNewUser, ledgers[0].BalanceAfter)
	assert.Equal(t, "user_register", ledgers[0].SourceType)
	assert.Equal(t, user.Id, ledgers[0].SourceId)
}
