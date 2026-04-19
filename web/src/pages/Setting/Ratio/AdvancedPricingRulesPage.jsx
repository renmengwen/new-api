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

import React, { useEffect, useState } from 'react';
import { Card, Empty, Modal, Space, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../../helpers';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import AdvancedPricingModelList from './components/advanced-pricing/AdvancedPricingModelList';
import AdvancedPricingPreview from './components/advanced-pricing/AdvancedPricingPreview';
import AdvancedPricingSummary from './components/advanced-pricing/AdvancedPricingSummary';
import MediaTaskRuleEditor from './components/advanced-pricing/MediaTaskRuleEditor';
import TextSegmentRuleEditor from './components/advanced-pricing/TextSegmentRuleEditor';
import {
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
  useAdvancedPricingRulesState,
} from './hooks/useAdvancedPricingRulesState';

const FALLBACK_MODEL_OPTION_KEYS = [
  'AdvancedPricingMode',
  'AdvancedPricingRules',
  'ModelPrice',
  'ModelRatio',
  'CompletionRatio',
  'CompletionRatioMeta',
  'CacheRatio',
  'CreateCacheRatio',
  'ImageRatio',
  'AudioRatio',
  'AudioCompletionRatio',
];

const parseOptionMap = (rawValue) => {
  if (!rawValue || typeof rawValue !== 'string') {
    return {};
  }

  try {
    const parsedValue = JSON.parse(rawValue);
    return parsedValue &&
      typeof parsedValue === 'object' &&
      !Array.isArray(parsedValue)
      ? parsedValue
      : {};
  } catch {
    return {};
  }
};

const buildFallbackEnabledModelNames = ({ options, initialModelName = '' }) => {
  const names = new Set();

  if (initialModelName) {
    names.add(initialModelName);
  }

  const parsedAdvancedPricingConfig = parseOptionMap(options?.AdvancedPricingConfig);
  const canonicalBillingModeMap =
    parsedAdvancedPricingConfig.billing_mode &&
    typeof parsedAdvancedPricingConfig.billing_mode === 'object' &&
    !Array.isArray(parsedAdvancedPricingConfig.billing_mode)
      ? parsedAdvancedPricingConfig.billing_mode
      : {};
  const canonicalRulesMap =
    parsedAdvancedPricingConfig.rules &&
    typeof parsedAdvancedPricingConfig.rules === 'object' &&
    !Array.isArray(parsedAdvancedPricingConfig.rules)
      ? parsedAdvancedPricingConfig.rules
      : {};

  Object.keys(canonicalBillingModeMap).forEach((modelName) => names.add(modelName));
  Object.keys(canonicalRulesMap).forEach((modelName) => names.add(modelName));

  FALLBACK_MODEL_OPTION_KEYS.forEach((key) => {
    Object.keys(parseOptionMap(options?.[key])).forEach((modelName) =>
      names.add(modelName),
    );
  });

  return Array.from(names)
    .filter(Boolean)
    .sort((leftName, rightName) => leftName.localeCompare(rightName));
};

export default function AdvancedPricingRulesPage(props) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [enabledModels, setEnabledModels] = useState([]);
  const [loadingModels, setLoadingModels] = useState(true);
  const [shouldUseFallbackEnabledModels, setShouldUseFallbackEnabledModels] =
    useState(false);
  const resolvedEnabledModels = shouldUseFallbackEnabledModels
    ? buildFallbackEnabledModelNames({
        options: props.options,
        initialModelName: props.initialModelName,
      })
    : enabledModels;

  useEffect(() => {
    let active = true;

    const loadEnabledModels = async () => {
      const fallbackEnabledModels = buildFallbackEnabledModelNames({
        options: props.options,
        initialModelName: props.initialModelName,
      });
      setLoadingModels(true);
      try {
        const response = await API.get('/api/channel/models_enabled');
        const { success, message, data } = response.data;

        if (!active) {
          return;
        }

        if (success) {
          setShouldUseFallbackEnabledModels(false);
          setEnabledModels(Array.isArray(data) ? data : []);
        } else {
          setShouldUseFallbackEnabledModels(true);
          setEnabledModels(fallbackEnabledModels);
          showError(message);
        }
      } catch (error) {
        if (active) {
          setShouldUseFallbackEnabledModels(true);
          setEnabledModels(fallbackEnabledModels);
          console.error('获取启用模型失败:', error);
          showError(t('获取启用模型失败'));
        }
      } finally {
        if (active) {
          setLoadingModels(false);
        }
      }
    };

    loadEnabledModels();

    return () => {
      active = false;
    };
  }, [props.initialModelName, props.options, t]);

  const {
    loading,
    modelSearchText,
    setModelSearchText,
    filteredModels,
    selectedModel,
    selectedModelName,
    setSelectedModelName,
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
  } = useAdvancedPricingRulesState({
    options: props.options,
    refresh: props.refresh,
    t,
    candidateModelNames: resolvedEnabledModels,
    selectedModelName: props.selectedModelName,
    onSelectedModelChange: props.onSelectedModelChange,
    initialSelectedModelName: props.initialModelName,
    initialSelectionVersion: props.initialModelSelectionKey,
  });

  const handleBackToPricing = (modelName = selectedModel?.name) => {
    if (typeof props.onBackToPricing === 'function') {
      props.onBackToPricing(modelName);
    }
  };

  const handleConfirmEffectiveModeChange = (nextMode) => {
    if (!selectedModel || nextMode === selectedModel.selectedMode) {
      return;
    }

    if (
      nextMode === 'advanced' &&
      !selectedModel.hasAdvancedPricing
    ) {
      handleEffectiveModeChange(nextMode);
      return;
    }

    Modal.confirm({
      title: t('确认切换当前生效模式？'),
      content: t(
        '切换不会删除另一套配置；保存后，新请求会按新的计费模式结算，历史日志和账单不会回滚。',
      ),
      onOk: () => handleEffectiveModeChange(nextMode),
    });
  };

  if (loadingModels) {
    return (
      <Spin spinning={true}>
        <div style={{ minHeight: 240 }} />
      </Spin>
    );
  }

  return (
    <Space vertical align='start' style={{ width: '100%' }}>
      <div className='text-sm text-gray-500'>
        {t(
          '高级定价规则用于承载固定价格页无法表达的复杂模型规则。左侧选择模型，右侧查看摘要、编辑规则并做命中预览。',
        )}
      </div>

      <div
        style={{
          width: '100%',
          display: 'grid',
          gap: 16,
          gridTemplateColumns: isMobile
            ? 'minmax(0, 1fr)'
            : 'minmax(280px, 320px) minmax(0, 1fr)',
          alignItems: 'start',
        }}
      >
        <AdvancedPricingModelList
          models={filteredModels}
          selectedModelName={selectedModelName}
          searchText={modelSearchText}
          onSearchTextChange={setModelSearchText}
          onSelectModel={setSelectedModelName}
        />

        <Space vertical align='start' style={{ width: '100%' }}>
          <AdvancedPricingSummary
            selectedModel={selectedModel}
            loading={loading}
            onBackToPricing={handleBackToPricing}
            onSave={handleSave}
            onEffectiveModeChange={handleConfirmEffectiveModeChange}
            onRuleTypeChange={handleRuleTypeChange}
          />

          {!selectedModel ? (
            <Card style={{ width: '100%' }}>
              <Empty
                title={t('暂无模型')}
                description={t('请先从左侧列表选择一个模型')}
              />
            </Card>
          ) : selectedAdvancedConfig.ruleType === TEXT_SEGMENT_RULE_TYPE ? (
            <>
              <TextSegmentRuleEditor
                config={selectedAdvancedConfig}
                rules={selectedAdvancedConfig.rules}
                validationErrors={validationErrors}
                onChange={handleTextSegmentRulesChange}
                onConfigChange={handleTextSegmentConfigChange}
              />
              <AdvancedPricingPreview
                selectedModel={selectedModel}
                selectedAdvancedConfig={selectedAdvancedConfig}
                previewInput={previewInput}
                previewResult={previewResult}
                savePreview={savePreview}
                onPreviewInputChange={handlePreviewInputChange}
              />
            </>
          ) : selectedAdvancedConfig.ruleType === MEDIA_TASK_RULE_TYPE ? (
            <>
              <MediaTaskRuleEditor
                config={selectedAdvancedConfig}
                validationErrors={validationErrors}
                onChange={handleMediaTaskConfigChange}
              />
              <AdvancedPricingPreview
                selectedModel={selectedModel}
                selectedAdvancedConfig={selectedAdvancedConfig}
                previewInput={previewInput}
                previewResult={previewResult}
                savePreview={savePreview}
                onPreviewInputChange={handlePreviewInputChange}
              />
            </>
          ) : (
            <Card title={t('规则编辑器')} style={{ width: '100%' }}>
              <div className='text-sm text-gray-500'>
                {t('当前规则类型暂无可用编辑器。')}
              </div>
            </Card>
          )}
        </Space>
      </div>
    </Space>
  );
}
