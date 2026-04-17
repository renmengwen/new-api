package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// LogTaskConsumption 记录任务消费日志和统计信息（仅记录，不涉及实际扣费）。
// 实际扣费已由 BillingSession（PreConsumeBilling + SettleBilling）完成。
func LogTaskConsumption(c *gin.Context, info *relaycommon.RelayInfo) {
	tokenName := c.GetString("token_name")
	logContent := fmt.Sprintf("操作 %s", info.Action)
	// 支持任务仅按次计费
	if common.StringsContains(constant.TaskPricePatches, info.OriginModelName) {
		logContent = fmt.Sprintf("%s，按次计费", logContent)
	} else {
		if len(info.PriceData.OtherRatios) > 0 {
			var contents []string
			for key, ra := range info.PriceData.OtherRatios {
				if 1.0 != ra {
					contents = append(contents, fmt.Sprintf("%s: %.2f", key, ra))
				}
			}
			if len(contents) > 0 {
				logContent = fmt.Sprintf("%s, 计算参数：%s", logContent, strings.Join(contents, ", "))
			}
		}
	}
	other := make(map[string]interface{})
	other["request_path"] = c.Request.URL.Path
	other["model_price"] = info.PriceData.ModelPrice
	other["group_ratio"] = info.PriceData.GroupRatioInfo.GroupRatio
	appendTaskPriceDataAdvancedInfo(other, info.PriceData)
	if info.PriceData.GroupRatioInfo.HasSpecialRatio {
		other["user_group_ratio"] = info.PriceData.GroupRatioInfo.GroupSpecialRatio
	}
	if info.IsModelMapped {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = info.UpstreamModelName
	}
	model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
		ChannelId: info.ChannelId,
		ModelName: info.OriginModelName,
		TokenName: tokenName,
		Quota:     info.PriceData.Quota,
		Content:   logContent,
		TokenId:   info.TokenId,
		Group:     info.UsingGroup,
		Other:     other,
	})
	model.UpdateUserUsedQuotaAndRequestCount(info.UserId, info.PriceData.Quota)
	model.UpdateChannelUsedQuota(info.ChannelId, info.PriceData.Quota)
}

// ---------------------------------------------------------------------------
// 异步任务计费辅助函数
// ---------------------------------------------------------------------------

// resolveTokenKey 通过 TokenId 运行时获取令牌 Key（用于 Redis 缓存操作）。
// 如果令牌已被删除或查询失败，返回空字符串。
func resolveTokenKey(ctx context.Context, tokenId int, taskID string) string {
	token, err := model.GetTokenById(tokenId)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("获取令牌 key 失败 (tokenId=%d, task=%s): %s", tokenId, taskID, err.Error()))
		return ""
	}
	return token.Key
}

// taskIsSubscription 判断任务是否通过订阅计费。
func taskIsSubscription(task *model.Task) bool {
	return task.PrivateData.BillingSource == BillingSourceSubscription && task.PrivateData.SubscriptionId > 0
}

// taskAdjustFunding 调整任务的资金来源（钱包或订阅），delta > 0 表示扣费，delta < 0 表示退还。
func taskAdjustFunding(task *model.Task, delta int) error {
	if taskIsSubscription(task) {
		return model.PostConsumeUserSubscriptionDelta(task.PrivateData.SubscriptionId, int64(delta))
	}
	if delta > 0 {
		return applyQuotaLedgerEntry(quotaLedgerEntryInput{
			UserId:     task.UserId,
			Delta:      -delta,
			EntryType:  model.LedgerEntryConsume,
			SourceType: "task_billing",
			SourceId:   int(task.ID),
			Reason:     "task_adjust_consume",
		})
	}
	return applyQuotaLedgerEntry(quotaLedgerEntryInput{
		UserId:     task.UserId,
		Delta:      -delta,
		EntryType:  model.LedgerEntryRefund,
		SourceType: "task_billing",
		SourceId:   int(task.ID),
		Reason:     "task_adjust_refund",
	})
}

