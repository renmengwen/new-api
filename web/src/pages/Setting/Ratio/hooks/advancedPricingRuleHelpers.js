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

export const ADVANCED_PRICING_MODE_ADVANCED = 'advanced';
export const TEXT_SEGMENT_RULE_TYPE = 'text_segment';
export const MEDIA_TASK_RULE_TYPE = 'media_task';
export const FIXED_BILLING_MODE_PER_TOKEN = 'per_token';
export const FIXED_BILLING_MODE_PER_REQUEST = 'per_request';

const MILLION = 1000000;

export const hasValue = (value) =>
  value !== '' && value !== null && value !== undefined && value !== false;

export const parseOptionJSON = (rawValue) => {
  if (!rawValue || String(rawValue).trim() === '') {
    return {};
  }
  try {
    const parsed = JSON.parse(rawValue);
    return parsed && typeof parsed === 'object' ? parsed : {};
  } catch (error) {
    return {};
  }
};

export const toNullableNumber = (value) => {
  if (!hasValue(value) && value !== 0) {
    return null;
  }
  const num = Number(value);
  return Number.isFinite(num) ? num : null;
};

export const formatNumber = (value) => {
  const num = toNullableNumber(value);
  if (num === null) {
    return '';
  }
  return parseFloat(num.toFixed(12)).toString();
};

const buildRangeSummary = (label, minValue, maxValue) => {
  const min = toNullableNumber(minValue);
  const max = toNullableNumber(maxValue);

  if (min === null && max === null) {
    return '';
  }

  const minText = min === null ? '-∞' : formatNumber(min);
  const maxText = max === null ? '+∞' : formatNumber(max);
  return `${label}[${minText}, ${maxText}]`;
};

export const buildTextSegmentConditionSummary = (rule) => {
  const inputSummary = buildRangeSummary('输入', rule?.inputMin, rule?.inputMax);
  const outputSummary = buildRangeSummary(
    '输出',
    rule?.outputMin,
    rule?.outputMax,
  );
  const summaries = [inputSummary, outputSummary].filter(Boolean);

  return summaries.length > 0 ? summaries.join(' / ') : '未设置条件';
};

const isRangeMatch = (value, minValue, maxValue) => {
  const num = toNullableNumber(value) ?? 0;
  const min = toNullableNumber(minValue);
  const max = toNullableNumber(maxValue);

  if (min !== null && num < min) {
    return false;
  }
  if (max !== null && num > max) {
    return false;
  }
  return true;
};

const isRangeOverlap = (minAValue, maxAValue, minBValue, maxBValue) => {
  const minA = toNullableNumber(minAValue);
  const maxA = toNullableNumber(maxAValue);
  const minB = toNullableNumber(minBValue);
  const maxB = toNullableNumber(maxBValue);

  const normalizedMinA = minA === null ? Number.NEGATIVE_INFINITY : minA;
  const normalizedMaxA = maxA === null ? Number.POSITIVE_INFINITY : maxA;
  const normalizedMinB = minB === null ? Number.NEGATIVE_INFINITY : minB;
  const normalizedMaxB = maxB === null ? Number.POSITIVE_INFINITY : maxB;

  return normalizedMinA <= normalizedMaxB && normalizedMinB <= normalizedMaxA;
};

export const createEmptyTextSegmentRule = (seed = Date.now()) => ({
  id: `text_segment-${seed}`,
  enabled: true,
  priority: '',
  inputMin: '',
  inputMax: '',
  outputMin: '',
  outputMax: '',
  inputPrice: '',
  outputPrice: '',
  cacheReadPrice: '',
  cacheWritePrice: '',
});

export const normalizeTextSegmentRule = (rule, index = 0) => ({
  ...createEmptyTextSegmentRule(index),
  ...rule,
  id: rule?.id || `text_segment-${index + 1}`,
  enabled: rule?.enabled !== false,
});

