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
import { Button, Card, Empty, Radio, RadioGroup, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

import {
  BILLING_MODE_ADVANCED,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_PER_TOKEN,
} from '../../hooks/modelPricingEditorHelpers';

const { Text } = Typography;

const getBillingModeText = (billingMode, t) => {
  if (billingMode === BILLING_MODE_ADVANCED) {
    return t('高级规则');
  }
  if (billingMode === BILLING_MODE_PER_REQUEST) {
    return t('按次计费');
  }
  return t('按量计费');
};

const getBillingModeColor = (billingMode) => {
  if (billingMode === BILLING_MODE_ADVANCED) {
    return 'orange';
  }
  if (billingMode === BILLING_MODE_PER_REQUEST) {
    return 'teal';
  }
  return 'violet';
};

const getRuleTypeText = (ruleType, t) => {
  if (ruleType === 'media_task') {
    return t('媒体任务规则');
  }
  return t('文本分段规则');
};

export default function AdvancedPricingSummary({
  selectedModel,
  currentBillingMode,
  draftBillingMode,
  currentRuleType,
  onBillingModeChange,
  onSave,
  saving,
}) {
  const { t } = useTranslation();

  return (
    <Card
      title={t('当前模型摘要')}
      headerExtraContent={
        selectedModel ? (
          <Tag color='blue'>{selectedModel.name}</Tag>
        ) : null
      }
    >
      {!selectedModel ? (
        <Empty
          title={t('尚未选择模型')}
          description={t('请先从左侧列表选择一个模型')}
        />
      ) : (
        <Space vertical align='start' style={{ width: '100%' }}>
          <div>
            <Text type='tertiary'>{t('当前生效计费模式')}</Text>
            <div className='mt-2'>
              <Tag color={getBillingModeColor(currentBillingMode)}>
                {getBillingModeText(currentBillingMode, t)}
              </Tag>
            </div>
          </div>
          <div>
            <Text type='tertiary'>{t('当前规则类型')}</Text>
            <div className='mt-2'>
              <Tag color='blue'>{getRuleTypeText(currentRuleType, t)}</Tag>
            </div>
          </div>
          <div style={{ width: '100%' }}>
            <div className='mb-2 font-medium'>{t('保存后的计费模式')}</div>
            <RadioGroup
              type='button'
              value={draftBillingMode}
              onChange={(event) => onBillingModeChange(event.target.value)}
            >
              <Radio value={BILLING_MODE_PER_TOKEN}>{t('按量计费')}</Radio>
              <Radio value={BILLING_MODE_PER_REQUEST}>{t('按次计费')}</Radio>
              <Radio value={BILLING_MODE_ADVANCED}>{t('高级规则')}</Radio>
            </RadioGroup>
          </div>
          <div className='text-xs text-gray-500'>
            {t(
              '这个页面先提供高级规则壳层：可查看当前状态、切换规则类型，并保存 AdvancedPricingMode 与 AdvancedPricingRules。',
            )}
          </div>
          <Button type='primary' loading={saving} onClick={onSave}>
            {t('保存高级规则')}
          </Button>
        </Space>
      )}
    </Card>
  );
}