// taskAdjustTokenQuota 调整任务的令牌额度，delta > 0 表示扣费，delta < 0 表示退还。
// 需要通过 resolveTokenKey 运行时获取 key（不从 PrivateData 中读取）。
func taskAdjustTokenQuota(ctx context.Context, task *model.Task, delta int) {
	if task.PrivateData.TokenId <= 0 || delta == 0 {
		return
	}
	tokenKey := resolveTokenKey(ctx, task.PrivateData.TokenId, task.TaskID)
	if tokenKey == "" {
		return
	}
	var err error
	if delta > 0 {
		err = model.DecreaseTokenQuota(task.PrivateData.TokenId, tokenKey, delta)
	} else {
		err = model.IncreaseTokenQuota(task.PrivateData.TokenId, tokenKey, -delta)
	}
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("调整令牌额度失败 (delta=%d, task=%s): %s", delta, task.TaskID, err.Error()))
	}
}

// taskBillingOther 从 task 的 BillingContext 构建日志 Other 字段。
func taskBillingOther(task *model.Task) map[string]interface{} {
	other := make(map[string]interface{})
	if bc := task.PrivateData.BillingContext; bc != nil {
		other["model_price"] = bc.ModelPrice
		other["group_ratio"] = bc.GroupRatio
		appendTaskBillingContextAdvancedInfo(other, bc)
		if len(bc.OtherRatios) > 0 {
			for k, v := range bc.OtherRatios {
				other[k] = v
			}
		}
	}
	props := task.Properties
	if props.UpstreamModelName != "" && props.UpstreamModelName != props.OriginModelName {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = props.UpstreamModelName
	}
	return other
}

func appendTaskPriceDataAdvancedInfo(other map[string]interface{}, priceData types.PriceData) {
	if other == nil || priceData.BillingMode != types.BillingModeAdvanced {
		return
	}
	other["billing_mode"] = string(priceData.BillingMode)
	if priceData.AdvancedRuleType != "" {
		other["advanced_rule_type"] = string(priceData.AdvancedRuleType)
	}
	if priceData.AdvancedRuleSnapshot != nil {
		other["advanced_rule"] = priceData.AdvancedRuleSnapshot
	}
}

func appendTaskBillingContextAdvancedInfo(other map[string]interface{}, billingContext *model.TaskBillingContext) {
	if other == nil || billingContext == nil || billingContext.BillingMode != types.BillingModeAdvanced {
		return
	}
	other["billing_mode"] = string(billingContext.BillingMode)
	if billingContext.AdvancedRuleType != "" {
		other["advanced_rule_type"] = string(billingContext.AdvancedRuleType)
	}
	if billingContext.AdvancedRuleSnapshot != nil {
		other["advanced_rule"] = billingContext.AdvancedRuleSnapshot
	}
}

func taskBillingAdvancedSummary(billingContext *model.TaskBillingContext) string {
	if billingContext == nil || billingContext.BillingMode != types.BillingModeAdvanced || billingContext.AdvancedRuleSnapshot == nil {
		return ""
	}
	ruleType := billingContext.AdvancedRuleType
	if ruleType == "" {
		ruleType = billingContext.AdvancedRuleSnapshot.RuleType
	}
	matchSummary := billingContext.AdvancedRuleSnapshot.MatchSummary
	switch {
	case ruleType != "" && matchSummary != "":
		return fmt.Sprintf("advanced rule %s: %s", ruleType, matchSummary)
	case ruleType != "":
		return fmt.Sprintf("advanced rule %s", ruleType)
	case matchSummary != "":
		return fmt.Sprintf("advanced rule: %s", matchSummary)
	default:
		return "advanced rule"
	}
}

// taskModelName 从 BillingContext 或 Properties 中获取模型名称。
func taskModelName(task *model.Task) string {
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OriginModelName != "" {
		return bc.OriginModelName
	}
	return task.Properties.OriginModelName
}

