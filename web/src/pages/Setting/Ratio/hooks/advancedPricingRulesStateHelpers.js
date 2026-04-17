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

import {
  BILLING_MODE_PER_TOKEN,
  hasValue,
  resolveBillingMode,
} from './modelPricingEditorHelpers.js';

export const RULE_TYPE_TEXT_SEGMENT = 'text_segment';
export const RULE_TYPE_MEDIA_TASK = 'media_task';

export const parseOptionJSON = (rawValue) => {
  if (!rawValue || rawValue.trim() === '') {
    return {};
  }

  try {
    const parsed = JSON.parse(rawValue);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed
      : {};
  } catch (error) {
    console.error('Failed to parse advanced pricing JSON:', error);
    return {};
  }
};

export const reduceOptionsByKey = (items) =>
  Array.isArray(items)
    ? items.reduce((acc, item) => {
        if (item?.key) {
          acc[item.key] = item.value;
        }
        return acc;
      }, {})
    : {};

const formatSegmentLine = (segment) => {
  if (!segment || typeof segment !== 'object') {
    return '';
  }

  const start =
    segment.input_min ?? segment.start ?? segment.from ?? segment.min ?? '';
  const end =
    segment.input_max ?? segment.end ?? segment.to ?? segment.max ?? '';
  const price =
    segment.input_price ?? segment.price ?? segment.value ?? '';

  return `${start}-${end}: ${price}`.trim();
};

const cloneRule = (rule) => {
  if (!rule || typeof rule !== 'object' || Array.isArray(rule)) {
    return {};
  }
  return JSON.parse(JSON.stringify(rule));
};

const DRAFT_ROUND_TRIP_MODE = '__roundtrip_mode';
const DRAFT_ORIGINAL_CANONICAL_RULE = '__original_canonical_rule';
const DRAFT_ORIGINAL_SHELL_RULE = '__original_shell_rule';
const ROUND_TRIP_MODE_PRESERVE_CANONICAL = 'preserve_canonical';

const TEXT_SEGMENT_METADATA_FIELDS = [
  'display_name',
  'segment_basis',
  'billing_unit',
  'default_price',
  'note',
];
const MEDIA_TASK_METADATA_FIELDS = [
  'display_name',
  'task_type',
  'billing_unit',
  'note',
];
const TEXT_SEGMENT_UNSAFE_SHELL_FIELDS = ['segments_text'];
const MEDIA_TASK_UNSAFE_SHELL_FIELDS = ['unit_price'];

const valueToComparableString = (value) =>
  value === null || value === undefined ? '' : String(value);

const hasOnlyAllowedKeys = (value, allowedKeys) =>
  Object.keys(value).every((key) => allowedKeys.has(key));

const hasScalarDraftValue = (value) =>
  value !== null && value !== undefined && String(value).trim() !== '';

const extractTextSegmentShellFields = (segment = {}) => ({
  start: segment.input_min ?? segment.start ?? segment.from ?? segment.min,
  end: segment.input_max ?? segment.end ?? segment.to ?? segment.max,
  price: segment.input_price ?? segment.price ?? segment.value,
});

const isTextSegmentShellCompatible = (segment) => {
  if (!segment || typeof segment !== 'object' || Array.isArray(segment)) {
    return false;
  }

  const allowedKeys = new Set([
    'priority',
    'input_min',
    'input_max',
    'input_price',
    'start',
    'end',
    'from',
    'to',
    'min',
    'max',
    'price',
    'value',
  ]);
  const { start, end, price } = extractTextSegmentShellFields(segment);

  return (
    hasOnlyAllowedKeys(segment, allowedKeys) &&
    hasScalarDraftValue(start) &&
    hasScalarDraftValue(end) &&
    hasScalarDraftValue(price)
  );
};

const isMediaTaskSegmentShellCompatible = (segment) => {
  if (!segment || typeof segment !== 'object' || Array.isArray(segment)) {
    return false;
  }

  const allowedKeys = new Set(['priority', 'unit_price', 'remark']);

  return hasOnlyAllowedKeys(segment, allowedKeys) && hasScalarDraftValue(segment.unit_price);
};

