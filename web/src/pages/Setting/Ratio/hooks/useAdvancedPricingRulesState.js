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
  ADVANCED_PRICING_MODE_ADVANCED,
  ADVANCED_PRICING_MODE_FIXED,
  FIXED_BILLING_MODE_PER_REQUEST,
  FIXED_BILLING_MODE_PER_TOKEN,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
  buildTextSegmentPreview,
  getAdvancedRuleType,
  getEffectiveBillingModeForModel,
  getFixedBillingModeForModel,
  hasAdvancedPricingConfig,
  hasFixedPricingConfig,
  normalizeAdvancedPricingConfig,
  parseOptionJSON,
  validateTextSegmentRules,
} from './advancedPricingRuleHelpers';

const EMPTY_PREVIEW_INPUT = {
  inputTokens: '',
  outputTokens: '',
};

const MODEL_OPTION_KEYS = [
  'ModelPrice',
  'ModelRatio',
  'CompletionRatio',
  'CacheRatio',
  'CreateCacheRatio',
  'ImageRatio',
  'AudioRatio',
  'AudioCompletionRatio',
];

const NUMERIC_INPUT_REGEX = /^\d*$/;

const buildSourceMaps = (options = {}) =>
  MODEL_OPTION_KEYS.reduce((result, key) => {
    result[key] = parseOptionJSON(options[key]);
    return result;
  }, {});

const buildAdvancedPricingMap = (options = {}) => {
  const parsedValue = parseOptionJSON(options.AdvancedPricingRules);

  return Object.entries(parsedValue).reduce((result, [modelName, config]) => {
    result[modelName] = normalizeAdvancedPricingConfig(config);
    return result;
  }, {});
};

const buildBillingModeMap = (options = {}) => {
  const parsedValue = parseOptionJSON(options.ModelBillingMode);

  return Object.entries(parsedValue).reduce((result, [modelName, mode]) => {
    if (mode === ADVANCED_PRICING_MODE_ADVANCED) {
      result[modelName] = ADVANCED_PRICING_MODE_ADVANCED;
    }
    return result;
  }, {});
};

const serializeAdvancedPricingMap = (advancedPricingMap) =>
  Object.entries(advancedPricingMap).reduce((result, [modelName, config]) => {
    const normalizedConfig = normalizeAdvancedPricingConfig(config);
    if (
      normalizedConfig.ruleType !== TEXT_SEGMENT_RULE_TYPE ||
      normalizedConfig.rules.length > 0
    ) {
      result[modelName] = normalizedConfig;
    }
    return result;
  }, {});

