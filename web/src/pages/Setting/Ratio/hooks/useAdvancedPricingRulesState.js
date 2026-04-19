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

import { useEffect, useMemo, useRef, useState } from 'react';
import { API, showError, showSuccess } from '../../../../helpers';
import {
  ADVANCED_PRICING_MODE_ADVANCED,
  FIXED_BILLING_MODE_PER_REQUEST,
  FIXED_BILLING_MODE_PER_TOKEN,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
  buildAdvancedPricingConfigPayload,
  buildAdvancedPricingSaveMaps,
  buildMediaTaskPreview,
  buildTextSegmentPreview,
  getAdvancedRuleType,
  getAdvancedPricingMapValidationErrors,
  getAdvancedPricingValidationErrors,
  getEffectiveBillingModeForModel,
  getFixedBillingModeForModel,
  hasAdvancedPricingConfig,
  hasFixedPricingConfig,
  mergeAdvancedPricingDraftMap,
  mergeAdvancedPricingModeDraftMap,
  normalizeFixedBillingMode,
  normalizeAdvancedPricingConfig,
  parseOptionJSON,
  saveAdvancedPricingOptions,
} from './advancedPricingRuleHelpers';
import { resolveAdvancedPricingSelectedModelName } from './advancedPricingSelection';

const EMPTY_PREVIEW_INPUT = {
  inputTokens: '',
  outputTokens: '',
  serviceTier: '',
  rawAction: '',
  inferenceMode: '',
  inputVideo: '',
  audio: '',
  resolution: '',
  aspectRatio: '',
  outputDuration: '',
  inputVideoDuration: '',
  draft: '',
  usageTotalTokens: '',
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
const PREVIEW_NUMERIC_FIELDS = new Set([
  'inputTokens',
  'outputTokens',
  'outputDuration',
  'inputVideoDuration',
  'usageTotalTokens',
]);
const PREVIEW_BOOLEAN_FIELDS = new Set(['inputVideo', 'audio', 'draft']);

const buildSourceMaps = (options = {}) =>
  MODEL_OPTION_KEYS.reduce((result, key) => {
    result[key] = parseOptionJSON(options[key]);
    return result;
  }, {});

const parseAdvancedPricingConfigOption = (rawValue) => {
  const parsedValue = parseOptionJSON(rawValue);

  return {
    billing_mode:
      parsedValue?.billing_mode &&
      typeof parsedValue.billing_mode === 'object' &&
      !Array.isArray(parsedValue.billing_mode)
        ? parsedValue.billing_mode
        : {},
    rules:
      parsedValue?.rules &&
      typeof parsedValue.rules === 'object' &&
      !Array.isArray(parsedValue.rules)
        ? parsedValue.rules
        : {},
  };
};

const buildAdvancedPricingMap = (rulesMap = {}) => {
  return Object.entries(rulesMap).reduce((result, [modelName, config]) => {
    result[modelName] = normalizeAdvancedPricingConfig(config);
    return result;
  }, {});
};

const buildBillingModeMap = (modeMap = {}) => {
  return Object.entries(modeMap).reduce((result, [modelName, mode]) => {
    result[modelName] =
      mode === ADVANCED_PRICING_MODE_ADVANCED
        ? ADVANCED_PRICING_MODE_ADVANCED
        : normalizeFixedBillingMode(mode);
    return result;
  }, {});
};

export function useAdvancedPricingRulesState({
  options,
  refresh,
  t,
  candidateModelNames = [],
  selectedModelName: externalSelectedModelName,
  onSelectedModelChange,
  initialSelectedModelName = '',
  initialSelectionVersion = 0,
}) {
  const isControlledSelection = externalSelectedModelName !== undefined;
  const [selectedModelName, setSelectedModelNameState] = useState(
    isControlledSelection
      ? (externalSelectedModelName ?? '')
      : (initialSelectedModelName || ''),
  );
  const lastAppliedInitialSelectionVersionRef = useRef(null);
  const [modelSearchText, setModelSearchText] = useState('');
  const [loading, setLoading] = useState(false);
  const [previewInput, setPreviewInput] = useState(EMPTY_PREVIEW_INPUT);
  const [advancedPricingModeMap, setAdvancedPricingModeMap] = useState({});
  const [advancedPricingMap, setAdvancedPricingMap] = useState({});
  const dirtyModelNamesRef = useRef(new Set());

  const sourceMaps = useMemo(() => buildSourceMaps(options), [options]);
  const canonicalAdvancedPricingConfig = useMemo(
    () => parseAdvancedPricingConfigOption(options?.AdvancedPricingConfig),
    [options?.AdvancedPricingConfig],
  );
  const serverAdvancedPricingModeMap = useMemo(
    () =>
      buildBillingModeMap(
        Object.keys(canonicalAdvancedPricingConfig.billing_mode).length > 0
          ? canonicalAdvancedPricingConfig.billing_mode
          : parseOptionJSON(options.AdvancedPricingMode),
      ),
    [canonicalAdvancedPricingConfig, options?.AdvancedPricingMode],
  );
  const serverAdvancedPricingMap = useMemo(
    () =>
      buildAdvancedPricingMap(
        Object.keys(canonicalAdvancedPricingConfig.rules).length > 0
          ? canonicalAdvancedPricingConfig.rules
          : parseOptionJSON(options.AdvancedPricingRules),
      ),
    [canonicalAdvancedPricingConfig, options?.AdvancedPricingRules],
  );

  useEffect(() => {
    setAdvancedPricingModeMap((previous) =>
      mergeAdvancedPricingModeDraftMap(
        previous,
        serverAdvancedPricingModeMap,
        dirtyModelNamesRef.current,
      ),
    );
    setAdvancedPricingMap((previous) =>
      mergeAdvancedPricingDraftMap(
        previous,
        serverAdvancedPricingMap,
        dirtyModelNamesRef.current,
      ),
    );
  }, [
    options?.AdvancedPricingConfig,
    options?.AdvancedPricingMode,
    options?.AdvancedPricingRules,
    serverAdvancedPricingMap,
    serverAdvancedPricingModeMap,
  ]);

  const models = useMemo(() => {
    return Array.from(new Set((candidateModelNames || []).filter(Boolean)))
      .sort((leftName, rightName) => leftName.localeCompare(rightName))
      .map((modelName) => {
        const fixedBillingMode = getFixedBillingModeForModel(modelName, sourceMaps);
        const advancedConfig = normalizeAdvancedPricingConfig(
          advancedPricingMap[modelName],
        );
        const serverAdvancedConfig = normalizeAdvancedPricingConfig(
          serverAdvancedPricingMap[modelName],
        );

        return {
          name: modelName,
          fixedBillingMode,
          selectedMode: getEffectiveBillingModeForModel({
            selectedMode: advancedPricingModeMap[modelName],
            fixedBillingMode,
            advancedConfig,
          }),
          effectiveMode: getEffectiveBillingModeForModel({
            selectedMode: serverAdvancedPricingModeMap[modelName],
            fixedBillingMode,
            advancedConfig: serverAdvancedConfig,
          }),
          hasFixedPricing: hasFixedPricingConfig(modelName, sourceMaps),
          hasAdvancedPricing: hasAdvancedPricingConfig(advancedConfig),
          ruleType: getAdvancedRuleType(advancedConfig),
          advancedConfig,
        };
      });
  }, [
    advancedPricingMap,
    advancedPricingModeMap,
    candidateModelNames,
    serverAdvancedPricingMap,
    serverAdvancedPricingModeMap,
    sourceMaps,
  ]);

  const filteredModels = useMemo(() => {
    const keyword = modelSearchText.trim().toLowerCase();
    if (!keyword) {
      return models;
    }
    return models.filter((model) =>
      model.name.toLowerCase().includes(keyword),
    );
  }, [modelSearchText, models]);

  const modelNames = useMemo(
    () => models.map((model) => model.name),
    [models],
  );

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

  const validationErrors = useMemo(
    () => getAdvancedPricingValidationErrors(selectedAdvancedConfig),
    [selectedAdvancedConfig],
  );

  const previewResult = useMemo(() => {
    if (selectedAdvancedConfig.ruleType === MEDIA_TASK_RULE_TYPE) {
      return buildMediaTaskPreview(selectedAdvancedConfig.rules, previewInput);
    }
    if (selectedAdvancedConfig.ruleType === TEXT_SEGMENT_RULE_TYPE) {
      return buildTextSegmentPreview(selectedAdvancedConfig.rules, previewInput);
    }
    return null;
  }, [previewInput, selectedAdvancedConfig]);

  const savePreview = useMemo(() => {
    if (!selectedModel) {
      return null;
    }

    const savePayload = buildAdvancedPricingSaveMaps({
      latestModeMap:
        Object.keys(canonicalAdvancedPricingConfig.billing_mode).length > 0
          ? canonicalAdvancedPricingConfig.billing_mode
          : parseOptionJSON(options.AdvancedPricingMode),
      latestRulesMap:
        Object.keys(canonicalAdvancedPricingConfig.rules).length > 0
          ? canonicalAdvancedPricingConfig.rules
          : parseOptionJSON(options.AdvancedPricingRules),
      draftModeMap: advancedPricingModeMap,
      draftConfigMap: advancedPricingMap,
      dirtyModelNames: dirtyModelNamesRef.current,
      fixedBillingModes: models.reduce((result, model) => {
        result[model.name] = model.fixedBillingMode;
        return result;
      }, {}),
    });

    return {
      effectiveMode: selectedModel.selectedMode,
      configOptionKey: 'AdvancedPricingConfig',
      configEntry: buildAdvancedPricingConfigPayload({
        modeMap: savePayload.modeMap,
        rulesMap: savePayload.rulesMap,
      }),
    };
  }, [
    advancedPricingMap,
    advancedPricingModeMap,
    models,
    options.AdvancedPricingConfig,
    options.AdvancedPricingMode,
    options.AdvancedPricingRules,
    canonicalAdvancedPricingConfig,
    selectedModel,
  ]);

  useEffect(() => {
    const {
      nextSelectedModelName,
      nextAppliedInitialSelectionVersion,
    } = resolveAdvancedPricingSelectedModelName({
      currentSelectedModelName: selectedModelName,
      modelNames,
      isControlledSelection,
      externalSelectedModelName,
      initialSelectedModelName,
      initialSelectionVersion,
      lastAppliedInitialSelectionVersion:
        lastAppliedInitialSelectionVersionRef.current,
    });

    if (
      nextAppliedInitialSelectionVersion !==
      lastAppliedInitialSelectionVersionRef.current
    ) {
      lastAppliedInitialSelectionVersionRef.current =
        nextAppliedInitialSelectionVersion;
    }

    if (nextSelectedModelName !== selectedModelName) {
      setSelectedModelNameState(nextSelectedModelName);
    }
  }, [
    externalSelectedModelName,
    initialSelectedModelName,
    initialSelectionVersion,
    isControlledSelection,
    modelNames,
    selectedModelName,
  ]);

  const handleSelectedModelNameChange = (nextSelectedModelName) => {
    if (!isControlledSelection) {
      setSelectedModelNameState(nextSelectedModelName);
    }
    if (typeof onSelectedModelChange === 'function') {
      onSelectedModelChange(nextSelectedModelName);
    }
  };

  const updateSelectedModelConfig = (updater) => {
    if (!selectedModel) {
      return;
    }

    dirtyModelNamesRef.current = new Set(dirtyModelNamesRef.current).add(
      selectedModel.name,
    );

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
      return false;
    }

    if (
      nextMode === ADVANCED_PRICING_MODE_ADVANCED &&
      !selectedModel.hasAdvancedPricing
    ) {
      showError(t('请至少先保存一条高级规则，再切换为高级规则生效'));
      return false;
    }

    dirtyModelNamesRef.current = new Set(dirtyModelNamesRef.current).add(
      selectedModel.name,
    );

    setAdvancedPricingModeMap((previous) => {
      const normalizedMode =
        nextMode === ADVANCED_PRICING_MODE_ADVANCED
          ? ADVANCED_PRICING_MODE_ADVANCED
          : normalizeFixedBillingMode(nextMode || selectedModel.fixedBillingMode);

      return {
        ...previous,
        [selectedModel.name]: normalizedMode,
      };
    });
    return true;
  };

  const handleRuleTypeChange = (nextRuleType) => {
    updateSelectedModelConfig((config) =>
      normalizeAdvancedPricingConfig(
        nextRuleType === MEDIA_TASK_RULE_TYPE
          ? {
              ruleType: MEDIA_TASK_RULE_TYPE,
              displayName:
                config.ruleType === MEDIA_TASK_RULE_TYPE ? config.displayName : '',
              taskType:
                config.ruleType === MEDIA_TASK_RULE_TYPE ? config.taskType : '',
              billingUnit:
                config.ruleType === MEDIA_TASK_RULE_TYPE
                  ? config.billingUnit
                  : '',
              note: config.ruleType === MEDIA_TASK_RULE_TYPE ? config.note : '',
              rules: config.ruleType === MEDIA_TASK_RULE_TYPE ? config.rules : [],
            }
          : {
              ruleType: TEXT_SEGMENT_RULE_TYPE,
              rules:
                config.ruleType === TEXT_SEGMENT_RULE_TYPE ? config.rules : [],
            },
      ),
    );
  };

  const handleTextSegmentRulesChange = (nextRules) => {
    updateSelectedModelConfig((config) => ({
      ...config,
      ruleType: TEXT_SEGMENT_RULE_TYPE,
      rules: nextRules,
    }));
  };

  const handleTextSegmentConfigChange = (nextConfig) => {
    updateSelectedModelConfig(
      normalizeAdvancedPricingConfig({
        ...nextConfig,
        ruleType: TEXT_SEGMENT_RULE_TYPE,
      }),
    );
  };

  const handleMediaTaskConfigChange = (nextConfig) => {
    updateSelectedModelConfig(
      normalizeAdvancedPricingConfig({
        ...nextConfig,
        ruleType: MEDIA_TASK_RULE_TYPE,
      }),
    );
  };

  const handlePreviewInputChange = (field, value) => {
    if (PREVIEW_NUMERIC_FIELDS.has(field) && !NUMERIC_INPUT_REGEX.test(value)) {
      return;
    }

    if (PREVIEW_BOOLEAN_FIELDS.has(field)) {
      const normalizedValue =
        value === true || value === 'true'
          ? 'true'
          : value === false || value === 'false'
            ? 'false'
            : '';
      setPreviewInput((previous) => ({
        ...previous,
        [field]: normalizedValue,
      }));
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

    const allValidationErrors =
      getAdvancedPricingMapValidationErrors(advancedPricingMap);
    if (allValidationErrors.length > 0) {
      showError(allValidationErrors[0]);
      return false;
    }

    setLoading(true);
    try {
      const savePayload = buildAdvancedPricingSaveMaps({
        latestModeMap:
          Object.keys(canonicalAdvancedPricingConfig.billing_mode).length > 0
            ? canonicalAdvancedPricingConfig.billing_mode
            : parseOptionJSON(options.AdvancedPricingMode),
        latestRulesMap:
          Object.keys(canonicalAdvancedPricingConfig.rules).length > 0
            ? canonicalAdvancedPricingConfig.rules
            : parseOptionJSON(options.AdvancedPricingRules),
        draftModeMap: advancedPricingModeMap,
        draftConfigMap: advancedPricingMap,
        dirtyModelNames: dirtyModelNamesRef.current,
        fixedBillingModes: models.reduce((result, model) => {
          result[model.name] = model.fixedBillingMode;
          return result;
        }, {}),
      });

      await saveAdvancedPricingOptions({
        api: API,
        savePayload,
        saveFailureMessage: t('保存失败，请重试'),
      });
      showSuccess(t('保存成功'));
      dirtyModelNamesRef.current = new Set();
      if (typeof refresh === 'function') {
        try {
          await refresh();
        } catch (refreshError) {
          console.error('刷新高级定价规则失败:', refreshError);
          showError(refreshError.message || t('刷新失败，请手动重试'));
        }
      }
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
    setSelectedModelName: handleSelectedModelNameChange,
    selectedAdvancedConfig,
    validationErrors,
    previewInput,
    previewResult,
    savePreview,
    handleEffectiveModeChange,
    handleRuleTypeChange,
    handleTextSegmentRulesChange,
    handleTextSegmentConfigChange,
    handleMediaTaskConfigChange,
    handlePreviewInputChange,
    handleSave,
    fixedBillingModePerRequest: FIXED_BILLING_MODE_PER_REQUEST,
    fixedBillingModePerToken: FIXED_BILLING_MODE_PER_TOKEN,
  };
}

export {
  ADVANCED_PRICING_MODE_ADVANCED,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
};