const isTextRuleShellCompatible = (rule) => {
  if (!rule || typeof rule !== 'object' || Array.isArray(rule)) {
    return false;
  }

  const allowedKeys = new Set([
    'rule_type',
    'display_name',
    'segment_basis',
    'billing_unit',
    'default_price',
    'note',
    'segments',
  ]);

  return (
    hasOnlyAllowedKeys(rule, allowedKeys) &&
    Array.isArray(rule.segments) &&
    rule.segments.length > 0 &&
    rule.segments.every(isTextSegmentShellCompatible)
  );
};

const isMediaRuleShellCompatible = (rule) => {
  if (!rule || typeof rule !== 'object' || Array.isArray(rule)) {
    return false;
  }

  const allowedKeys = new Set([
    'rule_type',
    'display_name',
    'task_type',
    'billing_unit',
    'note',
    'unit_price',
    'segments',
  ]);

  if (!hasOnlyAllowedKeys(rule, allowedKeys)) {
    return false;
  }

  if (Array.isArray(rule.segments)) {
    return rule.segments.length === 1 && rule.segments.every(isMediaTaskSegmentShellCompatible);
  }

  return hasScalarDraftValue(rule.unit_price);
};

const shouldPreserveCanonicalShellState = (ruleType, rule) => {
  if (!rule || typeof rule !== 'object' || Array.isArray(rule)) {
    return false;
  }
  if (rule.rule_type !== ruleType) {
    return false;
  }

  if (ruleType === RULE_TYPE_MEDIA_TASK) {
    return Array.isArray(rule.segments) && !isMediaRuleShellCompatible(rule);
  }

  return Array.isArray(rule.segments) && !isTextRuleShellCompatible(rule);
};

const getDraftRoundTripState = (rule = {}) => {
  if (
    rule?.[DRAFT_ROUND_TRIP_MODE] !== ROUND_TRIP_MODE_PRESERVE_CANONICAL ||
    !rule?.[DRAFT_ORIGINAL_CANONICAL_RULE] ||
    !rule?.[DRAFT_ORIGINAL_SHELL_RULE]
  ) {
    return null;
  }

  const originalRule = cloneRule(rule[DRAFT_ORIGINAL_CANONICAL_RULE]);
  const originalShell = cloneRule(rule[DRAFT_ORIGINAL_SHELL_RULE]);

  if (!originalRule.rule_type || !originalShell.rule_type) {
    return null;
  }

  return {
    originalRule,
    originalShell,
  };
};

const withDraftRoundTripState = (draftRule, roundTripState) => {
  if (!roundTripState) {
    return draftRule;
  }

  return {
    ...draftRule,
    [DRAFT_ROUND_TRIP_MODE]: ROUND_TRIP_MODE_PRESERVE_CANONICAL,
    [DRAFT_ORIGINAL_CANONICAL_RULE]: cloneRule(roundTripState.originalRule),
    [DRAFT_ORIGINAL_SHELL_RULE]: cloneRule(roundTripState.originalShell),
  };
};

const shouldRetainDraftRoundTripState = (roundTripState, ruleType) =>
  Boolean(
    roundTripState &&
      roundTripState.originalRule.rule_type === ruleType &&
      roundTripState.originalShell.rule_type === ruleType,
  );

const buildSharedDraftFields = (normalized, firstSegment) => ({
  display_name:
    typeof normalized.display_name === 'string' ? normalized.display_name : '',
  note:
    typeof normalized.note === 'string'
      ? normalized.note
      : typeof firstSegment?.remark === 'string'
        ? firstSegment.remark
        : '',
});

const buildMediaTaskDraftFields = (normalized, shouldPreserveTypeSpecificFields) => {
  const firstSegment = Array.isArray(normalized.segments) ? normalized.segments[0] : null;
  const normalizedUnitPrice = normalized.unit_price ?? firstSegment?.unit_price;

  return {
    ...buildSharedDraftFields(normalized, firstSegment),
    rule_type: RULE_TYPE_MEDIA_TASK,
    task_type:
      shouldPreserveTypeSpecificFields &&
      typeof normalized.task_type === 'string' &&
      normalized.task_type
        ? normalized.task_type
        : 'image_generation',
    billing_unit:
      shouldPreserveTypeSpecificFields &&
      typeof normalized.billing_unit === 'string' &&
      normalized.billing_unit
        ? normalized.billing_unit
        : 'task',
    unit_price:
      shouldPreserveTypeSpecificFields && hasValue(normalizedUnitPrice)
        ? String(normalizedUnitPrice)
        : '',
  };
};

