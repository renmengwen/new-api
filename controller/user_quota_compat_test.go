package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type legacyUpdateUserAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setupLegacyUpdateUserQuotaTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.QuotaAccount{},
		&model.QuotaTransferOrder{},
		&model.QuotaLedger{},
		&model.AdminAuditLog{},
		&model.Log{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newLegacyUpdateUserContext(t *testing.T, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/user/", io.NopCloser(bytes.NewReader(payload)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 999)
	ctx.Set("role", common.RoleRootUser)
	return ctx, recorder
}

func seedLegacyUpdateUserQuotaTarget(t *testing.T, db *gorm.DB, username string, userType string, role int, quota int) model.User {
	t.Helper()

	user := model.User{
		Username:    username,
		Password:    "hashed-password",
		DisplayName: username,
		Role:        role,
		Status:      common.UserStatusEnabled,
		UserType:    userType,
		Group:       "default",
		Remark:      "before-remark",
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

func TestUpdateUserQuotaChangeCreatesLedgerForEndUser(t *testing.T) {
	db := setupLegacyUpdateUserQuotaTestDB(t)
	user := seedLegacyUpdateUserQuotaTarget(t, db, "legacy_end_user", model.UserTypeEndUser, common.RoleCommonUser, 600)

	ctx, recorder := newLegacyUpdateUserContext(t, map[string]any{
		"id":           user.Id,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"password":     "",
		"group":        user.Group,
		"role":         user.Role,
		"quota":        950,
		"remark":       user.Remark,
	})
	UpdateUser(ctx)

	var response legacyUpdateUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 950, account.Balance)
	require.Equal(t, 350, account.TotalAdjustedIn)

	var ledger model.QuotaLedger
	require.NoError(t, db.Where("source_type = ? AND source_id = ?", "admin_quota_adjust", user.Id).First(&ledger).Error)
	require.Equal(t, 600, ledger.BalanceBefore)
	require.Equal(t, 950, ledger.BalanceAfter)
	require.Equal(t, 350, ledger.Amount)
	require.Equal(t, "manual_adjust", ledger.Reason)
}

func TestUpdateUserQuotaChangeCreatesLedgerForAgentAndKeepsProfileFields(t *testing.T) {
	db := setupLegacyUpdateUserQuotaTestDB(t)
	user := seedLegacyUpdateUserQuotaTarget(t, db, "legacy_agent_user", model.UserTypeAgent, common.RoleCommonUser, 1200)

	ctx, recorder := newLegacyUpdateUserContext(t, map[string]any{
		"id":           user.Id,
		"username":     user.Username,
		"display_name": "Agent After",
		"password":     "",
		"group":        "vip",
		"role":         user.Role,
		"quota":        1000,
		"remark":       "after-remark",
	})
	UpdateUser(ctx)

	var response legacyUpdateUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	require.Equal(t, "Agent After", reloaded.DisplayName)
	require.Equal(t, "vip", reloaded.Group)
	require.Equal(t, "after-remark", reloaded.Remark)
	require.Equal(t, 1000, reloaded.Quota)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 1000, account.Balance)
	require.Equal(t, 200, account.TotalAdjustedOut)

	var ledger model.QuotaLedger
	require.NoError(t, db.Where("operator_user_id = ? AND account_id = ?", 999, account.Id).First(&ledger).Error)
	require.Equal(t, model.LedgerDirectionOut, ledger.Direction)
	require.Equal(t, 1200, ledger.BalanceBefore)
	require.Equal(t, 1000, ledger.BalanceAfter)
	require.Equal(t, 200, ledger.Amount)
	require.Equal(t, "manual_adjust", ledger.Reason)
}

func TestUpdateUserQuotaChangeReconcilesLegacyBalanceDriftBeforeAdjust(t *testing.T) {
	db := setupLegacyUpdateUserQuotaTestDB(t)
	user := seedLegacyUpdateUserQuotaTarget(t, db, "agents1", model.UserTypeAgent, common.RoleCommonUser, 100000000)
	require.NoError(t, db.Model(&model.QuotaAccount{}).
		Where("owner_type = ? AND owner_id = ?", model.QuotaOwnerTypeUser, user.Id).
		Updates(map[string]any{
			"balance":            0,
			"total_recharged":    0,
			"total_adjusted_in":  0,
			"total_adjusted_out": 0,
		}).Error)

	ctx, recorder := newLegacyUpdateUserContext(t, map[string]any{
		"id":           user.Id,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"password":     "agents1agents1",
		"group":        user.Group,
		"role":         user.Role,
		"quota":        1000000,
		"remark":       user.Remark,
	})
	UpdateUser(ctx)

	var response legacyUpdateUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, user.Id)
	require.NoError(t, err)
	require.Equal(t, 1000000, account.Balance)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 2)
	require.Equal(t, "quota_reconcile", ledgers[0].SourceType)
	require.Equal(t, 0, ledgers[0].BalanceBefore)
	require.Equal(t, 100000000, ledgers[0].BalanceAfter)
	require.Equal(t, model.LedgerDirectionOut, ledgers[1].Direction)
	require.Equal(t, 100000000, ledgers[1].BalanceBefore)
	require.Equal(t, 1000000, ledgers[1].BalanceAfter)
}
