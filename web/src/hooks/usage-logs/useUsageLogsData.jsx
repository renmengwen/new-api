/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@douyinfe/semi-ui';
import {
  API,
  MAX_EXCEL_EXPORT_ROWS,
  getTodayStartTimestamp,
  isAdmin,
  isAgentUser,
  showError,
  showInfo,
  showSuccess,
  timestamp2string,
  renderQuota,
  renderNumber,
  convertUSDToCurrency,
  getLogOther,
  copy,
  renderClaudeLogContent,
  renderLogContent,
  renderAudioModelPrice,
  renderClaudeModelPrice,
  renderModelPrice,
} from '../../helpers';
import {
  createSmartExportStatusNotifier,
  runSmartExport,
} from '../../helpers/smartExport';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';
import { useUserPermissions } from '../common/useUserPermissions';
import ParamOverrideEntry from '../../components/table/usage-logs/components/ParamOverrideEntry';
import { getLogsColumns } from '../../components/table/usage-logs/UsageLogsColumnDefs';
import {
  buildUsageLogExportRequest,
  createUsageLogCommittedQuery,
  getVisibleUsageLogColumnKeys,
} from './exportState';

const getAdvancedRuleSnapshot = (other) => {
  const snapshot = other?.advanced_rule;
  return snapshot && typeof snapshot === 'object' ? snapshot : null;
};

const getAdvancedPricingContext = (other) => {
  const context = other?.advanced_pricing_context;
  return context && typeof context === 'object' ? context : null;
};

const getAdvancedBillingUnit = (snapshot, other) => {
  const context = getAdvancedPricingContext(other);
  return String(context?.billing_unit || snapshot?.billing_unit || '').trim();
};

const getAdvancedRuleTypeLabel = (t, ruleType) => {
  switch (ruleType) {
    case 'text_segment':
      return t('文本分段规则');
    case 'media_task':
      return t('媒体任务规则');
    default:
      return ruleType || t('高级规则');
  }
};

const buildAdvancedConditionLines = (t, snapshot) => {
  const lines = [];
  if (snapshot?.match_summary) {
    lines.push(`${t('命中条件')}：${snapshot.match_summary}`);
  }
  if (Array.isArray(snapshot?.condition_tags) && snapshot.condition_tags.length > 0) {
    lines.push(`${t('条件标签')}：${snapshot.condition_tags.join(', ')}`);
  }
  return lines;
};

const buildAdvancedPricingContextLines = (other, snapshot) => {
  const context = getAdvancedPricingContext(other);
  if (!context) {
    return [];
  }

  const lines = [];
  const pushLine = (label, value) => {
    if (value === undefined || value === null || value === '') {
      return;
    }
    lines.push(`${label}: ${value}`);
  };

  pushLine('billing_unit', context?.billing_unit || snapshot?.billing_unit);
  pushLine('image_size_tier', context?.image_size_tier || snapshot?.image_size_tier);
  pushLine('tool_usage_type', context?.tool_usage_type || snapshot?.tool_usage_type);
  pushLine('tool_usage_count', context?.tool_usage_count);
  pushLine('image_count', context?.image_count);
  pushLine('live_duration_secs', context?.live_duration_secs);
  pushLine('free_quota', context?.free_quota);
  pushLine('overage_threshold', context?.overage_threshold);

  return lines;
};

const buildAdvancedPriceSummary = (t, other, snapshot) => {
  const priceSnapshot = snapshot?.price_snapshot || {};
  const billingUnit = getAdvancedBillingUnit(snapshot, other);
  const summary = [];

  switch (billingUnit) {
    case 'per_second':
    case 'per_image':
    case 'per_1000_calls': {
      const unitPrice = resolveAdvancedNonTokenUnitPrice(snapshot, other);
      if (unitPrice > 0) {
        return `${t('单价摘要')}：${renderAdvancedPrice(unitPrice)} / ${billingUnit}`;
      }
      return `${t('单价摘要')}：${t('未记录')}`;
    }
    default:
      break;
  }
  if (priceSnapshot.input_price !== undefined) {
    summary.push(`${t('输入')} ${renderAdvancedPrice(priceSnapshot.input_price)} / 1M tokens`);
  }
  if (priceSnapshot.output_price !== undefined) {
    summary.push(`${t('输出')} ${renderAdvancedPrice(priceSnapshot.output_price)} / 1M tokens`);
  }
  if (priceSnapshot.cache_read_price !== undefined) {
    summary.push(`${t('缓存读取')} ${renderAdvancedPrice(priceSnapshot.cache_read_price)} / 1M tokens`);
  }
  if (priceSnapshot.cache_create_price !== undefined) {
    summary.push(`${t('缓存创建')} ${renderAdvancedPrice(priceSnapshot.cache_create_price)} / 1M tokens`);
  }
  if (summary.length === 0 && other?.model_price !== undefined) {
    summary.push(`${t('单价')} ${renderAdvancedPrice(other.model_price)} / 1M tokens`);
  }
  if (summary.length === 0) {
    return `${t('单价摘要')}：${t('未记录')}`;
  }
  return `${t('单价摘要')}：${summary.join('；')}`;
};

const buildAdvancedBillingBasis = (t, log, other, snapshot) => {
  const thresholdSnapshot = snapshot?.threshold_snapshot || {};
  const lines = [];
  if ((other?.advanced_rule_type || snapshot?.rule_type) === 'media_task') {
    lines.push(`${t('实际计费依据')}：${t('按任务 usage.total_tokens 与命中的高级规则快照结算')}`);
    if (snapshot?.task_type) {
      lines.push(`${t('任务类型')}：${snapshot.task_type}`);
    }
    if (thresholdSnapshot.min_tokens !== undefined) {
      lines.push(`${t('最低 token 阈值')}：${renderNumber(thresholdSnapshot.min_tokens)}`);
    }
    return lines;
  }

  lines.push(`${t('实际计费依据')}：${t('按请求 token 与命中的高级规则快照结算')}`);
  if (log) {
    lines.push(
      `${t('本次用量')}：${t('输入')} ${renderNumber(log.prompt_tokens || 0)} tokens，${t('输出')} ${renderNumber(log.completion_tokens || 0)} tokens`,
    );
  }
  return lines;
};

