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
import { useTranslation } from 'react-i18next';
import { TEXT_SEGMENT_RULE_TYPE } from '../../hooks/advancedPricingRuleHelpers';

const { Text } = Typography;

export default function AdvancedPricingPreview({
  selectedModel,
  selectedAdvancedConfig,
  previewInput,
  previewResult,
  onPreviewInputChange,
}) {
  const { t } = useTranslation();

  return (
    <Card title={t('命中预览')}>
      {!selectedModel ? (
        <Empty
          title={t('暂无模型')}
          description={t('请先从左侧列表选择一个模型')}
        />
      ) : selectedAdvancedConfig.ruleType !== TEXT_SEGMENT_RULE_TYPE ? (
        <div className='text-sm text-gray-500'>
          {t('媒体任务规则预览将在后续批次补齐，本批先完成文本规则闭环。')}
        </div>
      ) : (
        <Space vertical align='start' style={{ width: '100%' }}>
          <div
            style={{
              width: '100%',
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
              gap: 12,
            }}
          >
            <Input
              value={previewInput.inputTokens}
              onChange={(value) => onPreviewInputChange('inputTokens', value)}
              placeholder={t('输入 input tokens')}
              addonBefore={t('输入 token')}
            />
            <Input
              value={previewInput.outputTokens}
              onChange={(value) => onPreviewInputChange('outputTokens', value)}
              placeholder={t('输入 output tokens')}
              addonBefore={t('输出 token')}
            />
          </div>

          {previewResult?.matchedRule ? (
            <Space vertical align='start' style={{ width: '100%' }}>
              <Space wrap>
                <Tag color='blue'>{previewResult.matchedRule.id}</Tag>
                <Tag color='cyan'>{previewResult.conditionSummary}</Tag>
              </Space>
              <div
                style={{
                  width: '100%',
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
                  gap: 12,
                }}
              >
                <div>
                  <Text type='tertiary'>{t('输入费用')}</Text>
                  <div>${previewResult.priceSummary.inputCost || '0'}</div>
                </div>
                <div>
                  <Text type='tertiary'>{t('输出费用')}</Text>
                  <div>${previewResult.priceSummary.outputCost || '0'}</div>
                </div>
                <div>
                  <Text type='tertiary'>{t('预估总费用')}</Text>
                  <div>${previewResult.priceSummary.totalCost || '0'}</div>
                </div>
                <div>
                  <Text type='tertiary'>{t('缓存读取单价')}</Text>
                  <div>${previewResult.priceSummary.cacheReadPrice || '-'}</div>
                </div>
                <div>
                  <Text type='tertiary'>{t('缓存创建单价')}</Text>
                  <div>${previewResult.priceSummary.cacheWritePrice || '-'}</div>
                </div>
              </div>
            </Space>
          ) : (
            <Empty
              title={t('未命中规则')}
              description={t('输入 token 后会在这里显示命中的 segment 和价格摘要')}
            />
          )}
        </Space>
      )}
    </Card>
  );
}
