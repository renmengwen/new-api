package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type updateUserSettingResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setupUserSettingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:user_setting?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newUpdateUserSettingContext(t *testing.T, userID int, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/user/setting", io.NopCloser(bytes.NewReader(payload)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", userID)
	return ctx, recorder
}

func TestUpdateUserSettingPreservesAllowedTokenGroups(t *testing.T) {
	db := setupUserSettingTestDB(t)

	user := model.User{
		Username: "user_setting_preserve",
		Password: "hashed-password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		Setting:  `{"allowed_token_groups_enabled":true,"allowed_token_groups":["default","vip"],"sidebar_modules":"{\"console\":{\"enabled\":true}}"}`,
		AffCode:  "usersettingpreserve",
	}
	require.NoError(t, db.Create(&user).Error)

	ctx, recorder := newUpdateUserSettingContext(t, user.Id, map[string]any{
		"notify_type":                     "email",
		"quota_warning_threshold":         10,
		"notification_email":              "demo@example.com",
		"accept_unset_model_ratio_model":  true,
		"record_ip_log":                   true,
	})
	UpdateUserSetting(ctx)

	var response updateUserSettingResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)

	settingMap := make(map[string]any)
	require.NoError(t, common.UnmarshalJsonStr(reloaded.Setting, &settingMap))
	require.Equal(t, true, settingMap["allowed_token_groups_enabled"])
	require.Equal(t, []any{"default", "vip"}, settingMap["allowed_token_groups"])
}
