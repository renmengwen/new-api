package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

var (
	settingAuditMetaResetModelRatio = service.SettingAuditActionMeta{
		ActionModule: "setting_ratio",
		ActionType:   "reset_model_ratio",
		ActionDesc:   "系统设置-分组与模型定价设置-重置模型倍率",
		TargetType:   "setting_group",
	}
	settingAuditMetaMigrateConsoleSetting = service.SettingAuditActionMeta{
		ActionModule: "setting_dashboard",
		ActionType:   "migrate_console_setting",
		ActionDesc:   "系统设置-仪表盘设置-配置迁移确认-确认迁移",
		TargetType:   "setting_group",
	}
	settingAuditMetaClearHistoryLogs = service.SettingAuditActionMeta{
		ActionModule: "setting_operation",
		ActionType:   "clear_history_logs",
		ActionDesc:   "系统设置-运营设置-日志设置-清除历史日志",
		TargetType:   "system_runtime",
	}
	settingAuditMetaFetchUpstreamRatio = service.SettingAuditActionMeta{
		ActionModule: "setting_ratio",
		ActionType:   "fetch_upstream_ratio",
		ActionDesc:   "系统设置-分组与模型定价设置-上游倍率同步-拉取上游倍率",
		TargetType:   "setting_group",
	}
	settingAuditMetaTestIoNetConnection = service.SettingAuditActionMeta{
		ActionModule: "setting_deployment",
		ActionType:   "test_connection",
		ActionDesc:   "系统设置-模型部署设置-测试连接",
		TargetType:   "system_runtime",
	}
	settingAuditMetaFetchCustomOAuthDiscovery = service.SettingAuditActionMeta{
		ActionModule: "setting_system",
		ActionType:   "fetch_custom_oauth_discovery",
		ActionDesc:   "系统设置-系统设置-自定义 OAuth-获取 Discovery 配置",
		TargetType:   "setting_group",
	}
	settingAuditMetaCreateCustomOAuthProvider = service.SettingAuditActionMeta{
		ActionModule: "setting_system",
		ActionType:   "create_custom_oauth_provider",
		ActionDesc:   "系统设置-系统设置-自定义 OAuth-新增提供商",
		TargetType:   "custom_oauth_provider",
	}
	settingAuditMetaUpdateCustomOAuthProvider = service.SettingAuditActionMeta{
		ActionModule: "setting_system",
		ActionType:   "update_custom_oauth_provider",
		ActionDesc:   "系统设置-系统设置-自定义 OAuth-编辑提供商",
		TargetType:   "custom_oauth_provider",
	}
	settingAuditMetaDeleteCustomOAuthProvider = service.SettingAuditActionMeta{
		ActionModule: "setting_system",
		ActionType:   "delete_custom_oauth_provider",
		ActionDesc:   "系统设置-系统设置-自定义 OAuth-删除提供商",
		TargetType:   "custom_oauth_provider",
	}
	settingAuditMetaRefreshPerformanceStats = service.SettingAuditActionMeta{
		ActionModule: "setting_performance",
		ActionType:   "refresh_performance_stats",
		ActionDesc:   "系统设置-性能设置-刷新统计",
		TargetType:   "system_runtime",
	}
	settingAuditMetaClearDiskCache = service.SettingAuditActionMeta{
		ActionModule: "setting_performance",
		ActionType:   "clear_disk_cache",
		ActionDesc:   "系统设置-性能设置-清理不活跃缓存",
		TargetType:   "system_runtime",
	}
	settingAuditMetaResetPerformanceStats = service.SettingAuditActionMeta{
		ActionModule: "setting_performance",
		ActionType:   "reset_performance_stats",
		ActionDesc:   "系统设置-性能设置-重置统计",
		TargetType:   "system_runtime",
	}
	settingAuditMetaForceGC = service.SettingAuditActionMeta{
		ActionModule: "setting_performance",
		ActionType:   "force_gc",
		ActionDesc:   "系统设置-性能设置-执行 GC",
		TargetType:   "system_runtime",
	}
	settingAuditMetaGetLogFiles = service.SettingAuditActionMeta{
		ActionModule: "setting_performance",
		ActionType:   "get_log_files",
		ActionDesc:   "系统设置-性能设置-查看日志文件",
		TargetType:   "system_runtime",
	}
	settingAuditMetaCleanupLogFiles = service.SettingAuditActionMeta{
		ActionModule: "setting_performance",
		ActionType:   "cleanup_log_files",
		ActionDesc:   "系统设置-性能设置-清理日志文件",
		TargetType:   "system_runtime",
	}
	settingAuditMetaRefreshAffinityCacheStats = service.SettingAuditActionMeta{
		ActionModule: "setting_model",
		ActionType:   "refresh_affinity_cache_stats",
		ActionDesc:   "系统设置-模型相关设置-渠道亲和性-刷新缓存统计",
		TargetType:   "channel_affinity_cache",
	}
	settingAuditMetaClearAffinityCache = service.SettingAuditActionMeta{
		ActionModule: "setting_model",
		ActionType:   "clear_affinity_cache",
		ActionDesc:   "系统设置-模型相关设置-渠道亲和性-清空缓存",
		TargetType:   "channel_affinity_cache",
	}
)

