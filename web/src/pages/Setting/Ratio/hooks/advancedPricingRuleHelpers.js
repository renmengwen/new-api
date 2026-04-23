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
const INTEGER_VALUE_REGEX = /^\d+$/;

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

const hasExplicitValue = (value) => hasValue(value) || value === 0;

const toNonNegativeInteger = (value) => {
  if (!hasExplicitValue(value)) {
    return null;
  }

  const normalizedValue = normalizeStringField(value).trim();
  if (normalizedValue === '') {
    return null;
  }
  if (!INTEGER_VALUE_REGEX.test(normalizedValue)) {
    return Number.NaN;
  }

  const numericValue = Number(normalizedValue);
  return Number.isSafeInteger(numericValue) ? numericValue : Number.NaN;
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
  const serviceTier = normalizeRuleServiceTier(rule);
  const inputModality = normalizeStringField(
    rule?.inputModality ?? rule?.input_modality,
  ).trim();
  const outputModality = normalizeStringField(
    rule?.outputModality ?? rule?.output_modality,
  ).trim();
  const imageSizeTier = normalizeStringField(
    rule?.imageSizeTier ?? rule?.image_size_tier,
  ).trim();
  const toolUsageType = normalizeStringField(
    rule?.toolUsageType ?? rule?.tool_usage_type,
  ).trim();
  const toolUsageCount = normalizeNumericField(
    rule?.toolUsageCount ?? rule?.tool_usage_count,
  );
  if (inputModality) {
    summaries.push(`input_modality=${inputModality}`);
  }
  if (outputModality) {
    summaries.push(`output_modality=${outputModality}`);
  }
  if (imageSizeTier) {
    summaries.push(`image_size_tier=${imageSizeTier}`);
  }
  if (toolUsageType) {
    summaries.push(`tool_usage_type=${toolUsageType}`);
  }
  if (toolUsageCount !== '') {
    summaries.push(`tool_usage_count>=${toolUsageCount}`);
  }

  if (serviceTier) {
    summaries.push(`服务层=${serviceTier}`);
  }

  return summaries.length > 0 ? summaries.join(' / ') : '未设置条件';
};

const buildTextSegmentPreviewConditionSummary = (rule) => {
  const inputSummary = buildRangeSummary('输入', rule?.inputMin, rule?.inputMax);
  const outputSummary = buildRangeSummary(
    '输出',
    rule?.outputMin,
    rule?.outputMax,
  );
  const summaries = [inputSummary, outputSummary].filter(Boolean);
  const serviceTier = normalizeRuleServiceTier(rule);
  const inputModality = normalizeStringField(
    rule?.inputModality ?? rule?.input_modality,
  ).trim();
  const outputModality = normalizeStringField(
    rule?.outputModality ?? rule?.output_modality,
  ).trim();

  if (inputModality) {
    summaries.push(`input_modality=${inputModality}`);
  }
  if (outputModality) {
    summaries.push(`output_modality=${outputModality}`);
  }
  if (serviceTier) {
    summaries.push(`服务层=${serviceTier}`);
  }

  return summaries.length > 0 ? summaries.join(' / ') : '未设置条件';
};

const hasTextSegmentCondition = (rule) =>
  hasExplicitValue(rule?.inputMin) ||
  hasExplicitValue(rule?.inputMax) ||
  hasExplicitValue(rule?.outputMin) ||
  hasExplicitValue(rule?.outputMax) ||
  normalizeRuleServiceTier(rule) !== '' ||
  normalizeStringField(rule?.inputModality ?? rule?.input_modality).trim() !==
    '' ||
  normalizeStringField(rule?.outputModality ?? rule?.output_modality).trim() !==
    '';

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

const hasPreviewTextSegmentCondition = (rule) =>
  hasTextSegmentCondition(rule) ||
  normalizeStringField(rule?.imageSizeTier ?? rule?.image_size_tier).trim() !==
    '' ||
  normalizeStringField(rule?.toolUsageType ?? rule?.tool_usage_type).trim() !==
    '' ||
  hasExplicitValue(rule?.toolUsageCount ?? rule?.tool_usage_count);

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

const normalizeRuleServiceTier = (rule) =>
  normalizeStringField(rule?.serviceTier ?? rule?.service_tier)
    .trim()
    .toLowerCase();

const normalizeComparableString = (value) =>
  normalizeStringField(value).trim().toLowerCase();

const isOptionalStringMatch = (previewValue, ruleValue) => {
  const normalizedRuleValue = normalizeComparableString(ruleValue);
  if (!normalizedRuleValue) {
    return true;
  }

  return normalizeComparableString(previewValue) === normalizedRuleValue;
};

const isServiceTierOverlap = (leftValue, rightValue) => {
  const normalizedLeftValue = normalizeStringField(leftValue).trim();
  const normalizedRightValue = normalizeStringField(rightValue).trim();

  if (!normalizedLeftValue || !normalizedRightValue) {
    return true;
  }

  return normalizedLeftValue === normalizedRightValue;
};

export const createEmptyTextSegmentRule = (seed = Date.now()) => ({
  id: `text_segment-${seed}`,
  enabled: true,
  priority: '',
  inputMin: '',
  inputMax: '',
  outputMin: '',
  outputMax: '',
  inputModality: '',
  outputModality: '',
  billingUnit: '',
  serviceTier: '',
  inputPrice: '',
  outputPrice: '',
  cacheReadPrice: '',
  cacheWritePrice: '',
  cacheStoragePrice: '',
  imageSizeTier: '',
  toolUsageType: '',
  toolUsageCount: '',
  toolOveragePrice: '',
  freeQuota: '',
  overageThreshold: '',
});

const omitObjectKeys = (value, keysToOmit) => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {};
  }

  return Object.entries(value).reduce((result, [key, entryValue]) => {
    if (!keysToOmit.has(key) && entryValue !== undefined) {
      result[key] = entryValue;
    }
    return result;
  }, {});
};

const TEXT_SEGMENT_RULE_EDITOR_ONLY_KEYS = new Set([
  'id',
  'enabled',
  'inputMin',
  'inputMax',
  'outputMin',
  'outputMax',
  'inputModality',
  'outputModality',
  'billingUnit',
  'serviceTier',
  'inputPrice',
  'outputPrice',
  'cacheReadPrice',
  'cacheWritePrice',
  'cacheStoragePrice',
  'imageSizeTier',
  'toolUsageType',
  'toolUsageCount',
  'toolOveragePrice',
  'freeQuota',
  'overageThreshold',
]);

const TEXT_SEGMENT_CONFIG_EDITOR_ONLY_KEYS = new Set([
  'ruleType',
  'displayName',
  'segmentBasis',
  'billingUnit',
  'defaultPrice',
  'note',
  'rules',
]);

const setSerializedStringField = (serializedValue, key, value) => {
  const normalizedValue = normalizeStringField(value).trim();

  if (normalizedValue) {
    serializedValue[key] = normalizedValue;
    return;
  }

  delete serializedValue[key];
};

const setSerializedNumberField = (serializedValue, key, value) => {
  const normalizedValue = toNullableNumber(value);

  if (normalizedValue !== null) {
    serializedValue[key] = normalizedValue;
    return;
  }

  delete serializedValue[key];
};

