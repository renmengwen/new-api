package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type userGroupsResponse struct {
	Success bool                      `json:"success"`
	Message string                    `json:"message"`
	Data    map[string]map[string]any `json:"data"`
}

func setupGroupControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:group_controller?mode=memory&cache=shared"), &gorm.Config{})
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

func withGroupControllerConfig(t *testing.T, usableGroupsJSON string, groupRatioJSON string) {
	t.Helper()

	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalGroupRatios := ratio_setting.GroupRatio2JSONString()
	originalGroupGroupRatios := ratio_setting.GroupGroupRatio2JSONString()

	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatios))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(originalGroupGroupRatios))
	})

	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(usableGroupsJSON))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(groupRatioJSON))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{"default":{"beta":1.7,"vip":2.3}}`))
}

func newUserGroupsContext(t *testing.T, target string, userID int, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, target, nil)
	ctx.Set("id", userID)
	ctx.Set("role", role)
	return ctx, recorder
}

func TestGetUserGroupsModeTokenUsesAllowedTokenGroupsWhenEnabled(t *testing.T) {
	db := setupGroupControllerTestDB(t)
	withGroupControllerConfig(t, `{"default":"Default","vip":"VIP","beta":"Beta"}`, `{"default":1,"vip":2,"beta":3}`)

	user := model.User{
		Username: "token_mode_whitelist_user",
		Password: "hashed-password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		Setting:  `{"allowed_token_groups_enabled":true,"allowed_token_groups":["default","beta"]}`,
		AffCode:  "tokenmodewhitelist",
	}
	require.NoError(t, db.Create(&user).Error)

	ctx, recorder := newUserGroupsContext(t, "/api/user/self/groups?mode=token", user.Id, user.Role)
	GetUserGroups(ctx)

	var response userGroupsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	require.Contains(t, response.Data, "default")
	require.Contains(t, response.Data, "beta")
	require.NotContains(t, response.Data, "vip")
	require.Equal(t, 1.7, response.Data["beta"]["ratio"])
}

func TestGetUserGroupsModeAssignableTokenReturnsOperatorDelegableGroups(t *testing.T) {
	db := setupGroupControllerTestDB(t)
	withGroupControllerConfig(t, `{"default":"Default","vip":"VIP","beta":"Beta"}`, `{"default":1,"vip":2,"beta":3}`)

	agent := model.User{
		Username: "assignable_agent_operator",
		Password: "hashed-password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		UserType: model.UserTypeAgent,
		Setting:  `{"allowed_token_groups_enabled":true,"allowed_token_groups":["default","beta"]}`,
		AffCode:  "assignableagentop",
	}
	require.NoError(t, db.Create(&agent).Error)

	ctx, recorder := newUserGroupsContext(t, "/api/user/self/groups?mode=assignable_token", agent.Id, agent.Role)
	GetUserGroups(ctx)

	var response userGroupsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	require.Contains(t, response.Data, "default")
	require.Contains(t, response.Data, "beta")
	require.NotContains(t, response.Data, "vip")
}

func TestGetUserGroupsModeTokenIncludesLegacyPrimaryGroupFromResolver(t *testing.T) {
	db := setupGroupControllerTestDB(t)
	withGroupControllerConfig(t, `{"default":"Default","beta":"Beta"}`, `{"default":1,"beta":3}`)

	user := model.User{
		Username: "token_mode_legacy_user",
		Password: "hashed-password",
		Group:    "legacy-private",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		Setting:  `{"allowed_token_groups_enabled":true,"allowed_token_groups":["legacy-private","beta"]}`,
		AffCode:  "tokenmodelegacy",
	}
	require.NoError(t, db.Create(&user).Error)

	ctx, recorder := newUserGroupsContext(t, "/api/user/self/groups?mode=token", user.Id, user.Role)
	GetUserGroups(ctx)

	var response userGroupsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	require.Contains(t, response.Data, "legacy-private")
	require.Contains(t, response.Data, "beta")
}
