package model

import (
	"errors"

	"gorm.io/gorm"
)

type QuotaAccount struct {
	Id               int    `json:"id"`
	OwnerType        string `json:"owner_type" gorm:"type:varchar(32);not null;uniqueIndex:uk_quota_accounts_owner,priority:1"`
	OwnerId          int    `json:"owner_id" gorm:"not null;uniqueIndex:uk_quota_accounts_owner,priority:2"`
	Balance          int    `json:"balance" gorm:"not null;default:0"`
	FrozenBalance    int    `json:"frozen_balance" gorm:"not null;default:0"`
	TotalRecharged   int    `json:"total_recharged" gorm:"not null;default:0"`
	TotalConsumed    int    `json:"total_consumed" gorm:"not null;default:0"`
	TotalAdjustedIn  int    `json:"total_adjusted_in" gorm:"not null;default:0"`
	TotalAdjustedOut int    `json:"total_adjusted_out" gorm:"not null;default:0"`
	Version          int    `json:"version" gorm:"not null;default:0"`
	Status           int    `json:"status" gorm:"not null;default:1;index:idx_quota_accounts_status"`
	CreatedAtTs      int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
	UpdatedAtTs      int64  `json:"updated_at" gorm:"column:updated_at;bigint;not null;default:0"`
}

type QuotaTransferOrder struct {
	Id               int    `json:"id"`
	OrderNo          string `json:"order_no" gorm:"type:varchar(64);not null;uniqueIndex:uk_qto_order_no"`
	FromAccountId    int    `json:"from_account_id" gorm:"not null;index:idx_qto_from_account_id"`
	ToAccountId      int    `json:"to_account_id" gorm:"not null;index:idx_qto_to_account_id"`
	TransferType     string `json:"transfer_type" gorm:"type:varchar(32);not null"`
	Amount           int    `json:"amount" gorm:"not null"`
	Status           int    `json:"status" gorm:"not null;default:1;index:idx_qto_status_created_at,priority:1"`
	OperatorUserId   int    `json:"operator_user_id" gorm:"not null;default:0;index:idx_qto_operator_user_id"`
	OperatorUserType string `json:"operator_user_type" gorm:"type:varchar(32);not null;default:''"`
	Reason           string `json:"reason" gorm:"type:text"`
	Remark           string `json:"remark" gorm:"type:text"`
	CreatedAtTs      int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0;index:idx_qto_status_created_at,priority:2"`
	CompletedAt      int64  `json:"completed_at" gorm:"bigint;not null;default:0"`
}

type QuotaLedger struct {
	Id               int    `json:"id"`
	BizNo            string `json:"biz_no" gorm:"type:varchar(64);not null;uniqueIndex:uk_ql_biz_no"`
	AccountId        int    `json:"account_id" gorm:"not null;index:idx_ql_account_id_created_at,priority:1"`
	TransferOrderId  int    `json:"transfer_order_id" gorm:"not null;default:0"`
	EntryType        string `json:"entry_type" gorm:"type:varchar(32);not null;index:idx_ql_entry_type_created_at,priority:1"`
	Direction        string `json:"direction" gorm:"type:varchar(16);not null"`
	Amount           int    `json:"amount" gorm:"not null"`
	BalanceBefore    int    `json:"balance_before" gorm:"not null"`
	BalanceAfter     int    `json:"balance_after" gorm:"not null"`
	SourceType       string `json:"source_type" gorm:"type:varchar(32);not null;default:'';index:idx_ql_source_type_source_id,priority:1"`
	SourceId         int    `json:"source_id" gorm:"not null;default:0;index:idx_ql_source_type_source_id,priority:2"`
	OperatorUserId   int    `json:"operator_user_id" gorm:"not null;default:0;index:idx_ql_operator_user_id_created_at,priority:1"`
	OperatorUserType string `json:"operator_user_type" gorm:"type:varchar(32);not null;default:''"`
	Reason           string `json:"reason" gorm:"type:text"`
	Remark           string `json:"remark" gorm:"type:text"`
	CreatedAtTs      int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0;index:idx_ql_account_id_created_at,priority:2;index:idx_ql_operator_user_id_created_at,priority:2;index:idx_ql_entry_type_created_at,priority:2"`
}

type QuotaAdjustmentBatch struct {
	Id             int    `json:"id"`
	BatchNo        string `json:"batch_no" gorm:"type:varchar(64);not null;uniqueIndex:uk_qab_batch_no"`
	OperatorUserId int    `json:"operator_user_id" gorm:"not null;index:idx_qab_operator_user_id_created_at,priority:1"`
	OperationType  string `json:"operation_type" gorm:"type:varchar(16);not null"`
	TargetCount    int    `json:"target_count" gorm:"not null;default:0"`
	Amount         int    `json:"amount" gorm:"not null;default:0"`
	Reason         string `json:"reason" gorm:"type:text"`
	Remark         string `json:"remark" gorm:"type:text"`
	Status         int    `json:"status" gorm:"not null;default:1"`
	CreatedAtTs    int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0;index:idx_qab_operator_user_id_created_at,priority:2"`
}

type QuotaAdjustmentBatchItem struct {
	Id             int    `json:"id"`
	BatchId        int    `json:"batch_id" gorm:"not null;index:idx_qabi_batch_id"`
	TargetUserId   int    `json:"target_user_id" gorm:"not null;index:idx_qabi_target_user_id"`
	QuotaAccountId int    `json:"quota_account_id" gorm:"not null"`
	QuotaLedgerId  int    `json:"quota_ledger_id" gorm:"not null;default:0"`
	Status         int    `json:"status" gorm:"not null;default:1"`
	ErrorMessage   string `json:"error_message" gorm:"type:text"`
	CreatedAtTs    int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
}

func GetQuotaAccountByOwner(ownerType string, ownerId int) (*QuotaAccount, error) {
	return getQuotaAccountByOwnerWithDB(DB, ownerType, ownerId)
}

func getQuotaAccountByOwnerWithDB(db *gorm.DB, ownerType string, ownerId int) (*QuotaAccount, error) {
	var account QuotaAccount
	err := db.Where("owner_type = ? AND owner_id = ?", ownerType, ownerId).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func InitQuotaAccount(ownerType string, ownerId int, initialBalance int) (*QuotaAccount, error) {
	return initQuotaAccountWithDB(DB, ownerType, ownerId, initialBalance)
}

func InitQuotaAccountTx(tx *gorm.DB, ownerType string, ownerId int, initialBalance int) (*QuotaAccount, error) {
	return initQuotaAccountWithDB(tx, ownerType, ownerId, initialBalance)
}

func initQuotaAccountWithDB(db *gorm.DB, ownerType string, ownerId int, initialBalance int) (*QuotaAccount, error) {
	account, err := getQuotaAccountByOwnerWithDB(db, ownerType, ownerId)
	if err == nil {
		return account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	account = &QuotaAccount{
		OwnerType:        ownerType,
		OwnerId:          ownerId,
		Balance:          initialBalance,
		TotalRecharged:   initialBalance,
		TotalAdjustedIn:  0,
		TotalAdjustedOut: 0,
		Status:           CommonStatusEnabled,
	}
	if err = db.Create(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}