export const normalizeTextSegmentRule = (rule, index = 0) => ({
  ...createEmptyTextSegmentRule(index),
  ...rule,
  id: rule?.id || `text_segment-${index + 1}`,
  enabled: rule?.enabled !== false,
  priority: normalizeNumericField(rule?.priority),
  inputMin: normalizeNumericField(rule?.inputMin ?? rule?.input_min),
  inputMax: normalizeNumericField(rule?.inputMax ?? rule?.input_max),
  outputMin: normalizeNumericField(rule?.outputMin ?? rule?.output_min),
  outputMax: normalizeNumericField(rule?.outputMax ?? rule?.output_max),
  inputModality: normalizeStringField(
    rule?.inputModality ?? rule?.input_modality,
  ),
  outputModality: normalizeStringField(
    rule?.outputModality ?? rule?.output_modality,
  ),
  billingUnit: normalizeStringField(rule?.billingUnit ?? rule?.billing_unit),
  serviceTier: normalizeStringField(rule?.serviceTier ?? rule?.service_tier),
  inputPrice: normalizeNumericField(rule?.inputPrice ?? rule?.input_price),
  outputPrice: normalizeNumericField(rule?.outputPrice ?? rule?.output_price),
  cacheReadPrice: normalizeNumericField(
    rule?.cacheReadPrice ?? rule?.cache_read_price,
  ),
  cacheWritePrice: normalizeNumericField(
    rule?.cacheWritePrice ?? rule?.cache_write_price,
  ),
  cacheStoragePrice: normalizeNumericField(
    rule?.cacheStoragePrice ?? rule?.cache_storage_price,
  ),
  imageSizeTier: normalizeStringField(
    rule?.imageSizeTier ?? rule?.image_size_tier,
  ),
  toolUsageType: normalizeStringField(
    rule?.toolUsageType ?? rule?.tool_usage_type,
  ),
  toolUsageCount: normalizeNumericField(
    rule?.toolUsageCount ?? rule?.tool_usage_count,
  ),
  toolOveragePrice: normalizeNumericField(
    rule?.toolOveragePrice ?? rule?.tool_overage_price,
  ),
  freeQuota: normalizeNumericField(rule?.freeQuota ?? rule?.free_quota),
  overageThreshold: normalizeNumericField(
    rule?.overageThreshold ?? rule?.overage_threshold,
  ),
});

const normalizeBooleanRuleValue = (value) => {
  if (value === true || value === 'true') {
    return 'true';
  }
  if (value === false || value === 'false') {
    return 'false';
  }
  return '';
};

const toBooleanRuleValue = (value) => {
  if (value === true || value === 'true') {
    return true;
  }
  if (value === false || value === 'false') {
    return false;
  }
  return null;
};

const normalizeStringField = (value) =>
  typeof value === 'string' ? value : value === null || value === undefined ? '' : String(value);

const normalizeMediaTaskTypeValue = (value) =>
  normalizeStringField(value).trim().toLowerCase();

const isCanonicalMediaTaskType = (value) =>
  value === 'video_generation' || value === 'image_generation';

const resolvePreviewMediaTaskType = (previewInput = {}) => {
  const explicitTaskType = normalizeMediaTaskTypeValue(previewInput?.taskType);
  if (explicitTaskType) {
    return explicitTaskType;
  }

  const rawAction = normalizeMediaTaskTypeValue(previewInput?.rawAction);
  switch (rawAction) {
    case 'image_generation':
      return 'image_generation';
    case 'video_generation':
      return 'video_generation';
    case 'generate':
    case 'textgenerate':
    case 'firsttailgenerate':
    case 'referencegenerate':
    case 'remixgenerate':
    case 'remix':
      return 'video_generation';
    default:
      return rawAction;
  }
};

export const getMediaTaskTypeDisplayLabel = (value, t = (label) => label) => {
  const normalizedValue = normalizeStringField(value).trim();
  switch (normalizeMediaTaskTypeValue(normalizedValue)) {
    case 'video_generation':
      return t('视频生成');
    case 'generate':
    case 'textgenerate':
    case 'firsttailgenerate':
    case 'referencegenerate':
    case 'remixgenerate':
    case 'remix':
      return `${t('视频生成')}（${t('旧值')} ${normalizedValue}）`;
    case 'image_generation':
      return t('图片生成');
    default:
      return normalizedValue;
  }
};

const normalizeNumericField = (value) => {
  const formatted = formatNumber(value);
  return formatted === '' ? '' : formatted;
};

export const createEmptyMediaTaskRule = (seed = Date.now()) => ({
  id: `media_task-${seed}`,
  priority: '',
  rawAction: '',
  inferenceMode: '',
  inputModality: '',
  outputModality: '',
  billingUnit: '',
  audio: '',
  inputVideo: '',
  resolution: '',
  aspectRatio: '',
  imageSizeTier: '',
  outputDurationMin: '',
  outputDurationMax: '',
  inputVideoDurationMin: '',
  inputVideoDurationMax: '',
  draft: '',
  draftCoefficient: '',
  toolUsageType: '',
  toolUsageCount: '',
  toolOveragePrice: '',
  freeQuota: '',
  overageThreshold: '',
  remark: '',
  unitPrice: '',
  minTokens: '',
});

export const normalizeMediaTaskRule = (rule, index = 0) => ({
  ...createEmptyMediaTaskRule(index),
  ...rule,
  id: rule?.id || `media_task-${index + 1}`,
  priority: normalizeNumericField(rule?.priority),
  rawAction: normalizeStringField(rule?.rawAction ?? rule?.raw_action),
  inferenceMode: normalizeStringField(
    rule?.inferenceMode ?? rule?.inference_mode,
  ),
  inputModality: normalizeStringField(
    rule?.inputModality ?? rule?.input_modality,
  ),
  outputModality: normalizeStringField(
    rule?.outputModality ?? rule?.output_modality,
  ),
  billingUnit: normalizeStringField(rule?.billingUnit ?? rule?.billing_unit),
  audio: normalizeBooleanRuleValue(rule?.audio),
  inputVideo: normalizeBooleanRuleValue(
    rule?.inputVideo ?? rule?.input_video,
  ),
  resolution: normalizeStringField(rule?.resolution),
  aspectRatio: normalizeStringField(rule?.aspectRatio ?? rule?.aspect_ratio),
  imageSizeTier: normalizeStringField(
    rule?.imageSizeTier ?? rule?.image_size_tier,
  ),
  outputDurationMin: normalizeNumericField(
    rule?.outputDurationMin ?? rule?.output_duration_min,
  ),
  outputDurationMax: normalizeNumericField(
    rule?.outputDurationMax ?? rule?.output_duration_max,
  ),
  inputVideoDurationMin: normalizeNumericField(
    rule?.inputVideoDurationMin ?? rule?.input_video_duration_min,
  ),
  inputVideoDurationMax: normalizeNumericField(
    rule?.inputVideoDurationMax ?? rule?.input_video_duration_max,
  ),
  draft: normalizeBooleanRuleValue(rule?.draft),
  draftCoefficient: normalizeNumericField(
    rule?.draftCoefficient ?? rule?.draft_coefficient,
  ),
  toolUsageType: normalizeStringField(
    rule?.toolUsageType ?? rule?.tool_usage_type,
  ),
  toolUsageCount: normalizeNumericField(
    rule?.toolUsageCount ?? rule?.tool_usage_count,
  ),
  toolOveragePrice: normalizeNumericField(
    rule?.toolOveragePrice ?? rule?.tool_overage_price,
  ),
  freeQuota: normalizeNumericField(rule?.freeQuota ?? rule?.free_quota),
  overageThreshold: normalizeNumericField(
    rule?.overageThreshold ?? rule?.overage_threshold,
  ),
  remark: normalizeStringField(rule?.remark),
  unitPrice: normalizeNumericField(rule?.unitPrice ?? rule?.unit_price),
  minTokens: normalizeNumericField(rule?.minTokens ?? rule?.min_tokens),
});

