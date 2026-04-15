package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMidjourneyQuotaLedgerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false

	db, err := gorm.Open(sqlite.Open("file:midjourney_quota_ledger?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Midjourney{}, &model.QuotaAccount{}, &model.QuotaLedger{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestRefundMidjourneyQuotaCreatesRefundLedger(t *testing.T) {
	db := setupMidjourneyQuotaLedgerTestDB(t)

	user := model.User{
		Username:    fmt.Sprintf("mj_refund_%d", time.Now().UnixNano()),
		Password:    "hashed-password",
		DisplayName: "mj refund",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		Quota:       200,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.QuotaAccount{
		OwnerType:      model.QuotaOwnerTypeUser,
		OwnerId:        user.Id,
		Balance:        200,
		TotalRecharged: 200,
		Status:         model.CommonStatusEnabled,
		CreatedAtTs:    common.GetTimestamp(),
		UpdatedAtTs:    common.GetTimestamp(),
	}).Error)

	task := model.Midjourney{
		UserId:     user.Id,
		Action:     "IMAGINE",
		MjId:       "mj_test_ledger",
		Status:     "FAILURE",
		Progress:   "100%",
		FailReason: "test failure",
		Quota:      60,
	}
	require.NoError(t, db.Create(&task).Error)

	require.NoError(t, refundMidjourneyQuota(&task))

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	require.Equal(t, 260, reloaded.Quota)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 260, account.Balance)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 1)
	require.Equal(t, model.LedgerEntryRefund, ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, 60, ledgers[0].Amount)
	require.Equal(t, "midjourney_refund", ledgers[0].SourceType)
	require.Equal(t, task.Id, ledgers[0].SourceId)
}
