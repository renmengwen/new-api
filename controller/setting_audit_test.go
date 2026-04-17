package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/ionet"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type settingAuditResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type fakeIoNetConnectionTester struct {
	result *ionet.MaxGPUResponse
	err    error
}

func (f fakeIoNetConnectionTester) GetMaxGPUsPerContainer() (*ionet.MaxGPUResponse, error) {
	return f.result, f.err
}

func setupSettingAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:setting_audit?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.Option{},
		&model.Log{},
		&model.AdminAuditLog{},
		&model.CustomOAuthProvider{},
		&model.UserOAuthBinding{},
	))

	originalOptionMap := common.OptionMap
	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()

	originalLogDir := *common.LogDir

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
		*common.LogDir = originalLogDir

		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newSettingAuditContext(t *testing.T, method string, target string, body any) (*gin.Context, *httptest.ResponseRecorder) {
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
	req.RemoteAddr = "203.0.113.9:12345"
	ctx.Request = req
	ctx.Set("id", 9001)
	ctx.Set("role", common.RoleRootUser)
	return ctx, recorder
}

func mustLoadSettingAuditPayload(t *testing.T, raw string) map[string]any {
	t.Helper()

	var payload map[string]any
	require.NoError(t, common.UnmarshalJsonStr(raw, &payload))
	return payload
}

