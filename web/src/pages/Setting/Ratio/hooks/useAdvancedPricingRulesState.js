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

import { useEffect, useMemo, useState } from 'react';

import { API, showError, showSuccess } from '../../../../helpers';
import {
  BILLING_MODE_PER_TOKEN,
  hasValue,
  resolveBillingMode,
} from './modelPricingEditorHelpers';

const RULE_TYPE_TEXT_SEGMENT = 'text_segment';
const RULE_TYPE_MEDIA_TASK = 'media_task';

const parseOptionJSON = (rawValue) => {
  if (!rawValue || rawValue.trim() === '') {
    return {};
  }

  try {
    const parsed = JSON.parse(rawValue);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed
      : {};
  } catch (error) {
    console.error('高级定价规则 JSON 解析失败:', error);
    return {};
  }
};

const reduceOptionsByKey = (items) =>
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

const buildRuleDraft = (ruleType, rule = {}) => {
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

export default function useAdvancedPricingRulesState({
  options,
  refresh,
  t,
  initialModelName = '',
  initialModelSelectionKey = 0,
}) {
  const [enabledModelNames, setEnabledModelNames] = useState([]);
  const [launchModelName, setLaunchModelName] = useState('');
  const [models, setModels] = useState([]);
  const [searchText, setSearchText] = useState('');
  const [selectedModelName, setSelectedModelName] = useState('');
  const [draftRules, setDraftRules] = useState({});
  const [draftBillingModes, setDraftBillingModes] = useState({});
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let active = true;

    const loadEnabledModels = async () => {
      try {
        const res = await API.get('/api/channel/models_enabled');
        const { success, message, data } = res?.data || {};
        if (!active) {
          return;
        }
        if (success) {
          setEnabledModelNames(Array.isArray(data) ? data.filter(Boolean) : []);
          return;
        }
        showError(message || 'Failed to load enabled models');
        setEnabledModelNames([]);
      } catch (error) {
        if (!active) {
          return;
        }
        console.error('Failed to load enabled models:', error);
        showError('Failed to load enabled models');
        setEnabledModelNames([]);
      }
    };

    loadEnabledModels();

    return () => {
      active = false;
    };
  }, [t]);

  useEffect(() => {
    if (!initialModelSelectionKey) {
      return;
    }
    setLaunchModelName(initialModelName || '');
  }, [initialModelName, initialModelSelectionKey]);

  useEffect(() => {
    if (launchModelName && selectedModelName === launchModelName) {
      setLaunchModelName('');
    }
  }, [launchModelName, selectedModelName]);

  useEffect(() => {
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

    const nextModels = Array.from(names)
      .filter(Boolean)
      .map((name) => buildModelState(name, sourceMaps))
      .sort((a, b) => a.name.localeCompare(b.name));

    setModels(nextModels);
    setDraftRules(
      nextModels.reduce((acc, model) => {
        acc[model.name] = buildRuleDraft(
          model.advancedRuleType || RULE_TYPE_TEXT_SEGMENT,
          model.rule,
        );
        return acc;
      }, {}),
    );
    setDraftBillingModes(
      nextModels.reduce((acc, model) => {
        acc[model.name] = model.billingMode;
        return acc;
      }, {}),
    );
    setSelectedModelName((previous) => {
      if (launchModelName && nextModels.some((model) => model.name === launchModelName)) {
        return launchModelName;
      }
      if (previous && nextModels.some((model) => model.name === previous)) {
        return previous;
      }
      return nextModels[0]?.name || '';
    });
  }, [enabledModelNames, launchModelName, options]);

  const filteredModels = useMemo(() => {
    const keyword = searchText.trim().toLowerCase();

    return models.filter((model) => {
      if (!keyword) {
        return true;
      }
      return model.name.toLowerCase().includes(keyword);
    });
  }, [models, searchText]);

  const selectedModel = useMemo(
    () => models.find((model) => model.name === selectedModelName) || null,
    [models, selectedModelName],
  );

  const selectedRule =
    draftRules[selectedModelName] || buildRuleDraft(RULE_TYPE_TEXT_SEGMENT);
  const currentRuleType = selectedRule.rule_type || RULE_TYPE_TEXT_SEGMENT;
  const currentBillingMode = selectedModel?.billingMode || BILLING_MODE_PER_TOKEN;
  const draftBillingMode =
    draftBillingModes[selectedModelName] || currentBillingMode;

  const previewPayload = selectedModel
    ? {
        AdvancedPricingMode: {
          [selectedModel.name]: draftBillingMode,
        },
        AdvancedPricingRules: {
          [selectedModel.name]: selectedRule,
        },
      }
    : null;

  const updateSelectedRuleType = (ruleType) => {
    if (!selectedModelName) {
      return;
    }

    setDraftRules((previous) => ({
      ...previous,
      [selectedModelName]: buildRuleDraft(ruleType, previous[selectedModelName]),
    }));
  };

  const updateSelectedRuleField = (field, value) => {
    if (!selectedModelName) {
      return;
    }

    setDraftRules((previous) => ({
      ...previous,
      [selectedModelName]: {
        ...buildRuleDraft(currentRuleType, previous[selectedModelName]),
        [field]: value,
      },
    }));
  };

  const updateSelectedBillingMode = (billingMode) => {
    if (!selectedModelName) {
      return;
    }

    setDraftBillingModes((previous) => ({
      ...previous,
      [selectedModelName]: billingMode,
    }));
  };

  const saveSelectedRule = async () => {
    if (!selectedModel) {
      showError(t('请先选择一个模型'));
      return false;
    }

    setSaving(true);
    try {
      const latestOptionsRes = await API.get('/api/option/');
      const {
        success: latestOptionsSuccess,
        message: latestOptionsMessage,
        data: latestOptionsData,
      } = latestOptionsRes?.data || {};

      if (!latestOptionsSuccess) {
        throw new Error(latestOptionsMessage || t('获取配置失败'));
      }

      const latestOptionsByKey = reduceOptionsByKey(latestOptionsData);
      const nextModeMap = parseOptionJSON(latestOptionsByKey.AdvancedPricingMode);
      const nextRulesMap = parseOptionJSON(latestOptionsByKey.AdvancedPricingRules);

      nextModeMap[selectedModel.name] = draftBillingMode;
      nextRulesMap[selectedModel.name] = selectedRule;

      const results = await Promise.all([
        API.put('/api/option/', {
          key: 'AdvancedPricingMode',
          value: JSON.stringify(nextModeMap, null, 2),
        }),
        API.put('/api/option/', {
          key: 'AdvancedPricingRules',
          value: JSON.stringify(nextRulesMap, null, 2),
        }),
      ]);

      for (const res of results) {
        if (!res?.data?.success) {
          throw new Error(res?.data?.message || t('保存失败，请重试'));
        }
      }

      showSuccess(t('高级定价规则已保存'));
      await refresh();
      return true;
    } catch (error) {
      console.error('保存高级定价规则失败:', error);
      showError(error.message || t('保存失败，请重试'));
      return false;
    } finally {
      setSaving(false);
    }
  };

  return {
    models,
    filteredModels,
    searchText,
    setSearchText,
    selectedModel,
    selectedModelName,
    setSelectedModelName,
    selectedRule,
    currentRuleType,
    currentBillingMode,
    draftBillingMode,
    updateSelectedRuleType,
    updateSelectedRuleField,
    updateSelectedBillingMode,
    previewPayload,
    saveSelectedRule,
    saving,
  };
}
