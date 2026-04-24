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
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRelayErrorLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	model.DB = db
	model.LOG_DB = db
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))
	return db
}

func TestRelayRecordsErrorLogWhenRequestValidationFails(t *testing.T) {
	db := setupRelayErrorLogTestDB(t)

	originalErrorLogEnabled := constant.ErrorLogEnabled
	constant.ErrorLogEnabled = true
	t.Cleanup(func() {
		constant.ErrorLogEnabled = originalErrorLogEnabled
	})

	user := model.User{
		Username: "relay_error_user",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", io.NopCloser(bytes.NewReader([]byte(`{"model":"gpt-test"`))))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", user.Id)
	ctx.Set("username", user.Username)
	ctx.Set("token_id", 42)
	ctx.Set("token_name", "test-token")
	ctx.Set("group", "default")
	ctx.Set("original_model", "gpt-test")
	ctx.Set(common.RequestIdKey, "req-relay-error-log")

	Relay(ctx, types.RelayFormatOpenAI)

	var logs []model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeError).Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, user.Id, logs[0].UserId)
	require.Equal(t, "gpt-test", logs[0].ModelName)
	require.Equal(t, "test-token", logs[0].TokenName)
	require.Equal(t, 42, logs[0].TokenId)
	require.Equal(t, "req-relay-error-log", logs[0].RequestId)
	require.Contains(t, logs[0].Content, "status_code=500")
}
