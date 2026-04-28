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
  resolveInitialVisibleModelNames,
  resolveVisibleModels,
} from './modelPricingVisibility';
import {
  resolveModelPricingBridgeSelection,
  resolveModelPricingSelectedModelName,
} from './modelPricingSelection';
import {
  BILLING_MODE_ADVANCED,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_PER_TOKEN,
  buildAdvancedPricingConfigPayloadForPricingEditor,
  buildAdvancedPricingModePayload,
  canUseAdvancedBilling,
  copyAdvancedPricingRulesForModels,
  getAdvancedRuleTypeText,
  hasValue,
  isBasePricingUnset,
  resolveBillingMode,
  shouldPersistAdvancedPricingMode,
} from './modelPricingEditorHelpers';

export const PAGE_SIZE = 10;
export const PRICE_SUFFIX = '$/1M tokens';
const EMPTY_CANDIDATE_MODEL_NAMES = [];

export { hasValue, isBasePricingUnset } from './modelPricingEditorHelpers';

const EMPTY_MODEL = {
  name: '',
  billingMode: BILLING_MODE_PER_TOKEN,
  explicitBillingMode: '',
  hasExplicitBillingMode: false,
  hasInvalidExplicitAdvancedMode: false,
  advancedRuleType: '',
  fixedPrice: '',
  inputPrice: '',
  completionPrice: '',
  lockedCompletionRatio: '',
  completionRatioLocked: false,
  cachePrice: '',
  createCachePrice: '',
  imagePrice: '',
  audioInputPrice: '',
  audioOutputPrice: '',
  rawRatios: {
    modelRatio: '',
    completionRatio: '',
    cacheRatio: '',
    createCacheRatio: '',
    imageRatio: '',
    audioRatio: '',
    audioCompletionRatio: '',
  },
  hasConflict: false,
};

const NUMERIC_INPUT_REGEX = /^(\d+(\.\d*)?|\.\d*)?$/;

const toNumericString = (value) => {
  if (!hasValue(value) && value !== 0) {
    return '';
  }
  const num = Number(value);
  return Number.isFinite(num) ? String(num) : '';
};

const toNumberOrNull = (value) => {
  if (!hasValue(value) && value !== 0) {
    return null;
  }
  const num = Number(value);
  return Number.isFinite(num) ? num : null;
};

const formatNumber = (value) => {
  const num = toNumberOrNull(value);
  if (num === null) {
    return '';
  }
  return parseFloat(num.toFixed(12)).toString();
};

const toNormalizedNumber = (value) => {
  const formatted = formatNumber(value);
  return formatted === '' ? null : Number(formatted);
};

const parseOptionJSON = (rawValue) => {
  if (!rawValue || rawValue.trim() === '') {
    return {};
  }
  try {
    const parsed = JSON.parse(rawValue);
    return parsed && typeof parsed === 'object' ? parsed : {};
  } catch (error) {
    console.error('JSON解析错误:', error);
    return {};
  }
};

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

const reduceOptionsByKey = (items) =>
  Array.isArray(items)
    ? items.reduce((acc, item) => {
        if (item?.key) {
          acc[item.key] = item.value;
        }
        return acc;
      }, {})
    : {};

const ratioToBasePrice = (ratio) => {
  const num = toNumberOrNull(ratio);
  if (num === null) return '';
  return formatNumber(num * 2);
};

const normalizeCompletionRatioMeta = (rawMeta) => {
  if (!rawMeta || typeof rawMeta !== 'object' || Array.isArray(rawMeta)) {
    return {
      locked: false,
      ratio: '',
    };
  }

  return {
    locked: Boolean(rawMeta.locked),
    ratio: toNumericString(rawMeta.ratio),
  };
};

const resolveAdvancedRuleType = (rawRuleSet) => {
  if (!rawRuleSet || typeof rawRuleSet !== 'object' || Array.isArray(rawRuleSet)) {
    return '';
  }

  return typeof rawRuleSet.rule_type === 'string' ? rawRuleSet.rule_type : '';
};