func syncTaskUsageToConsumeLog(ctx context.Context, task *model.Task, taskResult *relaycommon.TaskInfo) {
	if taskResult == nil || taskResult.TotalTokens <= 0 {
		return
	}

	completionTokens := taskResult.CompletionTokens
	promptTokens := 0
	if completionTokens <= 0 || completionTokens > taskResult.TotalTokens {
		completionTokens = taskResult.TotalTokens
	} else {
		promptTokens = taskResult.TotalTokens - completionTokens
	}

	if err := model.UpdateConsumeLogTokensByRequestID(task.UserId, task.PrivateData.RequestId, promptTokens, completionTokens); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("回写任务 token 用量到原始消费日志失败 (task=%s, request_id=%s): %s", task.TaskID, task.PrivateData.RequestId, err.Error()))
	}
}

// RefundTaskQuota 统一的任务失败退款逻辑。
// 当异步任务失败时，将预扣的 quota 退还给用户（支持钱包和订阅），并退还令牌额度。
func RefundTaskQuota(ctx context.Context, task *model.Task, reason string) {
	quota := task.Quota
	if quota == 0 {
		return
	}

	// 1. 退还资金来源（钱包或订阅）
	if err := taskAdjustFunding(task, -quota); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("退还资金来源失败 task %s: %s", task.TaskID, err.Error()))
		return
	}

	// 2. 退还令牌额度
	taskAdjustTokenQuota(ctx, task, -quota)

	// 3. 记录日志
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	other["reason"] = reason
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    task.UserId,
		LogType:   model.LogTypeRefund,
		Content:   "",
		ChannelId: task.ChannelId,
		ModelName: taskModelName(task),
		Quota:     quota,
		TokenId:   task.PrivateData.TokenId,
		Group:     task.Group,
		Other:     other,
	})
}

// RecalculateTaskQuota 通用的异步差额结算。
// actualQuota 是任务完成后的实际应扣额度，与预扣额度 (task.Quota) 做差额结算。
// reason 用于日志记录（例如 "token重算" 或 "adaptor调整"）。
func recalculateTaskQuota(ctx context.Context, task *model.Task, actualQuota int, reason string, allowZero bool) {
	if actualQuota < 0 || (!allowZero && actualQuota == 0) {
		return
	}
	preConsumedQuota := task.Quota
	quotaDelta := actualQuota - preConsumedQuota

	if quotaDelta == 0 {
		logger.LogInfo(ctx, fmt.Sprintf("任务 %s 预扣费准确（%s，%s）",
			task.TaskID, logger.LogQuota(actualQuota), reason))
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("任务 %s 差额结算：delta=%s（实际：%s，预扣：%s，%s）",
		task.TaskID,
		logger.LogQuota(quotaDelta),
		logger.LogQuota(actualQuota),
		logger.LogQuota(preConsumedQuota),
		reason,
	))

	// 调整资金来源
	if err := taskAdjustFunding(task, quotaDelta); err != nil {
		logger.LogError(ctx, fmt.Sprintf("差额结算资金调整失败 task %s: %s", task.TaskID, err.Error()))
		return
	}

	// 调整令牌额度
	taskAdjustTokenQuota(ctx, task, quotaDelta)

	task.Quota = actualQuota

	var logType int
	var logQuota int
	if quotaDelta > 0 {
		logType = model.LogTypeConsume
		logQuota = quotaDelta
		model.UpdateUserUsedQuotaAndRequestCount(task.UserId, quotaDelta)
		model.UpdateChannelUsedQuota(task.ChannelId, quotaDelta)
	} else {
		logType = model.LogTypeRefund
		logQuota = -quotaDelta
	}
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	appendTaskBillingModeFromReason(other, task.PrivateData.BillingContext, reason)
	//other["reason"] = reason
	other["pre_consumed_quota"] = preConsumedQuota
	other["actual_quota"] = actualQuota
	logContent := reason
	if summary := taskBillingAdvancedSummary(task.PrivateData.BillingContext); summary != "" {
		if logContent == "" {
			logContent = summary
		} else {
			logContent = fmt.Sprintf("%s | %s", reason, summary)
		}
	}
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    task.UserId,
		LogType:   logType,
		Content:   logContent,
		ChannelId: task.ChannelId,
		ModelName: taskModelName(task),
		Quota:     logQuota,
		TokenId:   task.PrivateData.TokenId,
		Group:     task.Group,
		Other:     other,
	})
}

