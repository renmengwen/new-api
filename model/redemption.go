package model

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrRedeemFailed is returned when redemption fails due to database error
var ErrRedeemFailed = errors.New("redeem.failed")

func redemptionUserForUpdateQuery(tx *gorm.DB, userId int) *gorm.DB {
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "quota").Where("id = ?", userId)
}

func redemptionQuotaAccountForUpdateQuery(tx *gorm.DB, ownerType string, ownerId int) *gorm.DB {
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("owner_type = ? AND owner_id = ?", ownerType, ownerId)
}

func applyRedemptionRechargeTx(tx *gorm.DB, redemption *Redemption, userId int) error {
	if redemption == nil || userId == 0 || redemption.Quota <= 0 {
		return errors.New("invalid redemption recharge payload")
	}

	user := &User{}
	if err := redemptionUserForUpdateQuery(tx, userId).First(user).Error; err != nil {
		return err
	}

	account := &QuotaAccount{}
	err := redemptionQuotaAccountForUpdateQuery(tx, QuotaOwnerTypeUser, userId).First(account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		account, err = InitQuotaAccountTx(tx, QuotaOwnerTypeUser, userId, user.Quota)
	}
	if err != nil {
		return err
	}

	now := common.GetTimestamp()
	if account.Balance != user.Quota {
		reconcileDelta := user.Quota - account.Balance
		reconcileLedger := &QuotaLedger{
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
			reconcileLedger.Direction = LedgerDirectionOut
			accountUpdates["total_adjusted_out"] = gorm.Expr("total_adjusted_out + ?", -reconcileDelta)
		} else {
			accountUpdates["total_adjusted_in"] = gorm.Expr("total_adjusted_in + ?", reconcileDelta)
		}
		if err := tx.Create(reconcileLedger).Error; err != nil {
			return err
		}
		if err := tx.Model(&QuotaAccount{}).Where("id = ?", account.Id).Updates(accountUpdates).Error; err != nil {
			return err
		}
		account.Balance = user.Quota
	}

	before := account.Balance
	after := before + redemption.Quota
	ledger := &QuotaLedger{
		BizNo:         fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000)),
		AccountId:     account.Id,
		EntryType:     LedgerEntryRecharge,
		Direction:     LedgerDirectionIn,
		Amount:        redemption.Quota,
		BalanceBefore: before,
		BalanceAfter:  after,
		SourceType:    "redemption_recharge",
		SourceId:      redemption.Id,
		Reason:        "redemption",
		CreatedAtTs:   now,
	}
	if err := tx.Create(ledger).Error; err != nil {
		return err
	}

	if err := tx.Model(&QuotaAccount{}).Where("id = ?", account.Id).Updates(map[string]interface{}{
		"balance":         after,
		"version":         gorm.Expr("version + 1"),
		"updated_at":      now,
		"total_recharged": gorm.Expr("total_recharged + ?", redemption.Quota),
	}).Error; err != nil {
		return err
	}

	return tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", after).Error
}

type Redemption struct {
	Id           int            `json:"id"`
	UserId       int            `json:"user_id"`
	Key          string         `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status       int            `json:"status" gorm:"default:1"`
	Name         string         `json:"name" gorm:"index"`
	Quota        int            `json:"quota" gorm:"default:100"`
	CreatedTime  int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime int64          `json:"redeemed_time" gorm:"bigint"`
	Count        int            `json:"count" gorm:"-:all"` // only for api request
	UsedUserId   int            `json:"used_user_id"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	ExpiredTime  int64          `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func SearchRedemptions(keyword string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Build query based on keyword type
	query := tx.Model(&Redemption{})

	// Only try to convert to ID if the string represents a valid integer
	if id, err := strconv.Atoi(keyword); err == nil {
		query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
	} else {
		query = query.Where("name LIKE ?", keyword+"%")
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	return &redemption, err
}

func Redeem(key string, userId int) (quota int, err error) {
	if key == "" {
		return 0, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return 0, errors.New("无效的 user id")
	}
	redemption := &Redemption{}

	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		if err := applyRedemptionRechargeTx(tx, redemption, userId); err != nil {
			return err
		}
		redemption.RedeemedTime = common.GetTimestamp()
		redemption.Status = common.RedemptionCodeStatusUsed
		redemption.UsedUserId = userId
		return tx.Save(redemption).Error
	})
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return 0, ErrRedeemFailed
	}
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(redemption.Quota), redemption.Id))
	return redemption.Quota, nil
}

func (redemption *Redemption) Insert() error {
	var err error
	err = DB.Create(redemption).Error
	return err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	var err error
	err = DB.Model(redemption).Select("name", "status", "quota", "redeemed_time", "expired_time").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}
