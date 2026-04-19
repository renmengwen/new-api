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

import React from 'react';
import {
  Banner,
  Button,
  Card,
  Empty,
  Radio,
  RadioGroup,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  ADVANCED_PRICING_MODE_ADVANCED,
  FIXED_BILLING_MODE_PER_REQUEST,
  FIXED_BILLING_MODE_PER_TOKEN,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
} from '../../hooks/advancedPricingRuleHelpers';

const { Text } = Typography;

const resolveMode = (mode, fixedBillingMode) => {
  if (mode === ADVANCED_PRICING_MODE_ADVANCED) {
    return ADVANCED_PRICING_MODE_ADVANCED;
  }
  if (mode === FIXED_BILLING_MODE_PER_REQUEST) {
    return FIXED_BILLING_MODE_PER_REQUEST;
  }
  if (mode === FIXED_BILLING_MODE_PER_TOKEN) {
    return FIXED_BILLING_MODE_PER_TOKEN;
  }
  return fixedBillingMode || FIXED_BILLING_MODE_PER_TOKEN;
};

const getModeLabel = (mode, fixedBillingMode, t) => {
  if (resolveMode(mode, fixedBillingMode) === ADVANCED_PRICING_MODE_ADVANCED) {
    return t('高级规则');
  }
  return resolveMode(mode, fixedBillingMode) === FIXED_BILLING_MODE_PER_REQUEST
    ? t('固定按次')
    : t('固定按量');
};

const getModeColor = (mode, fixedBillingMode) => {
  const resolvedMode = resolveMode(mode, fixedBillingMode);
  if (resolvedMode === ADVANCED_PRICING_MODE_ADVANCED) {
    return 'orange';
  }
  return resolvedMode === FIXED_BILLING_MODE_PER_REQUEST ? 'teal' : 'violet';
};

const getRuleTypeLabel = (ruleType, t) => {
  if (ruleType === MEDIA_TASK_RULE_TYPE) {
    return t('媒体任务');
  }
  if (ruleType === TEXT_SEGMENT_RULE_TYPE) {
    return t('文本分段');
  }
  return t('未设置');
};

