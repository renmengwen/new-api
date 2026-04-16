package controller

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreatePermissionTemplate(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	ctx, recorder := newAdminPermissionContext(t, http.MethodPost, "/api/admin/permission-templates", map[string]any{
		"profile_name": "Agent Manager",
		"profile_type": model.UserTypeAgent,
		"description":  "agent profile template",
		"status":       model.CommonStatusEnabled,
		"items": []map[string]any{
			{
				"resource_key": "user_management",
				"action_key":   "read",
				"allowed":      true,
			},
			{
				"resource_key": "quota_management",
				"action_key":   "ledger_read",
				"allowed":      true,
			},
		},
	})

	CreatePermissionTemplate(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var profile model.PermissionProfile
	require.NoError(t, db.Where("profile_name = ?", "Agent Manager").First(&profile).Error)
	require.Equal(t, model.UserTypeAgent, profile.ProfileType)

	var items []model.PermissionProfileItem
	require.NoError(t, db.Where("profile_id = ?", profile.Id).Order("id asc").Find(&items).Error)
	require.Len(t, items, 2)
	require.Equal(t, "user_management", items[0].ResourceKey)
}

func TestCreatePermissionTemplatePersistsMenuAndDataScopeItems(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	ctx, recorder := newAdminPermissionContext(t, http.MethodPost, "/api/admin/permission-templates", map[string]any{
		"profile_name": "Agent Console Template",
		"profile_type": model.UserTypeAgent,
		"description":  "agent template with menu and scope defaults",
		"status":       model.CommonStatusEnabled,
		"items": []map[string]any{
			{
				"resource_key": "user_management",
				"action_key":   "read",
				"allowed":      true,
			},
		},
		"menu_items": []map[string]any{
			{
				"section_key": "admin",
				"module_key":  "quota-ledger",
				"allowed":     true,
			},
		},
		"data_scope_items": []map[string]any{
			{
				"resource_key": "user_management",
				"scope_type":   model.ScopeTypeAssigned,
				"scope_value":  []int{101, 202},
			},
		},
	})

	CreatePermissionTemplate(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	menuItems, ok := response.Data["menu_items"].([]any)
	require.True(t, ok)
	require.Len(t, menuItems, 1)

	dataScopeItems, ok := response.Data["data_scope_items"].([]any)
	require.True(t, ok)
	require.Len(t, dataScopeItems, 1)

	var profile model.PermissionProfile
	require.NoError(t, db.Where("profile_name = ?", "Agent Console Template").First(&profile).Error)

	var items []model.PermissionProfileItem
	require.NoError(t, db.Where("profile_id = ?", profile.Id).Order("id asc").Find(&items).Error)
	require.Len(t, items, 3)
}

func TestUpdatePermissionTemplate(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	profile := model.PermissionProfile{
		ProfileName: "Template Update",
		ProfileType: model.UserTypeAdmin,
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

	ctx, recorder := newAdminPermissionContext(t, http.MethodPut, "/api/admin/permission-templates/"+strconv.Itoa(profile.Id), map[string]any{
		"profile_name": "Template Update 2",
		"profile_type": model.UserTypeAdmin,
		"description":  "updated",
		"status":       model.CommonStatusDisabled,
		"items": []map[string]any{
			{
				"resource_key": "permission_management",
				"action_key":   "bind_profile",
				"allowed":      true,
			},
		},
	})
	ctx.Params = append(ctx.Params, gin.Param{Key: "id", Value: strconv.Itoa(profile.Id)})

	UpdatePermissionTemplate(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var reloaded model.PermissionProfile
	require.NoError(t, db.First(&reloaded, profile.Id).Error)
	require.Equal(t, "Template Update 2", reloaded.ProfileName)
	require.Equal(t, model.CommonStatusDisabled, reloaded.Status)

	var items []model.PermissionProfileItem
	require.NoError(t, db.Where("profile_id = ?", profile.Id).Find(&items).Error)
	require.Len(t, items, 1)
	require.Equal(t, "permission_management", items[0].ResourceKey)
	require.Equal(t, "bind_profile", items[0].ActionKey)
}

func TestGetPermissionTemplatesReturnsItems(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	require.NoError(t, db.Create(&model.PermissionProfile{
		ProfileName: "End User Console",
		ProfileType: model.UserTypeEndUser,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminPermissionContext(t, http.MethodGet, "/api/admin/permission-templates?profile_type=end_user&p=1&page_size=10", nil)
	ctx.Request.URL.RawQuery = "profile_type=end_user&p=1&page_size=10"
	GetPermissionTemplates(ctx)

	var response adminPermissionPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
}

func TestDeletePermissionTemplateRejectsActiveBinding(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "permission_template_delete_blocked",
		Password:    "hashed-password",
		DisplayName: "Permission Template Delete Blocked",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Delete Blocked Template",
		ProfileType: model.UserTypeAdmin,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: "permission_management",
		ActionKey:   "bind_profile",
		Allowed:     true,
		ScopeType:   model.ScopeTypeAll,
		CreatedAtTs: common.GetTimestamp(),
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:        user.Id,
		ProfileId:     profile.Id,
		EffectiveFrom: common.GetTimestamp(),
		Status:        model.CommonStatusEnabled,
		CreatedAtTs:   common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminPermissionContext(
		t,
		http.MethodDelete,
		"/api/admin/permission-templates/"+strconv.Itoa(profile.Id),
		nil,
	)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(profile.Id)}}

	DeletePermissionTemplate(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "该模板正在被 1 个账号使用，无法删除", response.Message)

	var profileCount int64
	require.NoError(t, db.Model(&model.PermissionProfile{}).Where("id = ?", profile.Id).Count(&profileCount).Error)
	require.Equal(t, int64(1), profileCount)

	var itemCount int64
	require.NoError(t, db.Model(&model.PermissionProfileItem{}).Where("profile_id = ?", profile.Id).Count(&itemCount).Error)
	require.Equal(t, int64(1), itemCount)
}

func TestDeletePermissionTemplateAllowsHistoricalBindingOnly(t *testing.T) {
	db := setupAdminPermissionTestDB(t)

	user := model.User{
		Username:    "permission_template_delete_allowed",
		Password:    "hashed-password",
		DisplayName: "Permission Template Delete Allowed",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
	}
	require.NoError(t, db.Create(&user).Error)

	profile := model.PermissionProfile{
		ProfileName: "Delete Allowed Template",
		ProfileType: model.UserTypeAdmin,
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
	require.NoError(t, db.Model(&model.UserPermissionBinding{}).Create(map[string]any{
		"user_id":        user.Id,
		"profile_id":     profile.Id,
		"effective_from": common.GetTimestamp(),
		"status":         model.CommonStatusDisabled,
		"created_at":     common.GetTimestamp(),
	}).Error)

	ctx, recorder := newAdminPermissionContext(
		t,
		http.MethodDelete,
		"/api/admin/permission-templates/"+strconv.Itoa(profile.Id),
		nil,
	)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(profile.Id)}}

	DeletePermissionTemplate(ctx)

	var response adminPermissionAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var profileCount int64
	require.NoError(t, db.Model(&model.PermissionProfile{}).Where("id = ?", profile.Id).Count(&profileCount).Error)
	require.Zero(t, profileCount)

	var itemCount int64
	require.NoError(t, db.Model(&model.PermissionProfileItem{}).Where("profile_id = ?", profile.Id).Count(&itemCount).Error)
	require.Zero(t, itemCount)
}
