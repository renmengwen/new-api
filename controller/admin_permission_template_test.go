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