const buildModelState = (name, sourceMaps) => {
  const modelRatio = toNumericString(sourceMaps.ModelRatio[name]);
  const completionRatio = toNumericString(sourceMaps.CompletionRatio[name]);
  const completionRatioMeta = normalizeCompletionRatioMeta(
    sourceMaps.CompletionRatioMeta?.[name],
  );
  const cacheRatio = toNumericString(sourceMaps.CacheRatio[name]);
  const createCacheRatio = toNumericString(sourceMaps.CreateCacheRatio[name]);
  const imageRatio = toNumericString(sourceMaps.ImageRatio[name]);
  const audioRatio = toNumericString(sourceMaps.AudioRatio[name]);
  const audioCompletionRatio = toNumericString(
    sourceMaps.AudioCompletionRatio[name],
  );
  const fixedPrice = toNumericString(sourceMaps.ModelPrice[name]);
  const advancedRuleType = resolveAdvancedRuleType(
    sourceMaps.AdvancedPricingRules?.[name],
  );
  const billingModeState = resolveBillingMode({
    explicitMode: sourceMaps.AdvancedPricingMode[name],
    fixedPrice,
    advancedRuleType,
  });
  const inputPrice = ratioToBasePrice(modelRatio);
  const inputPriceNumber = toNumberOrNull(inputPrice);
  const audioInputPrice =
    inputPriceNumber !== null && hasValue(audioRatio)
      ? formatNumber(inputPriceNumber * Number(audioRatio))
      : '';

  return {
    ...EMPTY_MODEL,
    name,
    billingMode: billingModeState.billingMode,
    explicitBillingMode: billingModeState.explicitBillingMode,
    hasExplicitBillingMode: billingModeState.hasExplicitBillingMode,
    hasInvalidExplicitAdvancedMode:
      billingModeState.hasInvalidExplicitAdvancedMode,
    advancedRuleType,
    fixedPrice,
    inputPrice,
    completionRatioLocked: completionRatioMeta.locked,
    lockedCompletionRatio: completionRatioMeta.ratio,
    completionPrice:
      inputPriceNumber !== null &&
      hasValue(
        completionRatioMeta.locked
          ? completionRatioMeta.ratio
          : completionRatio,
      )
        ? formatNumber(
            inputPriceNumber *
              Number(
                completionRatioMeta.locked
                  ? completionRatioMeta.ratio
                  : completionRatio,
              ),
          )
        : '',
    cachePrice:
      inputPriceNumber !== null && hasValue(cacheRatio)
        ? formatNumber(inputPriceNumber * Number(cacheRatio))
        : '',
    createCachePrice:
      inputPriceNumber !== null && hasValue(createCacheRatio)
        ? formatNumber(inputPriceNumber * Number(createCacheRatio))
        : '',
    imagePrice:
      inputPriceNumber !== null && hasValue(imageRatio)
        ? formatNumber(inputPriceNumber * Number(imageRatio))
        : '',
    audioInputPrice,
    audioOutputPrice:
      toNumberOrNull(audioInputPrice) !== null && hasValue(audioCompletionRatio)
        ? formatNumber(Number(audioInputPrice) * Number(audioCompletionRatio))
        : '',
    rawRatios: {
      modelRatio,
      completionRatio,
      cacheRatio,
      createCacheRatio,
      imageRatio,
      audioRatio,
      audioCompletionRatio,
    },
    hasConflict:
      hasValue(fixedPrice) &&
      [
        modelRatio,
        completionRatio,
        cacheRatio,
        createCacheRatio,
        imageRatio,
        audioRatio,
        audioCompletionRatio,
      ].some(hasValue),
  };
};

export const getModelWarnings = (model, t) => {
  if (!model) {
    return [];
  }
  const warnings = [];
  const hasDerivedPricing = [
    model.inputPrice,
    model.completionPrice,
    model.cachePrice,
    model.createCachePrice,
    model.imagePrice,
    model.audioInputPrice,
    model.audioOutputPrice,
  ].some(hasValue);

  if (model.hasConflict) {
    warnings.push(
      t('当前模型同时存在按次价格和倍率配置，未生效配置会保留，仅当前计费模式生效。'),
    );
  }

  if (
    !hasValue(model.inputPrice) &&
    [
      model.rawRatios.completionRatio,
      model.rawRatios.cacheRatio,
      model.rawRatios.createCacheRatio,
      model.rawRatios.imageRatio,
      model.rawRatios.audioRatio,
      model.rawRatios.audioCompletionRatio,
    ].some(hasValue)
  ) {
    warnings.push(
      t(
        '当前模型存在未显式设置输入倍率的扩展倍率；填写输入价格后会自动换算为价格字段。',
      ),
    );
  }

  if (
    model.billingMode === BILLING_MODE_PER_TOKEN &&
    hasDerivedPricing &&
    !hasValue(model.inputPrice)
  ) {
    warnings.push(t('按量计费下需要先填写输入价格，才能保存其它价格项。'));
  }

  if (
    model.billingMode === BILLING_MODE_PER_TOKEN &&
    hasValue(model.audioOutputPrice) &&
    !hasValue(model.audioInputPrice)
  ) {
    warnings.push(t('填写音频补全价格前，需要先填写音频输入价格。'));
  }

  return warnings;
};

export const buildSummaryText = (model, t) => {
  if (model.billingMode === BILLING_MODE_ADVANCED) {
    return model.advancedRuleType
      ? `${t('高级规则')} · ${getAdvancedRuleTypeText(model.advancedRuleType, t)}`
      : t('高级规则');
  }

  if (
    model.billingMode === BILLING_MODE_PER_REQUEST &&
    hasValue(model.fixedPrice)
  ) {
    return `${t('按次')} $${model.fixedPrice} / ${t('次')}`;
  }

  if (hasValue(model.inputPrice)) {
    const extraCount = [
      model.completionPrice,
      model.cachePrice,
      model.createCachePrice,
      model.imagePrice,
      model.audioInputPrice,
      model.audioOutputPrice,
    ].filter(hasValue).length;
    const extraLabel =
      extraCount > 0 ? `，${t('额外价格项')} ${extraCount}` : '';
    return `${t('输入')} $${model.inputPrice}${extraLabel}`;
  }

  return t('未设置价格');
};

