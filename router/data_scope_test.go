package router

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type dataDashboardRouteResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    []model.QuotaData `json:"data"`
}

func TestDataDashboardRouteScopesAgentToSelfAndManagedUsers(t *testing.T) {
	db := setupDataDashboardRouterTestDB(t)
	agent, owned, other := seedDataDashboardScopeFixture(t, db, common.RoleCommonUser)
	router := newDataDashboardRouter(t, agent)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/data/?start_timestamp=1812000000&end_timestamp=1812003600",
		nil,
	)
	request.Header.Set("New-Api-User", strconv.Itoa(agent.Id))
	for _, sessionCookie := range loginDataDashboardUser(t, router).Cookies() {
		request.AddCookie(sessionCookie)
	}
	router.ServeHTTP(recorder, request)

	var response dataDashboardRouteResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	requireQuotaDataModels(t, response.Data, []string{"agent-model", owned.Username + "-model"})
	require.NotContains(t, quotaDataModelNames(response.Data), other.Username)
	require.NotContains(t, quotaDataModelNames(response.Data), "other-model")
	require.Contains(t, quotaDataModelNames(response.Data), owned.Username+"-model")
}

func TestDataDashboardRouteKeepsAdminRoleAgentScoped(t *testing.T) {
	db := setupDataDashboardRouterTestDB(t)
	agent, owned, _ := seedDataDashboardScopeFixture(t, db, common.RoleAdminUser)
	router := newDataDashboardRouter(t, agent)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/data/?start_timestamp=1812000000&end_timestamp=1812003600",
		nil,
	)
	request.Header.Set("New-Api-User", strconv.Itoa(agent.Id))
	for _, sessionCookie := range loginDataDashboardUser(t, router).Cookies() {
		request.AddCookie(sessionCookie)
	}
	router.ServeHTTP(recorder, request)

	var response dataDashboardRouteResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)
	requireQuotaDataModels(t, response.Data, []string{"agent-model", owned.Username + "-model"})
	require.NotContains(t, quotaDataModelNames(response.Data), "other-model")
}

func TestDataDashboardRouteRequiresReadSummaryPermissionForExplicitAgentProfile(t *testing.T) {
	db := setupDataDashboardRouterTestDB(t)
	agent, _, _ := seedDataDashboardScopeFixture(t, db, common.RoleCommonUser)
	bindDataDashboardPermissionProfile(t, db, agent.Id)
	router := newDataDashboardRouter(t, agent)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/data/?start_timestamp=1812000000&end_timestamp=1812003600",
		nil,
	)
	request.Header.Set("New-Api-User", strconv.Itoa(agent.Id))
	for _, sessionCookie := range loginDataDashboardUser(t, router).Cookies() {
		request.AddCookie(sessionCookie)
	}
	router.ServeHTTP(recorder, request)

	var response dataDashboardRouteResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "permission denied", response.Message)
}

func setupDataDashboardRouterTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dbName := "file:data_dashboard_router_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.QuotaData{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.UserPermissionOverride{},
		&model.UserMenuOverride{},
		&model.UserDataScopeOverride{},
		&model.AgentUserRelation{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func seedDataDashboardScopeFixture(t *testing.T, db *gorm.DB, agentRole int) (model.User, model.User, model.User) {
	t.Helper()

	agent := model.User{
		Id:          9101 + agentRole,
		Username:    "data_dashboard_agent_" + strconv.Itoa(agentRole),
		Password:    "hashed-password",
		DisplayName: "Data Dashboard Agent",
		Role:        agentRole,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
		AffCode:     "data_dashboard_agent_aff_" + strconv.Itoa(agentRole),
	}
	owned := model.User{
		Id:            9201 + agentRole,
		Username:      "data_dashboard_owned_" + strconv.Itoa(agentRole),
		Password:      "hashed-password",
		DisplayName:   "Data Dashboard Owned",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		UserType:      model.UserTypeEndUser,
		ParentAgentId: agent.Id,
		Group:         "default",
		AffCode:       "data_dashboard_owned_aff_" + strconv.Itoa(agentRole),
	}
	other := model.User{
		Id:          9301 + agentRole,
		Username:    "data_dashboard_other_" + strconv.Itoa(agentRole),
		Password:    "hashed-password",
		DisplayName: "Data Dashboard Other",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		AffCode:     "data_dashboard_other_aff_" + strconv.Itoa(agentRole),
	}
	require.NoError(t, db.Create(&[]model.User{agent, owned, other}).Error)
	require.NoError(t, db.Create(&model.AgentUserRelation{
		AgentUserId: agent.Id,
		EndUserId:   owned.Id,
		BindSource:  "manual",
		BindAt:      1812000000,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: 1812000000,
	}).Error)
	require.NoError(t, db.Create(&[]model.QuotaData{
		{
			UserID:    agent.Id,
			Username:  agent.Username,
			ModelName: "agent-model",
			CreatedAt: 1812000000,
			Count:     1,
			Quota:     10,
		},
		{
			UserID:    owned.Id,
			Username:  owned.Username,
			ModelName: owned.Username + "-model",
			CreatedAt: 1812000000,
			Count:     2,
			Quota:     20,
		},
		{
			UserID:    other.Id,
			Username:  other.Username,
			ModelName: "other-model",
			CreatedAt: 1812000000,
			Count:     3,
			Quota:     30,
		},
	}).Error)

	return agent, owned, other
}

func bindDataDashboardPermissionProfile(t *testing.T, db *gorm.DB, userId int) {
	t.Helper()

	profile := model.PermissionProfile{
		ProfileName: "data_dashboard_restricted_agent",
		ProfileType: model.UserTypeAgent,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: 1812000000,
		UpdatedAtTs: 1812000000,
	}
	require.NoError(t, db.Create(&profile).Error)
	require.NoError(t, db.Create(&model.PermissionProfileItem{
		ProfileId:   profile.Id,
		ResourceKey: "quota_management",
		ActionKey:   "ledger_read",
		Allowed:     true,
		ScopeType:   model.ScopeTypeAll,
		CreatedAtTs: 1812000000,
	}).Error)
	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:        userId,
		ProfileId:     profile.Id,
		Status:        model.CommonStatusEnabled,
		EffectiveFrom: 1812000000,
		CreatedAtTs:   1812000000,
	}).Error)
}

func newDataDashboardRouter(t *testing.T, user model.User) *gin.Engine {
	t.Helper()

	router := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("test-session", store))
	router.GET("/login-data-dashboard-user", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", user.Id)
		session.Set("username", user.Username)
		session.Set("role", user.Role)
		session.Set("user_type", user.GetUserType())
		session.Set("status", user.Status)
		session.Set("group", user.Group)
		require.NoError(t, session.Save())
		c.Status(http.StatusOK)
	})
	SetApiRouter(router)
	return router
}

func loginDataDashboardUser(t *testing.T, router *gin.Engine) *http.Response {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/login-data-dashboard-user", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	return recorder.Result()
}

func requireQuotaDataModels(t *testing.T, items []model.QuotaData, expectedModels []string) {
	t.Helper()

	actual := quotaDataModelNames(items)
	require.ElementsMatch(t, expectedModels, actual)
}

func quotaDataModelNames(items []model.QuotaData) []string {
	models := make([]string, 0, len(items))
	for _, item := range items {
		models = append(models, item.ModelName)
	}
	return models
}
