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
import TextSegmentRuleEditor from './components/advanced-pricing/TextSegmentRuleEditor';
import { useAdvancedPricingRulesState } from './hooks/useAdvancedPricingRulesState';

export default function AdvancedPricingRulesPage(props) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [enabledModels, setEnabledModels] = useState([]);
  const [loadingModels, setLoadingModels] = useState(true);

  useEffect(() => {
    let active = true;

    const loadEnabledModels = async () => {
      setLoadingModels(true);
      try {
        const response = await API.get('/api/channel/models_enabled');
        const { success, message, data } = response.data;

        if (!active) {
          return;
        }

        if (success) {
          setEnabledModels(Array.isArray(data) ? data : []);
        } else {
          setEnabledModels([]);
          showError(message);
        }
      } catch (error) {
        if (active) {
          console.error('获取启用模型失败:', error);
          setEnabledModels([]);
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
  }, [t]);

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
    handleEffectiveModeChange,
    handleRuleTypeChange,
    handleTextSegmentRulesChange,
    handlePreviewInputChange,
    handleSave,
  } = useAdvancedPricingRulesState({
    options: props.options,
    refresh: props.refresh,
    t,
    candidateModelNames: enabledModels,
    selectedModelName: props.selectedModelName || '',
    onSelectedModelChange: props.onSelectedModelChange,
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
          ) : selectedAdvancedConfig.ruleType === 'text_segment' ? (
            <>
              <TextSegmentRuleEditor
                rules={selectedAdvancedConfig.rules}
                validationErrors={validationErrors}
                onChange={handleTextSegmentRulesChange}
              />
              <AdvancedPricingPreview
                selectedModel={selectedModel}
                selectedAdvancedConfig={selectedAdvancedConfig}
                previewInput={previewInput}
                previewResult={previewResult}
                onPreviewInputChange={handlePreviewInputChange}
              />
            </>
          ) : (
            <>
              <Card title={t('媒体任务规则编辑器')} style={{ width: '100%' }}>
                <div className='text-sm text-gray-500'>
                  {t(
                    '媒体任务规则编辑器本批先保留页面壳子，已完成规则类型切换与状态摘要，后续批次再补齐完整编辑闭环。',
                  )}
                </div>
              </Card>
              <AdvancedPricingPreview
                selectedModel={selectedModel}
                selectedAdvancedConfig={selectedAdvancedConfig}
                previewInput={previewInput}
                previewResult={previewResult}
                onPreviewInputChange={handlePreviewInputChange}
              />
            </>
          )}
        </Space>
      </div>
    </Space>
  );
}
