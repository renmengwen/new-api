package service

var auditLogModuleLabels = map[string]string{
	"admin_management":    "管理员管理",
	"agent":               "代理管理",
	"user_management":     "用户管理",
	"permission":          "权限管理",
	"quota":               "额度管理",
	"setting_operation":   "运营设置",
	"setting_dashboard":   "仪表盘设置",
	"setting_chat":        "聊天设置",
	"setting_drawing":     "绘图设置",
	"setting_payment":     "支付设置",
	"setting_ratio":       "分组与模型定价设置",
	"setting_rate_limit":  "速率限制设置",
	"setting_model":       "模型相关设置",
	"setting_deployment":  "模型部署设置",
	"setting_performance": "性能设置",
	"setting_system":      "系统设置",
	"setting_misc":        "其他设置",
}

var auditLogActionLabels = map[string]string{
	"create":                               "创建",
	"update":                               "更新",
	"enable":                               "启用",
	"disable":                              "禁用",
	"delete":                               "删除",
	"bind_profile":                         "绑定权限模板",
	"clear_profile":                        "清空权限模板",
	"adjust":                               "额度调整",
	"adjust_batch":                         "批量额度调整",
	"save_general":                         "保存通用设置",
	"save_header_nav":                      "保存顶栏设置",
	"save_admin_sidebar":                   "保存管理员侧边栏设置",
	"save_sensitive_words":                 "保存屏蔽词过滤设置",
	"clear_history_logs":                   "清除历史日志",
	"save_log_settings":                    "保存日志设置",
	"save_monitoring":                      "保存监控设置",
	"save_quota":                           "保存额度设置",
	"save_checkin":                         "保存签到设置",
	"migrate_console_setting":              "确认迁移",
	"save_data_dashboard":                  "保存数据看板设置",
	"save_announcements":                   "保存公告设置",
	"toggle_announcements_enabled":         "启用或禁用公告",
	"save_api_info":                        "保存 API 信息设置",
	"toggle_api_info_enabled":              "启用或禁用 API 信息",
	"save_faq":                             "保存 FAQ 设置",
	"toggle_faq_enabled":                   "启用或禁用 FAQ",
	"save_uptime_groups":                   "保存 Uptime Kuma 设置",
	"toggle_uptime_kuma_enabled":           "启用或禁用 Uptime Kuma",
	"save_chats":                           "保存聊天设置",
	"save_drawing":                         "保存绘图设置",
	"save_payment_general":                 "更新服务器地址",
	"save_epay":                            "更新易支付设置",
	"save_stripe":                          "更新 Stripe 设置",
	"save_creem":                           "更新 Creem 设置",
	"save_waffo":                           "更新 Waffo 设置",
	"save_group_ratio":                     "保存分组相关设置",
	"save_model_ratio":                     "保存模型倍率设置",
	"reset_model_ratio":                    "重置模型倍率",
	"fetch_upstream_ratio":                 "拉取上游倍率",
	"apply_upstream_ratio":                 "应用同步",
	"save_model_rate_limit":                "保存模型速率限制",
	"save_global_model":                    "保存全局设置",
	"save_affinity":                        "保存渠道亲和性",
	"refresh_affinity_cache_stats":         "刷新缓存统计",
	"clear_affinity_cache":                 "清空缓存",
	"save_gemini":                          "保存 Gemini 设置",
	"save_claude":                          "保存 Claude 设置",
	"save_grok":                            "保存 Grok 设置",
	"save_deployment_setting":              "保存部署设置",
	"test_connection":                      "测试连接",
	"save_performance":                     "保存性能设置",
	"refresh_performance_stats":            "刷新统计",
	"clear_disk_cache":                     "清理不活跃缓存",
	"reset_performance_stats":              "重置统计",
	"force_gc":                             "执行 GC",
	"get_log_files":                        "查看日志文件",
	"cleanup_log_files":                    "清理日志文件",
	"save_server_url":                      "更新服务器地址",
	"save_worker":                          "更新 Worker 设置",
	"save_ssrf":                            "更新 SSRF 防护设置",
	"save_passkey":                         "保存 Passkey 设置",
	"save_email_whitelist":                 "保存邮箱域名白名单设置",
	"save_smtp":                            "保存 SMTP 设置",
	"save_oidc":                            "保存 OIDC 设置",
	"save_github_oauth":                    "保存 GitHub OAuth 设置",
	"save_discord_oauth":                   "保存 Discord OAuth 设置",
	"save_linuxdo_oauth":                   "保存 Linux DO OAuth 设置",
	"save_wechat_server":                   "保存 WeChat Server 设置",
	"save_telegram_login":                  "保存 Telegram 登录设置",
	"save_turnstile":                       "保存 Turnstile 设置",
	"toggle_password_login":                "允许密码登录",
	"toggle_password_register":             "允许密码注册",
	"toggle_email_verification":            "启用邮箱验证",
	"toggle_register":                      "允许新用户注册",
	"toggle_turnstile":                     "启用 Turnstile 验证",
	"toggle_github_oauth":                  "启用 GitHub 登录",
	"toggle_discord_oauth":                 "启用 Discord 登录",
	"toggle_linuxdo_oauth":                 "启用 Linux DO 登录",
	"toggle_wechat_auth":                   "启用微信登录",
	"toggle_telegram_oauth":                "启用 Telegram 登录",
	"toggle_oidc":                          "启用 OIDC 登录",
	"toggle_passkey":                       "启用或禁用 Passkey",
	"toggle_passkey_allow_insecure_origin": "允许不安全 Origin",
	"toggle_email_domain_restriction":      "启用或禁用邮箱域名白名单",
	"toggle_email_alias_restriction":       "启用或禁用邮箱别名限制",
	"toggle_smtp_ssl":                      "启用或禁用 SMTP SSL",
	"toggle_ssrf_protection":               "启用或禁用 SSRF 防护",
	"toggle_allow_private_ip":              "允许访问私有 IP",
	"toggle_apply_ip_filter_for_domain":    "对域名启用 IP 过滤",
	"create_custom_oauth_provider":         "新增提供商",
	"update_custom_oauth_provider":         "编辑提供商",
	"delete_custom_oauth_provider":         "删除提供商",
	"fetch_custom_oauth_discovery":         "获取 Discovery 配置",
	"save_notice":                          "设置公告",
	"save_terms":                           "设置用户协议",
	"save_privacy":                         "设置隐私政策",
	"save_system_name":                     "设置系统名称",
	"save_logo":                            "设置 Logo",
	"save_homepage_content":                "设置首页内容",
	"save_about":                           "设置关于",
	"save_footer":                          "设置页脚",
}

var auditLogTargetTypeLabels = map[string]string{
	"user":                   "用户",
	"option_key":             "配置项",
	"setting_group":          "设置分组",
	"channel_affinity_cache": "渠道亲和缓存",
	"system_runtime":         "系统运行态",
	"custom_oauth_provider":  "自定义 OAuth 提供商",
}

func GetAuditLogModuleLabel(value string) string {
	if label, ok := auditLogModuleLabels[value]; ok {
		return label
	}
	return value
}

func GetAuditLogActionLabel(value string) string {
	if label, ok := auditLogActionLabels[value]; ok {
		return label
	}
	return value
}

func GetAuditLogTargetTypeLabel(value string) string {
	if label, ok := auditLogTargetTypeLabels[value]; ok {
		return label
	}
	return value
}
