package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type adminQuotaAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type adminQuotaPageResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    common.PageInfo `json:"data"`
}

func setupAdminQuotaTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false

	db, err := gorm.Open(sqlite.Open("file:admin_quota?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.UserDataScopeOverride{},
		&model.AgentUserRelation{},
		&model.AgentQuotaPolicy{},
		&model.QuotaAccount{},
		&model.QuotaTransferOrder{},
		&model.QuotaLedger{},
		&model.QuotaAdjustmentBatch{},
		&model.QuotaAdjustmentBatchItem{},
		&model.AdminAuditLog{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newAdminQuotaContext(t *testing.T, method string, target string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		payload, err := common.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequest(method, target, io.NopCloser(bytes.NewReader(payload)))
		req.Header.Set("Content-Type", "application/json")
	}
	ctx.Request = req
	ctx.Set("id", 999)
	ctx.Set("role", common.RoleRootUser)
	return ctx, recorder
}

func newAdminQuotaContextWithOperator(t *testing.T, method string, target string, body any, operatorId int, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	ctx, recorder := newAdminQuotaContext(t, method, target, body)
	ctx.Set("id", operatorId)
	ctx.Set("role", role)
	return ctx, recorder
}

func seedQuotaUser(t *testing.T, db *gorm.DB, username string, quota int) model.User {
	t.Helper()

	user := model.User{
		Username:    username,
		Password:    "hashed-password",
		DisplayName: username,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		AffCode:     username + "_aff",
		Quota:       quota,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.QuotaAccount{
		OwnerType:      model.QuotaOwnerTypeUser,
		OwnerId:        user.Id,
		Balance:        quota,
		TotalRecharged: quota,
		Status:         model.CommonStatusEnabled,
		CreatedAtTs:    common.GetTimestamp(),
		UpdatedAtTs:    common.GetTimestamp(),
	}).Error)
	return user
}

func TestGetUserQuotaSummaryReturnsAccountState(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	user := seedQuotaUser(t, db, "quota_summary_user", 1200)

	ctx, recorder := newAdminQuotaContext(t, http.MethodGet, "/api/admin/users/"+strconv.Itoa(user.Id)+"/quota-summary", nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	GetUserQuotaSummary(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, float64(1200), response.Data["balance"])
	require.Equal(t, float64(user.Id), response.Data["user_id"])
}

func TestAdjustUserQuotaCreatesLedgerAndAudit(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	user := seedQuotaUser(t, db, "quota_adjust_user", 1000)

	ctx, recorder := newAdminQuotaContext(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": user.Id,
		"delta":          300,
		"reason":         "manual_adjust",
		"remark":         "phase1",
	})
	AdjustUserQuota(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 1300, account.Balance)
	require.Equal(t, 1000, account.TotalRecharged)
	require.Equal(t, 300, account.TotalAdjustedIn)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	require.Equal(t, 1300, reloaded.Quota)

	var ledger model.QuotaLedger
	require.NoError(t, db.Where("account_id = ?", account.Id).First(&ledger).Error)
	require.Equal(t, model.LedgerEntryAdjust, ledger.EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledger.Direction)
	require.Equal(t, 1000, ledger.BalanceBefore)
	require.Equal(t, 1300, ledger.BalanceAfter)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("target_type = ? AND target_id = ?", "user", user.Id).First(&audit).Error)
	require.Equal(t, "quota", audit.ActionModule)
}

func TestGetQuotaLedgerReturnsAdjustmentRecord(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	user := seedQuotaUser(t, db, "quota_ledger_user", 800)

	ctx, _ := newAdminQuotaContext(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": user.Id,
		"delta":          200,
		"reason":         "manual_adjust",
	})
	AdjustUserQuota(ctx)

	listCtx, recorder := newAdminQuotaContext(t, http.MethodGet, "/api/admin/quota/ledger?user_id="+strconv.Itoa(user.Id)+"&p=1&page_size=10", nil)
	GetQuotaLedger(listCtx)

	var response adminQuotaPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "manual_adjust", item["reason"])
	require.NotZero(t, item["created_at"])
}