export const sortTextSegmentRules = (rules = []) =>
  [...rules].sort((leftRule, rightRule) => {
    const leftPriority = toNullableNumber(leftRule?.priority);
    const rightPriority = toNullableNumber(rightRule?.priority);

    if (leftPriority === null && rightPriority === null) {
      return String(leftRule?.id || '').localeCompare(String(rightRule?.id || ''));
    }
    if (leftPriority === null) {
      return 1;
    }
    if (rightPriority === null) {
      return -1;
    }
    if (leftPriority !== rightPriority) {
      return leftPriority - rightPriority;
    }

    return String(leftRule?.id || '').localeCompare(String(rightRule?.id || ''));
  });

export const validateTextSegmentRules = (rules = []) => {
  const errors = [];
  const enabledRules = sortTextSegmentRules(rules).filter(
    (rule) => rule?.enabled !== false,
  );
  const priorityRuleMap = new Map();

  enabledRules.forEach((rule) => {
    const ruleId = rule?.id || '未命名规则';
    const priority = toNullableNumber(rule?.priority);
    const inputMin = toNullableNumber(rule?.inputMin);
    const inputMax = toNullableNumber(rule?.inputMax);
    const outputMin = toNullableNumber(rule?.outputMin);
    const outputMax = toNullableNumber(rule?.outputMax);

    if (priority === null) {
      errors.push(`${ruleId}: 请输入优先级`);
    } else if (priorityRuleMap.has(priority)) {
      errors.push(`优先级 ${priority} 重复: ${priorityRuleMap.get(priority)} / ${ruleId}`);
    } else {
      priorityRuleMap.set(priority, ruleId);
    }

    if (inputMin !== null && inputMax !== null && inputMin > inputMax) {
      errors.push(`${ruleId}: 输入最小值不能大于输入最大值`);
    }

    if (outputMin !== null && outputMax !== null && outputMin > outputMax) {
      errors.push(`${ruleId}: 输出最小值不能大于输出最大值`);
    }

    if (!hasValue(rule?.inputPrice) && rule?.inputPrice !== 0) {
      errors.push(`${ruleId}: 请输入输入价格`);
    }
  });

  for (let index = 0; index < enabledRules.length; index += 1) {
    const currentRule = enabledRules[index];
    const currentRuleId = currentRule?.id || '未命名规则';

    for (
      let compareIndex = index + 1;
      compareIndex < enabledRules.length;
      compareIndex += 1
    ) {
      const compareRule = enabledRules[compareIndex];
      const compareRuleId = compareRule?.id || '未命名规则';

      const inputOverlap = isRangeOverlap(
        currentRule?.inputMin,
        currentRule?.inputMax,
        compareRule?.inputMin,
        compareRule?.inputMax,
      );
      const outputOverlap = isRangeOverlap(
        currentRule?.outputMin,
        currentRule?.outputMax,
        compareRule?.outputMin,
        compareRule?.outputMax,
      );

      if (inputOverlap && outputOverlap) {
        errors.push(`${currentRuleId} 与 ${compareRuleId} 的区间明显重叠`);
      }
    }
  }

  return errors;
};

export const findMatchingTextSegmentRule = (rules = [], previewInput = {}) => {
  const sortedRules = sortTextSegmentRules(rules).filter(
    (rule) => rule?.enabled !== false,
  );
  const inputTokens = toNullableNumber(previewInput?.inputTokens) ?? 0;
  const outputTokens = toNullableNumber(previewInput?.outputTokens) ?? 0;

  return (
    sortedRules.find(
      (rule) =>
        isRangeMatch(inputTokens, rule?.inputMin, rule?.inputMax) &&
        isRangeMatch(outputTokens, rule?.outputMin, rule?.outputMax),
    ) || null
  );
};