const renderAdvancedBillingDetailsBase = (t, other) => {
  const snapshot = getAdvancedRuleSnapshot(other);
  const lines = [
    t('高级规则计费'),
    `${t('规则类型')}：${getAdvancedRuleTypeLabel(
      t,
      other?.advanced_rule_type || snapshot?.rule_type,
    )}`,
    ...buildAdvancedConditionLines(t, snapshot),
    ...buildAdvancedPricingContextLines(other, snapshot),
    buildAdvancedPriceSummary(t, other, snapshot),
    ...buildAdvancedBillingBasis(t, null, other, snapshot),
  ];
  return lines.filter(Boolean).join('\n');
};

const renderAdvancedBillingProcessBase = (t, log, other) => {
  const snapshot = getAdvancedRuleSnapshot(other);
  const lines = [
    t('高级规则计费'),
    `${t('规则类型')}：${getAdvancedRuleTypeLabel(
      t,
      other?.advanced_rule_type || snapshot?.rule_type,
    )}`,
    ...buildAdvancedConditionLines(t, snapshot),
    ...buildAdvancedPricingContextLines(other, snapshot),
    buildAdvancedPriceSummary(t, other, snapshot),
    ...buildAdvancedBillingBasis(t, log, other, snapshot),
    `${t('分组倍率')}：${renderNumber(other?.group_ratio || 1)}x`,
  ];
  return lines.filter(Boolean).join('\n');
};

const toAdvancedNumber = (value) => {
  const num = Number(value);
  return Number.isFinite(num) ? num : null;
};

const renderAdvancedPrice = (usdAmount, digits = 6) => {
  const amount = toAdvancedNumber(usdAmount);
  return convertUSDToCurrency(amount === null ? 0 : amount, digits);
};

const getAdvancedGroupRatio = (other) => {
  return toAdvancedNumber(other?.group_ratio) ?? 1;
};

const getAdvancedLegacyInputPrice = (other) => {
  const modelRatio = toAdvancedNumber(other?.model_ratio);
  return modelRatio === null ? null : modelRatio * 2;
};

const getAdvancedPriceSnapshot = (snapshot) => snapshot?.price_snapshot || {};

const getAdvancedThresholdSnapshot = (snapshot) => snapshot?.threshold_snapshot || {};

const getAdvancedLegacyMediaUnitPrice = (snapshot) => {
  const matchSummary = snapshot?.match_summary;
  if (typeof matchSummary !== 'string' || matchSummary.length === 0) {
    return null;
  }
  const matched = matchSummary.match(/(?:^|,\s*)unit_price=([0-9]+(?:\.[0-9]+)?)/);
  return matched ? toAdvancedNumber(matched[1]) : null;
};

const resolveAdvancedNonTokenUnitPrice = (snapshot, other) => {
  const billingUnit = getAdvancedBillingUnit(snapshot, other);
  const priceSnapshot = getAdvancedPriceSnapshot(snapshot);

  if (billingUnit === 'per_1000_calls') {
    return (
      toAdvancedNumber(priceSnapshot.tool_overage_price) ??
      toAdvancedNumber(priceSnapshot.input_price) ??
      getAdvancedLegacyMediaUnitPrice(snapshot) ??
      0
    );
  }

  const snapshotTotal = [
    priceSnapshot.input_price,
    priceSnapshot.output_price,
    priceSnapshot.cache_read_price,
    priceSnapshot.cache_create_price,
    priceSnapshot.cache_storage_price,
  ].reduce((total, price) => total + (toAdvancedNumber(price) ?? 0), 0);

  return snapshotTotal || getAdvancedLegacyMediaUnitPrice(snapshot) || 0;
};

const getAdvancedActualUsageTokens = (log, other) => {
  const explicitUsage =
    toAdvancedNumber(other?.usage_total_tokens) ??
    toAdvancedNumber(other?.total_tokens) ??
    toAdvancedNumber(other?.actual_total_tokens);
  if (explicitUsage !== null) {
    return {
      actualTokens: explicitUsage,
      usageSource: 'usage.total_tokens',
    };
  }

  const promptTokens = toAdvancedNumber(log?.prompt_tokens) ?? 0;
  const completionTokens = toAdvancedNumber(log?.completion_tokens) ?? 0;
  const logTotalTokens = promptTokens + completionTokens;
  if (logTotalTokens > 0) {
    return {
      actualTokens: logTotalTokens,
      usageSource: '当前日志 token 合计',
    };
  }

  return {
    actualTokens: 0,
    usageSource: '',
  };
};

const buildAdvancedExtraChargeItems = (t, other, snapshot) => {
  const priceSnapshot = getAdvancedPriceSnapshot(snapshot);
  const baseInputPrice =
    toAdvancedNumber(priceSnapshot.input_price) ??
    getAdvancedLegacyInputPrice(other) ??
    0;
  const audioRatio = toAdvancedNumber(other?.audio_ratio);
  const audioCompletionRatio = toAdvancedNumber(other?.audio_completion_ratio);
  const derivedAudioInputPrice =
    toAdvancedNumber(other?.audio_input_price) ??
    (audioRatio !== null ? baseInputPrice * audioRatio : null);
  const derivedAudioOutputPrice =
    audioRatio !== null && audioCompletionRatio !== null
      ? baseInputPrice * audioRatio * audioCompletionRatio
      : null;

  const items = [];
  const pushTokenItem = (label, tokenCount, unitPrice) => {
    const tokens = toAdvancedNumber(tokenCount);
    const price = toAdvancedNumber(unitPrice);
    if (!tokens || !price) {
      return;
    }
    items.push({
      label,
      formulaPart: `${label} ${renderNumber(tokens)} tokens / 1M tokens * ${renderAdvancedPrice(price)}`,
      amount: (tokens / 1000000) * price,
      summary: `${label} ${renderAdvancedPrice(price)} / 1M tokens`,
    });
  };
  const pushCountItem = (label, callCount, unitPrice) => {
    const count = toAdvancedNumber(callCount);
    const price = toAdvancedNumber(unitPrice);
    if (!count || !price) {
      return;
    }
    items.push({
      label,
      formulaPart: `${label} ${renderNumber(count)} 次 * ${renderAdvancedPrice(price)}`,
      amount: count * price,
      summary: `${label} ${renderNumber(count)} 次 × ${renderAdvancedPrice(price)}`,
    });
  };

  pushTokenItem(
    t('缓存读取'),
    other?.cache_tokens,
    priceSnapshot.cache_read_price ?? (other?.cache_ratio ? baseInputPrice * other.cache_ratio : null),
  );
  pushTokenItem(
    t('缓存创建'),
    other?.cache_creation_tokens,
    priceSnapshot.cache_create_price ??
      (other?.cache_creation_ratio ? baseInputPrice * other.cache_creation_ratio : null),
  );
  pushTokenItem(
    t('缓存创建(5分钟)'),
    other?.cache_creation_tokens_5m,
    other?.cache_creation_ratio_5m ? baseInputPrice * other.cache_creation_ratio_5m : null,
  );
  pushTokenItem(
    t('缓存创建(1小时)'),
    other?.cache_creation_tokens_1h,
    other?.cache_creation_ratio_1h ? baseInputPrice * other.cache_creation_ratio_1h : null,
  );
  pushCountItem(t('联网搜索'), other?.web_search_call_count, other?.web_search_price);
  pushCountItem(t('文件搜索'), other?.file_search_call_count, other?.file_search_price);
  pushCountItem(t('图片生成'), other?.image_generation_call, other?.image_generation_call_price);
  pushTokenItem(
    t('音频输入'),
    other?.audio_input_token_count ?? other?.audio_input,
    derivedAudioInputPrice,
  );
  pushTokenItem(t('音频输出'), other?.audio_output, derivedAudioOutputPrice);

  return items;
};