const buildTextSegmentDraftFields = (normalized, shouldPreserveTypeSpecificFields) => ({
  ...buildSharedDraftFields(
    normalized,
    Array.isArray(normalized.segments) ? normalized.segments[0] : null,
  ),
  rule_type: RULE_TYPE_TEXT_SEGMENT,
  segment_basis:
    shouldPreserveTypeSpecificFields &&
    typeof normalized.segment_basis === 'string' &&
    normalized.segment_basis
      ? normalized.segment_basis
      : 'token',
  billing_unit:
    shouldPreserveTypeSpecificFields &&
    typeof normalized.billing_unit === 'string' &&
    normalized.billing_unit
      ? normalized.billing_unit
      : '1K tokens',
  default_price:
    shouldPreserveTypeSpecificFields && hasValue(normalized.default_price)
      ? String(normalized.default_price)
      : '',
  segments_text:
    shouldPreserveTypeSpecificFields && typeof normalized.segments_text === 'string'
      ? normalized.segments_text
      : shouldPreserveTypeSpecificFields && Array.isArray(normalized.segments)
        ? normalized.segments.map(formatSegmentLine).filter(Boolean).join('\n')
        : '',
});

export const buildRuleDraft = (ruleType, rule = {}) => {
  const normalized = cloneRule(rule);
  const roundTripState = getDraftRoundTripState(normalized);
  const shouldPreserveTypeSpecificFields = normalized.rule_type === ruleType;
  const nextDraft =
    ruleType === RULE_TYPE_MEDIA_TASK
      ? buildMediaTaskDraftFields(normalized, shouldPreserveTypeSpecificFields)
      : buildTextSegmentDraftFields(normalized, shouldPreserveTypeSpecificFields);

  if (shouldRetainDraftRoundTripState(roundTripState, ruleType)) {
    return withDraftRoundTripState(nextDraft, roundTripState);
  }

  if (shouldPreserveCanonicalShellState(ruleType, normalized)) {
    return withDraftRoundTripState(nextDraft, {
      originalRule: normalized,
      originalShell: nextDraft,
    });
  }

  return nextDraft;
};

const buildModelState = (name, sourceMaps) => {
  const rawRule = sourceMaps.AdvancedPricingRules[name];
  const advancedRuleType =
    typeof rawRule?.rule_type === 'string' ? rawRule.rule_type : '';
  const billingModeState = resolveBillingMode({
    explicitMode: sourceMaps.AdvancedPricingMode[name],
    fixedPrice: sourceMaps.ModelPrice[name],
    advancedRuleType,
  });

  return {
    name,
    billingMode: billingModeState.billingMode,
    explicitBillingMode: billingModeState.explicitBillingMode,
    hasExplicitBillingMode: billingModeState.hasExplicitBillingMode,
    advancedRuleType,
    rule: cloneRule(rawRule),
    hasBasePricing:
      hasValue(sourceMaps.ModelPrice[name]) || hasValue(sourceMaps.ModelRatio[name]),
  };
};

export const buildAdvancedPricingModels = ({
  options = {},
  enabledModelNames = [],
  launchModelName = '',
  previousSelectedModelName = '',
}) => {
  const sourceMaps = {
    AdvancedPricingMode: parseOptionJSON(options.AdvancedPricingMode),
    AdvancedPricingRules: parseOptionJSON(options.AdvancedPricingRules),
    ModelPrice: parseOptionJSON(options.ModelPrice),
    ModelRatio: parseOptionJSON(options.ModelRatio),
  };

  const names = new Set([
    ...enabledModelNames,
    ...Object.keys(sourceMaps.AdvancedPricingMode),
    ...Object.keys(sourceMaps.AdvancedPricingRules),
    ...Object.keys(sourceMaps.ModelPrice),
    ...Object.keys(sourceMaps.ModelRatio),
  ]);

  if (launchModelName) {
    names.add(launchModelName);
  }
  if (previousSelectedModelName) {
    names.add(previousSelectedModelName);
  }

  return Array.from(names)
    .filter(Boolean)
    .map((name) => buildModelState(name, sourceMaps))
    .sort((a, b) => a.name.localeCompare(b.name));
};

