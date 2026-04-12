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

type quotaApplyResult struct {
	TargetUserId    int
	TargetAccountId int
	TargetBefore    int
	TargetAfter     int
	OrderNo         string
	BizNo           string
	TargetLedgerId  int
	BeforeAudit     map[string]any
	AfterAudit      map[string]any
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
	if _, err := GetManagedEndUserForResource(userId, operatorUserId, operatorRole, ResourceQuotaManagement); err != nil {
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

	user, err := GetManagedEndUserForResource(req.TargetUserId, operator.Id, operator.Role, ResourceQuotaManagement)
	if err != nil {
		return nil, err
	}

	return adjustQuotaForResolvedUser(req, operator, user, "admin_quota_adjust", user.Id)
}

func AdjustUserQuotaForTarget(req AdjustUserQuotaRequest, targetUser *model.User) (map[string]any, error) {
	if targetUser == nil || targetUser.Id == 0 {
		return nil, errors.New("target user is required")
	}
	if req.Delta == 0 {
		return nil, errors.New("delta cannot be zero")
	}

	operator, err := ResolveOperatorUser(req.OperatorUserId, req.OperatorRole)
	if err != nil {
		return nil, err
	}
	req.OperatorUserType = operator.GetUserType()
	req.TargetUserId = targetUser.Id

	return adjustQuotaForResolvedUser(req, operator, targetUser, "admin_quota_adjust", targetUser.Id)
}

func adjustQuotaForResolvedUser(req AdjustUserQuotaRequest, operator *model.User, user *model.User, sourceType string, sourceId int) (map[string]any, error) {
	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := common.GetTimestamp()
	result, err := applyQuotaAdjustmentTx(tx, operator, user, req, sourceType, sourceId, now)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	beforeJSON, _ := common.Marshal(result.BeforeAudit)
	afterJSON, _ := common.Marshal(result.AfterAudit)
	auditErr := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		ActionModule:     "quota",
		ActionType:       "adjust",
		ActionDesc:       req.Reason,
		TargetType:       "user",
		TargetId:         result.TargetUserId,
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
		"target_user_id": result.TargetUserId,
		"balance_before": result.TargetBefore,
		"balance_after":  result.TargetAfter,
		"order_no":       result.OrderNo,
		"biz_no":         result.BizNo,
	}, nil
}

func applyQuotaAdjustmentTx(tx *gorm.DB, operator *model.User, targetUser *model.User, req AdjustUserQuotaRequest, sourceType string, sourceId int, now int64) (*quotaApplyResult, error) {
	if operator != nil && operator.GetUserType() == model.UserTypeAgent {
		return applyAgentQuotaTransferTx(tx, operator, targetUser, req, sourceType, sourceId, now)
	}
	return applyAdminQuotaAdjustmentTx(tx, targetUser, req, sourceType, sourceId, now)
}

func applyAdminQuotaAdjustmentTx(tx *gorm.DB, user *model.User, req AdjustUserQuotaRequest, sourceType string, sourceId int, now int64) (*quotaApplyResult, error) {
	account, err := ensureUserQuotaAccountWithDB(tx, user.Id)
	if err != nil {
		return nil, err
	}
	if err := reconcileQuotaAccountWithUserQuotaTx(tx, account, user, req, now); err != nil {
		return nil, err
	}
	if req.Delta < 0 && account.Balance < -req.Delta {
		return nil, errors.New("insufficient quota balance")
	}

	before := account.Balance
	after := before + req.Delta
	orderNo := fmt.Sprintf("qto_%d_%d", now, common.GetRandomInt(1000000))
	bizNo := fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000))

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
		SourceType:       sourceType,
		SourceId:         sourceId,
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
		return nil, err
	}
	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("quota", after).Error; err != nil {
		return nil, err
	}

	return &quotaApplyResult{
		TargetUserId:    user.Id,
		TargetAccountId: account.Id,
		TargetBefore:    before,
		TargetAfter:     after,
		OrderNo:         orderNo,
		BizNo:           bizNo,
		TargetLedgerId:  ledger.Id,
		BeforeAudit:     map[string]any{"quota": before},
		AfterAudit:      map[string]any{"quota": after, "delta": req.Delta},
	}, nil
}

