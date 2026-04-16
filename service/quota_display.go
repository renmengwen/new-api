package service

var quotaLedgerEntryTypeLabels = map[string]string{
	"opening":    "初始额度",
	"adjust":     "调额",
	"recharge":   "充值",
	"reclaim":    "回收",
	"consume":    "消费",
	"refund":     "退款",
	"reward":     "奖励",
	"commission": "佣金",
}

var quotaLedgerReasonLabels = map[string]string{
	"manual_adjust":         "手动调额",
	"legacy_user_update":    "手动调额",
	"managed_user_update":   "管理用户调额",
	"batch_adjust":          "批量调额",
	"batch_partial_adjust":  "批量部分调额",
	"agent_adjust":          "代理商调额",
	"agent_reclaim":         "代理商回收",
	"agent_batch_adjust":    "代理商批量调额",
	"sync_with_user_quota":  "同步用户额度",
	"checkin_reward":        "签到奖励",
	"aff_quota_transfer":    "推广返佣",
	"user_register":         "新用户注册赠送",
	"invitee_register":      "邀请注册赠送",
	"wallet_preconsume":     "钱包预扣",
	"wallet_settle_consume": "钱包结算扣费",
	"wallet_settle_refund":  "钱包结算退款",
	"wallet_refund":         "钱包退款",
	"post_consume_quota":    "后置扣费",
	"post_consume_refund":   "后置退款",
	"task_adjust_consume":   "任务重算扣费",
	"task_adjust_refund":    "任务重算退款",
	"midjourney_refund":     "绘图任务退款",
	"stripe":                "Stripe 支付",
	"creem":                 "Creem 支付",
	"waffo":                 "Waffo 支付",
	"epay":                  "易支付",
	"alipay":                "支付宝",
	"wxpay":                 "微信支付",
	"qqpay":                 "QQ支付",
	"wechat":                "微信支付",
}

var quotaLedgerSourceTypeLabels = map[string]string{
	"quota_reconcile":          "同步用户额度",
	"admin_quota_adjust":       "管理员调额",
	"admin_quota_adjust_batch": "批量调额",
	"wallet_settle":            "钱包结算",
	"task_billing":             "任务结算",
	"admin_user_create":        "创建用户",
	"admin_manager_create":     "创建管理员",
	"agent_create":             "创建代理商",
	"root_bootstrap":           "Root 初始化",
	"redemption_recharge":      "兑换码充值",
	"checkin_reward":           "签到奖励",
	"aff_quota_transfer":       "推广返佣",
	"user_register":            "新用户注册赠送",
	"invitee_register":         "邀请注册赠送",
	"midjourney_refund":        "绘图任务退款",
}

var quotaLedgerDirectionLabels = map[string]string{
	"in":  "入账",
	"out": "出账",
}

func GetQuotaEntryTypeLabel(entryType string) string {
	if entryType == "" {
		return "-"
	}
	if label, ok := quotaLedgerEntryTypeLabels[entryType]; ok {
		return label
	}
	return entryType
}

func GetQuotaReasonLabel(reason string) string {
	if reason == "" {
		return "-"
	}
	if label, ok := quotaLedgerReasonLabels[reason]; ok {
		return label
	}
	return reason
}

func GetQuotaSourceTypeLabel(sourceType string) string {
	if sourceType == "" {
		return "-"
	}
	if label, ok := quotaLedgerSourceTypeLabels[sourceType]; ok {
		return label
	}
	if label, ok := quotaLedgerReasonLabels[sourceType]; ok {
		return label
	}
	return sourceType
}

func GetQuotaDirectionLabel(direction string) string {
	if direction == "" {
		return "-"
	}
	if label, ok := quotaLedgerDirectionLabels[direction]; ok {
		return label
	}
	return direction
}
