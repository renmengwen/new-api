package controller

import (
	"bytes"
	"fmt"
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

type userValidationAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setupUserValidationTestDB(t *testing.T) *gorm.DB {
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
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newCreateUserContext(t *testing.T, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 1)
	ctx.Set("role", common.RoleRootUser)
	return ctx, recorder
}

func TestCreateUserReturnsFriendlyMessageForDuplicateUsername(t *testing.T) {
	db := setupUserValidationTestDB(t)

	require.NoError(t, db.Create(&model.User{
		Username:    "duplicate-user",
		Password:    "hashed-password",
		DisplayName: "Existing User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
	}).Error)

	ctx, recorder := newCreateUserContext(t, map[string]any{
		"username": "duplicate-user",
		"password": "12345678",
		"email":    "duplicate-user-new@example.com",
	})

	CreateUser(ctx)

	var response userValidationAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "用户名已存在，请更换后重试", response.Message)
}

func TestCreateUserReturnsFriendlyMessageForInvalidPasswordRule(t *testing.T) {
	setupUserValidationTestDB(t)

	ctx, recorder := newCreateUserContext(t, map[string]any{
		"username": "password-rule-user",
		"password": "1234567",
		"email":    "password-rule-user@example.com",
	})

	CreateUser(ctx)

	var response userValidationAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "密码长度需为 8 到 20 位", response.Message)
}