func TestGetQuotaLedgerIncludesAccountAndOperatorUsernames(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	operator := seedQuotaUser(t, db, "quota_ledger_operator", 0)
	operator.Role = common.RoleAdminUser
	operator.UserType = model.UserTypeAdmin
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", operator.Id).Updates(map[string]any{
		"role":      operator.Role,
		"user_type": operator.UserType,
	}).Error)
	grantPermissionActions(t, db, operator.Id, "admin",
		permissionGrant{Resource: "quota_management", Action: "adjust"},
		permissionGrant{Resource: "quota_management", Action: "ledger_read"},
	)

	target := seedQuotaUser(t, db, "quota_ledger_target", 300)

	ctx, _ := newAdminQuotaContextWithOperator(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": target.Id,
		"delta":          50,
		"reason":         "manual_adjust",
	}, operator.Id, common.RoleAdminUser)
	AdjustUserQuota(ctx)

	listCtx, recorder := newAdminQuotaContextWithOperator(t, http.MethodGet, "/api/admin/quota/ledger?user_id="+strconv.Itoa(target.Id)+"&p=1&page_size=10", nil, operator.Id, common.RoleAdminUser)
	GetQuotaLedger(listCtx)

	var response adminQuotaPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, target.Username, item["account_username"])
	require.Equal(t, operator.Username, item["operator_username"])
}

func TestAdjustUserQuotaBatchCreatesBatchItemsAndAudits(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	userA := seedQuotaUser(t, db, "quota_batch_user_a", 500)
	userB := seedQuotaUser(t, db, "quota_batch_user_b", 700)

	ctx, recorder := newAdminQuotaContext(t, http.MethodPost, "/api/admin/quota/adjust/batch", map[string]any{
		"target_user_ids": []int{userA.Id, userB.Id},
		"delta":           100,
		"reason":          "batch_adjust",
		"remark":          "phase1-batch",
	})
	AdjustUserQuotaBatch(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var batch model.QuotaAdjustmentBatch
	require.NoError(t, db.Order("id desc").First(&batch).Error)
	require.Equal(t, 2, batch.TargetCount)
	require.Equal(t, 100, batch.Amount)

	var itemCount int64
	require.NoError(t, db.Model(&model.QuotaAdjustmentBatchItem{}).Where("batch_id = ?", batch.Id).Count(&itemCount).Error)
	require.Equal(t, int64(2), itemCount)

	accountA, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userA.Id)
	require.NoError(t, err)
	require.Equal(t, 600, accountA.Balance)

	accountB, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userB.Id)
	require.NoError(t, err)
	require.Equal(t, 800, accountB.Balance)

	var auditCount int64
	require.NoError(t, db.Model(&model.AdminAuditLog{}).Where("action_module = ? AND action_type = ?", "quota", "adjust_batch").Count(&auditCount).Error)
	require.Equal(t, int64(1), auditCount)
}

func TestAdjustUserQuotaBatchReturnsPartialSuccessDetails(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	userA := seedQuotaUser(t, db, "quota_batch_fail_user_a", 50)
	userB := seedQuotaUser(t, db, "quota_batch_fail_user_b", 200)

	ctx, recorder := newAdminQuotaContext(t, http.MethodPost, "/api/admin/quota/adjust/batch", map[string]any{
		"target_user_ids": []int{userA.Id, userB.Id},
		"delta":           -100,
		"reason":          "batch_partial_adjust",
		"remark":          "phase1-partial",
	})
	AdjustUserQuotaBatch(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, float64(2), response.Data["target_count"])
	require.Equal(t, float64(1), response.Data["success_count"])
	require.Equal(t, float64(1), response.Data["failed_count"])

	failedItems, ok := response.Data["failed_items"].([]any)
	require.True(t, ok)
	require.Len(t, failedItems, 1)

	failedItem, ok := failedItems[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(userA.Id), failedItem["target_user_id"])
	require.Contains(t, failedItem["error_message"], "insufficient quota balance")

	accountA, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userA.Id)
	require.NoError(t, err)
	require.Equal(t, 50, accountA.Balance)

	accountB, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userB.Id)
	require.NoError(t, err)
	require.Equal(t, 100, accountB.Balance)

	var successItems int64
	require.NoError(t, db.Model(&model.QuotaAdjustmentBatchItem{}).
		Where("target_user_id = ? AND status = ?", userB.Id, model.CommonStatusEnabled).
		Count(&successItems).Error)
	require.Equal(t, int64(1), successItems)

	var failedBatchItems []model.QuotaAdjustmentBatchItem
	require.NoError(t, db.Where("target_user_id = ? AND status = ?", userA.Id, model.CommonStatusDisabled).
		Find(&failedBatchItems).Error)
	require.Len(t, failedBatchItems, 1)
	require.Contains(t, failedBatchItems[0].ErrorMessage, "insufficient quota balance")
}