export const buildTextSegmentPreview = (rules = [], previewInput = {}) => {
  const matchedRule = findMatchingTextSegmentRule(rules, previewInput);
  const inputTokens = toNullableNumber(previewInput?.inputTokens) ?? 0;
  const outputTokens = toNullableNumber(previewInput?.outputTokens) ?? 0;

  if (!matchedRule) {
    return {
      matchedRule: null,
      conditionSummary: '未命中任何规则',
      priceSummary: {
        inputCost: '',
        outputCost: '',
        totalCost: '',
        cacheReadPrice: '',
        cacheWritePrice: '',
      },
    };
  }

  const inputPrice = toNullableNumber(matchedRule?.inputPrice) ?? 0;
  const outputPrice = toNullableNumber(matchedRule?.outputPrice) ?? 0;

  return {
    matchedRule,
    conditionSummary: buildTextSegmentConditionSummary(matchedRule),
    priceSummary: {
      inputCost: formatNumber((inputTokens * inputPrice) / MILLION),
      outputCost: formatNumber((outputTokens * outputPrice) / MILLION),
      totalCost: formatNumber(
        (inputTokens * inputPrice + outputTokens * outputPrice) / MILLION,
      ),
      cacheReadPrice: formatNumber(matchedRule?.cacheReadPrice),
      cacheWritePrice: formatNumber(matchedRule?.cacheWritePrice),
    },
  };
};

export const normalizeAdvancedPricingConfig = (rawConfig) => {
  if (!rawConfig || typeof rawConfig !== 'object' || Array.isArray(rawConfig)) {
    return {
      ruleType: TEXT_SEGMENT_RULE_TYPE,
      rules: [],
    };
  }

  const ruleType =
    rawConfig.ruleType === MEDIA_TASK_RULE_TYPE ||
    rawConfig.ruleType === 'media-task'
      ? MEDIA_TASK_RULE_TYPE
      : TEXT_SEGMENT_RULE_TYPE;
  const rules = Array.isArray(rawConfig.rules)
    ? rawConfig.rules.map((rule, index) =>
        ruleType === TEXT_SEGMENT_RULE_TYPE
          ? normalizeTextSegmentRule(rule, index)
          : rule,
      )
    : [];

  return {
    ...rawConfig,
    ruleType,
    rules,
  };
};

export const hasAdvancedPricingConfig = (config) =>
  Array.isArray(config?.rules) && config.rules.length > 0;

export const canUseAdvancedPricingMode = (config) =>
  hasAdvancedPricingConfig(normalizeAdvancedPricingConfig(config));

export const getAdvancedRuleType = (config) =>
  normalizeAdvancedPricingConfig(config).ruleType;

export const normalizeFixedBillingMode = (mode) =>
  mode === FIXED_BILLING_MODE_PER_REQUEST || mode === 'per-request'
    ? FIXED_BILLING_MODE_PER_REQUEST
    : FIXED_BILLING_MODE_PER_TOKEN;

export const getFixedBillingModeForModel = (modelName, sourceMaps) =>
  hasValue(sourceMaps?.ModelPrice?.[modelName])
    ? FIXED_BILLING_MODE_PER_REQUEST
    : FIXED_BILLING_MODE_PER_TOKEN;

export const hasFixedPricingConfig = (modelName, sourceMaps) =>
  [
    sourceMaps?.ModelPrice?.[modelName],
    sourceMaps?.ModelRatio?.[modelName],
    sourceMaps?.CompletionRatio?.[modelName],
    sourceMaps?.CacheRatio?.[modelName],
    sourceMaps?.CreateCacheRatio?.[modelName],
    sourceMaps?.ImageRatio?.[modelName],
    sourceMaps?.AudioRatio?.[modelName],
    sourceMaps?.AudioCompletionRatio?.[modelName],
  ].some((value) => hasValue(value));

export const getEffectiveBillingModeForModel = ({
  selectedMode,
  fixedBillingMode,
  advancedConfig,
}) =>
  selectedMode === ADVANCED_PRICING_MODE_ADVANCED
    ? canUseAdvancedPricingMode(advancedConfig)
      ? ADVANCED_PRICING_MODE_ADVANCED
      : normalizeFixedBillingMode(fixedBillingMode)
    : normalizeFixedBillingMode(selectedMode || fixedBillingMode);