func TestUpdateOptionCreatesAuditLogForMappedSettingKey(t *testing.T) {
	db := setupSettingAuditTestDB(t)
	require.NoError(t, db.Create(&model.Option{Key: "Notice", Value: "old notice"}).Error)

	common.OptionMapRWMutex.Lock()
	common.OptionMap["Notice"] = "old notice"
	common.OptionMapRWMutex.Unlock()

	ctx, recorder := newSettingAuditContext(t, http.MethodPut, "/api/option/", map[string]any{
		"key":   "Notice",
		"value": "new notice",
	})

	UpdateOption(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_misc", "save_notice").First(&audit).Error)
	require.Equal(t, "系统设置-其他设置-设置公告", audit.ActionDesc)
	require.Equal(t, "option_key", audit.TargetType)
	require.Equal(t, 0, audit.TargetId)
	require.Equal(t, 9001, audit.OperatorUserId)
	require.Equal(t, model.UserTypeRoot, audit.OperatorUserType)
	require.Equal(t, "203.0.113.9", audit.Ip)

	beforePayload := mustLoadSettingAuditPayload(t, audit.BeforeJSON)
	afterPayload := mustLoadSettingAuditPayload(t, audit.AfterJSON)
	require.Equal(t, "Notice", beforePayload["key"])
	require.Equal(t, "old notice", beforePayload["value"])
	require.Equal(t, "Notice", afterPayload["key"])
	require.Equal(t, "new notice", afterPayload["value"])
}

func TestUpdateOptionMasksSensitiveValuesInAuditLog(t *testing.T) {
	db := setupSettingAuditTestDB(t)
	require.NoError(t, db.Create(&model.Option{Key: "GitHubClientSecret", Value: "old-secret"}).Error)

	common.OptionMapRWMutex.Lock()
	common.OptionMap["GitHubClientSecret"] = "old-secret"
	common.OptionMapRWMutex.Unlock()

	ctx, recorder := newSettingAuditContext(t, http.MethodPut, "/api/option/", map[string]any{
		"key":   "GitHubClientSecret",
		"value": "new-secret",
	})

	UpdateOption(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_system", "save_github_oauth").First(&audit).Error)

	beforePayload := mustLoadSettingAuditPayload(t, audit.BeforeJSON)
	afterPayload := mustLoadSettingAuditPayload(t, audit.AfterJSON)
	require.Equal(t, "***", beforePayload["value"])
	require.Equal(t, "***", afterPayload["value"])
}

func TestUpdateOptionPrefersRequestAuditContext(t *testing.T) {
	db := setupSettingAuditTestDB(t)
	require.NoError(t, db.Create(&model.Option{Key: "ServerAddress", Value: "https://old.example.com"}).Error)

	common.OptionMapRWMutex.Lock()
	common.OptionMap["ServerAddress"] = "https://old.example.com"
	common.OptionMapRWMutex.Unlock()

	ctx, recorder := newSettingAuditContext(t, http.MethodPut, "/api/option/", map[string]any{
		"key":          "ServerAddress",
		"value":        "https://pay.example.com",
		"audit_module": "setting_payment",
		"audit_type":   "save_payment_general",
		"audit_desc":   "系统设置-支付设置-通用-更新服务器地址",
	})

	UpdateOption(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_payment", "save_payment_general").First(&audit).Error)
	require.Equal(t, "系统设置-支付设置-通用-更新服务器地址", audit.ActionDesc)
	require.Equal(t, "option_key", audit.TargetType)
	require.Equal(t, 0, audit.TargetId)

	beforePayload := mustLoadSettingAuditPayload(t, audit.BeforeJSON)
	afterPayload := mustLoadSettingAuditPayload(t, audit.AfterJSON)
	require.Equal(t, "ServerAddress", beforePayload["key"])
	require.Equal(t, "https://old.example.com", beforePayload["value"])
	require.Equal(t, "ServerAddress", afterPayload["key"])
	require.Equal(t, "https://pay.example.com", afterPayload["value"])
}

func TestUpdateOptionRejectsInvalidAdvancedPricingConfigBeforePersisting(t *testing.T) {
	db := setupSettingAuditTestDB(t)
	originalValue := `{
      "billing_mode": {
        "gpt-5": "advanced"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "segments": [
            {
              "priority": 10,
              "input_min": 0,
              "input_max": 100,
              "input_price": 1.2
            }
          ]
        }
      }
    }`
	require.NoError(t, db.Create(&model.Option{Key: "AdvancedPricingConfig", Value: originalValue}).Error)

	common.OptionMapRWMutex.Lock()
	common.OptionMap["AdvancedPricingConfig"] = originalValue
	common.OptionMapRWMutex.Unlock()

	ctx, recorder := newSettingAuditContext(t, http.MethodPut, "/api/option/", map[string]any{
		"key":   "AdvancedPricingConfig",
		"value": `{"rules":{"gpt-5":{"rule_type":"text_segment","segments":[]}}}`,
	})

	UpdateOption(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Contains(t, response.Message, "advanced pricing config update failed")

	var option model.Option
	require.NoError(t, db.Where("key = ?", "AdvancedPricingConfig").First(&option).Error)
	require.JSONEq(t, originalValue, option.Value)
}

func TestUpdateOptionCreatesAuditLogForAdvancedPricingConfig(t *testing.T) {
	db := setupSettingAuditTestDB(t)
	originalValue := `{
      "billing_mode": {
        "gpt-5": "advanced"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "segments": [
            {
              "priority": 10,
              "input_min": 0,
              "input_max": 100,
              "input_price": 1.2
            }
          ]
        }
      }
    }`
	updatedValue := `{
      "billing_mode": {
        "gpt-5": "advanced"
      },
      "rules": {
        "gpt-5": {
          "rule_type": "text_segment",
          "segments": [
            {
              "priority": 10,
              "input_min": 0,
              "input_max": 200,
              "input_price": 2.4
            }
          ]
        }
      }
    }`

	require.NoError(t, db.Create(&model.Option{Key: "AdvancedPricingConfig", Value: originalValue}).Error)

	common.OptionMapRWMutex.Lock()
	common.OptionMap["AdvancedPricingConfig"] = originalValue
	common.OptionMapRWMutex.Unlock()

	ctx, recorder := newSettingAuditContext(t, http.MethodPut, "/api/option/", map[string]any{
		"key":   "AdvancedPricingConfig",
		"value": updatedValue,
	})

	UpdateOption(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_ratio", "save_model_ratio").First(&audit).Error)
	require.Equal(t, "option_key", audit.TargetType)

	beforePayload := mustLoadSettingAuditPayload(t, audit.BeforeJSON)
	afterPayload := mustLoadSettingAuditPayload(t, audit.AfterJSON)
	require.Equal(t, "AdvancedPricingConfig", beforePayload["key"])
	require.Equal(t, "AdvancedPricingConfig", afterPayload["key"])
	require.JSONEq(t, originalValue, beforePayload["value"].(string))
	require.JSONEq(t, updatedValue, afterPayload["value"].(string))
}

func TestResetModelRatioCreatesAuditLog(t *testing.T) {
	db := setupSettingAuditTestDB(t)
	require.NoError(t, db.Create(&model.Option{Key: "ModelRatio", Value: `{"gpt-4":2}`}).Error)

	common.OptionMapRWMutex.Lock()
	common.OptionMap["ModelRatio"] = `{"gpt-4":2}`
	common.OptionMapRWMutex.Unlock()

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/option/rest_model_ratio", nil)
	ResetModelRatio(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_ratio", "reset_model_ratio").First(&audit).Error)
	require.Equal(t, "setting_group", audit.TargetType)
}

func TestMigrateConsoleSettingCreatesAuditLog(t *testing.T) {
	db := setupSettingAuditTestDB(t)
	require.NoError(t, db.Create(&model.Option{Key: "Announcements", Value: `[{"content":"hello"}]`}).Error)
	require.NoError(t, db.Create(&model.Option{Key: "ApiInfo", Value: `[{"url":"https://example.com","description":"ok","route":"/v1","color":"blue"}]`}).Error)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/option/migrate_console_setting", nil)
	MigrateConsoleSetting(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_dashboard", "migrate_console_setting").First(&audit).Error)
	require.Equal(t, "setting_group", audit.TargetType)
}

func TestCustomOAuthProviderMutationsCreateAuditLogs(t *testing.T) {
	db := setupSettingAuditTestDB(t)

	createCtx, createRecorder := newSettingAuditContext(t, http.MethodPost, "/api/custom-oauth-provider/", map[string]any{
		"name":                   "Internal SSO",
		"slug":                   "internal-sso",
		"enabled":                true,
		"client_id":              "client-id",
		"client_secret":          "client-secret",
		"authorization_endpoint": "https://sso.example.com/auth",
		"token_endpoint":         "https://sso.example.com/token",
		"user_info_endpoint":     "https://sso.example.com/userinfo",
		"scopes":                 "openid profile email",
	})
	CreateCustomOAuthProvider(createCtx)

	var createResponse settingAuditResponse
	require.NoError(t, common.Unmarshal(createRecorder.Body.Bytes(), &createResponse))
	require.True(t, createResponse.Success)

	var provider model.CustomOAuthProvider
	require.NoError(t, db.Where("slug = ?", "internal-sso").First(&provider).Error)

	updateCtx, updateRecorder := newSettingAuditContext(t, http.MethodPut, "/api/custom-oauth-provider/"+strconv.Itoa(provider.Id), map[string]any{
		"name":       "Internal SSO Updated",
		"enabled":    false,
		"auth_style": 1,
	})
	updateCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(provider.Id)}}
	UpdateCustomOAuthProvider(updateCtx)

	var updateResponse settingAuditResponse
	require.NoError(t, common.Unmarshal(updateRecorder.Body.Bytes(), &updateResponse))
	require.True(t, updateResponse.Success)

	deleteCtx, deleteRecorder := newSettingAuditContext(t, http.MethodDelete, "/api/custom-oauth-provider/"+strconv.Itoa(provider.Id), nil)
	deleteCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(provider.Id)}}
	DeleteCustomOAuthProvider(deleteCtx)

	var deleteResponse settingAuditResponse
	require.NoError(t, common.Unmarshal(deleteRecorder.Body.Bytes(), &deleteResponse))
	require.True(t, deleteResponse.Success)

	var audits []model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ?", "setting_system").Order("id asc").Find(&audits).Error)
	require.Len(t, audits, 3)
	require.Equal(t, "create_custom_oauth_provider", audits[0].ActionType)
	require.Equal(t, "update_custom_oauth_provider", audits[1].ActionType)
	require.Equal(t, "delete_custom_oauth_provider", audits[2].ActionType)
	require.Equal(t, "custom_oauth_provider", audits[0].TargetType)
	require.Equal(t, provider.Id, audits[0].TargetId)

	createAfter := mustLoadSettingAuditPayload(t, audits[0].AfterJSON)
	require.Equal(t, "Internal SSO", createAfter["name"])
	_, hasClientSecret := createAfter["client_secret"]
	require.False(t, hasClientSecret)
}

func TestFetchCustomOAuthDiscoveryCreatesAuditLog(t *testing.T) {
	db := setupSettingAuditTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authorization_endpoint":"https://issuer.example.com/auth","token_endpoint":"https://issuer.example.com/token","userinfo_endpoint":"https://issuer.example.com/userinfo"}`))
	}))
	defer server.Close()

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/custom-oauth-provider/discovery", map[string]any{
		"well_known_url": server.URL,
	})
	FetchCustomOAuthDiscovery(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_system", "fetch_custom_oauth_discovery").First(&audit).Error)
	require.Equal(t, "setting_group", audit.TargetType)
}