func applyAgentQuotaTransferTx(tx *gorm.DB, operator *model.User, targetUser *model.User, req AdjustUserQuotaRequest, sourceType string, sourceId int, now int64) (*quotaApplyResult, error) {
	agentUser, err := getUserByIdWithDB(tx, operator.Id)
	if err != nil {
		return nil, err
	}
	agentAccount, err := ensureUserQuotaAccountWithDB(tx, agentUser.Id)
	if err != nil {
		return nil, err
	}
	targetAccount, err := ensureUserQuotaAccountWithDB(tx, targetUser.Id)
	if err != nil {
		return nil, err
	}
	if err := reconcileQuotaAccountWithUserQuotaTx(tx, agentAccount, agentUser, req, now); err != nil {
		return nil, err
	}
	if err := reconcileQuotaAccountWithUserQuotaTx(tx, targetAccount, targetUser, req, now); err != nil {
		return nil, err
	}

	policy, err := getAgentQuotaPolicyWithDB(tx, agentUser.Id)
	if err != nil {
		return nil, err
	}
	if policy.MaxSingleAdjustAmount > 0 && absInt(req.Delta) > policy.MaxSingleAdjustAmount {
		return nil, errors.New("exceeds agent max single adjust amount")
	}

	amount := absInt(req.Delta)
	targetBefore := targetAccount.Balance
	agentBefore := agentAccount.Balance
	targetAfter := targetBefore
	agentAfter := agentBefore
	transferType := model.TransferTypeAgentRecharge

	fromAccount := agentAccount
	toAccount := targetAccount
	fromUserId := agentUser.Id
	toUserId := targetUser.Id

	if req.Delta > 0 {
		if !policy.AllowRechargeUser {
			return nil, errors.New("agent recharge user is disabled")
		}
		if agentAccount.Balance < amount {
			return nil, errors.New("insufficient agent quota balance")
		}
		agentAfter -= amount
		targetAfter += amount
	} else {
		if !policy.AllowReclaimQuota {
			return nil, errors.New("agent reclaim quota is disabled")
		}
		if targetAccount.Balance < amount {
			return nil, errors.New("insufficient quota balance")
		}
		transferType = model.TransferTypeAgentReclaim
		fromAccount = targetAccount
		toAccount = agentAccount
		fromUserId = targetUser.Id
		toUserId = agentUser.Id
		targetAfter -= amount
		agentAfter += amount
	}

	orderNo := fmt.Sprintf("qto_%d_%d", now, common.GetRandomInt(1000000))
	order := &model.QuotaTransferOrder{
		OrderNo:          orderNo,
		FromAccountId:    fromAccount.Id,
		ToAccountId:      toAccount.Id,
		TransferType:     transferType,
		Amount:           amount,
		Status:           model.CommonStatusEnabled,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           req.Reason,
		Remark:           req.Remark,
		CreatedAtTs:      now,
		CompletedAt:      now,
	}
	if err := tx.Create(order).Error; err != nil {
		return nil, err
	}

	fromBefore := fromAccount.Balance
	fromAfter := fromBefore - amount
	fromBizNo := fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000))
	fromLedger := &model.QuotaLedger{
		BizNo:            fromBizNo,
		AccountId:        fromAccount.Id,
		TransferOrderId:  order.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionOut,
		Amount:           amount,
		BalanceBefore:    fromBefore,
		BalanceAfter:     fromAfter,
		SourceType:       sourceType,
		SourceId:         sourceId,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           req.Reason,
		Remark:           req.Remark,
		CreatedAtTs:      now,
	}
	if err := tx.Create(fromLedger).Error; err != nil {
		return nil, err
	}

	toBefore := toAccount.Balance
	toAfter := toBefore + amount
	toBizNo := fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000))
	toLedger := &model.QuotaLedger{
		BizNo:            toBizNo,
		AccountId:        toAccount.Id,
		TransferOrderId:  order.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           amount,
		BalanceBefore:    toBefore,
		BalanceAfter:     toAfter,
		SourceType:       sourceType,
		SourceId:         sourceId,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           req.Reason,
		Remark:           req.Remark,
		CreatedAtTs:      now,
	}
	if err := tx.Create(toLedger).Error; err != nil {
		return nil, err
	}

	if err := updateQuotaAccountBalanceTx(tx, fromAccount.Id, fromAfter, -amount, now); err != nil {
		return nil, err
	}
	if err := updateQuotaAccountBalanceTx(tx, toAccount.Id, toAfter, amount, now); err != nil {
		return nil, err
	}
	if err := tx.Model(&model.User{}).Where("id = ?", fromUserId).Update("quota", fromAfter).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&model.User{}).Where("id = ?", toUserId).Update("quota", toAfter).Error; err != nil {
		return nil, err
	}

	targetLedgerId := toLedger.Id
	targetBizNo := toBizNo
	if req.Delta < 0 {
		targetLedgerId = fromLedger.Id
		targetBizNo = fromBizNo
	}

	return &quotaApplyResult{
		TargetUserId:    targetUser.Id,
		TargetAccountId: targetAccount.Id,
		TargetBefore:    targetBefore,
		TargetAfter:     targetAfter,
		OrderNo:         orderNo,
		BizNo:           targetBizNo,
		TargetLedgerId:  targetLedgerId,
		BeforeAudit: map[string]any{
			"quota":       targetBefore,
			"agent_quota": agentBefore,
		},
		AfterAudit: map[string]any{
			"quota":       targetAfter,
			"agent_quota": agentAfter,
			"delta":       req.Delta,
		},
	}, nil
}