const shouldPreservePreviousValue = (preserveModelNames, modelName) => {
  if (!modelName || !preserveModelNames) {
    return false;
  }

  if (preserveModelNames instanceof Set) {
    return preserveModelNames.has(modelName);
  }

  if (Array.isArray(preserveModelNames)) {
    return preserveModelNames.includes(modelName);
  }

  if (typeof preserveModelNames === 'object') {
    return Boolean(preserveModelNames[modelName]);
  }

  return false;
};

export const buildAdvancedPricingDraftRules = ({
  models = [],
  previousDraftRules = {},
  preserveDraftRuleModelNames = null,
}) =>
  models.reduce((acc, model) => {
    const previousDraft = previousDraftRules[model.name];
    acc[model.name] =
      shouldPreservePreviousValue(preserveDraftRuleModelNames, model.name) &&
      previousDraft && typeof previousDraft === 'object' && !Array.isArray(previousDraft)
        ? cloneRule(previousDraft)
        : buildRuleDraft(model.advancedRuleType || RULE_TYPE_TEXT_SEGMENT, model.rule);
    return acc;
  }, {});

export const buildAdvancedPricingDraftBillingModes = ({
  models = [],
  previousDraftBillingModes = {},
  preserveDraftBillingModeModelNames = null,
}) =>
  models.reduce((acc, model) => {
    acc[model.name] =
      shouldPreservePreviousValue(
        preserveDraftBillingModeModelNames,
        model.name,
      ) &&
      typeof previousDraftBillingModes[model.name] === 'string'
        ? previousDraftBillingModes[model.name]
        : model.billingMode;
    return acc;
  }, {});

export const resolveAdvancedPricingSelectedModelName = ({
  models = [],
  launchModelName = '',
  previousSelectedModelName = '',
}) => {
  if (launchModelName && models.some((model) => model.name === launchModelName)) {
    return launchModelName;
  }
  if (
    previousSelectedModelName &&
    models.some((model) => model.name === previousSelectedModelName)
  ) {
    return previousSelectedModelName;
  }
  return models[0]?.name || '';
};

export const buildAdvancedPricingState = ({
  options = {},
  enabledModelNames = [],
  launchModelName = '',
  previousDraftRules = {},
  previousDraftBillingModes = {},
  previousSelectedModelName = '',
  preserveDraftRuleModelNames = null,
  preserveDraftBillingModeModelNames = null,
}) => {
  const models = buildAdvancedPricingModels({
    options,
    enabledModelNames,
    launchModelName,
    previousSelectedModelName,
  });

  return {
    models,
    draftRules: buildAdvancedPricingDraftRules({
      models,
      previousDraftRules,
      preserveDraftRuleModelNames,
    }),
    draftBillingModes: buildAdvancedPricingDraftBillingModes({
      models,
      previousDraftBillingModes,
      preserveDraftBillingModeModelNames,
    }),
    selectedModelName: resolveAdvancedPricingSelectedModelName({
      models,
      launchModelName,
      previousSelectedModelName,
    }),
    defaultBillingMode: BILLING_MODE_PER_TOKEN,
  };
};

const parseAdvancedPricingNumber = (value) => {
  if (value === null || value === undefined) {
    return null;
  }
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : null;
  }
  if (typeof value !== 'string') {
    return null;
  }
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : null;
};

const parseAdvancedPricingInteger = (value) => {
  const parsed = parseAdvancedPricingNumber(value);
  if (parsed === null || !Number.isInteger(parsed)) {
    return null;
  }
  return parsed;
};

const parseAdvancedPricingString = (value) => {
  if (typeof value !== 'string') {
    return '';
  }
  return value.trim();
};

