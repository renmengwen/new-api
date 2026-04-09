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

func newAdminPermissionContextWithOperator(t *testing.T, method string, target string, body any, operatorId int, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	ctx, recorder := newAdminPermissionContext(t, method, target, body)
	ctx.Set("id", operatorId)
	ctx.Set("role", role)
	return ctx, recorder
}

func TestListUserPermissionTargets(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "end_user_permission_list",
		Password:    "hashed-password",
		DisplayName: "End User Permission List",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "End User Backend",
		ProfileType: model.UserTypeEndUser,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:        user.Id,
		ProfileId:     profile.Id,
		Status:        model.CommonStatusEnabled,
		EffectiveFrom: common.GetTimestamp(),
		CreatedAtTs:   common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodGet, "/api/admin/user-permissions/users?user_type=end_user&p=1&page_size=10", nil)
	ctx.Request.URL.RawQuery = "user_type=end_user&p=1&page_size=10"
	GetUserPermissionTargets(ctx)

	var response adminPermissionPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, profile.ProfileName, firstItem["profile_name"])
	require.Equal(t, model.UserTypeEndUser, firstItem["user_type"])
}

func TestListUserPermissionTargetsNormalizesLegacyRootUserType(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "legacy_root_permission_list",
		Password:    "hashed-password",
		DisplayName: "Legacy Root Permission List",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodGet, "/api/admin/user-permissions/users?p=1&page_size=10", nil)
	ctx.Request.URL.RawQuery = "p=1&page_size=10"
	GetUserPermissionTargets(ctx)

	var response adminPermissionPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)

	var found map[string]any
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		require.True(t, ok)
		if int(item["id"].(float64)) == user.Id {
			found = item
			break
		}
	}
	require.NotNil(t, found)
	require.Equal(t, model.UserTypeRoot, found["user_type"])
}