export default function AdvancedPricingSummary({
  selectedModel,
  loading,
  onBackToPricing,
  onSave,
  onEffectiveModeChange,
  onRuleTypeChange,
}) {
  const { t } = useTranslation();
  const hasPendingModeChange =
    selectedModel && selectedModel.effectiveMode !== selectedModel.selectedMode;
  const effectiveModeLabel = selectedModel
    ? getModeLabel(
        selectedModel.effectiveMode,
        selectedModel.fixedBillingMode,
        t,
      )
    : t('未设置');
  const selectedModeLabel = selectedModel
    ? getModeLabel(
        selectedModel.selectedMode,
        selectedModel.fixedBillingMode,
        t,
      )
    : t('未设置');

  const advancedCapabilityTags = Array.from(
    selectedModel?.advancedConfig?.rules?.reduce((result, rule) => {
      [
        ['inputModality', rule?.inputModality ?? rule?.input_modality],
        ['outputModality', rule?.outputModality ?? rule?.output_modality],
        ['billingUnit', rule?.billingUnit ?? rule?.billing_unit],
        ['imageSizeTier', rule?.imageSizeTier ?? rule?.image_size_tier],
        ['toolUsageType', rule?.toolUsageType ?? rule?.tool_usage_type],
      ].forEach(([key, value]) => {
        const normalizedValue = String(value || '').trim();
        if (normalizedValue) {
          result.add(`${key}: ${normalizedValue}`);
        }
      });

      const cacheStoragePrice =
        rule?.cacheStoragePrice ?? rule?.cache_storage_price;
      if (
        cacheStoragePrice !== '' &&
        cacheStoragePrice !== null &&
        cacheStoragePrice !== undefined
      ) {
        result.add(`cacheStoragePrice: ${cacheStoragePrice}`);
      }

      return result;
    }, new Set()) || [],
  );

  return (
    <Card
      title={t('当前模型摘要')}
      headerExtraContent={
        <Space>
          <Button type='tertiary' onClick={() => onBackToPricing(selectedModel?.name)}>
            {t('返回价格设置')}
          </Button>
          <Button type='primary' loading={loading} onClick={onSave}>
            {t('保存高级规则')}
          </Button>
        </Space>
      }
    >
      {!selectedModel ? (
        <Empty
          title={t('暂无模型')}
          description={t('请先从左侧列表选择一个模型')}
        />
      ) : (
        <Space vertical align='start' style={{ width: '100%' }}>
          <Space wrap>
            <Tag size='large' color='blue'>
              {selectedModel.name}
            </Tag>
            <Tag color={getModeColor(selectedModel.effectiveMode, selectedModel.fixedBillingMode)}>
              {t('当前生效')}: {effectiveModeLabel}
            </Tag>
            {hasPendingModeChange ? (
              <Tag
                color={getModeColor(selectedModel.selectedMode, selectedModel.fixedBillingMode)}
              >
                {t('本地草稿')}: {selectedModeLabel}
              </Tag>
            ) : null}
            <Tag color='cyan'>
              {t('规则类型')}: {getRuleTypeLabel(selectedModel.ruleType, t)}
            </Tag>
          </Space>

          <div
            style={{
              width: '100%',
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
              gap: 12,
            }}
          >
            <div>
              <Text type='tertiary'>{t('当前模型')}</Text>
              <div>{selectedModel.name}</div>
            </div>
            <div>
              <Text type='tertiary'>{t('当前生效模式')}</Text>
              <div>{effectiveModeLabel}</div>
            </div>
            <div>
              <Text type='tertiary'>{t('本地草稿模式')}</Text>
              <div>
                {selectedModeLabel}
                {hasPendingModeChange ? (
                  <Tag size='small' color='yellow' style={{ marginLeft: 8 }}>
                    {t('本地未保存')}
                  </Tag>
                ) : null}
              </div>
            </div>
            <div>
              <Text type='tertiary'>{t('规则类型')}</Text>
              <div>{getRuleTypeLabel(selectedModel.ruleType, t)}</div>
            </div>
            <div>
              <Text type='tertiary'>{t('固定价格配置')}</Text>
              <div>
                {selectedModel.hasFixedPricing
                  ? t('已配置')
                  : t('未配置')}
              </div>
            </div>
            <div>
              <Text type='tertiary'>{t('高级规则配置')}</Text>
              <div>
                {selectedModel.hasAdvancedPricing
                  ? t('已配置')
                  : t('未配置')}
              </div>
            </div>
          </div>

          <div>
            <div className='mb-2 font-medium text-gray-700'>
              {t('切换本地草稿模式')}
            </div>
            <RadioGroup
              type='button'
              value={selectedModel.selectedMode}
              onChange={(event) => onEffectiveModeChange(event.target.value)}
            >
              <Radio value={selectedModel.fixedBillingMode || FIXED_BILLING_MODE_PER_TOKEN}>
                {t('固定价格生效')}
              </Radio>
              <Radio
                value={ADVANCED_PRICING_MODE_ADVANCED}
                disabled={!selectedModel.hasAdvancedPricing}
              >
                {t('高级规则生效')}
              </Radio>
            </RadioGroup>
            <div className='mt-2 text-xs text-gray-500'>
              {selectedModel.hasAdvancedPricing
                ? t('切换不会删除另一套配置，保存后新请求才会按新模式结算。')
                : t('当前还没有高级规则配置，需先保存至少一条规则后才能切到高级规则生效。')}
            </div>
          </div>

          <div>
            <div className='mb-2 font-medium text-gray-700'>
              {t('规则类型')}
            </div>
            <RadioGroup
              type='button'
              value={selectedModel.ruleType || TEXT_SEGMENT_RULE_TYPE}
              onChange={(event) => onRuleTypeChange(event.target.value)}
            >
              <Radio value={TEXT_SEGMENT_RULE_TYPE}>
                {t('文本分段')}
              </Radio>
              <Radio value={MEDIA_TASK_RULE_TYPE}>{t('媒体任务')}</Radio>
            </RadioGroup>
            <div className='mt-2 text-xs text-gray-500'>
              {t('规则类型会决定右侧加载哪种编辑器；当前已支持文本分段与媒体任务两种规则编辑。')}
            </div>
          </div>

          {advancedCapabilityTags.length > 0 ? (
            <div>
              <div className='mb-2 font-medium text-gray-700'>
                {t('规则能力')}
              </div>
              <Space wrap>
                {advancedCapabilityTags.map((tag) => (
                  <Tag key={tag} color='cyan'>
                    {tag}
                  </Tag>
                ))}
              </Space>
            </div>
          ) : null}

          {hasPendingModeChange ? (
            <Banner
              type='warning'
              bordered
              fullMode={false}
              closeIcon={null}
              title={t('本地未保存')}
              description={t(
                '当前生效仍为 {{effectiveMode}}，本地草稿已切换为 {{selectedMode}}。保存后新请求才会按新模式结算。',
                {
                  effectiveMode: effectiveModeLabel,
                  selectedMode: selectedModeLabel,
                },
              )}
            />
          ) : selectedModel.hasAdvancedPricing &&
            selectedModel.effectiveMode !== ADVANCED_PRICING_MODE_ADVANCED ? (
            <Banner
              type='warning'
              bordered
              fullMode={false}
              closeIcon={null}
              title={t('高级规则已配置但当前未生效')}
              description={t('如需让新请求按规则结算，请先切换为“高级规则生效”并保存。')}
            />
          ) : null}
        </Space>
      )}
    </Card>
  );
}