func reconcileQuotaAccountWithUserQuotaTx(tx *gorm.DB, account *model.QuotaAccount, user *model.User, req AdjustUserQuotaRequest, now int64) error {
	if account == nil || user == nil || account.Balance == user.Quota {
		return nil
	}

	diff := user.Quota - account.Balance
	order := &model.QuotaTransferOrder{
		OrderNo:          fmt.Sprintf("qto_%d_%d", now, common.GetRandomInt(1000000)),
		FromAccountId:    0,
		ToAccountId:      account.Id,
		TransferType:     model.TransferTypeAdminAdjust,
		Amount:           absInt(diff),
		Status:           model.CommonStatusEnabled,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           "sync_with_user_quota",
		Remark:           req.Remark,
		CreatedAtTs:      now,
		CompletedAt:      now,
	}
	if diff < 0 {
		order.FromAccountId = account.Id
		order.ToAccountId = 0
	}
	if err := tx.Create(order).Error; err != nil {
		return err
	}

	ledger := &model.QuotaLedger{
		BizNo:            fmt.Sprintf("ql_%d_%d", now, common.GetRandomInt(1000000)),
		AccountId:        account.Id,
		TransferOrderId:  order.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           absInt(diff),
		BalanceBefore:    account.Balance,
		BalanceAfter:     user.Quota,
		SourceType:       "quota_reconcile",
		SourceId:         user.Id,
		OperatorUserId:   req.OperatorUserId,
		OperatorUserType: req.OperatorUserType,
		Reason:           "sync_with_user_quota",
		Remark:           req.Remark,
		CreatedAtTs:      now,
	}
	if diff < 0 {
		ledger.Direction = model.LedgerDirectionOut
	}
	if err := tx.Create(ledger).Error; err != nil {
		return err
	}

	accountUpdates := map[string]any{
		"balance":    user.Quota,
		"version":    gorm.Expr("version + 1"),
		"updated_at": now,
	}
	if diff > 0 {
		accountUpdates["total_adjusted_in"] = gorm.Expr("total_adjusted_in + ?", diff)
	} else {
		accountUpdates["total_adjusted_out"] = gorm.Expr("total_adjusted_out + ?", -diff)
	}
	if err := tx.Model(&model.QuotaAccount{}).Where("id = ?", account.Id).Updates(accountUpdates).Error; err != nil {
		return err
	}
	account.Balance = user.Quota
	return nil
}