func TestAgentCannotReadOrAdjustUnownedUserQuota(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	agent := seedQuotaUser(t, db, "quota_agent_operator", 200)
	agent.Role = common.RoleAdminUser
	agent.UserType = model.UserTypeAgent
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", agent.Id).Updates(map[string]any{
		"role":      agent.Role,
		"user_type": agent.UserType,
	}).Error)
	require.NoError(t, db.Create(&model.AgentQuotaPolicy{
		AgentUserId:           agent.Id,
		AllowRechargeUser:     true,
		AllowReclaimQuota:     true,
		MaxSingleAdjustAmount: 0,
		Status:                model.CommonStatusEnabled,
		UpdatedAtTs:           common.GetTimestamp(),
	}).Error)

	ownedUser := seedQuotaUser(t, db, "quota_owned_user", 400)
	unownedUser := seedQuotaUser(t, db, "quota_unowned_user", 500)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", ownedUser.Id).Update("parent_agent_id", agent.Id).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   ownedUser.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "quota_management", Action: "read_summary"},
		permissionGrant{Resource: "quota_management", Action: "adjust"},
	)

	summaryCtx, summaryRecorder := newAdminQuotaContextWithOperator(t, http.MethodGet, "/api/admin/users/"+strconv.Itoa(unownedUser.Id)+"/quota-summary", nil, agent.Id, common.RoleAdminUser)
	summaryCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(unownedUser.Id)}}
	GetUserQuotaSummary(summaryCtx)

	var summaryResponse adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(summaryRecorder.Body.Bytes(), &summaryResponse))
	require.False(t, summaryResponse.Success)

	adjustCtx, adjustRecorder := newAdminQuotaContextWithOperator(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": unownedUser.Id,
		"delta":          50,
		"reason":         "agent_adjust",
	}, agent.Id, common.RoleAdminUser)
	AdjustUserQuota(adjustCtx)

	var adjustResponse adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(adjustRecorder.Body.Bytes(), &adjustResponse))
	require.False(t, adjustResponse.Success)

	ownedAdjustCtx, ownedAdjustRecorder := newAdminQuotaContextWithOperator(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": ownedUser.Id,
		"delta":          50,
		"reason":         "agent_adjust",
	}, agent.Id, common.RoleAdminUser)
	AdjustUserQuota(ownedAdjustCtx)

	var ownedAdjustResponse adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(ownedAdjustRecorder.Body.Bytes(), &ownedAdjustResponse))
	require.True(t, ownedAdjustResponse.Success)
}

func TestAgentAdjustUserQuotaRejectsWhenRechargeExceedsAgentBalance(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	agent := seedQuotaUser(t, db, "agent_balance_guard", 80)
	agent.Role = common.RoleAdminUser
	agent.UserType = model.UserTypeAgent
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", agent.Id).Updates(map[string]any{
		"role":      agent.Role,
		"user_type": agent.UserType,
	}).Error)
	require.NoError(t, db.Create(&model.AgentQuotaPolicy{
		AgentUserId:           agent.Id,
		AllowRechargeUser:     true,
		AllowReclaimQuota:     true,
		MaxSingleAdjustAmount: 0,
		Status:                model.CommonStatusEnabled,
		UpdatedAtTs:           common.GetTimestamp(),
	}).Error)

	target := seedQuotaUser(t, db, "agent_balance_guard_target", 20)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", target.Id).Update("parent_agent_id", agent.Id).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   target.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "quota_management", Action: "adjust"},
	)

	ctx, recorder := newAdminQuotaContextWithOperator(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": target.Id,
		"delta":          100,
		"reason":         "agent_adjust",
	}, agent.Id, common.RoleAdminUser)
	AdjustUserQuota(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Contains(t, response.Message, "insufficient agent quota balance")
}

