package service

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type AdjustUserQuotaRequest struct {
	OperatorUserId   int    `json:"operator_user_id"`
	OperatorRole     int    `json:"operator_role"`
	OperatorUserType string `json:"operator_user_type"`
	TargetUserId     int    `json:"target_user_id"`
	Delta            int    `json:"delta"`
	Reason           string `json:"reason"`
	Remark           string `json:"remark"`
	IP               string `json:"ip"`
}

type AdjustUserQuotaBatchRequest struct {
	OperatorUserId   int    `json:"operator_user_id"`
	OperatorRole     int    `json:"operator_role"`
	OperatorUserType string `json:"operator_user_type"`
	TargetUserIds    []int  `json:"target_user_ids"`
	Delta            int    `json:"delta"`
	Reason           string `json:"reason"`
	Remark           string `json:"remark"`
	IP               string `json:"ip"`
}

type QuotaBatchFailureItem struct {
	TargetUserId int    `json:"target_user_id"`
	Username     string `json:"username,omitempty"`
	ErrorMessage string `json:"error_message"`
}

func GetUserQuotaSummary(userId int) (map[string]any, error) {
	user, err := model.GetUserById(userId, false)
	if err != nil {
		return nil, err
	}

	account, err := ensureUserQuotaAccount(userId)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"user_id":            user.Id,
		"username":           user.Username,
		"user_type":          user.GetUserType(),
		"balance":            account.Balance,
		"frozen_balance":     account.FrozenBalance,
		"status":             account.Status,
		"total_recharged":    account.TotalRecharged,
		"total_consumed":     account.TotalConsumed,
		"total_adjusted_in":  account.TotalAdjustedIn,
		"total_adjusted_out": account.TotalAdjustedOut,
	}, nil
}

func GetScopedUserQuotaSummary(userId int, operatorUserId int, operatorRole int) (map[string]any, error) {
	if _, err := GetManagedEndUser(userId, operatorUserId, operatorRole); err != nil {
		return nil, err
	}
	return GetUserQuotaSummary(userId)
}

func AdjustUserQuota(req AdjustUserQuotaRequest) (map[string]any, error) {
	if req.TargetUserId == 0 {
		return nil, errors.New("target_user_id is required")
	}
	if req.Delta == 0 {
		return nil, errors.New("delta cannot be zero")
	}

	operator, err := ResolveOperatorUser(req.OperatorUserId, req.OperatorRole)
	if err != nil {
		return nil, err
	}
	req.OperatorUserType = operator.GetUserType()

	user, err := GetManagedEndUser(req.TargetUserId, operator.Id, operator.Role)
	if err != nil {
		return nil, err
	}

	account, err := ensureUserQuotaAccount(user.Id)
	if err != nil {
		return nil, err
	}

	if req.Delta < 0 && account.Balance < -req.Delta {
		return nil, errors.New("insufficient quota balance")
	}

	before := account.Balance
	after := before + req.Delta
	now := common.GetTimestamp()
	orderNo := fmt.Sprintf("qto_%d_%d", now, common.GetRandomInt(1000000))
	bizNo := fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000))

	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	order := &model.QuotaTransferOrder{
		OrderNo:          orderNo,
		FromAccountId:    0,
		ToAccountId:      account.Id,
		TransferType:     model.TransferTypeAdminAdjust,
		Amount:           absInt(req.Delta),
		Status:           model.CommonStatusEnabled,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           req.Reason,
		Remark:           req.Remark,
		CreatedAtTs:      now,
		CompletedAt:      now,
	}
	if req.Delta < 0 {
		order.FromAccountId = account.Id
		order.ToAccountId = 0
	}
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	ledger := &model.QuotaLedger{
		BizNo:            bizNo,
		AccountId:        account.Id,
		TransferOrderId:  order.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           absInt(req.Delta),
		BalanceBefore:    before,
		BalanceAfter:     after,
		SourceType:       "admin_quota_adjust",
		SourceId:         user.Id,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           req.Reason,
		Remark:           req.Remark,
		CreatedAtTs:      now,
	}
	if req.Delta < 0 {
		ledger.Direction = model.LedgerDirectionOut
	}
	if err := tx.Create(ledger).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	accountUpdates := map[string]any{
		"balance":    after,
		"version":    gorm.Expr("version + 1"),
		"updated_at": now,
	}
	if req.Delta > 0 {
		accountUpdates["total_adjusted_in"] = gorm.Expr("total_adjusted_in + ?", req.Delta)
	} else {
		accountUpdates["total_adjusted_out"] = gorm.Expr("total_adjusted_out + ?", -req.Delta)
	}
	if err := tx.Model(&model.QuotaAccount{}).Where("id = ?", account.Id).Updates(accountUpdates).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("quota", after).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	beforeJSON, _ := common.Marshal(map[string]any{"quota": before})
	afterJSON, _ := common.Marshal(map[string]any{"quota": after, "delta": req.Delta})
	auditErr := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		ActionModule:     "quota",
		ActionType:       "adjust",
		ActionDesc:       req.Reason,
		TargetType:       "user",
		TargetId:         user.Id,
		BeforeJSON:       string(beforeJSON),
		AfterJSON:        string(afterJSON),
		IP:               req.IP,
	})
	if auditErr != nil {
		tx.Rollback()
		return nil, auditErr
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return map[string]any{
		"target_user_id": user.Id,
		"balance_before": before,
		"balance_after":  after,
		"order_no":       orderNo,
		"biz_no":         bizNo,
	}, nil
}

