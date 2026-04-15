package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func AppendUserOpeningQuotaLedgerTx(tx *gorm.DB, userId int, quota int, sourceType string) error {
	return appendUserOpeningQuotaLedgerTx(tx, userId, quota, sourceType)
}

func appendUserOpeningQuotaLedger(userId int, quota int, sourceType string) error {
	if userId == 0 || quota <= 0 || sourceType == "" {
		return nil
	}
	return appendUserQuotaLedgerSnapshot(userQuotaLedgerInput{
		UserId:     userId,
		Delta:      quota,
		EntryType:  LedgerEntryOpening,
		SourceType: sourceType,
		SourceId:   userId,
		Reason:     sourceType,
	}, 0, quota)
}

func appendUserOpeningQuotaLedgerTx(tx *gorm.DB, userId int, quota int, sourceType string) error {
	if userId == 0 || quota <= 0 || sourceType == "" {
		return nil
	}
	return appendUserQuotaLedgerSnapshotTx(tx, userQuotaLedgerInput{
		UserId:     userId,
		Delta:      quota,
		EntryType:  LedgerEntryOpening,
		SourceType: sourceType,
		SourceId:   userId,
		Reason:     sourceType,
	}, 0, quota)
}

func hasUserOpeningQuotaLedger(userId int) (bool, error) {
	if userId == 0 {
		return false, nil
	}

	account, err := GetQuotaAccountByOwner(QuotaOwnerTypeUser, userId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var count int64
	if err := DB.Model(&QuotaLedger{}).
		Where("account_id = ? AND entry_type = ?", account.Id, LedgerEntryOpening).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
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
	if err := userQuotaReconcileUserForUpdateQuery(tx, input.UserId).First(user).Error; err != nil {
		return err
	}

	account, err := getUserQuotaAccountForReconcileTx(tx, input.UserId, user.Quota)
	if err != nil {
		return err
	}

	now := common.GetTimestamp()
	if err := reconcileUserQuotaAccountBalanceTx(tx, user, account, now); err != nil {
		return err
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

func userQuotaReconcileUserForUpdateQuery(tx *gorm.DB, userId int) *gorm.DB {
	query := tx.Select("id", "quota").Where("id = ?", userId)
	if common.UsingSQLite {
		return query
	}
	return query.Clauses(clause.Locking{Strength: "UPDATE"})
}

func userQuotaReconcileAccountForUpdateQuery(tx *gorm.DB, ownerType string, ownerId int) *gorm.DB {
	query := tx.Where("owner_type = ? AND owner_id = ?", ownerType, ownerId)
	if common.UsingSQLite {
		return query
	}
	return query.Clauses(clause.Locking{Strength: "UPDATE"})
}

func getUserQuotaAccountForReconcileTx(tx *gorm.DB, userId int, initialBalance int) (*QuotaAccount, error) {
	account := &QuotaAccount{}
	err := userQuotaReconcileAccountForUpdateQuery(tx, QuotaOwnerTypeUser, userId).First(account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return InitQuotaAccountTx(tx, QuotaOwnerTypeUser, userId, initialBalance)
	}
	return account, err
}

func reconcileUserQuotaAccountBalanceTx(tx *gorm.DB, user *User, account *QuotaAccount, now int64) error {
	if tx == nil || user == nil || account == nil || account.Balance == user.Quota {
		return nil
	}

	reconcileDelta := user.Quota - account.Balance
	ledger := &QuotaLedger{
		BizNo:         fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000)),
		AccountId:     account.Id,
		EntryType:     LedgerEntryAdjust,
		Direction:     LedgerDirectionIn,
		Amount:        absInt(reconcileDelta),
		BalanceBefore: account.Balance,
		BalanceAfter:  user.Quota,
		SourceType:    "quota_reconcile",
		SourceId:      user.Id,
		Reason:        "sync_with_user_quota",
		CreatedAtTs:   now,
	}
	accountUpdates := map[string]interface{}{
		"balance":    user.Quota,
		"version":    gorm.Expr("version + 1"),
		"updated_at": now,
	}
	if reconcileDelta < 0 {
		ledger.Direction = LedgerDirectionOut
		accountUpdates["total_adjusted_out"] = gorm.Expr("total_adjusted_out + ?", -reconcileDelta)
	} else {
		accountUpdates["total_adjusted_in"] = gorm.Expr("total_adjusted_in + ?", reconcileDelta)
	}
	if err := tx.Create(ledger).Error; err != nil {
		return err
	}
	if err := tx.Model(&QuotaAccount{}).Where("id = ?", account.Id).Updates(accountUpdates).Error; err != nil {
		return err
	}
	account.Balance = user.Quota
	return nil
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