func TestGetUserPermissionDetailReturnsMergedOverrides(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "permission_detail_user",
		Password:    "hashed-password",
		DisplayName: "Permission Detail User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "End User Template",
		ProfileType: model.UserTypeEndUser,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: "user_management",
		ActionKey:   "read",
		Allowed:     true,
		ScopeType:   model.ScopeTypeAll,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:        user.Id,
		ProfileId:     profile.Id,
		Status:        model.CommonStatusEnabled,
		EffectiveFrom: common.GetTimestamp(),
		CreatedAtTs:   common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionOverride{
		UserId:      user.Id,
		ResourceKey: "user_management",
		ActionKey:   "read",
		Effect:      model.PermissionEffectDeny,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.UserMenuOverride{
		UserId:      user.Id,
		SectionKey:  "admin",
		ModuleKey:   "user-permissions",
		Effect:      model.MenuEffectShow,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.UserDataScopeOverride{
		UserId:      user.Id,
		ResourceKey: "user_management",
		ScopeType:   model.ScopeTypeAgentOnly,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodGet, "/api/admin/user-permissions/users/"+strconv.Itoa(user.Id), nil)
	ctx.Params = append(ctx.Params, gin.Param{Key: "id", Value: strconv.Itoa(user.Id)})
	GetUserPermissionDetail(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	effectiveActions, ok := response.Data["effective_actions"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, effectiveActions["user_management.read"])

	menuOverrides, ok := response.Data["menu_overrides"].([]any)
	require.True(t, ok)
	require.Len(t, menuOverrides, 1)

	dataScopeOverrides, ok := response.Data["data_scope_overrides"].([]any)
	require.True(t, ok)
	require.Len(t, dataScopeOverrides, 1)

	effectiveDataScopes, ok := response.Data["effective_data_scopes"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, model.ScopeTypeAgentOnly, effectiveDataScopes["user_management"])
}

func TestUpdateUserPermissionOverridesPersistsOverrides(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "permission_override_target",
		Password:    "hashed-password",
		DisplayName: "Permission Override Target",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodPut, "/api/admin/user-permissions/users/"+strconv.Itoa(user.Id)+"/overrides", map[string]any{
		"action_overrides": []map[string]any{
			{
				"resource_key": "quota_management",
				"action_key":   "ledger_read",
				"effect":       model.PermissionEffectAllow,
			},
		},
		"menu_overrides": []map[string]any{
			{
				"section_key": "admin",
				"module_key":  "quota-ledger",
				"effect":      model.MenuEffectShow,
			},
		},
		"data_scope_overrides": []map[string]any{
			{
				"resource_key": "quota_management",
				"scope_type":   model.ScopeTypeAssigned,
				"scope_value":  []int{101, 202},
			},
		},
	})
	ctx.Params = append(ctx.Params, gin.Param{Key: "id", Value: strconv.Itoa(user.Id)})

	UpdateUserPermissionOverrides(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var actionRows []model.UserPermissionOverride
	require.NoError(t, db.Where("user_id = ?", user.Id).Find(&actionRows).Error)
	require.Len(t, actionRows, 1)
	require.Equal(t, model.PermissionEffectAllow, actionRows[0].Effect)

	var menuRows []model.UserMenuOverride
	require.NoError(t, db.Where("user_id = ?", user.Id).Find(&menuRows).Error)
	require.Len(t, menuRows, 1)
	require.Equal(t, "quota-ledger", menuRows[0].ModuleKey)

	var dataScopeRows []model.UserDataScopeOverride
	require.NoError(t, db.Where("user_id = ?", user.Id).Find(&dataScopeRows).Error)
	require.Len(t, dataScopeRows, 1)
	require.Equal(t, model.ScopeTypeAssigned, dataScopeRows[0].ScopeType)
	require.JSONEq(t, `[101,202]`, dataScopeRows[0].ScopeValueJSON)
}

func TestAgentUserPermissionTargetsOnlyReturnManagedEndUsers(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	agent := model.User{
		Username:    "agent_permission_operator",
		Password:    "hashed-password",
		DisplayName: "Agent Permission Operator",
		AffCode:     "agent_perm_operator",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
	}
	require.NoError(t, db.Create(&agent).Error)
	grantPermissionActions(t, db, agent.Id, model.UserTypeAgent, permissionGrant{Resource: "permission_management", Action: "read"})

	ownedUser := model.User{
		Username:      "agent_owned_end_user",
		Password:      "hashed-password",
		DisplayName:   "Owned End User",
		AffCode:       "agent_owned_end_user",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		UserType:      model.UserTypeEndUser,
		Group:         "default",
		ParentAgentId: agent.Id,
	}
	require.NoError(t, db.Create(&ownedUser).Error)

	adminUser := model.User{
		Username:    "agent_visible_admin_should_not_happen",
		Password:    "hashed-password",
		DisplayName: "Admin Should Not Show",
		AffCode:     "agent_admin_hidden",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
	}
	require.NoError(t, db.Create(&adminUser).Error)

	rootUser := model.User{
		Username:    "agent_visible_root_should_not_happen",
		Password:    "hashed-password",
		DisplayName: "Root Should Not Show",
		AffCode:     "agent_root_hidden",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeRoot,
		Group:       "default",
	}
	require.NoError(t, db.Create(&rootUser).Error)

	ctx, recorder := newAdminPermissionContextWithOperator(t, http.MethodGet, "/api/admin/user-permissions/users?p=1&page_size=10", nil, agent.Id, agent.Role)
	ctx.Request.URL.RawQuery = "p=1&page_size=10"
	GetUserPermissionTargets(ctx)

	var response adminPermissionPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, ownedUser.Username, firstItem["username"])
	require.Equal(t, model.UserTypeEndUser, firstItem["user_type"])
}

func TestAgentCannotReadUnmanagedUserPermissionDetail(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	agent := model.User{
		Username:    "agent_permission_detail_operator",
		Password:    "hashed-password",
		DisplayName: "Agent Permission Detail Operator",
		AffCode:     "agent_perm_detail",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
	}
	require.NoError(t, db.Create(&agent).Error)
	grantPermissionActions(t, db, agent.Id, model.UserTypeAgent, permissionGrant{Resource: "permission_management", Action: "read"})

	adminUser := model.User{
		Username:    "detail_admin_not_visible",
		Password:    "hashed-password",
		DisplayName: "Detail Admin",
		AffCode:     "detail_admin_hidden",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
	}
	require.NoError(t, db.Create(&adminUser).Error)

	ctx, recorder := newAdminPermissionContextWithOperator(t, http.MethodGet, "/api/admin/user-permissions/users/"+strconv.Itoa(adminUser.Id), nil, agent.Id, agent.Role)
	ctx.Params = append(ctx.Params, gin.Param{Key: "id", Value: strconv.Itoa(adminUser.Id)})
	GetUserPermissionDetail(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
}