export const buildOptionalFieldToggles = (model) => ({
  completionPrice:
    model.completionRatioLocked || hasValue(model.completionPrice),
  cachePrice: hasValue(model.cachePrice),
  createCachePrice: hasValue(model.createCachePrice),
  imagePrice: hasValue(model.imagePrice),
  audioInputPrice: hasValue(model.audioInputPrice),
  audioOutputPrice: hasValue(model.audioOutputPrice),
});

const serializeModel = (model, t) => {
  const result = {
    ModelPrice: null,
    ModelRatio: null,
    CompletionRatio: null,
    CacheRatio: null,
    CreateCacheRatio: null,
    ImageRatio: null,
    AudioRatio: null,
    AudioCompletionRatio: null,
  };

  if (hasValue(model.fixedPrice)) {
    result.ModelPrice = toNormalizedNumber(model.fixedPrice);
  }

  const inputPrice = toNumberOrNull(model.inputPrice);
  const completionPrice = toNumberOrNull(model.completionPrice);
  const cachePrice = toNumberOrNull(model.cachePrice);
  const createCachePrice = toNumberOrNull(model.createCachePrice);
  const imagePrice = toNumberOrNull(model.imagePrice);
  const audioInputPrice = toNumberOrNull(model.audioInputPrice);
  const audioOutputPrice = toNumberOrNull(model.audioOutputPrice);

  const hasDependentPrice = [
    completionPrice,
    cachePrice,
    createCachePrice,
    imagePrice,
    audioInputPrice,
    audioOutputPrice,
  ].some((value) => value !== null);

  if (inputPrice === null) {
    if (hasDependentPrice) {
      throw new Error(
        t(
          '模型 {{name}} 缺少输入价格，无法计算补全/缓存/图片/音频价格对应的倍率',
          {
            name: model.name,
          },
        ),
      );
    }

    if (hasValue(model.rawRatios.modelRatio)) {
      result.ModelRatio = toNormalizedNumber(model.rawRatios.modelRatio);
    }
    if (hasValue(model.rawRatios.completionRatio)) {
      result.CompletionRatio = toNormalizedNumber(
        model.rawRatios.completionRatio,
      );
    }
    if (hasValue(model.rawRatios.cacheRatio)) {
      result.CacheRatio = toNormalizedNumber(model.rawRatios.cacheRatio);
    }
    if (hasValue(model.rawRatios.createCacheRatio)) {
      result.CreateCacheRatio = toNormalizedNumber(
        model.rawRatios.createCacheRatio,
      );
    }
    if (hasValue(model.rawRatios.imageRatio)) {
      result.ImageRatio = toNormalizedNumber(model.rawRatios.imageRatio);
    }
    if (hasValue(model.rawRatios.audioRatio)) {
      result.AudioRatio = toNormalizedNumber(model.rawRatios.audioRatio);
    }
    if (hasValue(model.rawRatios.audioCompletionRatio)) {
      result.AudioCompletionRatio = toNormalizedNumber(
        model.rawRatios.audioCompletionRatio,
      );
    }
    return result;
  }

  result.ModelRatio = toNormalizedNumber(inputPrice / 2);

  if (!model.completionRatioLocked && completionPrice !== null) {
    result.CompletionRatio = toNormalizedNumber(completionPrice / inputPrice);
  } else if (
    model.completionRatioLocked &&
    hasValue(model.rawRatios.completionRatio)
  ) {
    result.CompletionRatio = toNormalizedNumber(
      model.rawRatios.completionRatio,
    );
  }
  if (cachePrice !== null) {
    result.CacheRatio = toNormalizedNumber(cachePrice / inputPrice);
  }
  if (createCachePrice !== null) {
    result.CreateCacheRatio = toNormalizedNumber(createCachePrice / inputPrice);
  }
  if (imagePrice !== null) {
    result.ImageRatio = toNormalizedNumber(imagePrice / inputPrice);
  }
  if (audioInputPrice !== null) {
    result.AudioRatio = toNormalizedNumber(audioInputPrice / inputPrice);
  }
  if (audioOutputPrice !== null) {
    if (audioInputPrice === null || audioInputPrice === 0) {
      throw new Error(
        t('模型 {{name}} 缺少音频输入价格，无法计算音频补全倍率', {
          name: model.name,
        }),
      );
    }
    result.AudioCompletionRatio = toNormalizedNumber(
      audioOutputPrice / audioInputPrice,
    );
  }

  return result;
};