const parseTextSegmentLine = (line, index) => {
  const trimmed = line.trim();
  if (!trimmed) {
    return null;
  }

  const matches = trimmed.match(/^(\d+)\s*-\s*(\d+)\s*:\s*(-?\d+(?:\.\d+)?)$/);
  if (!matches) {
    throw new Error(`Invalid advanced pricing segment on line ${index + 1}`);
  }

  const start = parseAdvancedPricingInteger(matches[1]);
  const end = parseAdvancedPricingInteger(matches[2]);
  const price = parseAdvancedPricingNumber(matches[3]);

  if (start === null || end === null || price === null) {
    throw new Error(`Invalid advanced pricing segment on line ${index + 1}`);
  }
  if (end < start) {
    throw new Error(`Advanced pricing segment end must be >= start on line ${index + 1}`);
  }
  if (price < 0) {
    throw new Error(`Advanced pricing segment price cannot be negative on line ${index + 1}`);
  }

  return {
    priority: (index + 1) * 10,
    input_min: start,
    input_max: end,
    input_price: price,
  };
};

const getUnsupportedRoundTripError = (ruleType) =>
  ruleType === RULE_TYPE_MEDIA_TASK
    ? 'Advanced media task pricing rule cannot safely round-trip through the simplified editor. Only metadata fields can be edited.'
    : 'Advanced text pricing rule cannot safely round-trip through the simplified editor. Only metadata fields can be edited.';

const assertUnsafeShellFieldsUnchanged = ({
  draftRule,
  originalShell,
  ruleType,
}) => {
  if (originalShell.rule_type !== ruleType) {
    throw new Error(getUnsupportedRoundTripError(ruleType));
  }

  const unsafeFields =
    ruleType === RULE_TYPE_MEDIA_TASK
      ? MEDIA_TASK_UNSAFE_SHELL_FIELDS
      : TEXT_SEGMENT_UNSAFE_SHELL_FIELDS;

  for (const field of unsafeFields) {
    if (
      valueToComparableString(draftRule[field]) !==
      valueToComparableString(originalShell[field])
    ) {
      throw new Error(getUnsupportedRoundTripError(ruleType));
    }
  }
};

const applyChangedStringMetadata = ({
  targetRule,
  draftRule,
  originalShell,
  field,
}) => {
  if (
    valueToComparableString(draftRule[field]) ===
    valueToComparableString(originalShell[field])
  ) {
    return;
  }

  const normalizedValue = parseAdvancedPricingString(draftRule[field]);

  if (normalizedValue) {
    targetRule[field] = normalizedValue;
    return;
  }

  delete targetRule[field];
};

const applyChangedNumberMetadata = ({
  targetRule,
  draftRule,
  originalShell,
  field,
}) => {
  if (
    valueToComparableString(draftRule[field]) ===
    valueToComparableString(originalShell[field])
  ) {
    return;
  }

  const normalizedValue = parseAdvancedPricingNumber(draftRule[field]);

  if (normalizedValue !== null) {
    targetRule[field] = normalizedValue;
    return;
  }

  delete targetRule[field];
};

const mergeMetadataIntoPreservedCanonicalRule = ({
  draftRule,
  originalRule,
  originalShell,
  ruleType,
}) => {
  const mergedRule = cloneRule(originalRule);
  const stringFields =
    ruleType === RULE_TYPE_MEDIA_TASK
      ? MEDIA_TASK_METADATA_FIELDS
      : TEXT_SEGMENT_METADATA_FIELDS.filter((field) => field !== 'default_price');

  stringFields.forEach((field) =>
    applyChangedStringMetadata({
      targetRule: mergedRule,
      draftRule,
      originalShell,
      field,
    }),
  );

  if (ruleType === RULE_TYPE_TEXT_SEGMENT) {
    applyChangedNumberMetadata({
      targetRule: mergedRule,
      draftRule,
      originalShell,
      field: 'default_price',
    });
  }

  return mergedRule;
};

