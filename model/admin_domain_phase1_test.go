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

func TestUserInsertInitializesQuotaAccount(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&QuotaAccount{}))

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
		DB.Delete(&QuotaAccount{}, "owner_type = ? AND owner_id = ?", QuotaOwnerTypeUser, user.Id)
		DB.Delete(&User{}, "id = ?", user.Id)
	})

	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Id, account.OwnerId)
	assert.Equal(t, QuotaOwnerTypeUser, account.OwnerType)
	assert.Equal(t, common.QuotaForNewUser, account.Balance)
}