func TestPerformanceActionsCreateAuditLogs(t *testing.T) {
	db := setupSettingAuditTestDB(t)

	logDir := t.TempDir()
	*common.LogDir = logDir
	require.NoError(t, os.WriteFile(filepath.Join(logDir, "oneapi-20240101000000.log"), []byte("legacy"), 0o644))

	statsCtx, statsRecorder := newSettingAuditContext(t, http.MethodGet, "/api/performance/stats", nil)
	GetPerformanceStats(statsCtx)
	var statsResponse settingAuditResponse
	require.NoError(t, common.Unmarshal(statsRecorder.Body.Bytes(), &statsResponse))
	require.True(t, statsResponse.Success)

	gcCtx, gcRecorder := newSettingAuditContext(t, http.MethodPost, "/api/performance/gc", nil)
	ForceGC(gcCtx)
	var gcResponse settingAuditResponse
	require.NoError(t, common.Unmarshal(gcRecorder.Body.Bytes(), &gcResponse))
	require.True(t, gcResponse.Success)

	cleanupCtx, cleanupRecorder := newSettingAuditContext(t, http.MethodDelete, "/api/performance/logs?mode=by_count&value=0", nil)
	cleanupCtx.Request = httptest.NewRequest(http.MethodDelete, "/api/performance/logs?mode=by_count&value=1", nil)
	cleanupCtx.Request.RemoteAddr = "203.0.113.9:12345"
	cleanupCtx.Set("id", 9001)
	cleanupCtx.Set("role", common.RoleRootUser)
	CleanupLogFiles(cleanupCtx)
	var cleanupResponse settingAuditResponse
	require.NoError(t, common.Unmarshal(cleanupRecorder.Body.Bytes(), &cleanupResponse))
	require.True(t, cleanupResponse.Success)

	var audits []model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ?", "setting_performance").Order("id asc").Find(&audits).Error)
	require.Len(t, audits, 3)
	require.Equal(t, "refresh_performance_stats", audits[0].ActionType)
	require.Equal(t, "force_gc", audits[1].ActionType)
	require.Equal(t, "cleanup_log_files", audits[2].ActionType)
}

