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

  const start = segment.start ?? segment.from ?? segment.min ?? '';
  const end = segment.end ?? segment.to ?? segment.max ?? '';
  const price = segment.price ?? segment.value ?? '';

  return `${start}-${end}: ${price}`.trim();
};

const cloneRule = (rule) => {
  if (!rule || typeof rule !== 'object' || Array.isArray(rule)) {
    return {};
  }
  return JSON.parse(JSON.stringify(rule));
};

export const buildRuleDraft = (ruleType, rule = {}) => {
  const normalized = cloneRule(rule);
  const shouldPreserveTypeSpecificFields = normalized.rule_type === ruleType;
  const sharedFields = {
    display_name:
      typeof normalized.display_name === 'string' ? normalized.display_name : '',
    note: typeof normalized.note === 'string' ? normalized.note : '',
  };

  if (ruleType === RULE_TYPE_MEDIA_TASK) {
    return {
      ...sharedFields,
      rule_type: ruleType,
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
        shouldPreserveTypeSpecificFields && hasValue(normalized.unit_price)
          ? String(normalized.unit_price)
          : '',
    };
  }

  return {
    ...sharedFields,
    rule_type: ruleType,
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
      shouldPreserveTypeSpecificFields &&
      typeof normalized.segments_text === 'string'
        ? normalized.segments_text
        : shouldPreserveTypeSpecificFields && Array.isArray(normalized.segments)
          ? normalized.segments.map(formatSegmentLine).filter(Boolean).join('\n')
          : '',
  };
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

  return Array.from(names)
    .filter(Boolean)
    .map((name) => buildModelState(name, sourceMaps))
    .sort((a, b) => a.name.localeCompare(b.name));
};

export const buildAdvancedPricingDraftRules = ({
  models = [],
  previousDraftRules = {},
}) =>
  models.reduce((acc, model) => {
    const previousDraft = previousDraftRules[model.name];
    acc[model.name] =
      previousDraft && typeof previousDraft === 'object' && !Array.isArray(previousDraft)
        ? cloneRule(previousDraft)
        : buildRuleDraft(model.advancedRuleType || RULE_TYPE_TEXT_SEGMENT, model.rule);
    return acc;
  }, {});

export const buildAdvancedPricingDraftBillingModes = ({
  models = [],
  previousDraftBillingModes = {},
}) =>
  models.reduce((acc, model) => {
    acc[model.name] =
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
}) => {
  const models = buildAdvancedPricingModels({
    options,
    enabledModelNames,
    launchModelName,
  });

  return {
    models,
    draftRules: buildAdvancedPricingDraftRules({
      models,
      previousDraftRules,
    }),
    draftBillingModes: buildAdvancedPricingDraftBillingModes({
      models,
      previousDraftBillingModes,
    }),
    selectedModelName: resolveAdvancedPricingSelectedModelName({
      models,
      launchModelName,
      previousSelectedModelName,
    }),
    defaultBillingMode: BILLING_MODE_PER_TOKEN,
  };
};