const buildAdvancedExtraChargeLines = (t, log, other, snapshot) => {
  const items = buildAdvancedExtraChargeItems(t, other, snapshot);
  const lines = [];
  if (other?.ws || other?.audio) {
    if (other?.text_input > 0) {
      lines.push(`${t('文字输入')}：${renderNumber(other.text_input)}`);
    }
    if (other?.text_output > 0) {
      lines.push(`${t('文字输出')}：${renderNumber(other.text_output)}`);
    }
    if (other?.audio_input > 0) {
      lines.push(`${t('音频输入')}：${renderNumber(other.audio_input)}`);
    }
    if (other?.audio_output > 0) {
      lines.push(`${t('音频输出')}：${renderNumber(other.audio_output)}`);
    }
  }
  items.forEach((item) => {
    lines.push(`${t('附加收费')}：${item.summary}`);
  });
  return lines;
};

const buildAdvancedNonTokenFormula = (
  t,
  other,
  snapshot,
  unitPrice,
  groupRatio,
  multiplier = 1,
) => {
  const context = getAdvancedPricingContext(other);
  const billingUnit = getAdvancedBillingUnit(snapshot, other);
  const effectiveUnitPrice = unitPrice * multiplier;

  switch (billingUnit) {
    case 'per_second': {
      const liveDurationSecs = toAdvancedNumber(context?.live_duration_secs) ?? 0;
      const totalAmount = liveDurationSecs * effectiveUnitPrice * groupRatio;
      return [
        `${t('实际计费用量')}：${renderNumber(liveDurationSecs)} per_second`,
        `${t('最终计费公式')}：${renderNumber(liveDurationSecs)} per_second * ${renderAdvancedPrice(unitPrice)} * ${renderNumber(multiplier)} * ${t('分组倍率')} ${renderNumber(groupRatio)} = ${renderAdvancedPrice(totalAmount)}`,
      ];
    }
    case 'per_image': {
      const imageCount = toAdvancedNumber(context?.image_count) ?? 0;
      const totalAmount = imageCount * effectiveUnitPrice * groupRatio;
      return [
        `${t('实际计费用量')}：${renderNumber(imageCount)} per_image`,
        `${t('最终计费公式')}：${renderNumber(imageCount)} per_image * ${renderAdvancedPrice(unitPrice)} * ${renderNumber(multiplier)} * ${t('分组倍率')} ${renderNumber(groupRatio)} = ${renderAdvancedPrice(totalAmount)}`,
      ];
    }
    case 'per_1000_calls': {
      const toolUsageCount = toAdvancedNumber(context?.tool_usage_count) ?? 0;
      const freeQuota =
        toAdvancedNumber(context?.free_quota) ??
        toAdvancedNumber(snapshot?.threshold_snapshot?.free_quota) ??
        0;
      const overageThreshold =
        toAdvancedNumber(context?.overage_threshold) ??
        toAdvancedNumber(snapshot?.threshold_snapshot?.overage_threshold) ??
        1000;
      const billableCalls = Math.max(toolUsageCount - freeQuota, 0);
      const effectiveUsage =
        overageThreshold > 0 ? billableCalls / overageThreshold : 0;
      const totalAmount = effectiveUsage * effectiveUnitPrice * groupRatio;
      return [
        `${t('实际计费用量')}：tool_usage_count ${renderNumber(toolUsageCount)}`,
        `${t('生效计费用量')}：max(${renderNumber(toolUsageCount)} - ${renderNumber(freeQuota)}, 0) / ${renderNumber(overageThreshold)} per_1000_calls = ${renderNumber(effectiveUsage)}`,
        `${t('最终计费公式')}：${renderNumber(effectiveUsage)} per_1000_calls * ${renderAdvancedPrice(unitPrice)} * ${renderNumber(multiplier)} * ${t('分组倍率')} ${renderNumber(groupRatio)} = ${renderAdvancedPrice(totalAmount)}`,
      ];
    }
    default:
      return null;
  }
};