export const sortMediaTaskRules = (rules = []) =>
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

export const buildMediaTaskConditionSummary = (rule) => {
  const summaries = [];
  const rawAction = normalizeStringField(rule?.rawAction ?? rule?.raw_action);

  if (rawAction.trim()) {
    summaries.push(`raw_action=${rawAction.trim()}`);
  }

  if (rule?.inferenceMode) {
    summaries.push(`inference_mode=${rule.inferenceMode}`);
  }
  if (normalizeStringField(rule?.inputModality ?? rule?.input_modality).trim()) {
    summaries.push(
      `input_modality=${normalizeStringField(
        rule?.inputModality ?? rule?.input_modality,
      ).trim()}`,
    );
  }
  if (
    normalizeStringField(rule?.outputModality ?? rule?.output_modality).trim()
  ) {
    summaries.push(
      `output_modality=${normalizeStringField(
        rule?.outputModality ?? rule?.output_modality,
      ).trim()}`,
    );
  }

  const audioValue = toBooleanRuleValue(rule?.audio);
  if (audioValue !== null) {
    summaries.push(`audio=${audioValue ? 'yes' : 'no'}`);
  }

  const inputVideoValue = toBooleanRuleValue(rule?.inputVideo);
  if (inputVideoValue !== null) {
    summaries.push(`input_video=${inputVideoValue ? 'yes' : 'no'}`);
  }

  if (rule?.resolution) {
    summaries.push(`resolution=${rule.resolution}`);
  }
  if (rule?.aspectRatio) {
    summaries.push(`aspect_ratio=${rule.aspectRatio}`);
  }
  if (normalizeStringField(rule?.imageSizeTier ?? rule?.image_size_tier).trim()) {
    summaries.push(
      `image_size_tier=${normalizeStringField(
        rule?.imageSizeTier ?? rule?.image_size_tier,
      ).trim()}`,
    );
  }
  if (normalizeStringField(rule?.toolUsageType ?? rule?.tool_usage_type).trim()) {
    summaries.push(
      `tool_usage_type=${normalizeStringField(
        rule?.toolUsageType ?? rule?.tool_usage_type,
      ).trim()}`,
    );
  }
  if (hasValue(rule?.toolUsageCount) || rule?.toolUsageCount === 0) {
    summaries.push(
      `tool_usage_count>=${normalizeNumericField(
        rule?.toolUsageCount ?? rule?.tool_usage_count,
      )}`,
    );
  }

  const outputDurationSummary = buildRangeSummary(
    'output_duration',
    rule?.outputDurationMin,
    rule?.outputDurationMax,
  );
  if (outputDurationSummary) {
    summaries.push(outputDurationSummary);
  }

  const inputVideoDurationSummary = buildRangeSummary(
    'input_video_duration',
    rule?.inputVideoDurationMin,
    rule?.inputVideoDurationMax,
  );
  if (inputVideoDurationSummary) {
    summaries.push(inputVideoDurationSummary);
  }

  const draftValue = toBooleanRuleValue(rule?.draft);
  if (draftValue !== null) {
    summaries.push(`draft=${draftValue ? 'yes' : 'no'}`);
  }

  if (hasValue(rule?.minTokens) || rule?.minTokens === 0) {
    summaries.push(`min_tokens>=${normalizeNumericField(rule.minTokens)}`);
  }

  return summaries.length > 0 ? summaries.join(' / ') : 'no filters';
};

const validateRangeOrder = (errors, ruleId, fieldName, minValue, maxValue) => {
  const hasMin = hasExplicitValue(minValue);
  const hasMax = hasExplicitValue(maxValue);

  if (!hasMin && !hasMax) {
    return;
  }
  if (hasMin !== hasMax) {
    errors.push(`${ruleId}: ${fieldName} requires both min and max`);
    return;
  }

  const min = toNonNegativeInteger(minValue);
  const max = toNonNegativeInteger(maxValue);
  if (Number.isNaN(min) || Number.isNaN(max)) {
    errors.push(`${ruleId}: ${fieldName} must use non-negative integers`);
    return;
  }

  if (min > max) {
    errors.push(`${ruleId}: ${fieldName} range is invalid`);
  }
};

export const validateMediaTaskConfig = (config) => {
  const errors = [];
  const normalizedConfig = normalizeAdvancedPricingConfig(config);

  if (normalizedConfig.ruleType !== MEDIA_TASK_RULE_TYPE) {
    return errors;
  }

  if (!normalizeStringField(normalizedConfig.taskType).trim()) {
    errors.push('task_type is required');
  }
  if (!normalizeStringField(normalizedConfig.billingUnit).trim()) {
    errors.push('billing_unit is required');
  }
  if (normalizedConfig.rules.length === 0) {
    errors.push('segments are required');
  }

  const priorityRuleMap = new Map();
  sortMediaTaskRules(normalizedConfig.rules).forEach((rule) => {
    const ruleId = rule?.id || 'media_task';
    const priority = toNullableNumber(rule?.priority);

    if (priority === null) {
      errors.push(`${ruleId}: priority is required`);
    } else if (priorityRuleMap.has(priority)) {
      errors.push(
        `priority ${priority} duplicated: ${priorityRuleMap.get(priority)} / ${ruleId}`,
      );
    } else {
      priorityRuleMap.set(priority, ruleId);
    }

    if (!hasValue(rule?.unitPrice) && rule?.unitPrice !== 0) {
      errors.push(`${ruleId}: unit_price is required`);
    }

    validateRangeOrder(
      errors,
      ruleId,
      'output_duration',
      rule?.outputDurationMin,
      rule?.outputDurationMax,
    );
    validateRangeOrder(
      errors,
      ruleId,
      'input_video_duration',
      rule?.inputVideoDurationMin,
      rule?.inputVideoDurationMax,
    );
  });

  return errors;
};

const appendSerializedIntegerRange = (
  serializedRule,
  minKey,
  maxKey,
  minValue,
  maxValue,
) => {
  const hasMin = hasExplicitValue(minValue);
  const hasMax = hasExplicitValue(maxValue);

  if (!hasMin || !hasMax) {
    return;
  }

  const min = toNonNegativeInteger(minValue);
  const max = toNonNegativeInteger(maxValue);
  if (Number.isNaN(min) || Number.isNaN(max) || min > max) {
    return;
  }

  serializedRule[minKey] = min;
  serializedRule[maxKey] = max;
};