func getSettingOptionCurrentValue(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return common.OptionMap[key]
}

func marshalSettingAuditPayload(payload any) string {
	raw, err := common.Marshal(payload)
	if err != nil {
		common.SysError("marshal setting audit payload failed: " + err.Error())
		return "{}"
	}
	return string(raw)
}

func createSettingAuditLog(c *gin.Context, meta service.SettingAuditActionMeta, targetID int, beforeJSON string, afterJSON string) {
	if meta.ActionModule == "" || meta.ActionType == "" {
		return
	}

	operatorUserID := c.GetInt("id")
	operatorRole := c.GetInt("role")
	if err := service.CreateAdminAuditLog(service.AuditLogInput{
		OperatorUserId:   operatorUserID,
		OperatorUserType: service.UserTypeFromRole(operatorRole),
		ActionModule:     meta.ActionModule,
		ActionType:       meta.ActionType,
		ActionDesc:       meta.ActionDesc,
		TargetType:       meta.TargetType,
		TargetId:         targetID,
		BeforeJSON:       beforeJSON,
		AfterJSON:        afterJSON,
		IP:               c.ClientIP(),
	}); err != nil {
		common.SysError("create setting audit log failed: " + err.Error())
	}
}

func resolveSettingOptionAuditMeta(option OptionUpdateRequest) (service.SettingAuditActionMeta, bool) {
	fallbackMeta, hasFallback := service.LookupSettingAuditMeta(option.Key)

	auditModule := strings.TrimSpace(option.AuditModule)
	auditType := strings.TrimSpace(option.AuditType)
	auditDesc := strings.TrimSpace(option.AuditDesc)
	if auditModule != "" && auditType != "" {
		meta := service.SettingAuditActionMeta{
			ActionModule: auditModule,
			ActionType:   auditType,
			ActionDesc:   auditDesc,
			TargetType:   "option_key",
		}
		if hasFallback && fallbackMeta.TargetType != "" {
			meta.TargetType = fallbackMeta.TargetType
		}
		if meta.ActionDesc == "" && hasFallback {
			meta.ActionDesc = fallbackMeta.ActionDesc
		}
		return meta, true
	}

	if hasFallback {
		return fallbackMeta, true
	}

	return service.SettingAuditActionMeta{}, false
}

func createSettingOptionAuditLog(c *gin.Context, option OptionUpdateRequest, beforeValue string, afterValue string) {
	meta, ok := resolveSettingOptionAuditMeta(option)
	if !ok {
		return
	}
	createSettingAuditLog(
		c,
		meta,
		0,
		service.MarshalSettingOptionAuditValue(option.Key, beforeValue),
		service.MarshalSettingOptionAuditValue(option.Key, afterValue),
	)
}

func marshalCustomOAuthProviderAuditPayload(provider *model.CustomOAuthProvider, includeSecretUpdated bool) string {
	if provider == nil {
		return ""
	}

	payload := map[string]any{
		"id":                     provider.Id,
		"name":                   provider.Name,
		"slug":                   provider.Slug,
		"icon":                   provider.Icon,
		"enabled":                provider.Enabled,
		"client_id":              provider.ClientId,
		"authorization_endpoint": provider.AuthorizationEndpoint,
		"token_endpoint":         provider.TokenEndpoint,
		"user_info_endpoint":     provider.UserInfoEndpoint,
		"scopes":                 provider.Scopes,
		"user_id_field":          provider.UserIdField,
		"username_field":         provider.UsernameField,
		"display_name_field":     provider.DisplayNameField,
		"email_field":            provider.EmailField,
		"well_known":             provider.WellKnown,
		"auth_style":             provider.AuthStyle,
		"access_policy":          provider.AccessPolicy,
		"access_denied_message":  provider.AccessDeniedMessage,
	}
	if includeSecretUpdated {
		payload["client_secret_updated"] = true
	}
	return marshalSettingAuditPayload(payload)
}
