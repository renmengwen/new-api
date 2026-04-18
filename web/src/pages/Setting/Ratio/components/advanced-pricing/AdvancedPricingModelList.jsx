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
import { Card, Empty, Input, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  ADVANCED_PRICING_MODE_ADVANCED,
  FIXED_BILLING_MODE_PER_REQUEST,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
} from '../../hooks/advancedPricingRuleHelpers';

const { Text } = Typography;

const getModeLabel = (model, t) => {
  if (model?.effectiveMode === ADVANCED_PRICING_MODE_ADVANCED) {
    return t('高级规则');
  }
  return model?.fixedBillingMode === FIXED_BILLING_MODE_PER_REQUEST
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

export default function AdvancedPricingModelList({
  models,
  selectedModelName,
  searchText,
  onSearchTextChange,
  onSelectModel,
}) {
  const { t } = useTranslation();

  return (
    <Card
      title={t('模型列表')}
      bodyStyle={{ padding: 12 }}
      style={{ height: '100%' }}
    >
      <Input
        prefix={<IconSearch />}
        placeholder={t('搜索模型名称')}
        value={searchText}
        onChange={onSearchTextChange}
        showClear
        style={{ marginBottom: 12 }}
      />

      {models.length === 0 ? (
        <Empty
          title={t('暂无模型')}
          description={t('当前没有可编辑的已启用模型')}
        />
      ) : (
        <Space vertical align='stretch' style={{ width: '100%' }}>
          {models.map((model) => {
            const selected = model.name === selectedModelName;

            return (
              <div
                key={model.name}
                onClick={() => onSelectModel(model.name)}
                role='button'
                tabIndex={0}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    onSelectModel(model.name);
                  }
                }}
                style={{
                  cursor: 'pointer',
                  padding: 12,
                  borderRadius: 8,
                  border: selected
                    ? '1px solid var(--semi-color-primary)'
                    : '1px solid var(--semi-color-border)',
                  background: selected
                    ? 'var(--semi-color-primary-light-default)'
                    : 'var(--semi-color-bg-1)',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    gap: 12,
                    marginBottom: 8,
                  }}
                >
                  <Text strong>{model.name}</Text>
                  {selected ? (
                    <Tag color='blue'>{t('当前编辑')}</Tag>
                  ) : null}
                </div>
                <Space wrap>
                  <Tag
                    color={
                      model.effectiveMode === ADVANCED_PRICING_MODE_ADVANCED
                        ? 'orange'
                        : model.fixedBillingMode === FIXED_BILLING_MODE_PER_REQUEST
                          ? 'teal'
                          : 'violet'
                    }
                  >
                    {getModeLabel(model, t)}
                  </Tag>
                  <Tag color='cyan'>{getRuleTypeLabel(model.ruleType, t)}</Tag>
                  <Tag color={model.hasFixedPricing ? 'green' : 'grey'}>
                    {model.hasFixedPricing
                      ? t('已配置固定价格')
                      : t('未配置固定价格')}
                  </Tag>
                  <Tag color={model.hasAdvancedPricing ? 'blue' : 'grey'}>
                    {model.hasAdvancedPricing
                      ? t('已配置高级规则')
                      : t('未配置高级规则')}
                  </Tag>
                </Space>
              </div>
            );
          })}
        </Space>
      )}
    </Card>
  );
}