func TestAgentAdjustUserQuotaTransfersBalanceAndCreatesDualLedger(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	agent := seedQuotaUser(t, db, "agent_transfer_operator", 500)
	agent.Role = common.RoleAdminUser
	agent.UserType = model.UserTypeAgent
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", agent.Id).Updates(map[string]any{
		"role":      agent.Role,
		"user_type": agent.UserType,
	}).Error)
	require.NoError(t, db.Create(&model.AgentQuotaPolicy{
		AgentUserId:           agent.Id,
		AllowRechargeUser:     true,
		AllowReclaimQuota:     true,
		MaxSingleAdjustAmount: 0,
		Status:                model.CommonStatusEnabled,
		UpdatedAtTs:           common.GetTimestamp(),
	}).Error)

	target := seedQuotaUser(t, db, "agent_transfer_target", 100)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", target.Id).Update("parent_agent_id", agent.Id).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   target.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "quota_management", Action: "adjust"},
	)

	ctx, recorder := newAdminQuotaContextWithOperator(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": target.Id,
		"delta":          120,
		"reason":         "agent_adjust",
	}, agent.Id, common.RoleAdminUser)
	AdjustUserQuota(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	agentAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, agent.Id)
	require.NoError(t, err)
	targetAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, target.Id)
	require.NoError(t, err)
	require.Equal(t, 380, agentAccount.Balance)
	require.Equal(t, 220, targetAccount.Balance)

	var order model.QuotaTransferOrder
	require.NoError(t, db.Where("transfer_type = ?", model.TransferTypeAgentRecharge).First(&order).Error)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("transfer_order_id = ?", order.Id).Order("account_id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 2)
	require.Equal(t, model.LedgerDirectionOut, ledgers[0].Direction)
	require.Equal(t, agentAccount.Id, ledgers[0].AccountId)
	require.Equal(t, 500, ledgers[0].BalanceBefore)
	require.Equal(t, 380, ledgers[0].BalanceAfter)
	require.Equal(t, model.LedgerDirectionIn, ledgers[1].Direction)
	require.Equal(t, targetAccount.Id, ledgers[1].AccountId)
	require.Equal(t, 100, ledgers[1].BalanceBefore)
	require.Equal(t, 220, ledgers[1].BalanceAfter)
}

func TestAgentReclaimQuotaReturnsBalanceAndCreatesDualLedger(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	agent := seedQuotaUser(t, db, "agent_reclaim_operator", 200)
	agent.Role = common.RoleAdminUser
	agent.UserType = model.UserTypeAgent
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", agent.Id).Updates(map[string]any{
		"role":      agent.Role,
		"user_type": agent.UserType,
	}).Error)
	require.NoError(t, db.Create(&model.AgentQuotaPolicy{
		AgentUserId:           agent.Id,
		AllowRechargeUser:     true,
		AllowReclaimQuota:     true,
		MaxSingleAdjustAmount: 0,
		Status:                model.CommonStatusEnabled,
		UpdatedAtTs:           common.GetTimestamp(),
	}).Error)

	target := seedQuotaUser(t, db, "agent_reclaim_target", 140)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", target.Id).Update("parent_agent_id", agent.Id).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   target.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "quota_management", Action: "adjust"},
	)

	ctx, recorder := newAdminQuotaContextWithOperator(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": target.Id,
		"delta":          -60,
		"reason":         "agent_reclaim",
	}, agent.Id, common.RoleAdminUser)
	AdjustUserQuota(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	agentAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, agent.Id)
	require.NoError(t, err)
	targetAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, target.Id)
	require.NoError(t, err)
	require.Equal(t, 260, agentAccount.Balance)
	require.Equal(t, 80, targetAccount.Balance)

	var order model.QuotaTransferOrder
	require.NoError(t, db.Where("transfer_type = ?", model.TransferTypeAgentReclaim).First(&order).Error)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("transfer_order_id = ?", order.Id).Order("account_id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 2)
	require.Equal(t, model.LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, agentAccount.Id, ledgers[0].AccountId)
	require.Equal(t, 200, ledgers[0].BalanceBefore)
	require.Equal(t, 260, ledgers[0].BalanceAfter)
	require.Equal(t, model.LedgerDirectionOut, ledgers[1].Direction)
	require.Equal(t, targetAccount.Id, ledgers[1].AccountId)
	require.Equal(t, 140, ledgers[1].BalanceBefore)
	require.Equal(t, 80, ledgers[1].BalanceAfter)
}

