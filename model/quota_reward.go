package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

type userQuotaLedgerInput struct {
	UserId           int
	Delta            int
	EntryType        string
	SourceType       string
	SourceId         int
	OperatorUserId   int
	OperatorUserType string
	Reason           string
	Remark           string
}

func applyUserQuotaLedger(input userQuotaLedgerInput) error {
	if input.UserId == 0 || input.Delta == 0 {
		return nil
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := applyUserQuotaLedgerTx(tx, input); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	gopool.Go(func() {
		if input.Delta > 0 {
			_ = cacheIncrUserQuota(input.UserId, int64(input.Delta))
			return
		}
		_ = cacheDecrUserQuota(input.UserId, int64(-input.Delta))
	})

	return nil
}

func applyUserQuotaLedgerTx(tx *gorm.DB, input userQuotaLedgerInput) error {
	if input.UserId == 0 || input.Delta == 0 {
		return nil
	}

	user := &User{}
	if err := tx.Select("id", "quota").Where("id = ?", input.UserId).First(user).Error; err != nil {
		return err
	}

	account, err := getQuotaAccountByOwnerWithDB(tx, QuotaOwnerTypeUser, input.UserId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		account, err = InitQuotaAccountTx(tx, QuotaOwnerTypeUser, input.UserId, user.Quota)
	}
	if err != nil {
		return err
	}

	now := common.GetTimestamp()
	if account.Balance != user.Quota {
		if err := tx.Model(&QuotaAccount{}).Where("id = ?", account.Id).Updates(map[string]interface{}{
			"balance":    user.Quota,
			"version":    gorm.Expr("version + 1"),
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		account.Balance = user.Quota
	}

	if input.Delta < 0 && account.Balance < -input.Delta {
		return errors.New("insufficient quota balance")
	}

	before := account.Balance
	after := before + input.Delta
	ledger := &QuotaLedger{
		BizNo:            fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000)),
		AccountId:        account.Id,
		EntryType:        input.EntryType,
		Direction:        LedgerDirectionIn,
		Amount:           absInt(input.Delta),
		BalanceBefore:    before,
		BalanceAfter:     after,
		SourceType:       input.SourceType,
		SourceId:         input.SourceId,
		OperatorUserId:   input.OperatorUserId,
		OperatorUserType: input.OperatorUserType,
		Reason:           input.Reason,
		Remark:           input.Remark,
		CreatedAtTs:      now,
	}
	if input.Delta < 0 {
		ledger.Direction = LedgerDirectionOut
	}
	if err := tx.Create(ledger).Error; err != nil {
		return err
	}

	accountUpdates := map[string]interface{}{
		"balance":    after,
		"version":    gorm.Expr("version + 1"),
		"updated_at": now,
	}
	switch input.EntryType {
	case LedgerEntryConsume:
		accountUpdates["total_consumed"] = gorm.Expr("total_consumed + ?", absInt(input.Delta))
	case LedgerEntryRecharge, LedgerEntryRefund, LedgerEntryReward, LedgerEntryCommission:
		accountUpdates["total_recharged"] = gorm.Expr("total_recharged + ?", absInt(input.Delta))
	case LedgerEntryAdjust:
		if input.Delta > 0 {
			accountUpdates["total_adjusted_in"] = gorm.Expr("total_adjusted_in + ?", input.Delta)
		} else {
			accountUpdates["total_adjusted_out"] = gorm.Expr("total_adjusted_out + ?", -input.Delta)
		}
	}
	if err := tx.Model(&QuotaAccount{}).Where("id = ?", account.Id).Updates(accountUpdates).Error; err != nil {
		return err
	}

	return tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", after).Error
}

func appendUserQuotaLedgerSnapshot(input userQuotaLedgerInput, balanceBefore int, balanceAfter int) error {
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := appendUserQuotaLedgerSnapshotTx(tx, input, balanceBefore, balanceAfter); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func appendUserQuotaLedgerSnapshotTx(tx *gorm.DB, input userQuotaLedgerInput, balanceBefore int, balanceAfter int) error {
	if input.UserId == 0 || input.Delta == 0 {
		return nil
	}

	account, err := getQuotaAccountByOwnerWithDB(tx, QuotaOwnerTypeUser, input.UserId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		account, err = InitQuotaAccountTx(tx, QuotaOwnerTypeUser, input.UserId, balanceAfter)
	}
	if err != nil {
		return err
	}

	now := common.GetTimestamp()
	ledger := &QuotaLedger{
		BizNo:            fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000)),
		AccountId:        account.Id,
		EntryType:        input.EntryType,
		Direction:        LedgerDirectionIn,
		Amount:           absInt(input.Delta),
		BalanceBefore:    balanceBefore,
		BalanceAfter:     balanceAfter,
		SourceType:       input.SourceType,
		SourceId:         input.SourceId,
		OperatorUserId:   input.OperatorUserId,
		OperatorUserType: input.OperatorUserType,
		Reason:           input.Reason,
		Remark:           input.Remark,
		CreatedAtTs:      now,
	}
	return tx.Create(ledger).Error
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