export const serializeMediaTaskRule = (rule) => {
  const normalizedRule = normalizeMediaTaskRule(rule);
  const serializedRule = {};
  const priority = toNullableNumber(normalizedRule.priority);
  const draftCoefficient = toNullableNumber(normalizedRule.draftCoefficient);
  const unitPrice = toNullableNumber(normalizedRule.unitPrice);
  const minTokens = toNullableNumber(normalizedRule.minTokens);
  const toolUsageCount = toNullableNumber(normalizedRule.toolUsageCount);
  const toolOveragePrice = toNullableNumber(normalizedRule.toolOveragePrice);
  const audio = toBooleanRuleValue(normalizedRule.audio);
  const inputVideo = toBooleanRuleValue(normalizedRule.inputVideo);
  const draft = toBooleanRuleValue(normalizedRule.draft);

  if (priority !== null) {
    serializedRule.priority = priority;
  }
  if (normalizedRule.rawAction.trim()) {
    serializedRule.raw_action = normalizedRule.rawAction.trim();
  }
  if (normalizedRule.inferenceMode.trim()) {
    serializedRule.inference_mode = normalizedRule.inferenceMode.trim();
  }
  if (normalizedRule.inputModality.trim()) {
    serializedRule.input_modality = normalizedRule.inputModality.trim();
  }
  if (normalizedRule.outputModality.trim()) {
    serializedRule.output_modality = normalizedRule.outputModality.trim();
  }
  if (normalizedRule.billingUnit.trim()) {
    serializedRule.billing_unit = normalizedRule.billingUnit.trim();
  }
  if (audio !== null) {
    serializedRule.audio = audio;
  }
  if (inputVideo !== null) {
    serializedRule.input_video = inputVideo;
  }
  if (normalizedRule.resolution.trim()) {
    serializedRule.resolution = normalizedRule.resolution.trim();
  }
  if (normalizedRule.aspectRatio.trim()) {
    serializedRule.aspect_ratio = normalizedRule.aspectRatio.trim();
  }
  if (normalizedRule.imageSizeTier.trim()) {
    serializedRule.image_size_tier = normalizedRule.imageSizeTier.trim();
  }
  appendSerializedIntegerRange(
    serializedRule,
    'output_duration_min',
    'output_duration_max',
    normalizedRule.outputDurationMin,
    normalizedRule.outputDurationMax,
  );
  appendSerializedIntegerRange(
    serializedRule,
    'input_video_duration_min',
    'input_video_duration_max',
    normalizedRule.inputVideoDurationMin,
    normalizedRule.inputVideoDurationMax,
  );
  if (draft !== null) {
    serializedRule.draft = draft;
  }
  if (draftCoefficient !== null) {
    serializedRule.draft_coefficient = draftCoefficient;
  }
  if (normalizedRule.toolUsageType.trim()) {
    serializedRule.tool_usage_type = normalizedRule.toolUsageType.trim();
  }
  if (toolUsageCount !== null) {
    serializedRule.tool_usage_count = toolUsageCount;
  }
  if (toolOveragePrice !== null) {
    serializedRule.tool_overage_price = toolOveragePrice;
  }
  const freeQuota = toNullableNumber(normalizedRule.freeQuota);
  if (freeQuota !== null) {
    serializedRule.free_quota = freeQuota;
  }
  const overageThreshold = toNullableNumber(normalizedRule.overageThreshold);
  if (overageThreshold !== null) {
    serializedRule.overage_threshold = overageThreshold;
  }
  if (normalizedRule.remark.trim()) {
    serializedRule.remark = normalizedRule.remark.trim();
  }
  if (unitPrice !== null) {
    serializedRule.unit_price = unitPrice;
  }
  if (minTokens !== null) {
    serializedRule.min_tokens = minTokens;
  }

  return serializedRule;
};

export const serializeTextSegmentRule = (rule) => {
  const normalizedRule = normalizeTextSegmentRule(rule);
  const serializedRule = omitObjectKeys(
    normalizedRule,
    TEXT_SEGMENT_RULE_EDITOR_ONLY_KEYS,
  );

  setSerializedNumberField(serializedRule, 'priority', normalizedRule.priority);
  setSerializedNumberField(serializedRule, 'input_min', normalizedRule.inputMin);
  setSerializedNumberField(serializedRule, 'input_max', normalizedRule.inputMax);
  setSerializedNumberField(serializedRule, 'output_min', normalizedRule.outputMin);
  setSerializedNumberField(serializedRule, 'output_max', normalizedRule.outputMax);
  setSerializedStringField(
    serializedRule,
    'input_modality',
    normalizedRule.inputModality,
  );
  setSerializedStringField(
    serializedRule,
    'output_modality',
    normalizedRule.outputModality,
  );
  setSerializedStringField(
    serializedRule,
    'billing_unit',
    normalizedRule.billingUnit,
  );
  setSerializedStringField(
    serializedRule,
    'service_tier',
    normalizeRuleServiceTier(normalizedRule),
  );
  setSerializedNumberField(serializedRule, 'input_price', normalizedRule.inputPrice);
  setSerializedNumberField(
    serializedRule,
    'output_price',
    normalizedRule.outputPrice,
  );
  setSerializedNumberField(
    serializedRule,
    'cache_read_price',
    normalizedRule.cacheReadPrice,
  );
  setSerializedNumberField(
    serializedRule,
    'cache_write_price',
    normalizedRule.cacheWritePrice,
  );
  setSerializedNumberField(
    serializedRule,
    'cache_storage_price',
    normalizedRule.cacheStoragePrice,
  );
  setSerializedStringField(
    serializedRule,
    'image_size_tier',
    normalizedRule.imageSizeTier,
  );
  setSerializedStringField(
    serializedRule,
    'tool_usage_type',
    normalizedRule.toolUsageType,
  );
  setSerializedNumberField(
    serializedRule,
    'tool_usage_count',
    normalizedRule.toolUsageCount,
  );
  setSerializedNumberField(
    serializedRule,
    'tool_overage_price',
    normalizedRule.toolOveragePrice,
  );
  setSerializedNumberField(serializedRule, 'free_quota', normalizedRule.freeQuota);
  setSerializedNumberField(
    serializedRule,
    'overage_threshold',
    normalizedRule.overageThreshold,
  );

  return serializedRule;
};

export const serializeAdvancedPricingConfig = (config) => {
  const normalizedConfig = normalizeAdvancedPricingConfig(config);

  if (normalizedConfig.ruleType !== MEDIA_TASK_RULE_TYPE) {
    const serializedConfig = omitObjectKeys(
      normalizedConfig,
      TEXT_SEGMENT_CONFIG_EDITOR_ONLY_KEYS,
    );

    serializedConfig.rule_type = TEXT_SEGMENT_RULE_TYPE;
    serializedConfig.segments = sortTextSegmentRules(normalizedConfig.rules)
      .filter((rule) => rule?.enabled !== false)
      .map((rule) => serializeTextSegmentRule(rule));
    setSerializedStringField(
      serializedConfig,
      'display_name',
      normalizedConfig.displayName,
    );
    setSerializedStringField(
      serializedConfig,
      'segment_basis',
      normalizedConfig.segmentBasis,
    );
    setSerializedStringField(
      serializedConfig,
      'billing_unit',
      normalizedConfig.billingUnit,
    );
    setSerializedNumberField(
      serializedConfig,
      'default_price',
      normalizedConfig.defaultPrice,
    );
    setSerializedStringField(serializedConfig, 'note', normalizedConfig.note);
    return serializedConfig;
  }

  const serializedConfig = {
    rule_type: MEDIA_TASK_RULE_TYPE,
    segments: sortMediaTaskRules(normalizedConfig.rules).map((rule) =>
      serializeMediaTaskRule(rule),
    ),
  };

  if (normalizedConfig.displayName.trim()) {
    serializedConfig.display_name = normalizedConfig.displayName.trim();
  }
  if (normalizedConfig.taskType.trim()) {
    serializedConfig.task_type = normalizedConfig.taskType.trim();
  }
  if (normalizedConfig.billingUnit.trim()) {
    serializedConfig.billing_unit = normalizedConfig.billingUnit.trim();
  }
  if (normalizedConfig.note.trim()) {
    serializedConfig.note = normalizedConfig.note.trim();
  }

  return serializedConfig;
};

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

export const getTextSegmentRuleEditorMeta = (config = {}, rules = []) => {
  const defaultPrice = normalizeNumericField(
    config?.defaultPrice ?? config?.default_price,
  );
  const normalizedRules = Array.isArray(rules) ? rules : [];

  return {
    ruleType: TEXT_SEGMENT_RULE_TYPE,
    totalRules: normalizedRules.length,
    enabledRules: normalizedRules.filter((rule) => rule?.enabled !== false)
      .length,
    hasDefaultPrice: defaultPrice !== '',
    defaultPrice,
  };
};