const buildAdvancedTextSegmentFormula = (t, log, other, snapshot) => {
  const priceSnapshot = getAdvancedPriceSnapshot(snapshot);
  const groupRatio = getAdvancedGroupRatio(other);
  const inputTokens = toAdvancedNumber(log?.prompt_tokens) ?? 0;
  const outputTokens = toAdvancedNumber(log?.completion_tokens) ?? 0;
  const inputPrice =
    toAdvancedNumber(priceSnapshot.input_price) ??
    getAdvancedLegacyInputPrice(other) ??
    0;
  const outputPrice =
    toAdvancedNumber(priceSnapshot.output_price) ??
    (inputPrice && toAdvancedNumber(other?.completion_ratio) !== null
      ? inputPrice * Number(other.completion_ratio)
      : 0);
  const extraItems = buildAdvancedExtraChargeItems(t, other, snapshot);
  const billingUnit = getAdvancedBillingUnit(snapshot, other);
  const multiplier =
    toAdvancedNumber(snapshot?.threshold_snapshot?.draft_coefficient) ??
    toAdvancedNumber(other?.draft_coefficient) ??
    1;
  const nonTokenUnitPrice = resolveAdvancedNonTokenUnitPrice(snapshot, other);
  const nonTokenFormula = buildAdvancedNonTokenFormula(
    t,
    other,
    snapshot,
    nonTokenUnitPrice,
    groupRatio,
    multiplier,
  );
  if (nonTokenFormula) {
    return nonTokenFormula;
  }
  const settledQuota = toAdvancedNumber(other?.advanced_charged_quota);
  const settledQuotaPerUnit = toAdvancedNumber(other?.quota_per_unit);
  if (settledQuota !== null && settledQuotaPerUnit !== null && settledQuotaPerUnit > 0) {
    const settledAmount = settledQuota / settledQuotaPerUnit;
    return [
      `${t('最终计费公式')}：${renderNumber(settledQuota)} quota / ${renderNumber(settledQuotaPerUnit)} = ${renderAdvancedPrice(settledAmount)}`,
    ];
  }

  const baseParts = [];
  let baseAmount = 0;

  if (inputPrice > 0) {
    baseParts.push(`${t('输入')} ${renderNumber(inputTokens)} tokens / 1M tokens * ${renderAdvancedPrice(inputPrice)}`);
    baseAmount += (inputTokens / 1000000) * inputPrice;
  }
  if (outputPrice > 0) {
    baseParts.push(`${t('输出')} ${renderNumber(outputTokens)} tokens / 1M tokens * ${renderAdvancedPrice(outputPrice)}`);
    baseAmount += (outputTokens / 1000000) * outputPrice;
  }

  extraItems.forEach((item) => {
    baseParts.push(item.formulaPart);
    baseAmount += item.amount;
  });

  const formula = baseParts.length > 0 ? baseParts.join(' + ') : t('暂无可展示的高级计费公式');
  return [
    `${t('本次用量')}：${t('输入')} ${renderNumber(inputTokens)} tokens，${t('输出')} ${renderNumber(outputTokens)} tokens`,
    `${t('最终计费公式')}：(${formula}) * ${t('分组倍率')} ${renderNumber(groupRatio)} = ${renderAdvancedPrice(
      baseAmount * groupRatio,
    )}`,
  ];
};

const resolveAdvancedUnitPrice = (snapshot, other) => {
  const priceSnapshot = getAdvancedPriceSnapshot(snapshot);
  return (
    toAdvancedNumber(priceSnapshot.input_price) ??
    toAdvancedNumber(priceSnapshot.output_price) ??
    getAdvancedLegacyMediaUnitPrice(snapshot) ??
    toAdvancedNumber(other?.model_price) ??
    getAdvancedLegacyInputPrice(other) ??
    0
  );
};

const buildAdvancedMediaTaskFormula = (t, log, other, snapshot) => {
  const thresholdSnapshot = getAdvancedThresholdSnapshot(snapshot);
  const { actualTokens, usageSource } = getAdvancedActualUsageTokens(log, other);
  const minTokens = toAdvancedNumber(thresholdSnapshot.min_tokens) ?? 0;
  const effectiveTokens = Math.max(actualTokens, minTokens);
  const unitPrice = resolveAdvancedUnitPrice(snapshot, other);
  const groupRatio = getAdvancedGroupRatio(other);
  const actualLabel = usageSource ? `${renderNumber(actualTokens)} tokens（${usageSource}）` : t('未记录');

  return [
    `${t('实际计费用量')}：${actualLabel}`,
    `${t('最低 token 阈值')}：${renderNumber(minTokens)} tokens`,
    `${t('生效计费用量')}：${renderNumber(effectiveTokens)} tokens（${t(
      '取实际计费用量与最低阈值较大值',
    )}）`,
    `${t('最终计费公式')}：${renderNumber(
      effectiveTokens,
    )} tokens / 1M tokens * ${renderAdvancedPrice(unitPrice)} * ${t('分组倍率')} ${renderNumber(
      groupRatio,
    )} = ${renderAdvancedPrice(((effectiveTokens / 1000000) * unitPrice) * groupRatio)}`,
  ];
};

const renderAdvancedBillingDetails = (t, other) => {
  const snapshot = getAdvancedRuleSnapshot(other);
  if (!snapshot) {
    return renderAdvancedBillingDetailsBase(t, other);
  }
  const lines = [
    t('高级规则计费'),
    `${t('规则类型')}：${getAdvancedRuleTypeLabel(t, other?.advanced_rule_type || snapshot?.rule_type)}`,
    ...buildAdvancedConditionLines(t, snapshot),
    buildAdvancedPriceSummary(t, other, snapshot),
    ...buildAdvancedExtraChargeLines(t, null, other, snapshot),
  ];
  return lines.filter(Boolean).join('\n');
};

const renderAdvancedBillingProcess = (t, log, other) => {
  const snapshot = getAdvancedRuleSnapshot(other);
  if (!snapshot) {
    return renderAdvancedBillingProcessBase(t, log, other);
  }
  const ruleType = other?.advanced_rule_type || snapshot?.rule_type;
  const groupRatio = getAdvancedGroupRatio(other);
  const formulaLines =
    ruleType === 'media_task'
      ? buildAdvancedMediaTaskFormula(t, log, other, snapshot)
      : buildAdvancedTextSegmentFormula(t, log, other, snapshot);
  const lines = [
    t('高级规则计费'),
    `${t('规则类型')}：${getAdvancedRuleTypeLabel(t, ruleType)}`,
    ...buildAdvancedConditionLines(t, snapshot),
    buildAdvancedPriceSummary(t, other, snapshot),
    ...buildAdvancedExtraChargeLines(t, log, other, snapshot),
    ...formulaLines,
    `${t('分组倍率')}：${renderNumber(groupRatio)}x`,
  ];
  return lines.filter(Boolean).join('\n');
};