export const buildPreviewRows = (model, t) => {
  if (!model) return [];
  const rows = shouldPersistAdvancedPricingMode({
    model,
    dirtyModeNames: model.billingModeDirty ? [model.name] : [],
  })
    ? [
        {
          key: 'AdvancedPricingMode',
          label: 'AdvancedPricingMode',
          value: model.billingMode,
        },
      ]
    : [];

  if (hasValue(model.fixedPrice)) {
    rows.push({
      key: 'ModelPrice',
      label: 'ModelPrice',
      value: model.fixedPrice,
    });
  }

  const inputPrice = toNumberOrNull(model.inputPrice);
  if (inputPrice === null) {
    return rows.concat([
      {
        key: 'ModelRatio',
        label: 'ModelRatio',
        value: hasValue(model.rawRatios.modelRatio)
          ? model.rawRatios.modelRatio
          : t('空'),
      },
      {
        key: 'CompletionRatio',
        label: 'CompletionRatio',
        value: hasValue(model.rawRatios.completionRatio)
          ? model.rawRatios.completionRatio
          : t('空'),
      },
      {
        key: 'CacheRatio',
        label: 'CacheRatio',
        value: hasValue(model.rawRatios.cacheRatio)
          ? model.rawRatios.cacheRatio
          : t('空'),
      },
      {
        key: 'CreateCacheRatio',
        label: 'CreateCacheRatio',
        value: hasValue(model.rawRatios.createCacheRatio)
          ? model.rawRatios.createCacheRatio
          : t('空'),
      },
      {
        key: 'ImageRatio',
        label: 'ImageRatio',
        value: hasValue(model.rawRatios.imageRatio)
          ? model.rawRatios.imageRatio
          : t('空'),
      },
      {
        key: 'AudioRatio',
        label: 'AudioRatio',
        value: hasValue(model.rawRatios.audioRatio)
          ? model.rawRatios.audioRatio
          : t('空'),
      },
      {
        key: 'AudioCompletionRatio',
        label: 'AudioCompletionRatio',
        value: hasValue(model.rawRatios.audioCompletionRatio)
          ? model.rawRatios.audioCompletionRatio
          : t('空'),
      },
    ]);
  }

  const completionPrice = toNumberOrNull(model.completionPrice);
  const cachePrice = toNumberOrNull(model.cachePrice);
  const createCachePrice = toNumberOrNull(model.createCachePrice);
  const imagePrice = toNumberOrNull(model.imagePrice);
  const audioInputPrice = toNumberOrNull(model.audioInputPrice);
  const audioOutputPrice = toNumberOrNull(model.audioOutputPrice);

  return rows.concat([
    {
      key: 'ModelRatio',
      label: 'ModelRatio',
      value: formatNumber(inputPrice / 2),
    },
    {
      key: 'CompletionRatio',
      label: 'CompletionRatio',
      value: model.completionRatioLocked
        ? `${model.lockedCompletionRatio || t('空')} (${t('后端固定')})`
        : completionPrice !== null
          ? formatNumber(completionPrice / inputPrice)
          : t('空'),
    },
    {
      key: 'CacheRatio',
      label: 'CacheRatio',
      value:
        cachePrice !== null ? formatNumber(cachePrice / inputPrice) : t('空'),
    },
    {
      key: 'CreateCacheRatio',
      label: 'CreateCacheRatio',
      value:
        createCachePrice !== null
          ? formatNumber(createCachePrice / inputPrice)
          : t('空'),
    },
    {
      key: 'ImageRatio',
      label: 'ImageRatio',
      value:
        imagePrice !== null ? formatNumber(imagePrice / inputPrice) : t('空'),
    },
    {
      key: 'AudioRatio',
      label: 'AudioRatio',
      value:
        audioInputPrice !== null
          ? formatNumber(audioInputPrice / inputPrice)
          : t('空'),
    },
    {
      key: 'AudioCompletionRatio',
      label: 'AudioCompletionRatio',
      value:
        audioOutputPrice !== null &&
        audioInputPrice !== null &&
        audioInputPrice !== 0
          ? formatNumber(audioOutputPrice / audioInputPrice)
          : t('空'),
    },
  ]);
};