func TestAgentAdjustUserQuotaBatchTransfersQuotaWithPartialFailures(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	agent := seedQuotaUser(t, db, "agent_batch_operator", 150)
	agent.Role = common.RoleAdminUser
	agent.UserType = model.UserTypeAgent
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", agent.Id).Updates(map[string]any{
		"role":      agent.Role,
		"user_type": agent.UserType,
	}).Error)
	require.NoError(t, db.Create(&model.AgentQuotaPolicy{
		AgentUserId:           agent.Id,
		AllowRechargeUser:     true,
		AllowReclaimQuota:     true,
		MaxSingleAdjustAmount: 0,
		Status:                model.CommonStatusEnabled,
		UpdatedAtTs:           common.GetTimestamp(),
	}).Error)

	userA := seedQuotaUser(t, db, "agent_batch_target_a", 20)
	userB := seedQuotaUser(t, db, "agent_batch_target_b", 30)
	for _, target := range []model.User{userA, userB} {
		require.NoError(t, db.Model(&model.User{}).Where("id = ?", target.Id).Update("parent_agent_id", agent.Id).Error)
		require.NoError(t, db.Create(&model.AgentUserRelation{
			AgentUserId: agent.Id,
			EndUserId:   target.Id,
			BindSource:  "manual",
			BindAt:      common.GetTimestamp(),
			Status:      model.CommonStatusEnabled,
			CreatedAtTs: common.GetTimestamp(),
		}).Error)
	}
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "quota_management", Action: "adjust_batch"},
	)

	ctx, recorder := newAdminQuotaContextWithOperator(t, http.MethodPost, "/api/admin/quota/adjust/batch", map[string]any{
		"target_user_ids": []int{userA.Id, userB.Id},
		"delta":           100,
		"reason":          "agent_batch_adjust",
	}, agent.Id, common.RoleAdminUser)
	AdjustUserQuotaBatch(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, float64(1), response.Data["success_count"])
	require.Equal(t, float64(1), response.Data["failed_count"])

	agentAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, agent.Id)
	require.NoError(t, err)
	accountA, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userA.Id)
	require.NoError(t, err)
	accountB, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userB.Id)
	require.NoError(t, err)
	require.Equal(t, 50, agentAccount.Balance)
	require.Equal(t, 120, accountA.Balance)
	require.Equal(t, 30, accountB.Balance)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Order("id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 2)
}

func TestAgentQuotaScopeOverrideAllAllowsUnownedUserQuota(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	agent := seedQuotaUser(t, db, "quota_scope_all_agent", 0)
	agent.Role = common.RoleAdminUser
	agent.UserType = model.UserTypeAgent
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", agent.Id).Updates(map[string]any{
		"role":      agent.Role,
		"user_type": agent.UserType,
	}).Error)

	unownedUser := seedQuotaUser(t, db, "quota_scope_all_target", 500)
	require.NoError(t, db.Create(&model.UserDataScopeOverride{
		UserId:      agent.Id,
		ResourceKey: "quota_management",
		ScopeType:   model.ScopeTypeAll,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "quota_management", Action: "read_summary"},
	)

	summaryCtx, summaryRecorder := newAdminQuotaContextWithOperator(t, http.MethodGet, "/api/admin/users/"+strconv.Itoa(unownedUser.Id)+"/quota-summary", nil, agent.Id, common.RoleAdminUser)
	summaryCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(unownedUser.Id)}}
	GetUserQuotaSummary(summaryCtx)

	var summaryResponse adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(summaryRecorder.Body.Bytes(), &summaryResponse))
	require.True(t, summaryResponse.Success)
	require.Equal(t, float64(unownedUser.Id), summaryResponse.Data["user_id"])
}

