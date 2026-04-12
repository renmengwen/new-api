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

func setupTopUpEpayTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false

	db, err := gorm.Open(sqlite.Open("file:topup_epay_ledger?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.QuotaAccount{}, &model.QuotaLedger{}, &model.Log{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func seedTopUpEpayUser(t *testing.T, db *gorm.DB, quota int) model.User {
	t.Helper()

	user := model.User{
		Username:    fmt.Sprintf("epay_notify_user_%d", time.Now().UnixNano()),
		Password:    "hashed-password",
		DisplayName: "epay notify user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Email:       "epay-notify@example.com",
		Quota:       quota,
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func TestProcessEpayTopUpSuccessCreatesLedger(t *testing.T) {
	db := setupTopUpEpayTestDB(t)

	user := seedTopUpEpayUser(t, db, 90)
	tradeNo := fmt.Sprintf("epay_notify_%d", time.Now().UnixNano())
	topUp := model.TopUp{
		UserId:        user.Id,
		Amount:        2,
		Money:         2,
		TradeNo:       tradeNo,
		PaymentMethod: "epay",
		CreateTime:    common.GetTimestamp(),
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(&topUp).Error)

	require.NoError(t, processEpayTopUpSuccess(tradeNo))

	var reloadedTopUp model.TopUp
	require.NoError(t, db.First(&reloadedTopUp, topUp.Id).Error)
	require.Equal(t, common.TopUpStatusSuccess, reloadedTopUp.Status)

	var reloadedUser model.User
	require.NoError(t, db.First(&reloadedUser, user.Id).Error)
	expectedIncrease := int(2 * common.QuotaPerUnit)
	require.Equal(t, 90+expectedIncrease, reloadedUser.Quota)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 90+expectedIncrease, account.Balance)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("account_id = ?", account.Id).Find(&ledgers).Error)
	require.Len(t, ledgers, 1)
	require.Equal(t, model.LedgerEntryRecharge, ledgers[0].EntryType)
	require.Equal(t, "topup_epay", ledgers[0].SourceType)
}
