package service

import "github.com/QuantumNous/new-api/common"

type SettingAuditActionMeta struct {
	ActionModule string
	ActionType   string
	ActionDesc   string
	TargetType   string
}

func registerSettingAuditMeta(target map[string]SettingAuditActionMeta, meta SettingAuditActionMeta, keys ...string) {
	for _, key := range keys {
		target[key] = meta
	}
}

var settingOptionAuditMeta = func() map[string]SettingAuditActionMeta {
	meta := make(map[string]SettingAuditActionMeta, 160)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_operation", "save_general", "系统设置-运营设置-保存通用设置", "option_key"},
		"TopUpLink",
		"general_setting.docs_link",
		"general_setting.quota_display_type",
		"general_setting.custom_currency_symbol",
		"general_setting.custom_currency_exchange_rate",
		"QuotaPerUnit",
		"RetryTimes",
		"USDExchangeRate",
		"DisplayTokenStatEnabled",
		"DefaultCollapseSidebar",
		"DemoSiteEnabled",
		"SelfUseModeEnabled",
		"token_setting.max_user_tokens",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_operation", "save_header_nav", "系统设置-运营设置-顶栏管理-保存设置", "option_key"},
		"HeaderNavModules",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_operation", "save_admin_sidebar", "系统设置-运营设置-管理员侧边栏-保存设置", "option_key"},
		"SidebarModulesAdmin",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_operation", "save_sensitive_words", "系统设置-运营设置-保存屏蔽词过滤设置", "option_key"},
		"CheckSensitiveEnabled",
		"CheckSensitiveOnPromptEnabled",
		"SensitiveWords",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_operation", "save_log_settings", "系统设置-运营设置-保存日志设置", "option_key"},
		"LogConsumeEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_operation", "save_monitoring", "系统设置-运营设置-保存监控设置", "option_key"},
		"ChannelDisableThreshold",
		"QuotaRemindThreshold",
		"AutomaticDisableChannelEnabled",
		"AutomaticEnableChannelEnabled",
		"AutomaticDisableKeywords",
		"AutomaticDisableStatusCodes",
		"AutomaticRetryStatusCodes",
		"monitor_setting.auto_test_channel_enabled",
		"monitor_setting.auto_test_channel_minutes",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_operation", "save_quota", "系统设置-运营设置-保存额度设置", "option_key"},
		"QuotaForNewUser",
		"PreConsumedQuota",
		"QuotaForInviter",
		"QuotaForInvitee",
		"quota_setting.enable_free_model_pre_consume",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_operation", "save_checkin", "系统设置-运营设置-保存签到设置", "option_key"},
		"checkin_setting.enabled",
		"checkin_setting.min_quota",
		"checkin_setting.max_quota",
	)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "save_data_dashboard", "系统设置-仪表盘设置-保存数据看板设置", "option_key"},
		"DataExportEnabled",
		"DataExportInterval",
		"DataExportDefaultTime",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "save_announcements", "系统设置-仪表盘设置-系统公告-保存设置", "option_key"},
		"console_setting.announcements",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "toggle_announcements_enabled", "系统设置-仪表盘设置-系统公告-启用/禁用", "option_key"},
		"console_setting.announcements_enabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "save_api_info", "系统设置-仪表盘设置-API 信息-保存设置", "option_key"},
		"console_setting.api_info",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "toggle_api_info_enabled", "系统设置-仪表盘设置-API 信息-启用/禁用", "option_key"},
		"console_setting.api_info_enabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "save_faq", "系统设置-仪表盘设置-FAQ-保存设置", "option_key"},
		"console_setting.faq",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "toggle_faq_enabled", "系统设置-仪表盘设置-FAQ-启用/禁用", "option_key"},
		"console_setting.faq_enabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "save_uptime_groups", "系统设置-仪表盘设置-Uptime Kuma-保存设置", "option_key"},
		"console_setting.uptime_kuma_groups",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_dashboard", "toggle_uptime_kuma_enabled", "系统设置-仪表盘设置-Uptime Kuma-启用/禁用", "option_key"},
		"console_setting.uptime_kuma_enabled",
	)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_chat", "save_chats", "系统设置-聊天设置-保存聊天设置", "option_key"},
		"Chats",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_drawing", "save_drawing", "系统设置-绘图设置-保存绘图设置", "option_key"},
		"DrawingEnabled",
		"MjNotifyEnabled",
		"MjAccountFilterEnabled",
		"MjForwardUrlEnabled",
		"MjModeClearEnabled",
		"MjActionCheckSuccessEnabled",
	)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_payment", "save_epay", "系统设置-支付设置-易支付-更新支付设置", "option_key"},
		"PayAddress",
		"EpayId",
		"EpayKey",
		"Price",
		"MinTopUp",
		"TopupGroupRatio",
		"CustomCallbackAddress",
		"PayMethods",
		"payment_setting.amount_options",
		"payment_setting.amount_discount",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_payment", "save_stripe", "系统设置-支付设置-Stripe-更新 Stripe 设置", "option_key"},
		"StripeApiSecret",
		"StripeWebhookSecret",
		"StripePriceId",
		"StripeUnitPrice",
		"StripeMinTopUp",
		"StripePromotionCodesEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_payment", "save_creem", "系统设置-支付设置-Creem-更新 Creem 设置", "option_key"},
		"CreemApiKey",
		"CreemWebhookSecret",
		"CreemProducts",
		"CreemTestMode",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_payment", "save_waffo", "系统设置-支付设置-Waffo-更新 Waffo 设置", "option_key"},
		"WaffoEnabled",
		"WaffoApiKey",
		"WaffoPrivateKey",
		"WaffoPublicCert",
		"WaffoSandboxPublicCert",
		"WaffoSandboxApiKey",
		"WaffoSandboxPrivateKey",
		"WaffoSandbox",
		"WaffoMerchantId",
		"WaffoCurrency",
		"WaffoUnitPrice",
		"WaffoMinTopUp",
		"WaffoNotifyUrl",
		"WaffoReturnUrl",
		"WaffoPayMethods",
	)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_ratio", "save_group_ratio", "系统设置-分组与模型定价设置-保存分组相关设置", "option_key"},
		"GroupRatio",
		"UserUsableGroups",
		"GroupGroupRatio",
		"group_ratio_setting.group_special_usable_group",
		"AutoGroups",
		"DefaultUseAutoGroup",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_ratio", "save_model_ratio", "系统设置-分组与模型定价设置-保存模型倍率设置", "option_key"},
		"ModelPrice",
		"ModelRatio",
		"CacheRatio",
		"CreateCacheRatio",
		"CompletionRatio",
		"ImageRatio",
		"AudioRatio",
		"AudioCompletionRatio",
		"AdvancedPricingMode",
		"AdvancedPricingRules",
		"ExposeRatioEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_rate_limit", "save_model_rate_limit", "系统设置-速率限制设置-保存模型速率限制", "option_key"},
		"ModelRequestRateLimitEnabled",
		"ModelRequestRateLimitCount",
		"ModelRequestRateLimitSuccessCount",
		"ModelRequestRateLimitDurationMinutes",
		"ModelRequestRateLimitGroup",
	)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_model", "save_global_model", "系统设置-模型相关设置-全局设置-保存", "option_key"},
		"global.pass_through_request_enabled",
		"global.thinking_model_blacklist",
		"global.chat_completions_to_responses_policy",
		"general_setting.ping_interval_enabled",
		"general_setting.ping_interval_seconds",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_model", "save_affinity", "系统设置-模型相关设置-渠道亲和性-保存", "option_key"},
		"channel_affinity_setting.enabled",
		"channel_affinity_setting.switch_on_success",
		"channel_affinity_setting.max_entries",
		"channel_affinity_setting.default_ttl_seconds",
		"channel_affinity_setting.rules",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_model", "save_gemini", "系统设置-模型相关设置-Gemini-保存", "option_key"},
		"gemini.safety_settings",
		"gemini.version_settings",
		"gemini.supported_imagine_models",
		"gemini.thinking_adapter_enabled",
		"gemini.thinking_adapter_budget_tokens_percentage",
		"gemini.function_call_thought_signature_enabled",
		"gemini.remove_function_response_id_enabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_model", "save_claude", "系统设置-模型相关设置-Claude-保存", "option_key"},
		"claude.model_headers_settings",
		"claude.thinking_adapter_enabled",
		"claude.default_max_tokens",
		"claude.thinking_adapter_budget_tokens_percentage",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_model", "save_grok", "系统设置-模型相关设置-Grok-保存", "option_key"},
		"grok.violation_deduction_enabled",
		"grok.violation_deduction_amount",
	)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_deployment", "save_deployment_setting", "系统设置-模型部署设置-保存设置", "option_key"},
		"model_deployment.ionet.api_key",
		"model_deployment.ionet.enabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_performance", "save_performance", "系统设置-性能设置-保存性能设置", "option_key"},
		"performance_setting.disk_cache_enabled",
		"performance_setting.disk_cache_threshold_mb",
		"performance_setting.disk_cache_max_size_mb",
		"performance_setting.disk_cache_path",
		"performance_setting.monitor_enabled",
		"performance_setting.monitor_cpu_threshold",
		"performance_setting.monitor_memory_threshold",
		"performance_setting.monitor_disk_threshold",
	)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_worker", "系统设置-系统设置-更新 Worker 设置", "option_key"},
		"WorkerUrl",
		"WorkerValidKey",
		"WorkerAllowHttpImageRequestEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_ssrf", "系统设置-系统设置-更新 SSRF 防护设置", "option_key"},
		"fetch_setting.domain_filter_mode",
		"fetch_setting.domain_list",
		"fetch_setting.ip_filter_mode",
		"fetch_setting.ip_list",
		"fetch_setting.allowed_ports",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_password_login", "系统设置-登录注册-允许通过密码进行登录", "option_key"},
		"PasswordLoginEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_password_register", "系统设置-登录注册-允许通过密码进行注册", "option_key"},
		"PasswordRegisterEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_email_verification", "系统设置-登录注册-密码注册时启用邮箱验证", "option_key"},
		"EmailVerificationEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_register", "系统设置-登录注册-允许新用户注册", "option_key"},
		"RegisterEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_turnstile", "系统设置-登录注册-允许 Turnstile 用户验证", "option_key"},
		"TurnstileCheckEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_github_oauth", "系统设置-登录注册-允许通过 GitHub 登录注册", "option_key"},
		"GitHubOAuthEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_discord_oauth", "系统设置-登录注册-允许通过 Discord 登录注册", "option_key"},
		"discord.enabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_linuxdo_oauth", "系统设置-登录注册-允许通过 Linux DO 登录注册", "option_key"},
		"LinuxDOOAuthEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_wechat_auth", "系统设置-登录注册-允许通过微信登录注册", "option_key"},
		"WeChatAuthEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_telegram_oauth", "系统设置-登录注册-允许通过 Telegram 登录", "option_key"},
		"TelegramOAuthEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_oidc", "系统设置-登录注册-允许通过 OIDC 登录", "option_key"},
		"oidc.enabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_passkey", "系统设置-系统设置-Passkey 启用/禁用", "option_key"},
		"passkey.enabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_passkey_allow_insecure_origin", "系统设置-系统设置-Passkey-允许不安全 Origin", "option_key"},
		"passkey.allow_insecure_origin",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_email_domain_restriction", "系统设置-系统设置-邮箱域名白名单启用/禁用", "option_key"},
		"EmailDomainRestrictionEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_email_alias_restriction", "系统设置-系统设置-邮箱别名限制启用/禁用", "option_key"},
		"EmailAliasRestrictionEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_smtp_ssl", "系统设置-系统设置-SMTP SSL 启用/禁用", "option_key"},
		"SMTPSSLEnabled",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_ssrf_protection", "系统设置-系统设置-SSRF 防护启用/禁用", "option_key"},
		"fetch_setting.enable_ssrf_protection",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_allow_private_ip", "系统设置-系统设置-SSRF-允许访问私有 IP", "option_key"},
		"fetch_setting.allow_private_ip",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "toggle_apply_ip_filter_for_domain", "系统设置-系统设置-SSRF-对域名启用 IP 过滤", "option_key"},
		"fetch_setting.apply_ip_filter_for_domain",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_email_whitelist", "系统设置-系统设置-保存邮箱域名白名单设置", "option_key"},
		"EmailDomainWhitelist",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_smtp", "系统设置-系统设置-保存 SMTP 设置", "option_key"},
		"SMTPServer",
		"SMTPAccount",
		"SMTPFrom",
		"SMTPPort",
		"SMTPToken",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_oidc", "系统设置-系统设置-保存 OIDC 设置", "option_key"},
		"oidc.well_known",
		"oidc.client_id",
		"oidc.client_secret",
		"oidc.authorization_endpoint",
		"oidc.token_endpoint",
		"oidc.user_info_endpoint",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_github_oauth", "系统设置-系统设置-保存 GitHub OAuth 设置", "option_key"},
		"GitHubClientId",
		"GitHubClientSecret",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_discord_oauth", "系统设置-系统设置-保存 Discord OAuth 设置", "option_key"},
		"discord.client_id",
		"discord.client_secret",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_linuxdo_oauth", "系统设置-系统设置-保存 Linux DO OAuth 设置", "option_key"},
		"LinuxDOClientId",
		"LinuxDOClientSecret",
		"LinuxDOMinimumTrustLevel",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_wechat_server", "系统设置-系统设置-保存 WeChat Server 设置", "option_key"},
		"WeChatServerAddress",
		"WeChatAccountQRCodeImageURL",
		"WeChatServerToken",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_telegram_login", "系统设置-系统设置-保存 Telegram 登录设置", "option_key"},
		"TelegramBotToken",
		"TelegramBotName",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_turnstile", "系统设置-系统设置-保存 Turnstile 设置", "option_key"},
		"TurnstileSiteKey",
		"TurnstileSecretKey",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_system", "save_passkey", "系统设置-系统设置-保存 Passkey 设置", "option_key"},
		"passkey.rp_display_name",
		"passkey.rp_id",
		"passkey.user_verification",
		"passkey.attachment_preference",
		"passkey.origins",
	)

	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_notice", "系统设置-其他设置-设置公告", "option_key"},
		"Notice",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_terms", "系统设置-其他设置-设置用户协议", "option_key"},
		"legal.user_agreement",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_privacy", "系统设置-其他设置-设置隐私政策", "option_key"},
		"legal.privacy_policy",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_system_name", "系统设置-其他设置-设置系统名称", "option_key"},
		"SystemName",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_logo", "系统设置-其他设置-设置 Logo", "option_key"},
		"Logo",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_homepage_content", "系统设置-其他设置-设置首页内容", "option_key"},
		"HomePageContent",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_about", "系统设置-其他设置-设置关于", "option_key"},
		"About",
	)
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_footer", "系统设置-其他设置-设置页脚", "option_key"},
		"Footer",
	)

	return meta
}()