func getAgentQuotaPolicyWithDB(db *gorm.DB, agentUserId int) (*model.AgentQuotaPolicy, error) {
	policy := &model.AgentQuotaPolicy{}
	err := db.Where("agent_user_id = ?", agentUserId).First(policy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.AgentQuotaPolicy{
			AgentUserId:           agentUserId,
			AllowRechargeUser:     true,
			AllowReclaimQuota:     true,
			MaxSingleAdjustAmount: 0,
			Status:                model.CommonStatusEnabled,
		}, nil
	}
	return policy, err
}

func updateQuotaAccountBalanceTx(tx *gorm.DB, accountId int, balance int, delta int, now int64) error {
	updates := map[string]any{
		"balance":    balance,
		"version":    gorm.Expr("version + 1"),
		"updated_at": now,
	}
	if delta > 0 {
		updates["total_adjusted_in"] = gorm.Expr("total_adjusted_in + ?", delta)
	} else if delta < 0 {
		updates["total_adjusted_out"] = gorm.Expr("total_adjusted_out + ?", -delta)
	}
	return tx.Model(&model.QuotaAccount{}).Where("id = ?", accountId).Updates(updates).Error
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
			ResourceQuotaManagement,
		)
		managedAccountSubQuery := model.DB.Model(&model.QuotaAccount{}).
			Select("id").
			Where("owner_type = ?", model.QuotaOwnerTypeUser).
			Where("owner_id IN (?)", managedUserSubQuery)
		ownAccountSubQuery := model.DB.Model(&model.QuotaAccount{}).
			Select("id").
			Where("owner_type = ?", model.QuotaOwnerTypeUser).
			Where("owner_id = ?", operator.Id)
		query = query.Where("(account_id IN (?) OR account_id IN (?))", managedAccountSubQuery, ownAccountSubQuery)
	}

	if userId > 0 {
		if operator.GetUserType() == model.UserTypeAgent && userId == operator.Id {
			account, err := ensureUserQuotaAccount(operator.Id)
			if err != nil {
				return nil, 0, err
			}
			query = query.Where("account_id = ?", account.Id)
		} else {
			managedUser, err := GetManagedEndUserForResource(userId, requesterUserId, requesterRole, ResourceQuotaManagement)
			if err != nil {
				return nil, 0, err
			}
			account, err := ensureUserQuotaAccount(managedUser.Id)
			if err != nil {
				return nil, 0, err
			}
			query = query.Where("account_id = ?", account.Id)
		}
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
		user, err := GetManagedEndUserForResource(userId, operator.Id, operator.Role, ResourceQuotaManagement)
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

		if err := applyBatchQuotaAdjustmentItem(batch.Id, operator, user, req, now); err != nil {
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

func applyBatchQuotaAdjustmentItem(batchId int, operator *model.User, user *model.User, req AdjustUserQuotaBatchRequest, now int64) error {
	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	result, err := applyQuotaAdjustmentTx(tx, operator, user, AdjustUserQuotaRequest{
		OperatorUserId:   req.OperatorUserId,
		OperatorRole:     req.OperatorRole,
		OperatorUserType: req.OperatorUserType,
		TargetUserId:     user.Id,
		Delta:            req.Delta,
		Reason:           req.Reason,
		Remark:           req.Remark,
		IP:               req.IP,
	}, "admin_quota_adjust_batch", batchId, now)
	if err != nil {
		tx.Rollback()
		return err
	}

	item := &model.QuotaAdjustmentBatchItem{
		BatchId:        batchId,
		TargetUserId:   user.Id,
		QuotaAccountId: result.TargetAccountId,
		QuotaLedgerId:  result.TargetLedgerId,
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