export const validateTextSegmentRules = (rules = []) => {
  const errors = [];
  const enabledRules = sortTextSegmentRules(rules).filter(
    (rule) => rule?.enabled !== false,
  );
  const matchableRules = enabledRules.filter((rule) => hasTextSegmentCondition(rule));
  const defaultRules = enabledRules.filter((rule) => !hasTextSegmentCondition(rule));
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

  if (defaultRules.length > 1) {
    errors.push('default text segment can only be configured once');
  }

  for (let index = 0; index < matchableRules.length; index += 1) {
    const currentRule = matchableRules[index];
    const currentRuleId = currentRule?.id || '未命名规则';

    for (
      let compareIndex = index + 1;
      compareIndex < matchableRules.length;
      compareIndex += 1
    ) {
      const compareRule = matchableRules[compareIndex];
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
      const currentRuleServiceTier = normalizeRuleServiceTier(currentRule);
      const compareRuleServiceTier = normalizeRuleServiceTier(compareRule);
      const serviceTierOverlap =
        !currentRuleServiceTier ||
        !compareRuleServiceTier ||
        currentRuleServiceTier === compareRuleServiceTier;
      const currentInputModality = normalizeComparableString(
        currentRule?.inputModality ?? currentRule?.input_modality,
      );
      const compareInputModality = normalizeComparableString(
        compareRule?.inputModality ?? compareRule?.input_modality,
      );
      const inputModalityOverlap =
        !currentInputModality ||
        !compareInputModality ||
        currentInputModality === compareInputModality;
      const currentOutputModality = normalizeComparableString(
        currentRule?.outputModality ?? currentRule?.output_modality,
      );
      const compareOutputModality = normalizeComparableString(
        compareRule?.outputModality ?? compareRule?.output_modality,
      );
      const outputModalityOverlap =
        !currentOutputModality ||
        !compareOutputModality ||
        currentOutputModality === compareOutputModality;

      if (
        inputOverlap &&
        outputOverlap &&
        serviceTierOverlap &&
        inputModalityOverlap &&
        outputModalityOverlap
      ) {
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
  const defaultRule =
    sortedRules.find((rule) => !hasPreviewTextSegmentCondition(rule)) || null;
  const conditionalRules = sortedRules.filter((rule) =>
    hasPreviewTextSegmentCondition(rule),
  );
  const inputTokens = toNullableNumber(previewInput?.inputTokens) ?? 0;
  const outputTokens = toNullableNumber(previewInput?.outputTokens) ?? 0;
  const previewServiceTier = normalizeStringField(previewInput?.serviceTier)
    .trim()
    .toLowerCase();

  return (
    conditionalRules.find(
      (rule) =>
        isRangeMatch(inputTokens, rule?.inputMin, rule?.inputMax) &&
        isRangeMatch(outputTokens, rule?.outputMin, rule?.outputMax) &&
        (!normalizeRuleServiceTier(rule) ||
          normalizeRuleServiceTier(rule) === previewServiceTier) &&
        isOptionalStringMatch(
          previewInput?.inputModality,
          rule?.inputModality ?? rule?.input_modality,
        ) &&
        isOptionalStringMatch(
          previewInput?.outputModality,
          rule?.outputModality ?? rule?.output_modality,
        ) &&
        isOptionalStringMatch(
          previewInput?.imageSizeTier,
          rule?.imageSizeTier ?? rule?.image_size_tier,
        ) &&
        isOptionalStringMatch(
          previewInput?.toolUsageType,
          rule?.toolUsageType ?? rule?.tool_usage_type,
        ) &&
        isToolUsageCountMatch(
          previewInput?.toolUsageCount,
          rule?.toolUsageCount ?? rule?.tool_usage_count,
        ),
    ) ||
    defaultRule
  );
};

const isMediaTaskStringMatch = (previewValue, ruleValue) => {
  return isOptionalStringMatch(previewValue, ruleValue);
};

const isMediaTaskTypeMatch = (ruleSetTaskType, previewInput = {}) => {
  const normalizedRuleTaskType = normalizeMediaTaskTypeValue(ruleSetTaskType);
  if (!normalizedRuleTaskType) {
    return true;
  }

  const normalizedRawAction = normalizeMediaTaskTypeValue(previewInput?.rawAction);
  const previewTaskType = resolvePreviewMediaTaskType(previewInput);
  if (!previewTaskType && !normalizedRawAction) {
    return isCanonicalMediaTaskType(normalizedRuleTaskType);
  }
  if (previewTaskType && normalizedRuleTaskType === previewTaskType) {
    return true;
  }

  return normalizedRuleTaskType === normalizedRawAction;
};

const isMediaTaskBooleanMatch = (previewValue, ruleValue) => {
  const normalizedRuleValue = toBooleanRuleValue(ruleValue);
  if (normalizedRuleValue === null) {
    return true;
  }

  return toBooleanRuleValue(previewValue) === normalizedRuleValue;
};

const isMediaTaskRangeMatch = (previewValue, minValue, maxValue) => {
  const hasRangeFilter = hasExplicitValue(minValue) || hasExplicitValue(maxValue);
  if (!hasRangeFilter) {
    return true;
  }

  const normalizedPreviewValue = toNullableNumber(previewValue);
  if (normalizedPreviewValue === null) {
    return false;
  }

  return isRangeMatch(normalizedPreviewValue, minValue, maxValue);
};

const isToolUsageCountMatch = (previewValue, ruleValue) => {
  if (!hasExplicitValue(ruleValue)) {
    return true;
  }

  const normalizedPreviewValue = toNullableNumber(previewValue);
  const normalizedRuleValue = toNullableNumber(ruleValue);
  if (normalizedRuleValue === null) {
    return true;
  }
  if (normalizedPreviewValue === null) {
    return false;
  }
  return normalizedPreviewValue >= normalizedRuleValue;
};

const resolvePreviewBillingUnit = (value, fallback = 'per_million_tokens') => {
  const billingUnit = normalizeStringField(value).trim();
  return billingUnit || fallback;
};

const resolvePreviewImageCount = (previewInput = {}, fallbackValue = 0) =>
  toNullableNumber(previewInput?.imageCount) ??
  toNullableNumber(fallbackValue) ??
  0;

const resolvePreviewLiveDuration = (previewInput = {}, fallbackValue = 0) =>
  toNullableNumber(previewInput?.liveDurationSecs) ??
  toNullableNumber(fallbackValue) ??
  0;

const resolvePreviewCallCount = (previewInput = {}) =>
  toNullableNumber(previewInput?.toolUsageCount) ?? 0;

const buildPerSecondFormulaSummary = (quantity, unitPrice, multiplier = 1) =>
  `${formatNumber(quantity)} per_second × ${formatNumber(unitPrice)} × ${formatNumber(multiplier)}`;

const buildPerImageFormulaSummary = (quantity, unitPrice, multiplier = 1) =>
  `${formatNumber(quantity)} per_image × ${formatNumber(unitPrice)} × ${formatNumber(multiplier)}`;

const buildPer1000CallsFormulaSummary = (
  totalCount,
  freeQuota,
  unitPrice,
  multiplier = 1,
  overageThreshold = 1000,
) =>
  `max(${formatNumber(totalCount)} - ${formatNumber(freeQuota)}, 0) / ${formatNumber(overageThreshold)} per_1000_calls × ${formatNumber(unitPrice)} × ${formatNumber(multiplier)}`;

const findMatchingMediaTaskRule = (
  rules = [],
  previewInput = {},
  ruleSetTaskType = '',
) =>
  sortMediaTaskRules(rules).find(
    (rule) =>
      isMediaTaskTypeMatch(ruleSetTaskType, previewInput) &&
      isMediaTaskStringMatch(
        previewInput?.rawAction,
        rule?.rawAction ?? rule?.raw_action,
      ) &&
      isMediaTaskStringMatch(
        previewInput?.inferenceMode,
        rule?.inferenceMode ?? rule?.inference_mode,
      ) &&
      isMediaTaskStringMatch(
        previewInput?.inputModality,
        rule?.inputModality ?? rule?.input_modality,
      ) &&
      isMediaTaskStringMatch(
        previewInput?.outputModality,
        rule?.outputModality ?? rule?.output_modality,
      ) &&
      isMediaTaskStringMatch(
        previewInput?.imageSizeTier,
        rule?.imageSizeTier ?? rule?.image_size_tier,
      ) &&
      isMediaTaskStringMatch(
        previewInput?.toolUsageType,
        rule?.toolUsageType ?? rule?.tool_usage_type,
      ) &&
      isToolUsageCountMatch(
        previewInput?.toolUsageCount,
        rule?.toolUsageCount ?? rule?.tool_usage_count,
      ) &&
      isMediaTaskBooleanMatch(
        previewInput?.inputVideo,
        rule?.inputVideo ?? rule?.input_video,
      ) &&
      isMediaTaskBooleanMatch(previewInput?.audio, rule?.audio) &&
      isMediaTaskStringMatch(previewInput?.resolution, rule?.resolution) &&
      isMediaTaskStringMatch(
        previewInput?.aspectRatio,
        rule?.aspectRatio ?? rule?.aspect_ratio,
      ) &&
      isMediaTaskRangeMatch(
        previewInput?.outputDuration,
        rule?.outputDurationMin ?? rule?.output_duration_min,
        rule?.outputDurationMax ?? rule?.output_duration_max,
      ) &&
      isMediaTaskRangeMatch(
        previewInput?.inputVideoDuration,
        rule?.inputVideoDurationMin ?? rule?.input_video_duration_min,
        rule?.inputVideoDurationMax ?? rule?.input_video_duration_max,
      ) &&
      isMediaTaskBooleanMatch(previewInput?.draft, rule?.draft),
  ) || null;

export const buildTextSegmentPreview = (rules = [], previewInput = {}) => {
  const matchedRule = findMatchingTextSegmentRule(rules, previewInput);
  const inputTokens = toNullableNumber(previewInput?.inputTokens) ?? 0;
  const outputTokens = toNullableNumber(previewInput?.outputTokens) ?? 0;
  const previewToolUsageCount = resolvePreviewCallCount(previewInput);
  const previewImageCount = resolvePreviewImageCount(previewInput);
  const previewLiveDuration = resolvePreviewLiveDuration(previewInput);
  const previewImageSizeTier = normalizeStringField(previewInput?.imageSizeTier).trim();
  const previewToolUsageType = normalizeStringField(previewInput?.toolUsageType).trim();

  if (!matchedRule) {
    return {
      matchedRule: null,
      matchedSegmentPreview: null,
      conditionSummary: '未命中任何规则',
      formulaSummary: '',
      logPreview: {
        detailSummary: '',
        processSummary: '',
      },
      priceSummary: {
        inputCost: '',
        outputCost: '',
        totalCost: '',
        cacheReadPrice: '',
        cacheWritePrice: '',
        cacheStoragePrice: '',
        imageSizeTier: '',
        toolUsageType: '',
        toolUsageCount: '',
        toolOveragePrice: '',
        imageCount: '',
        liveDurationSecs: '',
        freeQuota: '',
        overageThreshold: '',
        billingUnit: '',
      },
    };
  }

  const inputPrice = toNullableNumber(matchedRule?.inputPrice) ?? 0;
  const outputPrice = toNullableNumber(matchedRule?.outputPrice) ?? 0;
  const billingUnit = resolvePreviewBillingUnit(
    matchedRule?.billingUnit ?? matchedRule?.billing_unit,
  );
  const freeQuota = toNullableNumber(matchedRule?.freeQuota ?? matchedRule?.free_quota) ?? 0;
  const overageThreshold =
    toNullableNumber(
      matchedRule?.overageThreshold ?? matchedRule?.overage_threshold,
    ) ?? 1000;
  const toolOveragePrice =
    toNullableNumber(matchedRule?.toolOveragePrice ?? matchedRule?.tool_overage_price) ??
    inputPrice;
  const unitPrice =
    billingUnit === 'per_1000_calls' ? toolOveragePrice : inputPrice + outputPrice;
  const perTokenTotalCost =
    (inputTokens * inputPrice + outputTokens * outputPrice) / MILLION;
  const matchedSegmentPreview = serializeTextSegmentRule(matchedRule);
  const conditionSummary = buildTextSegmentConditionSummary(matchedRule);
  const billableCallUnits =
    Math.max(previewToolUsageCount - freeQuota, 0) / overageThreshold;
  let formulaSummary = `(${formatNumber(inputTokens)} tokens × ${formatNumber(inputPrice)} + ${formatNumber(outputTokens)} tokens × ${formatNumber(outputPrice)}) / 1,000,000`;
  let processSummary = `输入 ${formatNumber(inputTokens)} tokens × ${formatNumber(inputPrice)} + 输出 ${formatNumber(outputTokens)} tokens × ${formatNumber(outputPrice)}`;
  let totalCost = perTokenTotalCost;

  switch (billingUnit) {
    case 'per_second':
      formulaSummary = buildPerSecondFormulaSummary(previewLiveDuration, unitPrice);
      totalCost = previewLiveDuration * unitPrice;
      processSummary = `${formulaSummary} = ${formatNumber(totalCost)}`;
      break;
    case 'per_image':
      formulaSummary = buildPerImageFormulaSummary(previewImageCount, unitPrice);
      totalCost = previewImageCount * unitPrice;
      processSummary = `${formulaSummary} = ${formatNumber(totalCost)}`;
      break;
    case 'per_1000_calls':
      formulaSummary = buildPer1000CallsFormulaSummary(
        previewToolUsageCount,
        freeQuota,
        unitPrice,
        1,
        overageThreshold,
      );
      totalCost = billableCallUnits * unitPrice;
      processSummary = `tool_usage_count ${formatNumber(previewToolUsageCount)}, free_quota ${formatNumber(freeQuota)}, overage_threshold ${formatNumber(overageThreshold)}: ${formulaSummary} = ${formatNumber(totalCost)}`;
      break;
    default:
      break;
  }

  return {
    matchedRule,
    matchedSegmentPreview,
    conditionSummary,
    formulaSummary,
    logPreview: {
      detailSummary: `命中文本分段规则：${conditionSummary}`,
      processSummary,
    },
    priceSummary: {
      inputCost: formatNumber((inputTokens * inputPrice) / MILLION),
      outputCost: formatNumber((outputTokens * outputPrice) / MILLION),
      totalCost: formatNumber(totalCost),
      cacheReadPrice: formatNumber(matchedRule?.cacheReadPrice),
      cacheWritePrice: formatNumber(matchedRule?.cacheWritePrice),
      cacheStoragePrice: formatNumber(matchedRule?.cacheStoragePrice),
      imageSizeTier: previewImageSizeTier,
      toolUsageType: previewToolUsageType,
      toolUsageCount: formatNumber(previewInput?.toolUsageCount),
      toolOveragePrice: formatNumber(
        matchedRule?.toolOveragePrice ?? matchedRule?.tool_overage_price,
      ),
      imageCount: formatNumber(previewImageCount),
      liveDurationSecs: formatNumber(previewLiveDuration),
      freeQuota: formatNumber(matchedRule?.freeQuota),
      overageThreshold: formatNumber(matchedRule?.overageThreshold),
      billingUnit: normalizeStringField(
        matchedRule?.billingUnit ?? matchedRule?.billing_unit,
      ).trim(),
    },
  };
};

export const buildMediaTaskPreview = (
  rules = [],
  previewInput = {},
  ruleSetTaskType = '',
) => {
  const matchedRule = findMatchingMediaTaskRule(
    rules,
    previewInput,
    ruleSetTaskType,
  );

  if (!matchedRule) {
    return {
      matchedRule: null,
      matchedSegmentPreview: null,
      conditionSummary: '未命中任何媒体任务规则',
      formulaSummary: '',
      logPreview: {
        detailSummary: '',
        processSummary: '',
      },
      priceSummary: {
        usageTotalTokens: '',
        billableTokens: '',
        minTokens: '',
        unitPrice: '',
        draftCoefficient: '',
        estimatedCost: '',
        imageSizeTier: '',
        toolUsageType: '',
        toolUsageCount: '',
        toolOveragePrice: '',
        imageCount: '',
        liveDurationSecs: '',
        freeQuota: '',
        overageThreshold: '',
        billingUnit: '',
      },
    };
  }

  const usageTotalTokens = toNullableNumber(previewInput?.usageTotalTokens) ?? 0;
  const minTokens = toNullableNumber(matchedRule?.minTokens) ?? 0;
  const billableTokens = Math.max(usageTotalTokens, minTokens);
  const unitPrice = toNullableNumber(matchedRule?.unitPrice) ?? 0;
  const isDraftTask = toBooleanRuleValue(previewInput?.draft) === true;
  const configuredDraftCoefficient = toNullableNumber(matchedRule?.draftCoefficient);
  const effectiveDraftCoefficient = isDraftTask
    ? configuredDraftCoefficient ?? 1
    : 1;
  const estimatedCost =
    (billableTokens * unitPrice * effectiveDraftCoefficient) / MILLION;
  const conditionSummary = buildMediaTaskConditionSummary(matchedRule);
  const billingUnit = resolvePreviewBillingUnit(
    matchedRule?.billingUnit ?? matchedRule?.billing_unit,
    'total_tokens',
  );
  const previewImageCount = resolvePreviewImageCount(previewInput, 1);
  const previewLiveDuration = resolvePreviewLiveDuration(previewInput);
  const previewToolUsageCount = resolvePreviewCallCount(previewInput);
  const freeQuota = toNullableNumber(matchedRule?.freeQuota ?? matchedRule?.free_quota) ?? 0;
  const overageThreshold =
    toNullableNumber(
      matchedRule?.overageThreshold ?? matchedRule?.overage_threshold,
    ) ?? 1000;
  const overageUnits =
    Math.max(previewToolUsageCount - freeQuota, 0) / overageThreshold;
  const toolOveragePrice =
    toNullableNumber(
      matchedRule?.toolOveragePrice ?? matchedRule?.tool_overage_price,
    ) ?? unitPrice;
  let formulaSummary =
    minTokens > 0
      ? `max(${formatNumber(usageTotalTokens)}, ${formatNumber(minTokens)}) × ${formatNumber(unitPrice)} × ${formatNumber(effectiveDraftCoefficient)} / 1,000,000`
      : `${formatNumber(billableTokens)} × ${formatNumber(unitPrice)} × ${formatNumber(effectiveDraftCoefficient)} / 1,000,000`;
  let processSummary =
    usageTotalTokens < minTokens
      ? `usage_total_tokens ${formatNumber(usageTotalTokens)} 低于 min_tokens ${formatNumber(minTokens)}，按 (${formatNumber(billableTokens)} × ${formatNumber(unitPrice)} × ${formatNumber(effectiveDraftCoefficient)}) / 1,000,000 = ${formatNumber(estimatedCost)} 结算`
      : `usage_total_tokens ${formatNumber(usageTotalTokens)} × ${formatNumber(unitPrice)} × ${formatNumber(effectiveDraftCoefficient)} / 1,000,000 = ${formatNumber(estimatedCost)}`;
  let resolvedEstimatedCost = estimatedCost;
  const matchedSegmentPreview = serializeMediaTaskRule(matchedRule);

  switch (billingUnit) {
    case 'per_second':
      formulaSummary = buildPerSecondFormulaSummary(
        previewLiveDuration,
        unitPrice,
        effectiveDraftCoefficient,
      );
      resolvedEstimatedCost =
        previewLiveDuration * unitPrice * effectiveDraftCoefficient;
      processSummary = `${formulaSummary} = ${formatNumber(resolvedEstimatedCost)}`;
      break;
    case 'per_image':
      formulaSummary = buildPerImageFormulaSummary(
        previewImageCount,
        unitPrice,
        effectiveDraftCoefficient,
      );
      resolvedEstimatedCost =
        previewImageCount * unitPrice * effectiveDraftCoefficient;
      processSummary = `${formulaSummary} = ${formatNumber(resolvedEstimatedCost)}`;
      break;
    case 'per_1000_calls':
      formulaSummary = buildPer1000CallsFormulaSummary(
        previewToolUsageCount,
        freeQuota,
        toolOveragePrice,
        effectiveDraftCoefficient,
        overageThreshold,
      );
      resolvedEstimatedCost =
        overageUnits * toolOveragePrice * effectiveDraftCoefficient;
      processSummary = `tool_usage_count ${formatNumber(previewToolUsageCount)}, free_quota ${formatNumber(freeQuota)}, overage_threshold ${formatNumber(overageThreshold)}: ${formulaSummary} = ${formatNumber(resolvedEstimatedCost)}`;
      break;
    default:
      break;
  }

  return {
    matchedRule,
    matchedSegmentPreview,
    conditionSummary,
    formulaSummary,
    logPreview: {
      detailSummary: `命中媒体任务规则 ${matchedRule?.id || '-'}：${conditionSummary}`,
      processSummary,
    },
    priceSummary: {
      usageTotalTokens: formatNumber(usageTotalTokens),
      billableTokens: formatNumber(billableTokens),
      minTokens: formatNumber(minTokens),
      unitPrice: formatNumber(unitPrice),
      draftCoefficient: formatNumber(effectiveDraftCoefficient),
      estimatedCost: formatNumber(resolvedEstimatedCost),
      imageSizeTier: normalizeStringField(
        previewInput?.imageSizeTier,
      ).trim(),
      toolUsageType: normalizeStringField(
        previewInput?.toolUsageType,
      ).trim(),
      toolUsageCount: formatNumber(previewInput?.toolUsageCount),
      toolOveragePrice: formatNumber(
        matchedRule?.toolOveragePrice ?? matchedRule?.tool_overage_price,
      ),
      imageCount: formatNumber(previewImageCount),
      liveDurationSecs: formatNumber(previewLiveDuration),
      freeQuota: formatNumber(matchedRule?.freeQuota),
      overageThreshold: formatNumber(matchedRule?.overageThreshold),
      billingUnit: normalizeStringField(
        matchedRule?.billingUnit ?? matchedRule?.billing_unit,
      ).trim(),
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
    rawConfig.rule_type === MEDIA_TASK_RULE_TYPE ||
    rawConfig.ruleType === 'media-task'
      ? MEDIA_TASK_RULE_TYPE
      : TEXT_SEGMENT_RULE_TYPE;

  if (ruleType === MEDIA_TASK_RULE_TYPE) {
    const rawSegments = Array.isArray(rawConfig.rules)
      ? rawConfig.rules
      : Array.isArray(rawConfig.segments)
        ? rawConfig.segments
        : [];

    return {
      ruleType,
      displayName: normalizeStringField(
        rawConfig.displayName ?? rawConfig.display_name,
      ),
      taskType: normalizeStringField(rawConfig.taskType ?? rawConfig.task_type),
      billingUnit: normalizeStringField(
        rawConfig.billingUnit ?? rawConfig.billing_unit,
      ),
      note: normalizeStringField(rawConfig.note),
      rules: rawSegments.map((rule, index) => normalizeMediaTaskRule(rule, index)),
    };
  }

  const rawSegments = Array.isArray(rawConfig.rules)
    ? rawConfig.rules
    : Array.isArray(rawConfig.segments)
      ? rawConfig.segments
      : [];

  return {
    ...rawConfig,
    ruleType,
    displayName: normalizeStringField(
      rawConfig.displayName ?? rawConfig.display_name,
    ),
    segmentBasis: normalizeStringField(
      rawConfig.segmentBasis ?? rawConfig.segment_basis,
    ),
    billingUnit: normalizeStringField(
      rawConfig.billingUnit ?? rawConfig.billing_unit,
    ),
    defaultPrice: normalizeNumericField(
      rawConfig.defaultPrice ?? rawConfig.default_price,
    ),
    note: normalizeStringField(rawConfig.note),
    rules: rawSegments.map((rule, index) => normalizeTextSegmentRule(rule, index)),
  };
};

export const hasAdvancedPricingConfig = (config) =>
  Array.isArray(serializeAdvancedPricingConfig(config)?.segments) &&
  serializeAdvancedPricingConfig(config).segments.length > 0;

export const getAdvancedPricingValidationErrors = (config) => {
  const normalizedConfig = normalizeAdvancedPricingConfig(config);

  if (!hasAdvancedPricingConfig(normalizedConfig)) {
    return [];
  }
  if (normalizedConfig.ruleType === TEXT_SEGMENT_RULE_TYPE) {
    return validateTextSegmentRules(normalizedConfig.rules);
  }
  if (normalizedConfig.ruleType === MEDIA_TASK_RULE_TYPE) {
    return validateMediaTaskConfig(normalizedConfig);
  }
  return [];
};

export const getAdvancedPricingMapValidationErrors = (advancedPricingMap = {}) =>
  Object.entries(advancedPricingMap).reduce((result, [modelName, config]) => {
    const validationErrors = getAdvancedPricingValidationErrors(config);

    validationErrors.forEach((error) => {
      result.push(`${modelName}: ${error}`);
    });
    return result;
  }, []);

export const mergeAdvancedPricingDraftMap = (
  previousDraftMap = {},
  nextServerMap = {},
  dirtyModelNames = new Set(),
) => {
  const mergedDraftMap = {};
  const modelNames = new Set([
    ...Object.keys(previousDraftMap),
    ...Object.keys(nextServerMap),
  ]);

  modelNames.forEach((modelName) => {
    if (
      dirtyModelNames.has(modelName) &&
      previousDraftMap[modelName] !== undefined
    ) {
      mergedDraftMap[modelName] = previousDraftMap[modelName];
      return;
    }

    if (nextServerMap[modelName] !== undefined) {
      mergedDraftMap[modelName] = nextServerMap[modelName];
    }
  });

  return mergedDraftMap;
};

export const mergeAdvancedPricingModeDraftMap = (
  previousModeMap = {},
  nextServerModeMap = {},
  dirtyModelNames = new Set(),
) => {
  const mergedModeMap = {};
  const modelNames = new Set([
    ...Object.keys(previousModeMap),
    ...Object.keys(nextServerModeMap),
  ]);

  modelNames.forEach((modelName) => {
    if (
      dirtyModelNames.has(modelName) &&
      previousModeMap[modelName] !== undefined
    ) {
      mergedModeMap[modelName] = previousModeMap[modelName];
      return;
    }

    if (nextServerModeMap[modelName] !== undefined) {
      mergedModeMap[modelName] = nextServerModeMap[modelName];
    }
  });

  return mergedModeMap;
};

export const serializeAdvancedPricingMap = (advancedPricingMap = {}) =>
  Object.entries(advancedPricingMap).reduce((result, [modelName, config]) => {
    const serializedConfig = serializeAdvancedPricingConfig(config);

    if (Array.isArray(serializedConfig?.segments) && serializedConfig.segments.length > 0) {
      result[modelName] = serializedConfig;
    }

    return result;
  }, {});

export const buildAdvancedPricingSaveMaps = ({
  latestModeMap = {},
  latestRulesMap = {},
  draftModeMap = {},
  draftConfigMap = {},
  dirtyModelNames = [],
  fixedBillingModes = {},
}) => {
  const dirtySet =
    dirtyModelNames instanceof Set ? dirtyModelNames : new Set(dirtyModelNames);
  const modeMap = Object.entries(latestModeMap || {}).reduce(
    (result, [modelName, mode]) => {
      result[modelName] =
        mode === ADVANCED_PRICING_MODE_ADVANCED
          ? ADVANCED_PRICING_MODE_ADVANCED
          : normalizeFixedBillingMode(mode);
      return result;
    },
    {},
  );
  const rulesMap =
    latestRulesMap &&
    typeof latestRulesMap === 'object' &&
    !Array.isArray(latestRulesMap)
      ? { ...latestRulesMap }
      : {};

  Object.keys(modeMap).forEach((modelName) => {
    if (
      modeMap[modelName] === ADVANCED_PRICING_MODE_ADVANCED &&
      !canUseAdvancedPricingMode(rulesMap[modelName])
    ) {
      delete modeMap[modelName];
    }
  });

  dirtySet.forEach((modelName) => {
    const nextConfig = normalizeAdvancedPricingConfig(draftConfigMap[modelName]);

    if (hasAdvancedPricingConfig(nextConfig)) {
      rulesMap[modelName] = serializeAdvancedPricingConfig(nextConfig);
    } else {
      delete rulesMap[modelName];
    }

    modeMap[modelName] = getEffectiveBillingModeForModel({
      selectedMode: draftModeMap[modelName],
      fixedBillingMode: normalizeFixedBillingMode(fixedBillingModes[modelName]),
      advancedConfig: nextConfig,
    });
  });

  return {
    modeMap,
    rulesMap,
  };
};

export const buildAdvancedPricingConfigPayload = ({
  modeMap = {},
  rulesMap = {},
}) => ({
  billing_mode:
    modeMap && typeof modeMap === 'object' && !Array.isArray(modeMap)
      ? { ...modeMap }
      : {},
  rules:
    rulesMap && typeof rulesMap === 'object' && !Array.isArray(rulesMap)
      ? { ...rulesMap }
      : {},
});

const assertAdvancedPricingSaveResponse = (
  response,
  saveFailureMessage = 'Save failed',
) => {
  if (!response?.data?.success) {
    throw new Error(response?.data?.message || saveFailureMessage);
  }
};

export const saveAdvancedPricingOptions = async ({
  api,
  savePayload,
  saveFailureMessage = 'Save failed',
}) => {
  const configResponse = await api.put('/api/option/', {
    key: 'AdvancedPricingConfig',
    value: JSON.stringify(
      buildAdvancedPricingConfigPayload({
        modeMap: savePayload.modeMap,
        rulesMap: savePayload.rulesMap,
      }),
      null,
      2,
    ),
  });
  assertAdvancedPricingSaveResponse(configResponse, saveFailureMessage);
};

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