var settingAuditSensitiveKeys = func() map[string]struct{} {
	keys := []string{
		"EpayKey",
		"StripeApiSecret",
		"StripeWebhookSecret",
		"CreemApiKey",
		"CreemWebhookSecret",
		"WaffoApiKey",
		"WaffoPrivateKey",
		"WaffoSandboxApiKey",
		"WaffoSandboxPrivateKey",
		"WorkerValidKey",
		"SMTPToken",
		"GitHubClientSecret",
		"discord.client_secret",
		"oidc.client_secret",
		"LinuxDOClientSecret",
		"WeChatServerToken",
		"TelegramBotToken",
		"TurnstileSecretKey",
		"TurnstileSiteKey",
		"model_deployment.ionet.api_key",
	}
	out := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		out[key] = struct{}{}
	}
	return out
}()

func LookupSettingAuditMeta(key string) (SettingAuditActionMeta, bool) {
	meta, ok := settingOptionAuditMeta[key]
	return meta, ok
}

func SanitizeSettingAuditValue(key, value string) string {
	if value == "" {
		return value
	}
	if _, ok := settingAuditSensitiveKeys[key]; ok {
		return "***"
	}
	return value
}

func MarshalSettingOptionAuditValue(key, value string) string {
	payload := map[string]string{
		"key":   key,
		"value": SanitizeSettingAuditValue(key, value),
	}
	raw, err := common.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