func TestChannelAffinityCacheActionsCreateAuditLogs(t *testing.T) {
	db := setupSettingAuditTestDB(t)

	statsCtx, statsRecorder := newSettingAuditContext(t, http.MethodGet, "/api/option/channel_affinity_cache", nil)
	GetChannelAffinityCacheStats(statsCtx)
	var statsResponse settingAuditResponse
	require.NoError(t, common.Unmarshal(statsRecorder.Body.Bytes(), &statsResponse))
	require.True(t, statsResponse.Success)

	clearCtx, clearRecorder := newSettingAuditContext(t, http.MethodDelete, "/api/option/channel_affinity_cache?all=true", nil)
	clearCtx.Request = httptest.NewRequest(http.MethodDelete, "/api/option/channel_affinity_cache?all=true", nil)
	clearCtx.Request.RemoteAddr = "203.0.113.9:12345"
	clearCtx.Set("id", 9001)
	clearCtx.Set("role", common.RoleRootUser)
	ClearChannelAffinityCache(clearCtx)
	var clearResponse settingAuditResponse
	require.NoError(t, common.Unmarshal(clearRecorder.Body.Bytes(), &clearResponse))
	require.True(t, clearResponse.Success)

	var audits []model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ?", "setting_model").Order("id asc").Find(&audits).Error)
	require.Len(t, audits, 2)
	require.Equal(t, "refresh_affinity_cache_stats", audits[0].ActionType)
	require.Equal(t, "clear_affinity_cache", audits[1].ActionType)
}

