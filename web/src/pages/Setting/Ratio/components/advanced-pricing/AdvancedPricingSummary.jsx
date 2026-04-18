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
  ADVANCED_PRICING_MODE_FIXED,
  FIXED_BILLING_MODE_PER_REQUEST,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
} from '../../hooks/advancedPricingRuleHelpers';

const { Text } = Typography;

const getModeLabel = (model, t) => {
  if (!model) {
    return t('未设置');
  }
  if (model.effectiveMode === ADVANCED_PRICING_MODE_ADVANCED) {
    return t('高级规则');
  }
  return model.fixedBillingMode === FIXED_BILLING_MODE_PER_REQUEST
    ? t('固定按次')
    : t('固定按量');
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
            <Tag
              color={
                selectedModel.effectiveMode === ADVANCED_PRICING_MODE_ADVANCED
                  ? 'orange'
                  : selectedModel.fixedBillingMode === FIXED_BILLING_MODE_PER_REQUEST
                    ? 'teal'
                    : 'violet'
              }
            >
              {getModeLabel(selectedModel, t)}
            </Tag>
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
              <div>{getModeLabel(selectedModel, t)}</div>
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
              {t('切换当前生效模式')}
            </div>
            <RadioGroup
              type='button'
              value={selectedModel.selectedMode}
              onChange={(event) => onEffectiveModeChange(event.target.value)}
            >
              <Radio value={ADVANCED_PRICING_MODE_FIXED}>
                {t('固定价格生效')}
              </Radio>
              <Radio value={ADVANCED_PRICING_MODE_ADVANCED}>
                {t('高级规则生效')}
              </Radio>
            </RadioGroup>
            <div className='mt-2 text-xs text-gray-500'>
              {t('切换不会删除另一套配置，保存后新请求才会按新模式结算。')}
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
              {t('规则类型会决定右侧加载哪种编辑器；本批媒体任务编辑器先保留浅壳。')}
            </div>
          </div>

          {selectedModel.hasAdvancedPricing &&
          selectedModel.selectedMode !== ADVANCED_PRICING_MODE_ADVANCED ? (
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
