package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type legacyUserDetailResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func TestCreateUserPersistsAllowedTokenGroupsInLegacyFlow(t *testing.T) {
	db := setupUserValidationTestDB(t)

	ctx, recorder := newCreateUserContext(t, map[string]any{
		"username":                     "legacy-allowed-user",
		"password":                     "12345678",
		"group":                        "default",
		"allowed_token_groups_enabled": true,
		"allowed_token_groups":         []string{"default", "vip"},
	})

	CreateUser(ctx)

	var response userValidationAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var user model.User
	require.NoError(t, db.Where("username = ?", "legacy-allowed-user").First(&user).Error)

	settingMap := make(map[string]any)
	require.NoError(t, common.UnmarshalJsonStr(user.Setting, &settingMap))
	require.Equal(t, true, settingMap["allowed_token_groups_enabled"])
	require.Equal(t, []any{"default", "vip"}, settingMap["allowed_token_groups"])
}

func TestUpdateUserPersistsAllowedTokenGroupsInLegacyFlow(t *testing.T) {
	db := setupLegacyUpdateUserQuotaTestDB(t)
	user := seedLegacyUpdateUserQuotaTarget(t, db, "legacy_upd_allow", model.UserTypeEndUser, common.RoleCommonUser, 600)

	ctx, recorder := newLegacyUpdateUserContext(t, map[string]any{
		"id":                           user.Id,
		"username":                     user.Username,
		"display_name":                 user.DisplayName,
		"password":                     "",
		"group":                        "default",
		"role":                         user.Role,
		"quota":                        user.Quota,
		"remark":                       user.Remark,
		"allowed_token_groups_enabled": true,
		"allowed_token_groups":         []string{"default", "vip"},
	})
	UpdateUser(ctx)

	var response legacyUpdateUserAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)

	settingMap := make(map[string]any)
	require.NoError(t, common.UnmarshalJsonStr(reloaded.Setting, &settingMap))
	require.Equal(t, true, settingMap["allowed_token_groups_enabled"])
	require.Equal(t, []any{"default", "vip"}, settingMap["allowed_token_groups"])
}

func TestGetUserReturnsAllowedTokenGroupsInLegacyFlow(t *testing.T) {
	db := setupUserValidationTestDB(t)

	user := model.User{
		Username:    "legacy_detail_allowed",
		Password:    "hashed-password",
		DisplayName: "legacy_detail_allowed",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		Setting:     `{"allowed_token_groups_enabled":true,"allowed_token_groups":["default","vip"]}`,
		AffCode:     "legacydetailallowed",
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/"+strconv.Itoa(user.Id), nil)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(user.Id)}}
	ctx.Set("id", 1)
	ctx.Set("role", common.RoleRootUser)

	GetUser(ctx)

	var response legacyUserDetailResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	require.Equal(t, true, response.Data["allowed_token_groups_enabled"])
	require.Equal(t, []any{"default", "vip"}, response.Data["allowed_token_groups"])
}