export function useModelPricingEditorState({
  options,
  refresh,
  t,
  candidateModelNames = EMPTY_CANDIDATE_MODEL_NAMES,
  filterMode = 'all',
  initialSelectedModelName = '',
  initialSelectionVersion = 0,
}) {
  const [models, setModels] = useState([]);
  const [initialVisibleModelNames, setInitialVisibleModelNames] = useState([]);
  const [selectedModelName, setSelectedModelName] = useState(
    initialSelectedModelName || '',
  );
  const lastAppliedInitialSelectionVersionRef = useRef(null);
  const pendingSelectionPageRef = useRef(null);
  const [selectedModelNames, setSelectedModelNames] = useState([]);
  const [searchText, setSearchText] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [conflictOnly, setConflictOnly] = useState(false);
  const [optionalFieldToggles, setOptionalFieldToggles] = useState({});
  const [billingModeDirtyNames, setBillingModeDirtyNames] = useState(new Set());
  const [advancedPricingRules, setAdvancedPricingRules] = useState({});
  const [copiedAdvancedRuleNames, setCopiedAdvancedRuleNames] = useState(
    new Set(),
  );

  useEffect(() => {
    const canonicalAdvancedPricingConfig = parseAdvancedPricingConfigOption(
      options.AdvancedPricingConfig,
    );
    const sourceMaps = {
      AdvancedPricingMode:
        Object.keys(canonicalAdvancedPricingConfig.billing_mode).length > 0
          ? canonicalAdvancedPricingConfig.billing_mode
          : parseOptionJSON(options.AdvancedPricingMode),
      AdvancedPricingRules:
        Object.keys(canonicalAdvancedPricingConfig.rules).length > 0
          ? canonicalAdvancedPricingConfig.rules
          : parseOptionJSON(options.AdvancedPricingRules),
      ModelPrice: parseOptionJSON(options.ModelPrice),
      ModelRatio: parseOptionJSON(options.ModelRatio),
      CompletionRatio: parseOptionJSON(options.CompletionRatio),
      CompletionRatioMeta: parseOptionJSON(options.CompletionRatioMeta),
      CacheRatio: parseOptionJSON(options.CacheRatio),
      CreateCacheRatio: parseOptionJSON(options.CreateCacheRatio),
      ImageRatio: parseOptionJSON(options.ImageRatio),
      AudioRatio: parseOptionJSON(options.AudioRatio),
      AudioCompletionRatio: parseOptionJSON(options.AudioCompletionRatio),
    };

    const names = new Set([
      ...candidateModelNames,
      ...Object.keys(sourceMaps.AdvancedPricingMode),
      ...Object.keys(sourceMaps.AdvancedPricingRules),
      ...Object.keys(sourceMaps.ModelPrice),
      ...Object.keys(sourceMaps.ModelRatio),
      ...Object.keys(sourceMaps.CompletionRatio),
      ...Object.keys(sourceMaps.CompletionRatioMeta),
      ...Object.keys(sourceMaps.CacheRatio),
      ...Object.keys(sourceMaps.CreateCacheRatio),
      ...Object.keys(sourceMaps.ImageRatio),
      ...Object.keys(sourceMaps.AudioRatio),
      ...Object.keys(sourceMaps.AudioCompletionRatio),
    ]);

    const nextModels = Array.from(names)
      .map((name) => buildModelState(name, sourceMaps))
      .sort((a, b) => a.name.localeCompare(b.name));
    const nextVisibleModelNames = resolveInitialVisibleModelNames({
      nextModels,
      filterMode,
      candidateModelNames,
      isBasePricingUnset,
    });

    setModels(nextModels);
    setInitialVisibleModelNames(nextVisibleModelNames);
    setOptionalFieldToggles(
      nextModels.reduce((acc, model) => {
        acc[model.name] = buildOptionalFieldToggles(model);
        return acc;
      }, {}),
    );
    setAdvancedPricingRules(sourceMaps.AdvancedPricingRules);
    setBillingModeDirtyNames(new Set());
    setCopiedAdvancedRuleNames(new Set());
    setSelectedModelName((previous) => {
      if (previous && nextModels.some((model) => model.name === previous)) {
        return previous;
      }
      const nextVisibleModels = resolveVisibleModels({
        models: nextModels,
        filterMode,
        candidateModelNames,
        initialVisibleModelNames: nextVisibleModelNames,
      });
      return nextVisibleModels[0]?.name || '';
    });
  }, [candidateModelNames, filterMode, options]);

  const visibleModels = useMemo(
    () =>
      resolveVisibleModels({
        models,
        filterMode,
        candidateModelNames,
        initialVisibleModelNames,
      }),
    [candidateModelNames, filterMode, initialVisibleModelNames, models],
  );
  const visibleModelNames = useMemo(
    () => visibleModels.map((model) => model.name),
    [visibleModels],
  );

  const filteredModels = useMemo(() => {
    return visibleModels.filter((model) => {
      const keyword = searchText.trim().toLowerCase();
      const keywordMatch = keyword
        ? model.name.toLowerCase().includes(keyword)
        : true;
      const conflictMatch = conflictOnly ? model.hasConflict : true;
      return keywordMatch && conflictMatch;
    });
  }, [conflictOnly, searchText, visibleModels]);

  const pagedData = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredModels.slice(start, start + PAGE_SIZE);
  }, [currentPage, filteredModels]);

  const selectedModel = useMemo(() => {
    const currentModel =
      visibleModels.find((model) => model.name === selectedModelName) || null;

    if (!currentModel) {
      return null;
    }

    return {
      ...currentModel,
      billingModeDirty: billingModeDirtyNames.has(currentModel.name),
    };
  }, [billingModeDirtyNames, selectedModelName, visibleModels]);

  const selectedWarnings = useMemo(
    () => getModelWarnings(selectedModel, t),
    [selectedModel, t],
  );

  const previewRows = useMemo(
    () => buildPreviewRows(selectedModel, t),
    [selectedModel, t],
  );

  useEffect(() => {
    const {
      nextSelectedModelName,
      nextAppliedInitialSelectionVersion,
      shouldSyncSelection,
    } = resolveModelPricingSelectedModelName({
      currentSelectedModelName: selectedModelName,
      modelNames: visibleModelNames,
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
      setSelectedModelName(nextSelectedModelName);
    }

    if (!shouldSyncSelection) {
      return;
    }

    const bridgeSelectionState = resolveModelPricingBridgeSelection({
      shouldSyncSelection,
      modelNames: visibleModelNames,
      selectedModelName: nextSelectedModelName,
      pageSize: PAGE_SIZE,
      searchText,
      conflictOnly,
    });

    if (bridgeSelectionState.pendingSelectionPage !== null) {
      pendingSelectionPageRef.current = bridgeSelectionState.pendingSelectionPage;
      if (bridgeSelectionState.shouldResetSearchText) {
        setSearchText('');
      }
      if (bridgeSelectionState.shouldResetConflictOnly) {
        setConflictOnly(false);
      }
      return;
    }

    if (bridgeSelectionState.nextCurrentPage !== null) {
      setCurrentPage(bridgeSelectionState.nextCurrentPage);
    }
  }, [
    conflictOnly,
    initialSelectedModelName,
    initialSelectionVersion,
    searchText,
    selectedModelName,
    visibleModelNames,
  ]);

  useEffect(() => {
    if (pendingSelectionPageRef.current !== null) {
      setCurrentPage(pendingSelectionPageRef.current);
      pendingSelectionPageRef.current = null;
      return;
    }

    setCurrentPage(1);
  }, [searchText, conflictOnly, filterMode, candidateModelNames]);

  useEffect(() => {
    setSelectedModelNames((previous) =>
      previous.filter((name) =>
        visibleModels.some((model) => model.name === name),
      ),
    );
  }, [visibleModels]);

  useEffect(() => {
    if (visibleModels.length === 0) {
      setSelectedModelName('');
      return;
    }
    if (!visibleModels.some((model) => model.name === selectedModelName)) {
      setSelectedModelName(visibleModels[0].name);
    }
  }, [selectedModelName, visibleModels]);

  const upsertModel = (name, updater) => {
    setModels((previous) =>
      previous.map((model) => {
        if (model.name !== name) return model;
        return typeof updater === 'function' ? updater(model) : updater;
      }),
    );
  };

  const markBillingModesDirty = (modelNames) => {
    setBillingModeDirtyNames((previous) => {
      const next = new Set(previous);
      modelNames.forEach((name) => {
        if (name) {
          next.add(name);
        }
      });
      return next;
    });
  };

  const isOptionalFieldEnabled = (model, field) => {
    if (!model) return false;
    const modelToggles = optionalFieldToggles[model.name];
    if (modelToggles && typeof modelToggles[field] === 'boolean') {
      return modelToggles[field];
    }
    return buildOptionalFieldToggles(model)[field];
  };

  const updateOptionalFieldToggle = (modelName, field, checked) => {
    setOptionalFieldToggles((prev) => ({
      ...prev,
      [modelName]: {
        ...(prev[modelName] || {}),
        [field]: checked,
      },
    }));
  };

  const handleOptionalFieldToggle = (field, checked) => {
    if (!selectedModel) return;

    updateOptionalFieldToggle(selectedModel.name, field, checked);

    if (checked) {
      return;
    }

    upsertModel(selectedModel.name, (model) => {
      const nextModel = { ...model, [field]: '' };

      if (field === 'audioInputPrice') {
        nextModel.audioOutputPrice = '';
        setOptionalFieldToggles((prev) => ({
          ...prev,
          [selectedModel.name]: {
            ...(prev[selectedModel.name] || {}),
            audioInputPrice: false,
            audioOutputPrice: false,
          },
        }));
      }

      return nextModel;
    });
  };

  const fillDerivedPricesFromBase = (model, nextInputPrice) => {
    const baseNumber = toNumberOrNull(nextInputPrice);
    if (baseNumber === null) {
      return model;
    }

    return {
      ...model,
      completionPrice:
        model.completionRatioLocked && hasValue(model.lockedCompletionRatio)
          ? formatNumber(baseNumber * Number(model.lockedCompletionRatio))
          : !hasValue(model.completionPrice) &&
              hasValue(model.rawRatios.completionRatio)
            ? formatNumber(baseNumber * Number(model.rawRatios.completionRatio))
            : model.completionPrice,
      cachePrice:
        !hasValue(model.cachePrice) && hasValue(model.rawRatios.cacheRatio)
          ? formatNumber(baseNumber * Number(model.rawRatios.cacheRatio))
          : model.cachePrice,
      createCachePrice:
        !hasValue(model.createCachePrice) &&
        hasValue(model.rawRatios.createCacheRatio)
          ? formatNumber(baseNumber * Number(model.rawRatios.createCacheRatio))
          : model.createCachePrice,
      imagePrice:
        !hasValue(model.imagePrice) && hasValue(model.rawRatios.imageRatio)
          ? formatNumber(baseNumber * Number(model.rawRatios.imageRatio))
          : model.imagePrice,
      audioInputPrice:
        !hasValue(model.audioInputPrice) && hasValue(model.rawRatios.audioRatio)
          ? formatNumber(baseNumber * Number(model.rawRatios.audioRatio))
          : model.audioInputPrice,
      audioOutputPrice:
        !hasValue(model.audioOutputPrice) &&
        hasValue(model.rawRatios.audioRatio) &&
        hasValue(model.rawRatios.audioCompletionRatio)
          ? formatNumber(
              baseNumber *
                Number(model.rawRatios.audioRatio) *
                Number(model.rawRatios.audioCompletionRatio),
            )
          : model.audioOutputPrice,
    };
  };

  const handleNumericFieldChange = (field, value) => {
    if (!selectedModel || !NUMERIC_INPUT_REGEX.test(value)) {
      return;
    }

    upsertModel(selectedModel.name, (model) => {
      const updatedModel = { ...model, [field]: value };

      if (field === 'inputPrice') {
        return fillDerivedPricesFromBase(updatedModel, value);
      }

      return updatedModel;
    });
  };

  const handleBillingModeChange = (value) => {
    if (!selectedModel) return;
    if (value === selectedModel.billingMode) {
      return;
    }
    if (value === BILLING_MODE_ADVANCED && !canUseAdvancedBilling(selectedModel)) {
      showError(t('当前模型未配置高级规则，无法切换到高级规则计费模式'));
      return;
    }

    markBillingModesDirty([selectedModel.name]);
    upsertModel(selectedModel.name, (model) => ({
      ...model,
      billingMode: value,
    }));
  };

  const addModel = (modelName) => {
    const trimmedName = modelName.trim();
    if (!trimmedName) {
      showError(t('请输入模型名称'));
      return false;
    }
    if (models.some((model) => model.name === trimmedName)) {
      showError(t('模型名称已存在'));
      return false;
    }

    const nextModel = {
      ...EMPTY_MODEL,
      name: trimmedName,
      rawRatios: { ...EMPTY_MODEL.rawRatios },
    };

    setModels((previous) => [nextModel, ...previous]);
    setOptionalFieldToggles((prev) => ({
      ...prev,
      [trimmedName]: buildOptionalFieldToggles(nextModel),
    }));
    setSelectedModelName(trimmedName);
    setCurrentPage(1);
    return true;
  };

  const deleteModel = (name) => {
    const nextModels = models.filter((model) => model.name !== name);
    setModels(nextModels);
    setOptionalFieldToggles((prev) => {
      const next = { ...prev };
      delete next[name];
      return next;
    });
    setSelectedModelNames((previous) =>
      previous.filter((item) => item !== name),
    );
    if (selectedModelName === name) {
      setSelectedModelName(nextModels[0]?.name || '');
    }
  };

  const applySelectedModelPricing = ({ copyAdvancedRules = true } = {}) => {
    if (!selectedModel) {
      showError(t('请先选择一个作为模板的模型'));
      return false;
    }
    if (selectedModelNames.length === 0) {
      showError(t('请先勾选需要批量设置的模型'));
      return false;
    }

    const sourceToggles = optionalFieldToggles[selectedModel.name] || {};
    const shouldCopyAdvancedRules =
      selectedModel.billingMode === BILLING_MODE_ADVANCED && copyAdvancedRules;
    const advancedPricingCopyState = shouldCopyAdvancedRules
      ? copyAdvancedPricingRulesForModels({
          sourceModelName: selectedModel.name,
          targetModelNames: selectedModelNames,
          rulesMap: advancedPricingRules,
        })
      : null;

    if (
      shouldCopyAdvancedRules &&
      !hasValue(advancedPricingCopyState.advancedRuleType)
    ) {
      showError(t('当前模型未配置高级规则，无法切换到高级规则计费模式'));
      return false;
    }

    const invalidAdvancedTargets =
      selectedModel.billingMode === BILLING_MODE_ADVANCED && !shouldCopyAdvancedRules
        ? selectedModelNames.filter((modelName) => {
            const targetModel = models.find((item) => item.name === modelName);
            return targetModel && !canUseAdvancedBilling(targetModel);
          })
        : [];

    if (invalidAdvancedTargets.length > 0) {
      showError(t('选中的模型里存在未配置高级规则的项，无法批量应用高级规则计费模式'));
      return false;
    }

    const changedBillingModeNames = selectedModelNames.filter((modelName) => {
      const targetModel = models.find((item) => item.name === modelName);
      return targetModel && targetModel.billingMode !== selectedModel.billingMode;
    });

    setModels((previous) =>
      previous.map((model) => {
        if (!selectedModelNames.includes(model.name)) {
          return model;
        }

        const nextModel = {
          ...model,
          billingMode: selectedModel.billingMode,
          fixedPrice: selectedModel.fixedPrice,
          inputPrice: selectedModel.inputPrice,
          completionPrice: selectedModel.completionPrice,
          cachePrice: selectedModel.cachePrice,
          createCachePrice: selectedModel.createCachePrice,
          imagePrice: selectedModel.imagePrice,
          audioInputPrice: selectedModel.audioInputPrice,
          audioOutputPrice: selectedModel.audioOutputPrice,
        };

        if (shouldCopyAdvancedRules) {
          nextModel.advancedRuleType = advancedPricingCopyState.advancedRuleType;
          nextModel.hasInvalidExplicitAdvancedMode = false;
        }

        if (
          nextModel.billingMode === BILLING_MODE_PER_TOKEN &&
          nextModel.completionRatioLocked &&
          hasValue(nextModel.inputPrice) &&
          hasValue(nextModel.lockedCompletionRatio)
        ) {
          nextModel.completionPrice = formatNumber(
            Number(nextModel.inputPrice) *
              Number(nextModel.lockedCompletionRatio),
          );
        }

        return nextModel;
      }),
    );

    setOptionalFieldToggles((previous) => {
      const next = { ...previous };
      selectedModelNames.forEach((modelName) => {
        const targetModel = models.find((item) => item.name === modelName);
        next[modelName] = {
          completionPrice: targetModel?.completionRatioLocked
            ? true
            : Boolean(sourceToggles.completionPrice),
          cachePrice: Boolean(sourceToggles.cachePrice),
          createCachePrice: Boolean(sourceToggles.createCachePrice),
          imagePrice: Boolean(sourceToggles.imagePrice),
          audioInputPrice: Boolean(sourceToggles.audioInputPrice),
          audioOutputPrice:
            Boolean(sourceToggles.audioInputPrice) &&
            Boolean(sourceToggles.audioOutputPrice),
        };
      });
      return next;
    });

    if (shouldCopyAdvancedRules) {
      setAdvancedPricingRules(advancedPricingCopyState.rulesMap);
      setCopiedAdvancedRuleNames((previous) => {
        const next = new Set(previous);
        selectedModelNames.forEach((modelName) => next.add(modelName));
        return next;
      });
    }

    markBillingModesDirty(changedBillingModeNames);

    showSuccess(
      t('已将模型 {{name}} 的价格配置批量应用到 {{count}} 个模型', {
        name: selectedModel.name,
        count: selectedModelNames.length,
      }),
    );
    return true;
  };

  const handleSubmit = async () => {
    setLoading(true);
    try {
      const latestOptionsRes = await API.get('/api/option/');
      const {
        success: latestOptionsSuccess,
        message: latestOptionsMessage,
        data: latestOptionsData,
      } = latestOptionsRes?.data || {};
      if (!latestOptionsSuccess) {
        throw new Error(latestOptionsMessage || t('保存失败，请重试'));
      }

      const latestOptionsByKey = reduceOptionsByKey(latestOptionsData);
      const latestCanonicalAdvancedPricingConfig =
        parseAdvancedPricingConfigOption(
          latestOptionsByKey.AdvancedPricingConfig,
        );
      const output = {
        ModelPrice: {},
        ModelRatio: {},
        CompletionRatio: {},
        CacheRatio: {},
        CreateCacheRatio: {},
        ImageRatio: {},
        AudioRatio: {},
        AudioCompletionRatio: {},
      };

      for (const model of models) {
        const serialized = serializeModel(model, t);
        Object.entries(serialized).forEach(([key, value]) => {
          if (value !== null) {
            output[key][model.name] = value;
          }
        });
      }

      const latestAdvancedPricingModeMap =
        Object.keys(latestCanonicalAdvancedPricingConfig.billing_mode).length >
        0
          ? latestCanonicalAdvancedPricingConfig.billing_mode
          : parseOptionJSON(latestOptionsByKey.AdvancedPricingMode);
      const latestAdvancedPricingRulesMap =
        Object.keys(latestCanonicalAdvancedPricingConfig.rules).length > 0
          ? latestCanonicalAdvancedPricingConfig.rules
          : parseOptionJSON(latestOptionsByKey.AdvancedPricingRules);

      if (copiedAdvancedRuleNames.size > 0) {
        output.AdvancedPricingConfig =
          buildAdvancedPricingConfigPayloadForPricingEditor({
            latestModeMap: latestAdvancedPricingModeMap,
            latestRulesMap: latestAdvancedPricingRulesMap,
            draftRulesMap: advancedPricingRules,
            copiedRuleNames: copiedAdvancedRuleNames,
            models,
            dirtyModeNames: billingModeDirtyNames,
          });
      } else {
        output.AdvancedPricingMode = buildAdvancedPricingModePayload({
          latestModeMap: latestAdvancedPricingModeMap,
          latestRulesMap: latestAdvancedPricingRulesMap,
          models,
          dirtyModeNames: billingModeDirtyNames,
        });
      }

      const requestQueue = Object.entries(output).map(([key, value]) =>
        API.put('/api/option/', {
          key,
          value: JSON.stringify(value, null, 2),
        }),
      );

      const results = await Promise.all(requestQueue);
      for (const res of results) {
        if (!res?.data?.success) {
          throw new Error(res?.data?.message || t('保存失败，请重试'));
        }
      }

      showSuccess(t('保存成功'));
      await refresh();
    } catch (error) {
      console.error('保存失败:', error);
      showError(error.message || t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  return {
    models,
    selectedModel,
    selectedModelName,
    selectedModelNames,
    setSelectedModelName,
    setSelectedModelNames,
    searchText,
    setSearchText,
    currentPage,
    setCurrentPage,
    loading,
    conflictOnly,
    setConflictOnly,
    filteredModels,
    pagedData,
    selectedWarnings,
    previewRows,
    isOptionalFieldEnabled,
    handleOptionalFieldToggle,
    handleNumericFieldChange,
    handleBillingModeChange,
    handleSubmit,
    addModel,
    deleteModel,
    applySelectedModelPricing,
  };
}