func TestDeleteHistoryLogsCreatesAuditLog(t *testing.T) {
	db := setupSettingAuditTestDB(t)
	require.NoError(t, db.Create(&model.Log{
		UserId:    1,
		CreatedAt: 10,
		Type:      model.LogTypeConsume,
		Username:  "tester",
	}).Error)

	ctx, recorder := newSettingAuditContext(t, http.MethodDelete, "/api/log/?target_timestamp=20", nil)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/log/?target_timestamp=20", nil)
	ctx.Request.RemoteAddr = "203.0.113.9:12345"
	ctx.Set("id", 9001)
	ctx.Set("role", common.RoleRootUser)

	DeleteHistoryLogs(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_operation", "clear_history_logs").First(&audit).Error)
}

func TestFetchUpstreamRatiosCreatesAuditLog(t *testing.T) {
	db := setupSettingAuditTestDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"model_ratio":{"gpt-4":2}}}`))
	}))
	defer server.Close()

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/ratio_sync/fetch", map[string]any{
		"upstreams": []map[string]any{
			{
				"name":     "local",
				"base_url": server.URL,
				"endpoint": "/api/ratio_config",
			},
		},
		"timeout": 1,
	})
	FetchUpstreamRatios(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_ratio", "fetch_upstream_ratio").First(&audit).Error)
}

func TestIoNetConnectionCreatesAuditLog(t *testing.T) {
	db := setupSettingAuditTestDB(t)

	originalFactory := newIoNetEnterpriseConnectionTester
	newIoNetEnterpriseConnectionTester = func(apiKey string) ioNetConnectionTester {
		require.Equal(t, "test-api-key", apiKey)
		return fakeIoNetConnectionTester{
			result: &ionet.MaxGPUResponse{
				Total: 3,
				Hardware: []ionet.MaxGPUInfo{
					{HardwareID: 1, Available: 1},
					{HardwareID: 2, Available: 2},
				},
			},
		}
	}
	t.Cleanup(func() {
		newIoNetEnterpriseConnectionTester = originalFactory
	})

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/deployments/settings/test-connection", map[string]any{
		"api_key": "test-api-key",
	})

	TestIoNetConnection(ctx)

	var response settingAuditResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var audit model.AdminAuditLog
	require.NoError(t, db.Where("action_module = ? AND action_type = ?", "setting_deployment", "test_connection").First(&audit).Error)
	require.Equal(t, "system_runtime", audit.TargetType)
}

func TestAuditCatalogMapsNoticeAndSensitiveKeys(t *testing.T) {
	noticeMeta, ok := service.LookupSettingAuditMeta("Notice")
	require.True(t, ok)
	require.Equal(t, "setting_misc", noticeMeta.ActionModule)
	require.Equal(t, "save_notice", noticeMeta.ActionType)

	require.Equal(t, "***", service.SanitizeSettingAuditValue("GitHubClientSecret", "secret"))
	require.Equal(t, "visible", service.SanitizeSettingAuditValue("Notice", "visible"))
}