export const useLogsData = () => {
  const { t } = useTranslation();

  // Define column keys for selection
  const COLUMN_KEYS = {
    TIME: 'time',
    CHANNEL: 'channel',
    USERNAME: 'username',
    TOKEN: 'token',
    GROUP: 'group',
    TYPE: 'type',
    MODEL: 'model',
    USE_TIME: 'use_time',
    PROMPT: 'prompt',
    COMPLETION: 'completion',
    COST: 'cost',
    RETRY: 'retry',
    IP: 'ip',
    DETAILS: 'details',
  };

  // Basic state
  const [logs, setLogs] = useState([]);
  const [expandData, setExpandData] = useState({});
  const [showStat, setShowStat] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadingStat, setLoadingStat] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [logCount, setLogCount] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [logType, setLogType] = useState(0);

  // User and admin
  const { loading: permissionLoading, hasActionPermission } =
    useUserPermissions();
  const canReadScopedUsageLogs = hasActionPermission(
    'quota_management',
    'ledger_read',
  );
  const canReadScopedUsageLogSummary = hasActionPermission(
    'quota_management',
    'read_summary',
  );
  const isAdminUser = isAdmin() || (isAgentUser() && canReadScopedUsageLogs);
  const canUseScopedUsageLogStat =
    isAdmin() || (isAgentUser() && canReadScopedUsageLogSummary);
  const canShowUserInfo = isAdmin();
  const canShowChannelAffinityUsageCache = isAdmin();
  const shouldWaitForAgentPermissions =
    isAgentUser() && !isAdmin() && permissionLoading;
  // Role-specific storage key to prevent different roles from overwriting each other
  const STORAGE_KEY = isAdminUser
    ? 'logs-table-columns-admin'
    : 'logs-table-columns-user';
  const BILLING_DISPLAY_MODE_STORAGE_KEY = isAdminUser
    ? 'logs-billing-display-mode-admin'
    : 'logs-billing-display-mode-user';

  // Statistics state
  const [stat, setStat] = useState({
    quota: 0,
    token: 0,
  });

  // Form state
  const [formApi, setFormApi] = useState(null);
  let now = new Date();
  const formInitValues = {
    username: '',
    token_name: '',
    model_name: '',
    channel: '',
    group: '',
    request_id: '',
    dateRange: [
      timestamp2string(getTodayStartTimestamp()),
      timestamp2string(now.getTime() / 1000 + 3600),
    ],
    logType: '0',
  };
  const [committedQuery, setCommittedQuery] = useState(() =>
    createUsageLogCommittedQuery(formInitValues),
  );
  const [listRequestsInFlight, setListRequestsInFlight] = useState(0);
  const isExportReady = listRequestsInFlight === 0;
  const [exportLoading, setExportLoading] = useState(false);

  // Get default column visibility based on user role
  const getDefaultColumnVisibility = () => {
    return {
      [COLUMN_KEYS.TIME]: true,
      [COLUMN_KEYS.CHANNEL]: isAdminUser,
      [COLUMN_KEYS.USERNAME]: isAdminUser,
      [COLUMN_KEYS.TOKEN]: true,
      [COLUMN_KEYS.GROUP]: true,
      [COLUMN_KEYS.TYPE]: true,
      [COLUMN_KEYS.MODEL]: true,
      [COLUMN_KEYS.USE_TIME]: true,
      [COLUMN_KEYS.PROMPT]: true,
      [COLUMN_KEYS.COMPLETION]: true,
      [COLUMN_KEYS.COST]: true,
      [COLUMN_KEYS.RETRY]: isAdminUser,
      [COLUMN_KEYS.IP]: true,
      [COLUMN_KEYS.DETAILS]: true,
    };
  };

  const getInitialVisibleColumns = () => {
    const defaults = getDefaultColumnVisibility();
    const savedColumns = localStorage.getItem(STORAGE_KEY);

    if (!savedColumns) {
      return defaults;
    }

    try {
      const parsed = JSON.parse(savedColumns);
      const merged = { ...defaults, ...parsed };

      if (!isAdminUser) {
        merged[COLUMN_KEYS.CHANNEL] = false;
        merged[COLUMN_KEYS.USERNAME] = false;
        merged[COLUMN_KEYS.RETRY] = false;
      }

      return merged;
    } catch (e) {
      console.error('Failed to parse saved column preferences', e);
      return defaults;
    }
  };

  const getInitialBillingDisplayMode = () => {
    const savedMode = localStorage.getItem(BILLING_DISPLAY_MODE_STORAGE_KEY);
    if (savedMode === 'price' || savedMode === 'ratio') {
      return savedMode;
    }
    return localStorage.getItem('quota_display_type') === 'TOKENS'
      ? 'ratio'
      : 'price';
  };

  // Column visibility state
  const [visibleColumns, setVisibleColumns] = useState(getInitialVisibleColumns);
  const [showColumnSelector, setShowColumnSelector] = useState(false);
  const [billingDisplayMode, setBillingDisplayMode] = useState(
    getInitialBillingDisplayMode,
  );

  // Compact mode
  const [compactMode, setCompactMode] = useTableCompactMode('logs');

  // User info modal state
  const [showUserInfo, setShowUserInfoModal] = useState(false);
  const [userInfoData, setUserInfoData] = useState(null);

  // Channel affinity usage cache stats modal state (admin only)
  const [
    showChannelAffinityUsageCacheModal,
    setShowChannelAffinityUsageCacheModal,
  ] = useState(false);
  const [channelAffinityUsageCacheTarget, setChannelAffinityUsageCacheTarget] =
    useState(null);
  const [showParamOverrideModal, setShowParamOverrideModal] = useState(false);
  const [paramOverrideTarget, setParamOverrideTarget] = useState(null);

  // Initialize default column visibility
  const initDefaultColumns = () => {
    const defaults = getDefaultColumnVisibility();
    setVisibleColumns(defaults);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(defaults));
  };

  // Handle column visibility change
  const handleColumnVisibilityChange = (columnKey, checked) => {
    const updatedColumns = { ...visibleColumns, [columnKey]: checked };
    setVisibleColumns(updatedColumns);
  };

  // Handle "Select All" checkbox
  const handleSelectAll = (checked) => {
    const allKeys = Object.keys(COLUMN_KEYS).map((key) => COLUMN_KEYS[key]);
    const updatedColumns = {};

    allKeys.forEach((key) => {
      if (
        (key === COLUMN_KEYS.CHANNEL ||
          key === COLUMN_KEYS.USERNAME ||
          key === COLUMN_KEYS.RETRY) &&
        !isAdminUser
      ) {
        updatedColumns[key] = false;
      } else {
        updatedColumns[key] = checked;
      }
    });

    setVisibleColumns(updatedColumns);
  };

  // Persist column settings to the role-specific STORAGE_KEY
  useEffect(() => {
    if (Object.keys(visibleColumns).length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(visibleColumns));
    }
  }, [visibleColumns]);

  useEffect(() => {
    localStorage.setItem(BILLING_DISPLAY_MODE_STORAGE_KEY, billingDisplayMode);
  }, [BILLING_DISPLAY_MODE_STORAGE_KEY, billingDisplayMode]);

  // 获取表单值的辅助函数，确保所有值都是字符串
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : formInitValues;
    return createUsageLogCommittedQuery(formValues, formInitValues.dateRange);
  };

  // Statistics functions
  const getLogSelfStat = async (query = committedQuery) => {
    const {
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      group,
      logType: currentLogType,
    } = query;
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let url = `/api/log/self/stat?type=${currentLogType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${group}`;
    url = encodeURI(url);
    let res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const getLogStat = async (query = committedQuery) => {
    const {
      username,
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      channel,
      group,
      logType: currentLogType,
    } = query;
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let url = `/api/log/stat?type=${currentLogType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${group}`;
    url = encodeURI(url);
    let res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const handleEyeClick = async (query = committedQuery) => {
    if (loadingStat) {
      return;
    }
    setLoadingStat(true);
    try {
      if (canUseScopedUsageLogStat) {
        await getLogStat(query);
      } else {
        await getLogSelfStat(query);
      }
      setShowStat(true);
    } catch (error) {
      showError(error);
    } finally {
      setLoadingStat(false);
    }
  };

  // User info function
  const showUserInfoFunc = async (userId) => {
    if (!canShowUserInfo) {
      return;
    }
    const res = await API.get(`/api/user/${userId}`);
    const { success, message, data } = res.data;
    if (success) {
      setUserInfoData(data);
      setShowUserInfoModal(true);
    } else {
      showError(message);
    }
  };

  const openChannelAffinityUsageCacheModal = (affinity) => {
    if (!canShowChannelAffinityUsageCache) {
      return;
    }
    const a = affinity || {};
    setChannelAffinityUsageCacheTarget({
      rule_name: a.rule_name || a.reason || '',
      using_group: a.using_group || '',
      key_hint: a.key_hint || '',
      key_fp: a.key_fp || '',
    });
    setShowChannelAffinityUsageCacheModal(true);
  };

  const openParamOverrideModal = (log, other) => {
    const lines = Array.isArray(other?.po) ? other.po.filter(Boolean) : [];
    if (lines.length === 0) {
      return;
    }
    setParamOverrideTarget({
      lines,
      modelName: log?.model_name || '',
      requestId: log?.request_id || '',
      requestPath: other?.request_path || '',
    });
    setShowParamOverrideModal(true);
  };

  // Format logs data
  const setLogsFormat = (logs) => {
    const requestConversionDisplayValue = (conversionChain) => {
      const chain = Array.isArray(conversionChain)
        ? conversionChain.filter(Boolean)
        : [];
      if (chain.length <= 1) {
        return t('原生格式');
      }
      return `${chain.join(' -> ')}`;
    };

    let expandDatesLocal = {};
    for (let i = 0; i < logs.length; i++) {
      logs[i].timestamp2string = timestamp2string(logs[i].created_at);
      logs[i].key = logs[i].id;
      let other = getLogOther(logs[i].other);
      let expandDataLocal = [];

      if (isAdminUser && (logs[i].type === 0 || logs[i].type === 2 || logs[i].type === 6)) {
        expandDataLocal.push({
          key: t('渠道信息'),
          value: `${logs[i].channel} - ${logs[i].channel_name || '[未知]'}`,
        });
      }
      if (logs[i].request_id) {
        expandDataLocal.push({
          key: t('Request ID'),
          value: logs[i].request_id,
        });
      }
      if (other?.ws || other?.audio) {
        expandDataLocal.push({
          key: t('语音输入'),
          value: other.audio_input,
        });
        expandDataLocal.push({
          key: t('语音输出'),
          value: other.audio_output,
        });
        expandDataLocal.push({
          key: t('文字输入'),
          value: other.text_input,
        });
        expandDataLocal.push({
          key: t('文字输出'),
          value: other.text_output,
        });
      }
      if (other?.cache_tokens > 0) {
        expandDataLocal.push({
          key: t('缓存 Tokens'),
          value: other.cache_tokens,
        });
      }
      if (other?.cache_creation_tokens > 0) {
        expandDataLocal.push({
          key: t('缓存创建 Tokens'),
          value: other.cache_creation_tokens,
        });
      }
      if (logs[i].type === 2) {
        expandDataLocal.push({
          key: t('日志详情'),
          value: other?.billing_mode === 'advanced'
            ? renderAdvancedBillingDetails(t, other)
            : other?.claude
              ? renderClaudeLogContent(
                other?.model_ratio,
                other.completion_ratio,
                other.model_price,
                other.group_ratio,
                other?.user_group_ratio,
                other.cache_ratio || 1.0,
                other.cache_creation_ratio || 1.0,
                other.cache_creation_tokens_5m || 0,
                other.cache_creation_ratio_5m ||
                  other.cache_creation_ratio ||
                  1.0,
                other.cache_creation_tokens_1h || 0,
                other.cache_creation_ratio_1h ||
                  other.cache_creation_ratio ||
                  1.0,
                billingDisplayMode,
              )
            : renderLogContent(
                other?.model_ratio,
                other.completion_ratio,
                other.model_price,
                other.group_ratio,
                other?.user_group_ratio,
                other.cache_ratio || 1.0,
                false,
                1.0,
                other.web_search || false,
                other.web_search_call_count || 0,
                other.file_search || false,
                other.file_search_call_count || 0,
                billingDisplayMode,
              ),
        });
        if (logs[i]?.content) {
          expandDataLocal.push({
            key: t('其他详情'),
            value: logs[i].content,
          });
        }
        if (isAdminUser && other?.reject_reason) {
          expandDataLocal.push({
            key: t('拦截原因'),
            value: other.reject_reason,
          });
        }
      }
      if (logs[i].type === 2) {
        let modelMapped =
          other?.is_model_mapped &&
          other?.upstream_model_name &&
          other?.upstream_model_name !== '';
        if (modelMapped) {
          expandDataLocal.push({
            key: t('请求并计费模型'),
            value: logs[i].model_name,
          });
          expandDataLocal.push({
            key: t('实际模型'),
            value: other.upstream_model_name,
          });
        }

        const isViolationFeeLog =
          other?.violation_fee === true ||
          Boolean(other?.violation_fee_code) ||
          Boolean(other?.violation_fee_marker);

        let content = '';
        if (!isViolationFeeLog) {
          if (other?.billing_mode === 'advanced') {
            content = renderAdvancedBillingProcess(t, logs[i], other);
          } else if (other?.ws || other?.audio) {
            content = renderAudioModelPrice(
              other?.text_input,
              other?.text_output,
              other?.model_ratio,
              other?.model_price,
              other?.completion_ratio,
              other?.audio_input,
              other?.audio_output,
              other?.audio_ratio,
              other?.audio_completion_ratio,
              other?.group_ratio,
              other?.user_group_ratio,
              other?.cache_tokens || 0,
              other?.cache_ratio || 1.0,
              billingDisplayMode,
            );
          } else if (other?.claude) {
            content = renderClaudeModelPrice(
              logs[i].prompt_tokens,
              logs[i].completion_tokens,
              other.model_ratio,
              other.model_price,
              other.completion_ratio,
              other.group_ratio,
              other?.user_group_ratio,
              other.cache_tokens || 0,
              other.cache_ratio || 1.0,
              other.cache_creation_tokens || 0,
              other.cache_creation_ratio || 1.0,
              other.cache_creation_tokens_5m || 0,
              other.cache_creation_ratio_5m ||
                other.cache_creation_ratio ||
                1.0,
              other.cache_creation_tokens_1h || 0,
              other.cache_creation_ratio_1h ||
                other.cache_creation_ratio ||
                1.0,
              billingDisplayMode,
            );
          } else {
            content = renderModelPrice(
              logs[i].prompt_tokens,
              logs[i].completion_tokens,
              other?.model_ratio,
              other?.model_price,
              other?.completion_ratio,
              other?.group_ratio,
              other?.user_group_ratio,
              other?.cache_tokens || 0,
              other?.cache_ratio || 1.0,
              other?.image || false,
              other?.image_ratio || 0,
              other?.image_output || 0,
              other?.web_search || false,
              other?.web_search_call_count || 0,
              other?.web_search_price || 0,
              other?.file_search || false,
              other?.file_search_call_count || 0,
              other?.file_search_price || 0,
              other?.audio_input_seperate_price || false,
              other?.audio_input_token_count || 0,
              other?.audio_input_price || 0,
              other?.image_generation_call || false,
              other?.image_generation_call_price || 0,
              billingDisplayMode,
            );
          }
          expandDataLocal.push({
            key: t('计费过程'),
            value: content,
          });
        }
        if (other?.reasoning_effort) {
          expandDataLocal.push({
            key: t('Reasoning Effort'),
            value: other.reasoning_effort,
          });
        }
      }
      if (logs[i].type === 6) {
        if (other?.task_id) {
          expandDataLocal.push({
            key: t('任务ID'),
            value: other.task_id,
          });
        }
        if (other?.reason) {
          expandDataLocal.push({
            key: t('失败原因'),
            value: (
              <div style={{ maxWidth: 600, whiteSpace: 'normal', wordBreak: 'break-word', lineHeight: 1.6 }}>
                {other.reason}
              </div>
            ),
          });
        }
      }
      if (other?.request_path) {
        expandDataLocal.push({
          key: t('请求路径'),
          value: other.request_path,
        });
      }
      if (isAdminUser && other?.stream_status) {
        const ss = other.stream_status;
        const isOk = ss.status === 'ok';
        const statusLabel = isOk ? '✓ ' + t('正常') : '✗ ' + t('异常');
        let streamValue = statusLabel + ' (' + (ss.end_reason || 'unknown') + ')';
        if (ss.error_count > 0) {
          streamValue += ` [${t('软错误')}: ${ss.error_count}]`;
        }
        if (ss.end_error) {
          streamValue += ` - ${ss.end_error}`;
        }
        expandDataLocal.push({
          key: t('流状态'),
          value: streamValue,
        });
        if (Array.isArray(ss.errors) && ss.errors.length > 0) {
          expandDataLocal.push({
            key: t('流错误详情'),
            value: (
              <div style={{ maxWidth: 600, whiteSpace: 'pre-line', wordBreak: 'break-word', lineHeight: 1.6 }}>
                {ss.errors.join('\n')}
              </div>
            ),
          });
        }
      }
      if (Array.isArray(other?.po) && other.po.length > 0) {
        expandDataLocal.push({
          key: t('参数覆盖'),
          value: (
            <ParamOverrideEntry
              count={other.po.length}
              t={t}
              onOpen={(event) => {
                event.stopPropagation();
                openParamOverrideModal(logs[i], other);
              }}
            />
          ),
        });
      }
      if (other?.billing_source === 'subscription') {
        const planId = other?.subscription_plan_id;
        const planTitle = other?.subscription_plan_title || '';
        const subscriptionId = other?.subscription_id;
        const unit = t('额度');
        const pre = other?.subscription_pre_consumed ?? 0;
        const postDelta = other?.subscription_post_delta ?? 0;
        const finalConsumed = other?.subscription_consumed ?? pre + postDelta;
        const remain = other?.subscription_remain;
        const total = other?.subscription_total;
        // Use multiple Description items to avoid an overlong single line.
        if (planId) {
          expandDataLocal.push({
            key: t('订阅套餐'),
            value: `#${planId} ${planTitle}`.trim(),
          });
        }
        if (subscriptionId) {
          expandDataLocal.push({
            key: t('订阅实例'),
            value: `#${subscriptionId}`,
          });
        }
        const settlementLines = [
          `${t('预扣')}：${pre} ${unit}`,
          `${t('结算差额')}：${postDelta > 0 ? '+' : ''}${postDelta} ${unit}`,
          `${t('最终抵扣')}：${finalConsumed} ${unit}`,
        ]
          .filter(Boolean)
          .join('\n');
        expandDataLocal.push({
          key: t('订阅结算'),
          value: (
            <div style={{ whiteSpace: 'pre-line' }}>{settlementLines}</div>
          ),
        });
        if (remain !== undefined && total !== undefined) {
          expandDataLocal.push({
            key: t('订阅剩余'),
            value: `${remain}/${total} ${unit}`,
          });
        }
        expandDataLocal.push({
          key: t('订阅说明'),
          value: t(
            'token 会按倍率换算成“额度/次数”，请求结束后再做差额结算（补扣/返还）。',
          ),
        });
      }
      if (isAdminUser && logs[i].type !== 6) {
        expandDataLocal.push({
          key: t('请求转换'),
          value: requestConversionDisplayValue(other?.request_conversion),
        });
      }
      if (isAdminUser && logs[i].type !== 6) {
        let localCountMode = '';
        if (other?.admin_info?.local_count_tokens) {
          localCountMode = t('本地计费');
        } else {
          localCountMode = t('上游返回');
        }
        expandDataLocal.push({
          key: t('计费模式'),
          value: localCountMode,
        });
      }
      expandDatesLocal[logs[i].key] = expandDataLocal;
    }

    setExpandData(expandDatesLocal);
    setLogs(logs);
  };

  // Load logs function
  const loadLogs = async (startIdx, pageSize, query = committedQuery) => {
    setLoading(true);
    setListRequestsInFlight((count) => count + 1);

    let url = '';
    const {
      username,
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      channel,
      group,
      request_id,
      logType: currentLogType,
    } = query;

    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    if (isAdminUser) {
      url = `/api/log/?p=${startIdx}&page_size=${pageSize}&type=${currentLogType}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${group}&request_id=${request_id}`;
    } else {
      url = `/api/log/self/?p=${startIdx}&page_size=${pageSize}&type=${currentLogType}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${group}&request_id=${request_id}`;
    }
    url = encodeURI(url);
    try {
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items;
        setActivePage(data.page);
        setPageSize(data.page_size);
        setLogCount(data.total);

        setLogsFormat(newPageData);
        return true;
      }

      showError(message);
      return false;
    } finally {
      setLoading(false);
      setListRequestsInFlight((count) => Math.max(0, count - 1));
    }
  };

  // Page handlers
  const handlePageChange = (page) => {
    setActivePage(page);
    loadLogs(page, pageSize, committedQuery).then((r) => {});
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    loadLogs(1, size, committedQuery)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  // Refresh function
  const refresh = async () => {
    const nextCommittedQuery = getFormValues();
    setActivePage(1);
    handleEyeClick(nextCommittedQuery);
    const didRefresh = await loadLogs(1, pageSize, nextCommittedQuery);
    if (didRefresh) {
      setCommittedQuery(nextCommittedQuery);
    }
  };

  // Copy text function
  const copyText = async (e, text) => {
    e.stopPropagation();
    if (await copy(text)) {
      showSuccess('已复制：' + text);
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  const getExportColumnKeys = () => {
    const allColumns = getLogsColumns({
      t,
      COLUMN_KEYS,
      copyText,
      showUserInfoFunc,
      openChannelAffinityUsageCacheModal,
      canShowChannelAffinityUsageCache,
      isAdminUser,
      billingDisplayMode,
    });

    return getVisibleUsageLogColumnKeys({
      allColumns,
      visibleColumns,
    });
  };

  const runExport = async () => {
    setExportLoading(true);
    try {
      await runSmartExport({
        url: isAdminUser ? '/api/log/export-auto' : '/api/log/self/export-auto',
        payload: {
          ...buildUsageLogExportRequest({
            committedQuery,
            visibleColumnKeys: getExportColumnKeys(),
          }),
          limit: logCount,
        },
        fallbackFileName: 'usage-logs.xlsx',
        onAsyncProgress: createSmartExportStatusNotifier({
          t,
          showInfo,
          showSuccess,
        }),
      });
    } catch (error) {
      showError(error);
    } finally {
      setExportLoading(false);
    }
  };

  const handleExport = async () => {
    if (loading || exportLoading || !isExportReady) {
      return;
    }

    if (!logCount) {
      showInfo(t('无可导出数据'));
      return;
    }

    if (logCount > MAX_EXCEL_EXPORT_ROWS) {
      Modal.confirm({
        title: t('导出 Excel'),
        content: t('当前筛选结果较大，导出可能切换为后台生成，是否继续？'),
        okText: t('继续导出'),
        cancelText: t('取消'),
        onOk: runExport,
      });
      return;
    }

    await runExport();
  };

  // Initialize data
  useEffect(() => {
    if (shouldWaitForAgentPermissions) {
      return;
    }
    const localPageSize =
      parseInt(localStorage.getItem('page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadLogs(activePage, localPageSize, committedQuery)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  }, [shouldWaitForAgentPermissions, isAdminUser]);

  // Initialize statistics when formApi is available
  useEffect(() => {
    if (!formApi || shouldWaitForAgentPermissions) {
      return;
    }
    handleEyeClick(committedQuery);
  }, [formApi, shouldWaitForAgentPermissions, canUseScopedUsageLogStat]);

  // Check if any record has expandable content
  const hasExpandableRows = () => {
    return logs.some(
      (log) => expandData[log.key] && expandData[log.key].length > 0,
    );
  };

  return {
    // Basic state
    logs,
    expandData,
    showStat,
    loading,
    loadingStat,
    activePage,
    logCount,
    pageSize,
    logType,
    stat,
    isAdminUser,

    // Form state
    formApi,
    setFormApi,
    formInitValues,
    getFormValues,
    committedQuery,
    isExportReady,

    // Column visibility
    visibleColumns,
    showColumnSelector,
    setShowColumnSelector,
    billingDisplayMode,
    setBillingDisplayMode,
    handleColumnVisibilityChange,
    handleSelectAll,
    initDefaultColumns,
    COLUMN_KEYS,

    // Compact mode
    compactMode,
    setCompactMode,

    // User info modal
    showUserInfo,
    setShowUserInfoModal,
    userInfoData,
    showUserInfoFunc,

    // Channel affinity usage cache stats modal
    showChannelAffinityUsageCacheModal,
    setShowChannelAffinityUsageCacheModal,
    channelAffinityUsageCacheTarget,
    openChannelAffinityUsageCacheModal,
    showParamOverrideModal,
    setShowParamOverrideModal,
    paramOverrideTarget,

    // Functions
    loadLogs,
    handlePageChange,
    handlePageSizeChange,
    refresh,
    copyText,
    handleEyeClick,
    setLogsFormat,
    hasExpandableRows,
    setLogType,
    openParamOverrideModal,
    handleExport,
    exportLoading,

    // Translation
    t,
  };
};