export function useAdvancedPricingRulesState({
  options,
  refresh,
  t,
  candidateModelNames = [],
  selectedModelName: externalSelectedModelName = '',
  onSelectedModelChange,
}) {
  const [selectedModelName, setSelectedModelName] = useState(
    externalSelectedModelName || '',
  );
  const [modelSearchText, setModelSearchText] = useState('');
  const [loading, setLoading] = useState(false);
  const [previewInput, setPreviewInput] = useState(EMPTY_PREVIEW_INPUT);
  const [modelBillingModeMap, setModelBillingModeMap] = useState({});
  const [advancedPricingMap, setAdvancedPricingMap] = useState({});

  const sourceMaps = useMemo(() => buildSourceMaps(options), [options]);

  useEffect(() => {
    setModelBillingModeMap(buildBillingModeMap(options));
    setAdvancedPricingMap(buildAdvancedPricingMap(options));
  }, [options]);

  const models = useMemo(() => {
    const modelNames = new Set(candidateModelNames);

    MODEL_OPTION_KEYS.forEach((key) => {
      Object.keys(sourceMaps[key] || {}).forEach((modelName) => {
        modelNames.add(modelName);
      });
    });

    Object.keys(modelBillingModeMap).forEach((modelName) => {
      modelNames.add(modelName);
    });
    Object.keys(advancedPricingMap).forEach((modelName) => {
      modelNames.add(modelName);
    });

    return Array.from(modelNames)
      .sort((leftName, rightName) => leftName.localeCompare(rightName))
      .map((modelName) => {
        const fixedBillingMode = getFixedBillingModeForModel(modelName, sourceMaps);
        const advancedConfig = normalizeAdvancedPricingConfig(
          advancedPricingMap[modelName],
        );

        return {
          name: modelName,
          fixedBillingMode,
          selectedMode:
            modelBillingModeMap[modelName] === ADVANCED_PRICING_MODE_ADVANCED
              ? ADVANCED_PRICING_MODE_ADVANCED
              : ADVANCED_PRICING_MODE_FIXED,
          effectiveMode: getEffectiveBillingModeForModel({
            selectedMode: modelBillingModeMap[modelName],
            fixedBillingMode,
          }),
          hasFixedPricing: hasFixedPricingConfig(modelName, sourceMaps),
          hasAdvancedPricing: hasAdvancedPricingConfig(advancedConfig),
          ruleType: getAdvancedRuleType(advancedConfig),
          advancedConfig,
        };
      });
  }, [advancedPricingMap, candidateModelNames, modelBillingModeMap, sourceMaps]);

  const filteredModels = useMemo(() => {
    const keyword = modelSearchText.trim().toLowerCase();
    if (!keyword) {
      return models;
    }
    return models.filter((model) =>
      model.name.toLowerCase().includes(keyword),
    );
  }, [modelSearchText, models]);

  const selectedModel = useMemo(
    () => models.find((model) => model.name === selectedModelName) || null,
    [models, selectedModelName],
  );

  const selectedAdvancedConfig = useMemo(
    () =>
      selectedModel
        ? normalizeAdvancedPricingConfig(advancedPricingMap[selectedModel.name])
        : normalizeAdvancedPricingConfig(null),
    [advancedPricingMap, selectedModel],
  );

  const validationErrors = useMemo(() => {
    if (selectedAdvancedConfig.ruleType !== TEXT_SEGMENT_RULE_TYPE) {
      return [];
    }
    return validateTextSegmentRules(selectedAdvancedConfig.rules);
  }, [selectedAdvancedConfig]);

  const previewResult = useMemo(() => {
    if (selectedAdvancedConfig.ruleType !== TEXT_SEGMENT_RULE_TYPE) {
      return null;
    }
    return buildTextSegmentPreview(selectedAdvancedConfig.rules, previewInput);
  }, [previewInput, selectedAdvancedConfig]);

  useEffect(() => {
    if (
      externalSelectedModelName &&
      models.some((model) => model.name === externalSelectedModelName)
    ) {
      setSelectedModelName(externalSelectedModelName);
      return;
    }

    if (!models.length) {
      setSelectedModelName('');
      return;
    }

    if (!models.some((model) => model.name === selectedModelName)) {
      setSelectedModelName(models[0].name);
    }
  }, [externalSelectedModelName, models, selectedModelName]);

  useEffect(() => {
    if (selectedModelName && typeof onSelectedModelChange === 'function') {
      onSelectedModelChange(selectedModelName);
    }
  }, [onSelectedModelChange, selectedModelName]);

  const updateSelectedModelConfig = (updater) => {
    if (!selectedModel) {
      return;
    }

    setAdvancedPricingMap((previous) => ({
      ...previous,
      [selectedModel.name]:
        typeof updater === 'function'
          ? updater(
              normalizeAdvancedPricingConfig(previous[selectedModel.name]),
            )
          : updater,
    }));
  };

  const handleEffectiveModeChange = (nextMode) => {
    if (!selectedModel) {
      return;
    }

    setModelBillingModeMap((previous) => {
      const nextMap = { ...previous };
      if (nextMode === ADVANCED_PRICING_MODE_ADVANCED) {
        nextMap[selectedModel.name] = ADVANCED_PRICING_MODE_ADVANCED;
      } else {
        delete nextMap[selectedModel.name];
      }
      return nextMap;
    });
  };

  const handleRuleTypeChange = (nextRuleType) => {
    updateSelectedModelConfig((config) => ({
      ...config,
      ruleType:
        nextRuleType === MEDIA_TASK_RULE_TYPE
          ? MEDIA_TASK_RULE_TYPE
          : TEXT_SEGMENT_RULE_TYPE,
    }));
  };

  const handleTextSegmentRulesChange = (nextRules) => {
    updateSelectedModelConfig((config) => ({
      ...config,
      ruleType: TEXT_SEGMENT_RULE_TYPE,
      rules: nextRules,
    }));
  };

  const handlePreviewInputChange = (field, value) => {
    if (!NUMERIC_INPUT_REGEX.test(value)) {
      return;
    }
    setPreviewInput((previous) => ({
      ...previous,
      [field]: value,
    }));
  };

  const handleSave = async () => {
    if (!selectedModel) {
      showError(t('请先选择模型'));
      return false;
    }

    if (
      selectedAdvancedConfig.ruleType === TEXT_SEGMENT_RULE_TYPE &&
      validationErrors.length > 0
    ) {
      showError(validationErrors[0]);
      return false;
    }

    setLoading(true);
    try {
      const responseList = await Promise.all([
        API.put('/api/option/', {
          key: 'ModelBillingMode',
          value: JSON.stringify(modelBillingModeMap, null, 2),
        }),
        API.put('/api/option/', {
          key: 'AdvancedPricingRules',
          value: JSON.stringify(
            serializeAdvancedPricingMap(advancedPricingMap),
            null,
            2,
          ),
        }),
      ]);

      for (const response of responseList) {
        if (!response?.data?.success) {
          throw new Error(response?.data?.message || t('保存失败，请重试'));
        }
      }

      showSuccess(t('保存成功'));
      await refresh();
      return true;
    } catch (error) {
      console.error('保存高级定价规则失败:', error);
      showError(error.message || t('保存失败，请重试'));
      return false;
    } finally {
      setLoading(false);
    }
  };

  return {
    loading,
    modelSearchText,
    setModelSearchText,
    models,
    filteredModels,
    selectedModel,
    selectedModelName,
    setSelectedModelName,
    selectedAdvancedConfig,
    validationErrors,
    previewInput,
    previewResult,
    handleEffectiveModeChange,
    handleRuleTypeChange,
    handleTextSegmentRulesChange,
    handlePreviewInputChange,
    handleSave,
    fixedBillingModePerRequest: FIXED_BILLING_MODE_PER_REQUEST,
    fixedBillingModePerToken: FIXED_BILLING_MODE_PER_TOKEN,
  };
}

export {
  ADVANCED_PRICING_MODE_ADVANCED,
  ADVANCED_PRICING_MODE_FIXED,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
};
