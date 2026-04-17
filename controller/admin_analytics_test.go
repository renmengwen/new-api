package controller

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type adminAnalyticsAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type adminAnalyticsModelItemResponse struct {
	ModelName        string  `json:"model_name"`
	CallCount        int64   `json:"call_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalCost        int64   `json:"total_cost"`
	AvgUseTime       float64 `json:"avg_use_time"`
	SuccessRate      float64 `json:"success_rate"`
}

type adminAnalyticsUserItemResponse struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	CallCount    int64  `json:"call_count"`
	ModelCount   int64  `json:"model_count"`
	TotalTokens  int64  `json:"total_tokens"`
	TotalCost    int64  `json:"total_cost"`
	LastCalledAt int64  `json:"last_called_at"`
}

type adminAnalyticsDailyItemResponse struct {
	BucketDay    string `json:"bucket_day"`
	CallCount    int64  `json:"call_count"`
	TotalCost    int64  `json:"total_cost"`
	ActiveUsers  int64  `json:"active_users"`
	ActiveModels int64  `json:"active_models"`
}

type adminAnalyticsModelPageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Page     int                               `json:"page"`
		PageSize int                               `json:"page_size"`
		Total    int                               `json:"total"`
		Items    []adminAnalyticsModelItemResponse `json:"items"`
	} `json:"data"`
}

type adminAnalyticsUserPageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Page     int                              `json:"page"`
		PageSize int                              `json:"page_size"`
		Total    int                              `json:"total"`
		Items    []adminAnalyticsUserItemResponse `json:"items"`
	} `json:"data"`
}

type adminAnalyticsDailyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Items []adminAnalyticsDailyItemResponse `json:"items"`
	} `json:"data"`
}

type adminAnalyticsSQLCapture struct {
	mu      sync.Mutex
	queries []string
}

type adminAnalyticsSQLCaptureLogger struct {
	base    gormlogger.Interface
	capture *adminAnalyticsSQLCapture
}

func setupAdminAnalyticsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dbName := "file:admin_analytics_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Token{},
		&model.Log{},
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

func newAdminAnalyticsSQLCaptureLogger() *adminAnalyticsSQLCaptureLogger {
	return &adminAnalyticsSQLCaptureLogger{
		base: gormlogger.Discard,
		capture: &adminAnalyticsSQLCapture{
			queries: make([]string, 0, 4),
		},
	}
}

func (l *adminAnalyticsSQLCaptureLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &adminAnalyticsSQLCaptureLogger{
		base:    l.base.LogMode(level),
		capture: l.capture,
	}
}

func (l *adminAnalyticsSQLCaptureLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.base.Info(ctx, msg, data...)
}

func (l *adminAnalyticsSQLCaptureLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.base.Warn(ctx, msg, data...)
}

func (l *adminAnalyticsSQLCaptureLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.base.Error(ctx, msg, data...)
}

func (l *adminAnalyticsSQLCaptureLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	l.capture.mu.Lock()
	l.capture.queries = append(l.capture.queries, sql)
	l.capture.mu.Unlock()
	l.base.Trace(ctx, begin, fc, err)
}

func (l *adminAnalyticsSQLCaptureLogger) Queries() []string {
	l.capture.mu.Lock()
	defer l.capture.mu.Unlock()

	queries := make([]string, len(l.capture.queries))
	copy(queries, l.capture.queries)
	return queries
}

func attachAdminAnalyticsSQLCaptureLogger(t *testing.T, db *gorm.DB) *adminAnalyticsSQLCaptureLogger {
	t.Helper()

	queryLogger := newAdminAnalyticsSQLCaptureLogger()
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	tracedDB := db.Session(&gorm.Session{Logger: queryLogger})
	model.DB = tracedDB
	model.LOG_DB = tracedDB
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
	})
	return queryLogger
}

func newAdminAnalyticsContext(t *testing.T, method string, target string, body any, operatorID int, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		payload, err := common.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequest(method, target, io.NopCloser(bytes.NewReader(payload)))
		req.Header.Set("Content-Type", "application/json")
	}

	ctx.Request = req
	ctx.Set("id", operatorID)
	ctx.Set("role", role)
	return ctx, recorder
}

func seedAdminAnalyticsOperator(t *testing.T, db *gorm.DB, username string) model.User {
	t.Helper()

	operator := model.User{
		Username:    username,
		Password:    "hashed-password",
		DisplayName: username,
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     username + "_aff",
	}
	require.NoError(t, db.Create(&operator).Error)
	return operator
}

type adminAnalyticsFixture struct {
	Now        int64
	Last7Start int64
	TodayStart int64
	Root       model.User
	Admin      model.User
	Agent      model.User
	OwnedA     model.User
	OwnedB     model.User
	Other      model.User
	OwnedToken model.Token
}

func seedAdminAnalyticsFixture(t *testing.T, db *gorm.DB) adminAnalyticsFixture {
	t.Helper()

	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.Local)
	last7Start := time.Date(2026, 4, 11, 0, 0, 0, 0, time.Local).Unix()
	todayStart := time.Date(2026, 4, 17, 0, 0, 0, 0, time.Local).Unix()

	users := []model.User{
		{
			Username:    "analytics_root",
			Password:    "hashed-password",
			DisplayName: "analytics_root",
			Role:        common.RoleRootUser,
			Status:      common.UserStatusEnabled,
			UserType:    model.UserTypeRoot,
			Group:       "default",
			AffCode:     "analytics_root_aff",
		},
		{
			Username:    "analytics_admin",
			Password:    "hashed-password",
			DisplayName: "analytics_admin",
			Role:        common.RoleAdminUser,
			Status:      common.UserStatusEnabled,
			UserType:    model.UserTypeAdmin,
			Group:       "default",
			AffCode:     "analytics_admin_aff",
		},
		{
			Username:    "analytics_agent",
			Password:    "hashed-password",
			DisplayName: "analytics_agent",
			Role:        common.RoleAdminUser,
			Status:      common.UserStatusEnabled,
			UserType:    model.UserTypeAgent,
			Group:       "default",
			AffCode:     "analytics_agent_aff",
		},
		{
			Username:    "analytics_owned_a",
			Password:    "hashed-password",
			DisplayName: "analytics_owned_a",
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
			UserType:    model.UserTypeEndUser,
			Group:       "default",
			AffCode:     "analytics_owned_a_aff",
		},
		{
			Username:    "analytics_owned_b",
			Password:    "hashed-password",
			DisplayName: "analytics_owned_b",
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
			UserType:    model.UserTypeEndUser,
			Group:       "default",
			AffCode:     "analytics_owned_b_aff",
		},
		{
			Username:    "analytics_other",
			Password:    "hashed-password",
			DisplayName: "analytics_other",
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
			UserType:    model.UserTypeEndUser,
			Group:       "default",
			AffCode:     "analytics_other_aff",
		},
	}
	require.NoError(t, db.Create(&users).Error)

	root := users[0]
	admin := users[1]
	agent := users[2]
	ownedA := users[3]
	ownedB := users[4]
	other := users[5]

	require.NoError(t, db.Model(&model.User{}).Where("id IN ?", []int{ownedA.Id, ownedB.Id}).
		Update("parent_agent_id", agent.Id).Error)
	require.NoError(t, db.Create([]model.AgentUserRelation{
		{
			AgentUserId: agent.Id,
			EndUserId:   ownedA.Id,
			BindSource:  "manual",
			BindAt:      now.Unix(),
			Status:      model.CommonStatusEnabled,
			CreatedAtTs: now.Unix(),
		},
		{
			AgentUserId: agent.Id,
			EndUserId:   ownedB.Id,
			BindSource:  "manual",
			BindAt:      now.Unix(),
			Status:      model.CommonStatusEnabled,
			CreatedAtTs: now.Unix(),
		},
	}).Error)

	tokens := []model.Token{
		{UserId: agent.Id, Key: "sk-agent-analytics-token-0000000000000000000001", Status: common.TokenStatusEnabled, Name: "agent-token", CreatedTime: now.Unix(), Group: "default"},
		{UserId: ownedA.Id, Key: "sk-owned-a-analytics-token-00000000000000000001", Status: common.TokenStatusEnabled, Name: "owned-a-token", CreatedTime: now.Unix(), Group: "default"},
		{UserId: ownedB.Id, Key: "sk-owned-b-analytics-token-00000000000000000001", Status: common.TokenStatusEnabled, Name: "owned-b-token", CreatedTime: now.Unix(), Group: "default"},
		{UserId: other.Id, Key: "sk-other-analytics-token-000000000000000000001", Status: common.TokenStatusEnabled, Name: "other-token", CreatedTime: now.Unix(), Group: "default"},
	}
	require.NoError(t, db.Create(&tokens).Error)

	logs := []model.Log{
		{
			UserId:           ownedA.Id,
			CreatedAt:        time.Date(2026, 4, 13, 9, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         ownedA.Username,
			TokenName:        tokens[1].Name,
			ModelName:        "claude-4",
			Quota:            100,
			PromptTokens:     10,
			CompletionTokens: 20,
			UseTime:          12,
			TokenId:          tokens[1].Id,
		},
		{
			UserId:           agent.Id,
			CreatedAt:        time.Date(2026, 4, 17, 9, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         agent.Username,
			TokenName:        tokens[0].Name,
			ModelName:        "agent-model",
			Quota:            50,
			PromptTokens:     5,
			CompletionTokens: 15,
			UseTime:          18,
			TokenId:          tokens[0].Id,
		},
		{
			UserId:           ownedB.Id,
			CreatedAt:        time.Date(2026, 4, 17, 10, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeError,
			Username:         ownedB.Username,
			TokenName:        tokens[2].Name,
			ModelName:        "gpt-4",
			Quota:            0,
			PromptTokens:     2,
			CompletionTokens: 0,
			UseTime:          30,
			TokenId:          tokens[2].Id,
		},
		{
			UserId:           other.Id,
			CreatedAt:        time.Date(2026, 4, 17, 11, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         other.Username,
			TokenName:        tokens[3].Name,
			ModelName:        "other-model",
			Quota:            25,
			PromptTokens:     8,
			CompletionTokens: 16,
			UseTime:          21,
			TokenId:          tokens[3].Id,
		},
		{
			UserId:           ownedA.Id,
			CreatedAt:        time.Date(2026, 4, 7, 12, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         ownedA.Username,
			TokenName:        tokens[1].Name,
			ModelName:        "claude-4",
			Quota:            80,
			PromptTokens:     8,
			CompletionTokens: 12,
			UseTime:          9,
			TokenId:          tokens[1].Id,
		},
		{
			UserId:           other.Id,
			CreatedAt:        time.Date(2026, 4, 10, 12, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         other.Username,
			TokenName:        tokens[3].Name,
			ModelName:        "prev-only-model",
			Quota:            20,
			PromptTokens:     4,
			CompletionTokens: 10,
			UseTime:          11,
			TokenId:          tokens[3].Id,
		},
	}
	require.NoError(t, db.Create(&logs).Error)

	return adminAnalyticsFixture{
		Now:        now.Unix(),
		Last7Start: last7Start,
		TodayStart: todayStart,
		Root:       root,
		Admin:      admin,
		Agent:      agent,
		OwnedA:     ownedA,
		OwnedB:     ownedB,
		Other:      other,
		OwnedToken: tokens[1],
	}
}

func TestGetAdminAnalyticsSummaryRequiresReadPermission(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	operator := seedAdminAnalyticsOperator(t, db, "analytics_summary_no_read")
	grantPermissionActions(t, db, operator.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourcePermissionManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceAgentManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceAdminManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceUserManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceQuotaManagement, Action: service.ActionReadSummary},
		permissionGrant{Resource: service.ResourceAuditManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionExport},
	)

	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, "/api/admin/analytics/summary", nil, operator.Id, operator.Role)
	GetAdminAnalyticsSummary(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Contains(t, response.Message, "permission denied")
}

func TestGetAdminAnalyticsSummaryRejectsCommonUserEvenWithAnalyticsPermission(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.OwnedA.Id, model.UserTypeEndUser,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/summary?date_preset=today&start_timestamp=" +
		strconv.FormatInt(fixture.TodayStart, 10) + "&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.OwnedA.Id, fixture.OwnedA.Role)
	GetAdminAnalyticsSummary(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Contains(t, response.Message, "permission denied")
}

func TestExportAdminAnalyticsRequiresExportPermission(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	operator := seedAdminAnalyticsOperator(t, db, "analytics_export_no_export")
	grantPermissionActions(t, db, operator.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourcePermissionManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceAgentManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceAdminManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceUserManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceQuotaManagement, Action: service.ActionLedgerRead},
		permissionGrant{Resource: service.ResourceAuditManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	ctx, recorder := newAdminAnalyticsContext(t, http.MethodPost, "/api/admin/analytics/export", map[string]any{}, operator.Id, operator.Role)
	ExportAdminAnalytics(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Contains(t, response.Message, "permission denied")
}

func TestExportAdminAnalyticsRequiresReadPermission(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionExport},
	)

	ctx, recorder := newAdminAnalyticsContext(t, http.MethodPost, "/api/admin/analytics/export", map[string]any{
		"view":            dto.AdminAnalyticsViewModels,
		"date_preset":     dto.AdminAnalyticsDatePresetToday,
		"start_timestamp": fixture.TodayStart,
		"end_timestamp":   fixture.Now,
		"limit":           10,
	}, fixture.Admin.Id, fixture.Admin.Role)
	ExportAdminAnalytics(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Contains(t, response.Message, "permission denied")
}

func TestGetAdminAnalyticsSummaryReturnsFourStatsForAdmin(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/summary?date_preset=today&start_timestamp=" +
		strconv.FormatInt(fixture.TodayStart, 10) + "&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsSummary(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, float64(3), response.Data["total_calls"])
	require.Equal(t, float64(75), response.Data["total_cost"])
	require.Equal(t, float64(3), response.Data["active_users"])
	require.Equal(t, float64(3), response.Data["active_models"])
}

func TestGetAdminAnalyticsSummaryReturnsWowForLast7Days(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/summary?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) + "&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsSummary(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	wow, ok := response.Data["wow"].(map[string]any)
	require.True(t, ok)
	totalCallsWow, ok := wow["total_calls"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(4), response.Data["total_calls"])
	require.Equal(t, float64(175), response.Data["total_cost"])
	require.Equal(t, float64(4), response.Data["active_users"])
	require.Equal(t, float64(4), response.Data["active_models"])
	require.Equal(t, float64(4), totalCallsWow["current"])
	require.Equal(t, float64(2), totalCallsWow["previous"])
}

func TestGetAdminAnalyticsSummaryReturnsTotalTokensAndWowForLast7Days(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/summary?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) + "&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsSummary(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, float64(76), response.Data["total_tokens"])

	wow, ok := response.Data["wow"].(map[string]any)
	require.True(t, ok)
	totalTokensWow, ok := wow["total_tokens"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(76), totalTokensWow["current"])
	require.Equal(t, float64(34), totalTokensWow["previous"])
}

func TestGetAdminAnalyticsSummaryOmitsWowOutsideLast7Days(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/summary?date_preset=today&start_timestamp=" +
		strconv.FormatInt(fixture.TodayStart, 10) + "&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsSummary(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	_, exists := response.Data["wow"]
	require.False(t, exists)
}

func TestGetAdminAnalyticsSummaryScopesAgentToOwnedUsersAndTokens(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/summary?date_preset=today&start_timestamp=" +
		strconv.FormatInt(fixture.TodayStart, 10) + "&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Agent.Id, fixture.Agent.Role)
	GetAdminAnalyticsSummary(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, float64(2), response.Data["total_calls"])
	require.Equal(t, float64(50), response.Data["total_cost"])
	require.Equal(t, float64(2), response.Data["active_users"])
	require.Equal(t, float64(2), response.Data["active_models"])
}

func TestGetAdminAnalyticsSummaryIncludesTokenOwnedByAgentScope(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)
	require.NoError(t, db.Create(&model.Log{
		UserId:       fixture.Other.Id,
		CreatedAt:    time.Date(2026, 4, 17, 11, 30, 0, 0, time.Local).Unix(),
		Type:         model.LogTypeConsume,
		Username:     fixture.Other.Username,
		TokenName:    fixture.OwnedToken.Name,
		ModelName:    "token-owned-model",
		Quota:        33,
		PromptTokens: 3,
		TokenId:      fixture.OwnedToken.Id,
	}).Error)

	target := "/api/admin/analytics/summary?date_preset=today&start_timestamp=" +
		strconv.FormatInt(fixture.TodayStart, 10) + "&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Agent.Id, fixture.Agent.Role)
	GetAdminAnalyticsSummary(ctx)

	var response adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, float64(3), response.Data["total_calls"])
	require.Equal(t, float64(83), response.Data["total_cost"])
	require.Equal(t, float64(3), response.Data["active_users"])
	require.Equal(t, float64(3), response.Data["active_models"])
}

func TestGetAdminAnalyticsModelsReturnsPagedAggregates(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)
	require.NoError(t, db.Create([]model.Log{
		{
			UserId:           fixture.OwnedA.Id,
			CreatedAt:        time.Date(2026, 4, 17, 8, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         fixture.OwnedA.Username,
			TokenName:        "owned-a-token",
			ModelName:        "claude-4",
			Quota:            60,
			PromptTokens:     6,
			CompletionTokens: 4,
			UseTime:          30,
			TokenId:          fixture.OwnedToken.Id,
		},
		{
			UserId:           fixture.OwnedB.Id,
			CreatedAt:        time.Date(2026, 4, 17, 8, 30, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeError,
			Username:         fixture.OwnedB.Username,
			TokenName:        "owned-b-token",
			ModelName:        "claude-4",
			Quota:            0,
			PromptTokens:     4,
			CompletionTokens: 0,
			UseTime:          20,
		},
	}).Error)

	target := "/api/admin/analytics/models?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10) +
		"&sort_by=call_count&sort_order=desc&p=1&page_size=2"
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsModels(ctx)

	var response adminAnalyticsModelPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Page)
	require.Equal(t, 2, response.Data.PageSize)
	require.Equal(t, 4, response.Data.Total)
	require.Len(t, response.Data.Items, 2)
	require.Equal(t, "claude-4", response.Data.Items[0].ModelName)
	require.EqualValues(t, 3, response.Data.Items[0].CallCount)
	require.EqualValues(t, 20, response.Data.Items[0].PromptTokens)
	require.EqualValues(t, 24, response.Data.Items[0].CompletionTokens)
	require.EqualValues(t, 160, response.Data.Items[0].TotalCost)
	require.InDelta(t, 62.0/3.0, response.Data.Items[0].AvgUseTime, 0.0001)
	require.InDelta(t, 2.0/3.0, response.Data.Items[0].SuccessRate, 0.0001)
}

func TestGetAdminAnalyticsUsersReturnsLastCallTime(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/users?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10) +
		"&sort_by=last_called_at&sort_order=desc&p=1&page_size=10"
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsUsers(ctx)

	var response adminAnalyticsUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 4, response.Data.Total)
	require.Len(t, response.Data.Items, 4)
	require.Equal(t, fixture.Other.Id, response.Data.Items[0].UserID)
	require.Equal(t, fixture.Other.Username, response.Data.Items[0].Username)
	require.EqualValues(t, 1, response.Data.Items[0].CallCount)
	require.EqualValues(t, time.Date(2026, 4, 17, 11, 0, 0, 0, time.Local).Unix(), response.Data.Items[0].LastCalledAt)
	require.EqualValues(t, 1, response.Data.Items[0].ModelCount)
	require.EqualValues(t, 25, response.Data.Items[0].TotalCost)
	require.Equal(t, fixture.OwnedA.Username, response.Data.Items[3].Username)
	require.EqualValues(t, 1, response.Data.Items[3].CallCount)
	require.EqualValues(t, time.Date(2026, 4, 13, 9, 0, 0, 0, time.Local).Unix(), response.Data.Items[3].LastCalledAt)
}

func TestGetAdminAnalyticsUsersReturnsTotalTokens(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/users?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10) +
		"&sort_by=last_called_at&sort_order=desc&p=1&page_size=10"
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsUsers(ctx)

	var response adminAnalyticsUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.NotEmpty(t, response.Data.Items)
	require.GreaterOrEqual(t, response.Data.Items[0].TotalTokens, int64(1))
	require.EqualValues(t, 24, response.Data.Items[0].TotalTokens)
}

func TestGetAdminAnalyticsModelsNegativePageSizeStillPaginates(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	extraLogs := make([]model.Log, 0, common.ItemsPerPage+2)
	for index := 0; index < common.ItemsPerPage+2; index++ {
		extraLogs = append(extraLogs, model.Log{
			UserId:           fixture.OwnedA.Id,
			CreatedAt:        time.Date(2026, 4, 17, 6, index, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         fixture.OwnedA.Username,
			TokenName:        fixture.OwnedToken.Name,
			ModelName:        "paged-model-" + strconv.Itoa(index),
			Quota:            10 + index,
			PromptTokens:     1,
			CompletionTokens: 2,
			UseTime:          3,
			TokenId:          fixture.OwnedToken.Id,
		})
	}
	require.NoError(t, db.Create(&extraLogs).Error)

	target := "/api/admin/analytics/models?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10) +
		"&sort_by=model_name&sort_order=asc&p=1&page_size=-1"
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsModels(ctx)

	var response adminAnalyticsModelPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, common.ItemsPerPage, response.Data.PageSize)
	require.Greater(t, response.Data.Total, common.ItemsPerPage)
	require.Len(t, response.Data.Items, common.ItemsPerPage)
}

func TestGetAdminAnalyticsUsersUsesCurrentUsernameAfterRename(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	renamedUsername := "analytics_owned_a_renamed"
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", fixture.OwnedA.Id).Update("username", renamedUsername).Error)

	target := "/api/admin/analytics/users?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10) +
		"&sort_by=user_id&sort_order=asc&p=1&page_size=10"
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsUsers(ctx)

	var response adminAnalyticsUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	found := false
	for _, item := range response.Data.Items {
		if item.UserID == fixture.OwnedA.Id {
			found = true
			require.Equal(t, renamedUsername, item.Username)
			break
		}
	}
	require.True(t, found)
}

func TestGetAdminAnalyticsUsersSortsByCurrentUsername(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	require.NoError(t, db.Model(&model.User{}).Where("id = ?", fixture.OwnedA.Id).Update("username", "000_current_first").Error)

	target := "/api/admin/analytics/users?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10) +
		"&sort_by=username&sort_order=asc&p=1&page_size=10"
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsUsers(ctx)

	var response adminAnalyticsUserPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.NotEmpty(t, response.Data.Items)
	require.Equal(t, fixture.OwnedA.Id, response.Data.Items[0].UserID)
	require.Equal(t, "000_current_first", response.Data.Items[0].Username)
}

func TestGetAdminAnalyticsModelsEscapesLiteralWildcardsInFilters(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	users := []model.User{
		{
			Username:    "Case_User",
			Password:    "hashed-password",
			DisplayName: "Case_User",
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
			UserType:    model.UserTypeEndUser,
			Group:       "default",
			AffCode:     "Case_User_aff",
		},
		{
			Username:    "CaseXUser",
			Password:    "hashed-password",
			DisplayName: "CaseXUser",
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
			UserType:    model.UserTypeEndUser,
			Group:       "default",
			AffCode:     "CaseXUser_aff",
		},
	}
	require.NoError(t, db.Create(&users).Error)

	logs := []model.Log{
		{
			UserId:           users[0].Id,
			CreatedAt:        time.Date(2026, 4, 17, 8, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         users[0].Username,
			TokenName:        "literal-token-a",
			ModelName:        "Literal 100% Model",
			Quota:            31,
			PromptTokens:     3,
			CompletionTokens: 5,
			UseTime:          7,
		},
		{
			UserId:           users[1].Id,
			CreatedAt:        time.Date(2026, 4, 17, 8, 30, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         users[1].Username,
			TokenName:        "literal-token-b",
			ModelName:        "Literal 100x Model",
			Quota:            41,
			PromptTokens:     4,
			CompletionTokens: 6,
			UseTime:          8,
		},
	}
	require.NoError(t, db.Create(&logs).Error)

	target := "/api/admin/analytics/models?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10) +
		"&sort_by=model_name&sort_order=asc&p=1&page_size=10" +
		"&model_keyword=" + url.QueryEscape("100% model") +
		"&username_keyword=" + url.QueryEscape("case_user")
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsModels(ctx)

	var response adminAnalyticsModelPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, "Literal 100% Model", response.Data.Items[0].ModelName)
	require.EqualValues(t, 1, response.Data.Items[0].CallCount)
	require.EqualValues(t, 31, response.Data.Items[0].TotalCost)
}

func TestGetAdminAnalyticsDailyReturnsChronologicalSeries(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	target := "/api/admin/analytics/daily?date_preset=last7days&start_timestamp=" +
		strconv.FormatInt(fixture.Last7Start, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	ctx, recorder := newAdminAnalyticsContext(t, http.MethodGet, target, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsDaily(ctx)

	var response adminAnalyticsDailyResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Items, 7)
	require.Equal(t, "2026-04-11", response.Data.Items[0].BucketDay)
	require.Equal(t, "2026-04-17", response.Data.Items[len(response.Data.Items)-1].BucketDay)
	require.EqualValues(t, 0, response.Data.Items[0].CallCount)
	require.Equal(t, "2026-04-13", response.Data.Items[2].BucketDay)
	require.EqualValues(t, 1, response.Data.Items[2].CallCount)
	require.EqualValues(t, 100, response.Data.Items[2].TotalCost)
	require.EqualValues(t, 1, response.Data.Items[2].ActiveUsers)
	require.EqualValues(t, 1, response.Data.Items[2].ActiveModels)
	require.EqualValues(t, 3, response.Data.Items[6].CallCount)
	require.EqualValues(t, 75, response.Data.Items[6].TotalCost)
	require.EqualValues(t, 3, response.Data.Items[6].ActiveUsers)
	require.EqualValues(t, 3, response.Data.Items[6].ActiveModels)
}

func TestGetOperationsAnalyticsDailyUsesGroupedDatabaseQuery(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	queryLogger := attachAdminAnalyticsSQLCaptureLogger(t, db)

	items, err := service.GetOperationsAnalyticsDaily(dto.AdminAnalyticsQuery{
		DatePreset:     dto.AdminAnalyticsDatePresetLast7Days,
		StartTimestamp: fixture.Last7Start,
		EndTimestamp:   fixture.Now,
	}, fixture.Admin.Id, fixture.Admin.Role)
	require.NoError(t, err)
	require.Len(t, items, 7)

	logQuery := ""
	for _, query := range queryLogger.Queries() {
		lowerQuery := strings.ToLower(query)
		if strings.Contains(lowerQuery, " from ") && strings.Contains(lowerQuery, "logs") {
			logQuery = lowerQuery
		}
	}
	require.NotEmpty(t, logQuery)
	require.Contains(t, logQuery, "group by")
	require.Contains(t, logQuery, "count(distinct")
	require.NotContains(t, logQuery, "select logs.created_at, logs.user_id, logs.model_name, logs.quota")
}

func TestOperationsAnalyticsDailyBucketExprAvoidsSessionTimezoneFunctions(t *testing.T) {
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	t.Cleanup(func() {
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
	})

	testCases := []struct {
		name            string
		usingSQLite     bool
		usingMySQL      bool
		usingPostgreSQL bool
	}{
		{name: "sqlite", usingSQLite: true},
		{name: "mysql", usingMySQL: true},
		{name: "postgres", usingPostgreSQL: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			common.UsingSQLite = testCase.usingSQLite
			common.UsingMySQL = testCase.usingMySQL
			common.UsingPostgreSQL = testCase.usingPostgreSQL

			expr := service.OperationsAnalyticsDailyBucketIDExpr(8 * 60 * 60)
			lowerExpr := strings.ToLower(expr)
			require.NotContains(t, lowerExpr, "from_unixtime")
			require.NotContains(t, lowerExpr, "to_timestamp")
			require.NotContains(t, lowerExpr, "localtime")
			require.Contains(t, lowerExpr, "created_at")
		})
	}
}

func TestBuildOperationsAnalyticsNaturalWeekRangesUsesFixedUTCPlus8(t *testing.T) {
	originalLocal := time.Local
	losAngeles, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	time.Local = losAngeles
	t.Cleanup(func() {
		time.Local = originalLocal
	})

	analyticsLocation := time.FixedZone("UTC+8", 8*60*60)
	endTimestamp := time.Date(2026, 3, 9, 10, 30, 0, 0, analyticsLocation).Unix()

	currentWeekStart, currentWeekEnd, previousWeekStart, previousWeekEnd := service.BuildOperationsAnalyticsNaturalWeekRanges(endTimestamp)

	require.Equal(t, time.Date(2026, 3, 9, 0, 0, 0, 0, analyticsLocation).Unix(), currentWeekStart)
	require.Equal(t, endTimestamp, currentWeekEnd)
	require.Equal(t, time.Date(2026, 3, 2, 0, 0, 0, 0, analyticsLocation).Unix(), previousWeekStart)
	require.Equal(t, time.Date(2026, 3, 2, 10, 30, 0, 0, analyticsLocation).Unix(), previousWeekEnd)
}

func TestGetAdminAnalyticsModelsIncludesEmptyModelBucketForReconciliation(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
	)

	require.NoError(t, db.Create(&model.Log{
		UserId:           fixture.Admin.Id,
		CreatedAt:        time.Date(2026, 4, 17, 11, 40, 0, 0, time.Local).Unix(),
		Type:             model.LogTypeConsume,
		Username:         fixture.Admin.Username,
		TokenName:        "admin-empty-model-token",
		ModelName:        "",
		Quota:            40,
		PromptTokens:     4,
		CompletionTokens: 8,
		UseTime:          10,
	}).Error)

	summaryTarget := "/api/admin/analytics/summary?date_preset=today&start_timestamp=" +
		strconv.FormatInt(fixture.TodayStart, 10) + "&end_timestamp=" + strconv.FormatInt(fixture.Now, 10)
	summaryCtx, summaryRecorder := newAdminAnalyticsContext(t, http.MethodGet, summaryTarget, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsSummary(summaryCtx)

	var summaryResponse adminAnalyticsAPIResponse
	require.NoError(t, common.Unmarshal(summaryRecorder.Body.Bytes(), &summaryResponse))
	require.True(t, summaryResponse.Success)

	modelsTarget := "/api/admin/analytics/models?date_preset=today&start_timestamp=" +
		strconv.FormatInt(fixture.TodayStart, 10) +
		"&end_timestamp=" + strconv.FormatInt(fixture.Now, 10) +
		"&sort_by=total_cost&sort_order=desc&p=1&page_size=20"
	modelsCtx, modelsRecorder := newAdminAnalyticsContext(t, http.MethodGet, modelsTarget, nil, fixture.Admin.Id, fixture.Admin.Role)
	GetAdminAnalyticsModels(modelsCtx)

	var modelsResponse adminAnalyticsModelPageResponse
	require.NoError(t, common.Unmarshal(modelsRecorder.Body.Bytes(), &modelsResponse))
	require.True(t, modelsResponse.Success)

	var totalCalls int64
	var totalCost int64
	foundEmptyBucket := false
	for _, item := range modelsResponse.Data.Items {
		totalCalls += item.CallCount
		totalCost += item.TotalCost
		if item.ModelName == "(empty)" {
			foundEmptyBucket = true
			require.EqualValues(t, 1, item.CallCount)
			require.EqualValues(t, 40, item.TotalCost)
		}
	}

	require.True(t, foundEmptyBucket)
	require.EqualValues(t, summaryResponse.Data["total_calls"], float64(totalCalls))
	require.EqualValues(t, summaryResponse.Data["total_cost"], float64(totalCost))
}

func TestExportAdminAnalyticsModelsUsesCurrentFilters(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	originalQuotaDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalQuotaDisplayType
	})
	grantPermissionActions(t, db, fixture.Admin.Id, model.UserTypeAdmin,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionExport},
	)
	require.NoError(t, db.Create([]model.Log{
		{
			UserId:           fixture.Other.Id,
			CreatedAt:        time.Date(2026, 4, 17, 11, 30, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         fixture.Other.Username,
			TokenName:        "other-token",
			ModelName:        "claude-4",
			Quota:            77,
			PromptTokens:     7,
			CompletionTokens: 14,
			UseTime:          19,
		},
		{
			UserId:           fixture.OwnedA.Id,
			CreatedAt:        time.Date(2026, 4, 17, 7, 0, 0, 0, time.Local).Unix(),
			Type:             model.LogTypeConsume,
			Username:         fixture.OwnedA.Username,
			TokenName:        fixture.OwnedToken.Name,
			ModelName:        "claude-3",
			Quota:            66,
			PromptTokens:     9,
			CompletionTokens: 11,
			UseTime:          16,
			TokenId:          fixture.OwnedToken.Id,
		},
	}).Error)

	ctx, recorder := newAdminAnalyticsContext(t, http.MethodPost, "/api/admin/analytics/export", map[string]any{
		"view":               "models",
		"date_preset":        "last7days",
		"start_timestamp":    fixture.Last7Start,
		"end_timestamp":      fixture.Now,
		"sort_by":            "total_cost",
		"sort_order":         "asc",
		"model_keyword":      "claude",
		"username_keyword":   fixture.OwnedA.Username,
		"quota_display_type": "TOKENS",
		"limit":              10,
	}, fixture.Admin.Id, fixture.Admin.Role)
	ExportAdminAnalytics(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("models")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, []string{"Model", "Call Count", "Prompt Tokens", "Completion Tokens", "Total Cost", "Avg Use Time", "Success Rate"}, rows[0])
	require.Equal(t, "claude-3", rows[1][0])
	require.Equal(t, "1", rows[1][1])
	require.Equal(t, "66", rows[1][4])
	require.Equal(t, "claude-4", rows[2][0])
	require.Equal(t, "100", rows[2][4])
}

func TestExportAdminAnalyticsUsersScopesAgentRows(t *testing.T) {
	db := setupAdminAnalyticsTestDB(t)
	fixture := seedAdminAnalyticsFixture(t, db)
	originalQuotaDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalQuotaDisplayType
	})
	grantPermissionActions(t, db, fixture.Agent.Id, model.UserTypeAgent,
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionRead},
		permissionGrant{Resource: service.ResourceAnalyticsManagement, Action: service.ActionExport},
	)
	require.NoError(t, db.Create(&model.Log{
		UserId:           fixture.Other.Id,
		CreatedAt:        time.Date(2026, 4, 17, 11, 30, 0, 0, time.Local).Unix(),
		Type:             model.LogTypeConsume,
		Username:         fixture.Other.Username,
		TokenName:        fixture.OwnedToken.Name,
		ModelName:        "token-owned-model",
		Quota:            33,
		PromptTokens:     3,
		CompletionTokens: 7,
		UseTime:          9,
		TokenId:          fixture.OwnedToken.Id,
	}).Error)

	ctx, recorder := newAdminAnalyticsContext(t, http.MethodPost, "/api/admin/analytics/export", map[string]any{
		"view":               "users",
		"date_preset":        "today",
		"start_timestamp":    fixture.TodayStart,
		"end_timestamp":      fixture.Now,
		"sort_by":            "user_id",
		"sort_order":         "asc",
		"quota_display_type": "USD",
		"limit":              10,
	}, fixture.Agent.Id, fixture.Agent.Role)
	ExportAdminAnalytics(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("users")
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.Equal(t, []string{"User ID", "Username", "Call Count", "Model Count", "Total Cost", "Last Called At"}, rows[0])
	require.Equal(t, fixture.Agent.Username, rows[1][1])
	require.Equal(t, fixture.OwnedB.Username, rows[2][1])
	require.Equal(t, fixture.Other.Username, rows[3][1])
	require.Equal(t, "$0.01", rows[3][4])
}

func TestNormalizeAdminAnalyticsQueryLocksPresetWindow(t *testing.T) {
	originalLocal := time.Local
	losAngeles, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	time.Local = losAngeles
	t.Cleanup(func() {
		time.Local = originalLocal
	})

	analyticsLocation := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, analyticsLocation).Unix()

	todayQuery, err := dto.NormalizeAdminAnalyticsQuery(dto.AdminAnalyticsQuery{
		DatePreset:     dto.AdminAnalyticsDatePresetToday,
		StartTimestamp: now - 48*60*60,
		EndTimestamp:   now,
	}, now)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 3, 8, 0, 0, 0, 0, analyticsLocation).Unix(), todayQuery.StartTimestamp)
	require.Equal(t, now, todayQuery.EndTimestamp)

	last7Query, err := dto.NormalizeAdminAnalyticsQuery(dto.AdminAnalyticsQuery{
		DatePreset:     dto.AdminAnalyticsDatePresetLast7Days,
		StartTimestamp: now - 30*24*60*60,
		EndTimestamp:   now,
	}, now)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 3, 2, 0, 0, 0, 0, analyticsLocation).Unix(), last7Query.StartTimestamp)
	require.Equal(t, now, last7Query.EndTimestamp)
}

func TestNormalizeAdminAnalyticsQueryRejectsOversizedCustomRange(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.Local).Unix()
	_, err := dto.NormalizeAdminAnalyticsQuery(dto.AdminAnalyticsQuery{
		DatePreset:     dto.AdminAnalyticsDatePresetCustom,
		StartTimestamp: now - 91*24*60*60,
		EndTimestamp:   now,
	}, now)
	require.ErrorContains(t, err, "cannot exceed 90 days")
}
