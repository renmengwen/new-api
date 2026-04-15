package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type setupAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setupSetupControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)

	previousSetup := constant.Setup
	previousOptionMap := common.OptionMap
	previousSelfUseModeEnabled := operation_setting.SelfUseModeEnabled
	previousDemoSiteEnabled := operation_setting.DemoSiteEnabled
	previousUsingSQLite := common.UsingSQLite
	previousUsingMySQL := common.UsingMySQL
	previousUsingPostgreSQL := common.UsingPostgreSQL
	previousRedisEnabled := common.RedisEnabled

	constant.Setup = false
	common.OptionMap = make(map[string]string)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	operation_setting.SelfUseModeEnabled = false
	operation_setting.DemoSiteEnabled = false

	db, err := gorm.Open(sqlite.Open("file:setup_controller?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.QuotaAccount{},
		&model.QuotaLedger{},
		&model.Option{},
		&model.Setup{},
	))

	t.Cleanup(func() {
		constant.Setup = previousSetup
		common.OptionMap = previousOptionMap
		operation_setting.SelfUseModeEnabled = previousSelfUseModeEnabled
		operation_setting.DemoSiteEnabled = previousDemoSiteEnabled
		common.UsingSQLite = previousUsingSQLite
		common.UsingMySQL = previousUsingMySQL
		common.UsingPostgreSQL = previousUsingPostgreSQL
		common.RedisEnabled = previousRedisEnabled

		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newSetupContext(t *testing.T, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/setup", io.NopCloser(bytes.NewReader(payload)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	return ctx, recorder
}

func TestPostSetupCreatesRootQuotaAccountAndOpeningLedger(t *testing.T) {
	db := setupSetupControllerTestDB(t)

	ctx, recorder := newSetupContext(t, map[string]any{
		"username":           "rootsetup01",
		"password":           "12345678",
		"confirmPassword":    "12345678",
		"SelfUseModeEnabled": true,
		"DemoSiteEnabled":    false,
	})

	PostSetup(ctx)

	var response setupAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var rootUser model.User
	require.NoError(t, db.Where("username = ?", "rootsetup01").First(&rootUser).Error)
	require.Equal(t, common.RoleRootUser, rootUser.Role)
	require.Equal(t, model.UserTypeRoot, rootUser.UserType)
	require.Equal(t, 100000000, rootUser.Quota)

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, rootUser.Id)
	require.NoError(t, err)
	require.Equal(t, 100000000, account.Balance)

	var ledgers []model.QuotaLedger
	require.NoError(t, db.Where("account_id = ?", account.Id).Order("id asc").Find(&ledgers).Error)
	require.Len(t, ledgers, 1)
	require.Equal(t, model.LedgerEntryOpening, ledgers[0].EntryType)
	require.Equal(t, model.LedgerDirectionIn, ledgers[0].Direction)
	require.Equal(t, 100000000, ledgers[0].Amount)
	require.Equal(t, "root_bootstrap", ledgers[0].SourceType)
	require.Equal(t, rootUser.Id, ledgers[0].SourceId)
}
