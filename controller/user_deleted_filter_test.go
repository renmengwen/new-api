package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type legacyUserPageResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    common.PageInfo `json:"data"`
}

func newLegacyUserListContext(method string, target string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	return ctx, recorder
}

func TestGetAllUsersExcludesSoftDeletedUsers(t *testing.T) {
	db := setupLegacyUpdateUserQuotaTestDB(t)
	activeUser := seedLegacyUpdateUserQuotaTarget(t, db, "legacy_list_active_user", model.UserTypeEndUser, common.RoleCommonUser, 600)
	deletedUser := seedLegacyUpdateUserQuotaTarget(t, db, "legacy_list_deleted_user", model.UserTypeEndUser, common.RoleCommonUser, 300)
	require.NoError(t, db.Delete(&model.User{}, deletedUser.Id).Error)

	ctx, recorder := newLegacyUserListContext(http.MethodGet, "/api/user/?p=1&page_size=10")
	GetAllUsers(ctx)

	var response legacyUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(activeUser.Id), item["id"])
}

func TestSearchUsersExcludesSoftDeletedUsers(t *testing.T) {
	db := setupLegacyUpdateUserQuotaTestDB(t)
	activeUser := seedLegacyUpdateUserQuotaTarget(t, db, "legacy_search_filter_active", model.UserTypeEndUser, common.RoleCommonUser, 600)
	deletedUser := seedLegacyUpdateUserQuotaTarget(t, db, "legacy_search_filter_deleted", model.UserTypeEndUser, common.RoleCommonUser, 300)
	require.NoError(t, db.Delete(&model.User{}, deletedUser.Id).Error)

	ctx, recorder := newLegacyUserListContext(
		http.MethodGet,
		"/api/user/search?keyword=legacy_search_filter&group=&p=1&page_size=10",
	)
	SearchUsers(ctx)

	var response legacyUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)

	items, ok := response.Data.Items.([]any)
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(activeUser.Id), item["id"])
}