// RecalculateTaskQuotaByTokens 根据实际 token 消耗重新计费（异步差额结算）。
// 当任务成功且返回了 totalTokens 时，根据模型倍率和分组倍率重新计算实际扣费额度，
// 与预扣费的差额进行补扣或退还。支持钱包和订阅计费来源。
func RecalculateTaskQuota(ctx context.Context, task *model.Task, actualQuota int, reason string) {
	recalculateTaskQuota(ctx, task, actualQuota, reason, false)
}

func appendTaskBillingModeFromReason(other map[string]interface{}, billingContext *model.TaskBillingContext, reason string) {
	if other == nil || billingContext == nil || billingContext.BillingMode == "" {
		return
	}
	if _, exists := other["billing_mode"]; exists {
		return
	}
	if !strings.Contains(reason, "billing_mode="+string(billingContext.BillingMode)) {
		return
	}
	other["billing_mode"] = string(billingContext.BillingMode)
}

type taskTokenBillingRatios struct {
	modelRatio         float64
	groupRatio         float64
	otherRatios        map[string]float64
	billingMode        types.BillingMode
	fromBillingContext bool
}

func resolveTaskTokenBillingRatios(task *model.Task) (taskTokenBillingRatios, bool) {
	if task == nil {
		return taskTokenBillingRatios{}, false
	}
	if bc := task.PrivateData.BillingContext; bc != nil &&
		bc.BillingMode == types.BillingModePerToken {
		return taskTokenBillingRatios{
			modelRatio:         bc.ModelRatio,
			groupRatio:         bc.GroupRatio,
			otherRatios:        bc.OtherRatios,
			billingMode:        bc.BillingMode,
			fromBillingContext: true,
		}, true
	}

	modelName := taskModelName(task)
	modelRatio, hasRatioSetting, _ := ratio_setting.GetModelRatio(modelName)
	if !hasRatioSetting || modelRatio <= 0 {
		return taskTokenBillingRatios{}, false
	}

	group := task.Group
	if group == "" {
		user, err := model.GetUserById(task.UserId, false)
		if err == nil {
			group = user.Group
		}
	}
	if group == "" {
		return taskTokenBillingRatios{}, false
	}

	groupRatio := ratio_setting.GetGroupRatio(group)
	if userGroupRatio, hasUserGroupRatio := ratio_setting.GetGroupGroupRatio(group, group); hasUserGroupRatio {
		groupRatio = userGroupRatio
	}

	return taskTokenBillingRatios{
		modelRatio:  modelRatio,
		groupRatio:  groupRatio,
		billingMode: types.BillingModePerToken,
	}, true
}

func calculateTaskQuotaByTokenRatios(totalTokens int, ratios taskTokenBillingRatios) int {
	quota := float64(totalTokens) * ratios.modelRatio * ratios.groupRatio
	for _, ratio := range ratios.otherRatios {
		if ratio > 0 && ratio != 1.0 {
			quota *= ratio
		}
	}
	return int(quota)
}

func RecalculateTaskQuotaByTokens(ctx context.Context, task *model.Task, totalTokens int) {
	if totalTokens <= 0 {
		return
	}

	ratios, ok := resolveTaskTokenBillingRatios(task)
	if !ok {
		return
	}

	resolvedActualQuota := calculateTaskQuotaByTokenRatios(totalTokens, ratios)
	resolvedReason := fmt.Sprintf("token recalculation: tokens=%d, modelRatio=%.2f, groupRatio=%.2f", totalTokens, ratios.modelRatio, ratios.groupRatio)
	if ratios.fromBillingContext {
		resolvedReason = fmt.Sprintf("%s, billing_mode=%s", resolvedReason, ratios.billingMode)
	}
	recalculateTaskQuota(ctx, task, resolvedActualQuota, resolvedReason, true)
}