func TestAgentQuotaLedgerOnlyReturnsOwnedUsers(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	agent := seedQuotaUser(t, db, "ledger_agent_operator", 0)
	agent.Role = common.RoleAdminUser
	agent.UserType = model.UserTypeAgent
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", agent.Id).Updates(map[string]any{
		"role":      agent.Role,
		"user_type": agent.UserType,
	}).Error)

	ownedUser := seedQuotaUser(t, db, "ledger_owned_user", 400)
	unownedUser := seedQuotaUser(t, db, "ledger_unowned_user", 500)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", ownedUser.Id).Update("parent_agent_id", agent.Id).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   ownedUser.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "quota_management", Action: "ledger_read"},
	)

	ownedAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, ownedUser.Id)
	require.NoError(t, err)
	unownedAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, unownedUser.Id)
	require.NoError(t, err)

	require.NoError(t, db.Create(&model.QuotaLedger{
		BizNo:            "owned_ledger_1",
		AccountId:        ownedAccount.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           10,
		BalanceBefore:    400,
		BalanceAfter:     410,
		OperatorUserId:   agent.Id,
		OperatorUserType: model.UserTypeAgent,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.QuotaLedger{
		BizNo:            "unowned_ledger_1",
		AccountId:        unownedAccount.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           10,
		BalanceBefore:    500,
		BalanceAfter:     510,
		OperatorUserId:   agent.Id,
		OperatorUserType: model.UserTypeAgent,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)

	listCtx, recorder := newAdminQuotaContextWithOperator(t, http.MethodGet, "/api/admin/quota/ledger?p=1&page_size=10", nil, agent.Id, common.RoleAdminUser)
	GetQuotaLedger(listCtx)

	var response adminQuotaPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(ownedAccount.Id), firstItem["account_id"])
}

func TestAgentQuotaLedgerIncludesOwnAccountEntries(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	agent := seedQuotaUser(t, db, "ledger_agent_self", 300)
	agent.Role = common.RoleAdminUser
	agent.UserType = model.UserTypeAgent
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", agent.Id).Updates(map[string]any{
		"role":      agent.Role,
		"user_type": agent.UserType,
	}).Error)

	ownedUser := seedQuotaUser(t, db, "ledger_agent_self_owned", 100)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", ownedUser.Id).Update("parent_agent_id", agent.Id).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   ownedUser.Id,
		BindSource:  "manual",
		BindAt:      common.GetTimestamp(),
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	grantPermissionActions(t, db, agent.Id, "agent",
		permissionGrant{Resource: "quota_management", Action: "ledger_read"},
	)

	agentAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, agent.Id)
	require.NoError(t, err)
	ownedAccount, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, ownedUser.Id)
	require.NoError(t, err)

	require.NoError(t, db.Create(&model.QuotaLedger{
		BizNo:            "agent_self_ledger_1",
		AccountId:        agentAccount.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionOut,
		Amount:           50,
		BalanceBefore:    300,
		BalanceAfter:     250,
		OperatorUserId:   agent.Id,
		OperatorUserType: model.UserTypeAgent,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.QuotaLedger{
		BizNo:            "agent_owned_ledger_1",
		AccountId:        ownedAccount.Id,
		EntryType:        model.LedgerEntryAdjust,
		Direction:        model.LedgerDirectionIn,
		Amount:           50,
		BalanceBefore:    100,
		BalanceAfter:     150,
		OperatorUserId:   agent.Id,
		OperatorUserType: model.UserTypeAgent,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)

	listCtx, recorder := newAdminQuotaContextWithOperator(t, http.MethodGet, "/api/admin/quota/ledger?p=1&page_size=10", nil, agent.Id, common.RoleAdminUser)
	GetQuotaLedger(listCtx)

	var response adminQuotaPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 2, response.Data.Total)
}

func TestAdjustUserQuotaRequiresActionPermissionForAdmin(t *testing.T) {
	db := setupAdminQuotaTestDB(t)
	operator := seedQuotaUser(t, db, "quota_operator_no_grant", 0)
	operator.Role = common.RoleAdminUser
	operator.UserType = model.UserTypeAdmin
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", operator.Id).Updates(map[string]any{
		"role":      operator.Role,
		"user_type": operator.UserType,
	}).Error)
	target := seedQuotaUser(t, db, "quota_target_for_deny", 100)

	ctx, recorder := newAdminQuotaContextWithOperator(t, http.MethodPost, "/api/admin/quota/adjust", map[string]any{
		"target_user_id": target.Id,
		"delta":          10,
		"reason":         "no_permission",
	}, operator.Id, operator.Role)
	AdjustUserQuota(ctx)

	var response adminQuotaAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
}