func ListQuotaLedger(pageInfo *common.PageInfo, requesterUserId int, requesterRole int, userId int, operatorUserId int, entryType string) ([]model.QuotaLedger, int64, error) {
	query := model.DB.Model(&model.QuotaLedger{})

	operator, err := ResolveOperatorUser(requesterUserId, requesterRole)
	if err != nil {
		return nil, 0, err
	}
	if operator.GetUserType() == model.UserTypeAgent {
		managedUserSubQuery := ApplyManagedEndUserScope(
			model.DB.Model(&model.User{}).Select("users.id"),
			operator,
		)
		managedAccountSubQuery := model.DB.Model(&model.QuotaAccount{}).
			Select("id").
			Where("owner_type = ?", model.QuotaOwnerTypeUser).
			Where("owner_id IN (?)", managedUserSubQuery)
		query = query.Where("account_id IN (?)", managedAccountSubQuery)
	}

	if userId > 0 {
		managedUser, err := GetManagedEndUser(userId, requesterUserId, requesterRole)
		if err != nil {
			return nil, 0, err
		}
		account, err := ensureUserQuotaAccount(managedUser.Id)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("account_id = ?", account.Id)
	}
	if operatorUserId > 0 {
		query = query.Where("operator_user_id = ?", operatorUserId)
	}
	if entryType != "" {
		query = query.Where("entry_type = ?", entryType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.QuotaLedger
	if err := query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func AdjustUserQuotaBatch(req AdjustUserQuotaBatchRequest) (map[string]any, error) {
	if len(req.TargetUserIds) == 0 {
		return nil, errors.New("target_user_ids is required")
	}
	if req.Delta == 0 {
		return nil, errors.New("delta cannot be zero")
	}

	operator, err := ResolveOperatorUser(req.OperatorUserId, req.OperatorRole)
	if err != nil {
		return nil, err
	}
	req.OperatorUserType = operator.GetUserType()

	now := common.GetTimestamp()
	batchNo := fmt.Sprintf("qab_%d_%d", now, common.GetRandomInt(1000000))
	batch := &model.QuotaAdjustmentBatch{
		BatchNo:        batchNo,
		OperatorUserId: req.OperatorUserId,
		OperationType:  batchOperationType(req.Delta),
		TargetCount:    len(req.TargetUserIds),
		Amount:         absInt(req.Delta),
		Reason:         req.Reason,
		Remark:         req.Remark,
		Status:         model.CommonStatusEnabled,
		CreatedAtTs:    now,
	}
	if err := model.DB.Create(batch).Error; err != nil {
		return nil, err
	}

	successCount := 0
	successUserIds := make([]int, 0, len(req.TargetUserIds))
	failedItems := make([]QuotaBatchFailureItem, 0)

	for _, userId := range req.TargetUserIds {
		user, err := GetManagedEndUser(userId, operator.Id, operator.Role)
		if err != nil {
			failure := QuotaBatchFailureItem{
				TargetUserId: userId,
				ErrorMessage: err.Error(),
			}
			failedItems = append(failedItems, failure)
			if itemErr := recordFailedBatchAdjustmentItem(batch.Id, userId, err.Error(), now); itemErr != nil {
				return nil, itemErr
			}
			continue
		}

		if err := applyBatchQuotaAdjustmentItem(batch.Id, user, req, now); err != nil {
			failedItems = append(failedItems, QuotaBatchFailureItem{
				TargetUserId: user.Id,
				Username:     user.Username,
				ErrorMessage: err.Error(),
			})
			if itemErr := recordFailedBatchAdjustmentItem(batch.Id, user.Id, err.Error(), now); itemErr != nil {
				return nil, itemErr
			}
			continue
		}

		successCount++
		successUserIds = append(successUserIds, user.Id)
	}

	failedCount := len(failedItems)
	afterJSON, _ := common.Marshal(map[string]any{
		"batch_id":         batch.Id,
		"target_count":     len(req.TargetUserIds),
		"delta":            req.Delta,
		"success_count":    successCount,
		"failed_count":     failedCount,
		"success_user_ids": successUserIds,
		"failed_items":     failedItems,
	})
	if err := CreateAdminAuditLog(AuditLogInput{
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		ActionModule:     "quota",
		ActionType:       "adjust_batch",
		ActionDesc:       req.Reason,
		TargetType:       "quota_batch",
		TargetId:         batch.Id,
		AfterJSON:        string(afterJSON),
		IP:               req.IP,
	}); err != nil {
		return nil, err
	}

	return map[string]any{
		"batch_id":         batch.Id,
		"batch_no":         batch.BatchNo,
		"target_count":     batch.TargetCount,
		"success_count":    successCount,
		"failed_count":     failedCount,
		"delta":            req.Delta,
		"success_user_ids": successUserIds,
		"failed_items":     failedItems,
	}, nil
}

func applyBatchQuotaAdjustmentItem(batchId int, user *model.User, req AdjustUserQuotaBatchRequest, now int64) error {
	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	account, err := ensureUserQuotaAccountWithDB(tx, user.Id)
	if err != nil {
		tx.Rollback()
		return err
	}
	if req.Delta < 0 && account.Balance < -req.Delta {
		tx.Rollback()
		return errors.New("insufficient quota balance")
	}

	before := account.Balance
	after := before + req.Delta
	order := &model.QuotaTransferOrder{
		OrderNo:          fmt.Sprintf("qto_%d_%d", common.GetTimestamp(), common.GetRandomInt(1000000)),
		FromAccountId:    0,
		ToAccountId:      account.Id,
		TransferType:     model.TransferTypeAdminAdjust,
		Amount:           absInt(req.Delta),
		Status:           model.CommonStatusEnabled,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           req.Reason,
		Remark:           req.Remark,
		CreatedAtTs:      now,
		CompletedAt:      now,
	}
	if req.Delta < 0 {
		order.FromAccountId = account.Id
		order.ToAccountId = 0
	}
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		return err
	}

	ledger := &model.QuotaLedger{
		BizNo:            fmt.Sprintf("ql_%d_%d", common.GetTimestamp(), common.GetRandomInt(1000000)),
		AccountId:        account.Id,
		TransferOrderId:  order.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           absInt(req.Delta),
		BalanceBefore:    before,
		BalanceAfter:     after,
		SourceType:       "admin_quota_adjust_batch",
		SourceId:         batchId,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           req.Reason,
		Remark:           req.Remark,
		CreatedAtTs:      now,
	}
	if req.Delta < 0 {
		ledger.Direction = model.LedgerDirectionOut
	}
	if err := tx.Create(ledger).Error; err != nil {
		tx.Rollback()
		return err
	}

	accountUpdates := map[string]any{
		"balance":    after,
		"version":    gorm.Expr("version + 1"),
		"updated_at": now,
	}
	if req.Delta > 0 {
		accountUpdates["total_adjusted_in"] = gorm.Expr("total_adjusted_in + ?", req.Delta)
	} else {
		accountUpdates["total_adjusted_out"] = gorm.Expr("total_adjusted_out + ?", -req.Delta)
	}
	if err := tx.Model(&model.QuotaAccount{}).Where("id = ?", account.Id).Updates(accountUpdates).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("quota", after).Error; err != nil {
		tx.Rollback()
		return err
	}

	item := &model.QuotaAdjustmentBatchItem{
		BatchId:        batchId,
		TargetUserId:   user.Id,
		QuotaAccountId: account.Id,
		QuotaLedgerId:  ledger.Id,
		Status:         model.CommonStatusEnabled,
		CreatedAtTs:    now,
	}
	if err := tx.Create(item).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func recordFailedBatchAdjustmentItem(batchId int, targetUserId int, errorMessage string, now int64) error {
	return model.DB.Model(&model.QuotaAdjustmentBatchItem{}).Create(map[string]any{
		"batch_id":         batchId,
		"target_user_id":   targetUserId,
		"quota_account_id": 0,
		"quota_ledger_id":  0,
		"status":           model.CommonStatusDisabled,
		"error_message":    errorMessage,
		"created_at":       now,
	}).Error
}

func ensureUserQuotaAccount(userId int) (*model.QuotaAccount, error) {
	return ensureUserQuotaAccountWithDB(model.DB, userId)
}

func ensureUserQuotaAccountWithDB(db *gorm.DB, userId int) (*model.QuotaAccount, error) {
	account, err := getQuotaAccountByOwnerWithDB(db, model.QuotaOwnerTypeUser, userId)
	if err == nil {
		return account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	user, userErr := getUserByIdWithDB(db, userId)
	if userErr != nil {
		return nil, userErr
	}
	if db == model.DB {
		return model.InitQuotaAccount(model.QuotaOwnerTypeUser, user.Id, user.Quota)
	}
	return model.InitQuotaAccountTx(db, model.QuotaOwnerTypeUser, user.Id, user.Quota)
}

func getQuotaAccountByOwnerWithDB(db *gorm.DB, ownerType string, ownerId int) (*model.QuotaAccount, error) {
	account := &model.QuotaAccount{}
	err := db.Where("owner_type = ? AND owner_id = ?", ownerType, ownerId).First(account).Error
	return account, err
}

func getUserByIdWithDB(db *gorm.DB, userId int) (*model.User, error) {
	user := &model.User{}
	var err error
	if db == model.DB {
		return model.GetUserById(userId, true)
	}
	err = db.First(user, "id = ?", userId).Error
	return user, err
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func batchOperationType(delta int) string {
	if delta >= 0 {
		return "increase"
	}
	return "decrease"
}
