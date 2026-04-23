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
  Button,
  Card,
  Empty,
  Input,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

import {
  BILLING_MODE_ADVANCED,
  BILLING_MODE_PER_REQUEST,
} from '../../hooks/modelPricingEditorHelpers';

const { Text } = Typography;

const MODEL_LIST_CONSTRAINT_STYLE = `
.advanced-pricing-model-list,
.advanced-pricing-model-list .semi-card-body,
.advanced-pricing-model-list-scroll,
.advanced-pricing-model-list-items,
.advanced-pricing-model-list-item,
.advanced-pricing-model-list-item .semi-button-content {
  min-width: 0;
  max-width: 100%;
}

.advanced-pricing-model-list {
  overflow: hidden;
}

.advanced-pricing-model-list .semi-card-body {
  overflow: hidden;
}

.advanced-pricing-model-list-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.advanced-pricing-model-list-item {
  box-sizing: border-box;
}

.advanced-pricing-model-list-item .semi-button-content {
  display: block;
  width: 100%;
  overflow: hidden;
}
`;

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

export default function AdvancedPricingModelList({
  models,
  searchText,
  onSearchTextChange,
  selectedModelName,
  onSelectModel,
}) {
  const { t } = useTranslation();

  return (
    <Card
      className='advanced-pricing-model-list'
      title={t('模型列表')}
      bodyStyle={{ padding: 0, minWidth: 0, overflow: 'hidden' }}
      style={{
        height: '100%',
        minWidth: 0,
        maxWidth: '100%',
        overflow: 'hidden',
      }}
    >
      <style>{MODEL_LIST_CONSTRAINT_STYLE}</style>
      <div style={{ padding: 16, borderBottom: '1px solid var(--semi-color-border)' }}>
        <Input
          prefix={<IconSearch />}
          placeholder={t('搜索模型名称')}
          value={searchText}
          onChange={onSearchTextChange}
          showClear
        />
      </div>
      {models.length === 0 ? (
        <div style={{ padding: 24 }}>
          <Empty
            title={t('暂无模型')}
            description={t('当前没有可编辑高级定价规则的模型')}
          />
        </div>
      ) : (
        <div
          className='advanced-pricing-model-list-scroll'
          style={{
            maxHeight: 720,
            overflowY: 'auto',
            overflowX: 'hidden',
            padding: 12,
          }}
        >
          <div
            className='advanced-pricing-model-list-items'
            style={{ width: '100%', minWidth: 0, maxWidth: '100%' }}
          >
            {models.map((model) => {
              const selected = model.name === selectedModelName;
              const billingMode =
                model.effectiveMode ?? model.selectedMode ?? model.billingMode;
              const ruleType = model.ruleType ?? model.advancedRuleType;
              const hasBasePricing =
                model.hasFixedPricing ?? model.hasBasePricing;

              return (
                <Button
                  key={model.name}
                  className='advanced-pricing-model-list-item'
                  theme='borderless'
                  type='tertiary'
                  onClick={() => onSelectModel(model.name)}
                  style={{
                    boxSizing: 'border-box',
                    display: 'flex',
                    width: '100%',
                    maxWidth: '100%',
                    minWidth: 0,
                    padding: 12,
                    height: 'auto',
                    justifyContent: 'flex-start',
                    overflow: 'hidden',
                    borderRadius: 12,
                    textAlign: 'left',
                    border: selected
                      ? '1px solid var(--semi-color-primary)'
                      : '1px solid var(--semi-color-border)',
                    background: selected
                      ? 'var(--semi-color-primary-light-default)'
                      : 'var(--semi-color-bg-1)',
                  }}
                >
                  <div style={{ width: '100%', minWidth: 0, overflow: 'hidden' }}>
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        gap: 8,
                        minWidth: 0,
                        maxWidth: '100%',
                      }}
                    >
                      <div style={{ flex: '1 1 auto', minWidth: 0, maxWidth: '100%' }}>
                        <Tooltip content={model.name}>
                          <Text
                            strong
                            style={{
                              display: 'block',
                              maxWidth: '100%',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {model.name}
                          </Text>
                        </Tooltip>
                      </div>
                      <Tag
                        color={getBillingModeColor(billingMode)}
                        style={{ flexShrink: 0 }}
                      >
                        {getBillingModeText(billingMode, t)}
                      </Tag>
                    </div>
                    <div
                      style={{
                        marginTop: 8,
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                        flexWrap: 'wrap',
                        minWidth: 0,
                        maxWidth: '100%',
                        overflow: 'hidden',
                      }}
                    >
                      <Tag
                        color={ruleType ? 'blue' : 'grey'}
                        style={{ maxWidth: '100%', overflow: 'hidden' }}
                      >
                        {ruleType
                          ? getRuleTypeText(ruleType, t)
                          : t('未配置规则')}
                      </Tag>
                      {hasBasePricing ? (
                        <Tag
                          color='cyan'
                          style={{ maxWidth: '100%', overflow: 'hidden' }}
                        >
                          {t('已有基础定价')}
                        </Tag>
                      ) : null}
                    </div>
                  </div>
                </Button>
              );
            })}
          </div>
        </div>
      )}
    </Card>
  );
}