export const normalizeAdvancedPricingDraftRule = (draftRule = {}) => {
  const ruleType =
    typeof draftRule?.rule_type === 'string' && draftRule.rule_type
      ? draftRule.rule_type
      : RULE_TYPE_TEXT_SEGMENT;
  const roundTripState = getDraftRoundTripState(draftRule);

  if (roundTripState) {
    if (roundTripState.originalRule.rule_type !== ruleType) {
      throw new Error(getUnsupportedRoundTripError(ruleType));
    }

    assertUnsafeShellFieldsUnchanged({
      draftRule,
      originalShell: roundTripState.originalShell,
      ruleType,
    });

    return mergeMetadataIntoPreservedCanonicalRule({
      draftRule,
      originalRule: roundTripState.originalRule,
      originalShell: roundTripState.originalShell,
      ruleType,
    });
  }

  const displayName = parseAdvancedPricingString(draftRule.display_name);
  const billingUnit = parseAdvancedPricingString(draftRule.billing_unit);
  const note = parseAdvancedPricingString(draftRule.note);

  if (ruleType === RULE_TYPE_MEDIA_TASK) {
    const unitPrice = parseAdvancedPricingNumber(draftRule.unit_price);
    const taskType = parseAdvancedPricingString(draftRule.task_type);
    if (unitPrice === null) {
      throw new Error('Advanced media task pricing requires unit_price');
    }
    if (unitPrice < 0) {
      throw new Error('Advanced media task unit_price cannot be negative');
    }

    const segment = {
      priority: 10,
      unit_price: unitPrice,
    };
    if (typeof draftRule.note === 'string' && draftRule.note.trim()) {
      segment.remark = note;
    }

    const normalizedRule = {
      rule_type: RULE_TYPE_MEDIA_TASK,
      segments: [segment],
    };
    if (displayName) {
      normalizedRule.display_name = displayName;
    }
    if (taskType) {
      normalizedRule.task_type = taskType;
    }
    if (billingUnit) {
      normalizedRule.billing_unit = billingUnit;
    }
    if (note) {
      normalizedRule.note = note;
    }

    return normalizedRule;
  }

  const rawLines =
    typeof draftRule.segments_text === 'string'
      ? draftRule.segments_text
          .split('\n')
          .map((line) => line.trim())
          .filter(Boolean)
      : [];

  if (rawLines.length === 0) {
    throw new Error('Advanced text pricing requires at least one segment');
  }

  const normalizedRule = {
    rule_type: RULE_TYPE_TEXT_SEGMENT,
    segments: rawLines
      .map((line, index) => parseTextSegmentLine(line, index))
      .filter(Boolean),
  };
  const segmentBasis = parseAdvancedPricingString(draftRule.segment_basis);
  const defaultPrice = parseAdvancedPricingNumber(draftRule.default_price);
  if (displayName) {
    normalizedRule.display_name = displayName;
  }
  if (segmentBasis) {
    normalizedRule.segment_basis = segmentBasis;
  }
  if (billingUnit) {
    normalizedRule.billing_unit = billingUnit;
  }
  if (defaultPrice !== null) {
    normalizedRule.default_price = defaultPrice;
  }
  if (note) {
    normalizedRule.note = note;
  }

  return normalizedRule;
};

export const buildAdvancedPricingSavePayload = ({
  modelName = '',
  billingMode = BILLING_MODE_PER_TOKEN,
  draftRule = {},
  latestModeMap = {},
  latestRulesMap = {},
}) => {
  if (!modelName) {
    throw new Error('Advanced pricing save requires a model name');
  }

  const normalizedRule = normalizeAdvancedPricingDraftRule(draftRule);
  const nextModeMap = {
    ...latestModeMap,
    [modelName]: billingMode,
  };
  const nextRulesMap = {
    ...latestRulesMap,
    [modelName]: normalizedRule,
  };

  return {
    normalizedRule,
    previewPayload: {
      AdvancedPricingMode: {
        [modelName]: billingMode,
      },
      AdvancedPricingRules: {
        [modelName]: normalizedRule,
      },
    },
    optionValue: JSON.stringify(
      {
        billing_mode: nextModeMap,
        rules: nextRulesMap,
      },
      null,
      2,
    ),
  };
};
