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
import { Card, Empty, Input, Radio, RadioGroup, Space, Tag } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  ADVANCED_PRICING_MODE_ADVANCED,
  FIXED_BILLING_MODE_PER_REQUEST,
  FIXED_BILLING_MODE_PER_TOKEN,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
  serializeMediaTaskRule,
  serializeTextSegmentRule,
} from '../../hooks/advancedPricingRuleHelpers';

const TEXT_SEGMENT_PREVIEW_PRICE_FIELDS = [
  { key: 'inputCost', label: '输入费用' },
  { key: 'outputCost', label: '输出费用' },
  { key: 'totalCost', label: '总费用' },
  { key: 'cacheReadPrice', label: '缓存读单价' },
  { key: 'cacheWritePrice', label: '缓存写单价' },
];

const MEDIA_TASK_PREVIEW_PRICE_FIELDS = [
  { key: 'usageTotalTokens', label: '本次上报 Token' },
  { key: 'billableTokens', label: '结算 Token' },
  { key: 'minTokens', label: '最低结算 Token' },
  { key: 'unitPrice', label: '单价' },
  { key: 'draftCoefficient', label: '草稿系数' },
  { key: 'estimatedCost', label: '预估费用' },
];

const BOOLEAN_PREVIEW_OPTIONS = [
  { value: '', label: '不限' },
  { value: 'true', label: '是' },
  { value: 'false', label: '否' },
];

const getEffectiveModeLabel = (mode, t) => {
  if (mode === ADVANCED_PRICING_MODE_ADVANCED) {
    return t('高级规则');
  }
  if (mode === FIXED_BILLING_MODE_PER_REQUEST) {
    return t('固定按次');
  }
  if (mode === FIXED_BILLING_MODE_PER_TOKEN) {
    return t('固定按量');
  }
  return mode || '-';
};

export default function AdvancedPricingPreview({
  selectedModel,
  selectedAdvancedConfig,
  previewInput,
  previewResult,
  savePreview,
  onPreviewInputChange,
}) {
  const { t } = useTranslation();
  const ruleType = selectedAdvancedConfig?.ruleType;
  const enabledRuleCount = Array.isArray(selectedAdvancedConfig?.rules)
    ? selectedAdvancedConfig.rules.filter(
        (rule) => ruleType !== TEXT_SEGMENT_RULE_TYPE || rule?.enabled !== false,
      ).length
    : 0;
  const matchedSegmentPreview =
    previewResult?.matchedSegmentPreview ||
    (previewResult?.matchedRule
      ? ruleType === MEDIA_TASK_RULE_TYPE
        ? serializeMediaTaskRule(previewResult.matchedRule)
        : serializeTextSegmentRule(previewResult.matchedRule)
      : null);
  const savePreviewConfigJson = JSON.stringify(
    savePreview?.configEntry ?? null,
    null,
    2,
  );
  const effectiveModeLabel = getEffectiveModeLabel(
    savePreview?.effectiveMode,
    t,
  );
  const mediaTaskPreviewLabels = {
    usageTotalTokens: t('本次上报 Token'),
    billableTokens: t('结算 Token'),
    minTokens: t('最低结算 Token'),
    unitPrice: t('单价'),
    draftCoefficient: t('草稿系数'),
    estimatedCost: t('预估费用'),
  };

  const renderSavePreviewCard = () => (
    <Card
      bodyStyle={{ padding: 16 }}
      style={{
        width: '100%',
        marginBottom: 16,
        background: 'var(--semi-color-fill-0)',
      }}
      title={t('保存预览')}
    >
      <Space vertical align='start' style={{ width: '100%' }}>
        <div className='text-sm text-gray-500'>
          {t('这里会预览当前模型保存后将写入的配置项，以及保存后会生效的计费模式。')}
        </div>

        <div
          style={{
            width: '100%',
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
            gap: 12,
          }}
        >
          <div>
            <div className='text-xs text-gray-500 mb-1'>{t('当前模型')}</div>
            <div className='font-medium'>{selectedModel?.name || '-'}</div>
          </div>
          <div>
            <div className='text-xs text-gray-500 mb-1'>{t('保存后生效模式')}</div>
            <div className='font-medium'>{effectiveModeLabel}</div>
          </div>
          <div>
            <div className='text-xs text-gray-500 mb-1'>{t('保存配置键')}</div>
            <div className='font-medium'>{savePreview?.configOptionKey || '-'}</div>
          </div>
        </div>

        <div style={{ width: '100%' }}>
          <div className='mb-1 font-medium text-gray-700'>{t('配置预览')}</div>
          <pre
            style={{
              margin: 0,
              padding: 16,
              borderRadius: 12,
              background: 'var(--semi-color-fill-1)',
              border: '1px solid var(--semi-color-border)',
              overflowX: 'auto',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}
          >
            {savePreviewConfigJson}
          </pre>
        </div>
      </Space>
    </Card>
  );

  const renderMatchedPreview = (priceFields) => (
    <>
      <div style={{ width: '100%' }}>
        <div className='mb-1 font-medium text-gray-700'>{t('匹配条件')}</div>
        <Tag color='cyan'>{previewResult?.conditionSummary || '-'}</Tag>
      </div>

      <div style={{ width: '100%' }}>
        <div className='mb-1 font-medium text-gray-700'>{t('预估计费公式')}</div>
        <Tag color='blue'>{previewResult?.formulaSummary || '-'}</Tag>
      </div>

      <div
        style={{
          width: '100%',
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))',
          gap: 12,
        }}
      >
        {priceFields.map((field) => (
          <div
            key={field.key}
            style={{
              padding: 12,
              borderRadius: 8,
              background: 'var(--semi-color-fill-0)',
              border: '1px solid var(--semi-color-border)',
            }}
          >
            <div className='text-xs text-gray-500 mb-1'>
              {mediaTaskPreviewLabels[field.key] || t(field.label)}
            </div>
            <div className='font-medium'>
              {previewResult?.priceSummary?.[field.key] || '-'}
            </div>
          </div>
        ))}
      </div>

      <Card
        bodyStyle={{ padding: 16 }}
        style={{
          width: '100%',
          background: 'var(--semi-color-fill-0)',
        }}
        title={t('日志说明预览')}
      >
        <Space vertical align='start' style={{ width: '100%' }}>
          <div>
            <div className='text-xs text-gray-500 mb-1'>{t('日志详情')}</div>
            <div className='font-medium'>
              {previewResult?.logPreview?.detailSummary || '-'}
            </div>
          </div>
          <div>
            <div className='text-xs text-gray-500 mb-1'>{t('计费过程')}</div>
            <div className='font-medium'>
              {previewResult?.logPreview?.processSummary || '-'}
            </div>
          </div>
        </Space>
      </Card>

      <div style={{ width: '100%' }}>
        <div className='mb-1 font-medium text-gray-700'>
          {t('命中规则 JSON')}
        </div>
        <pre
          style={{
            margin: 0,
            padding: 16,
            borderRadius: 12,
            background: 'var(--semi-color-fill-0)',
            border: '1px solid var(--semi-color-border)',
            overflowX: 'auto',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {JSON.stringify(matchedSegmentPreview, null, 2)}
        </pre>
      </div>
    </>
  );

  if (ruleType !== TEXT_SEGMENT_RULE_TYPE && ruleType !== MEDIA_TASK_RULE_TYPE) {
    return (
      <Card title={t('规则命中预览')}>
        <Empty
          title={t('暂无预览')}
          description={t('当前规则类型暂不支持此预览。')}
        />
      </Card>
    );
  }

  return (
    <Card title={t('规则命中预览')}>
      {renderSavePreviewCard()}
      {ruleType === TEXT_SEGMENT_RULE_TYPE ? (
        <Space vertical align='start' style={{ width: '100%' }}>
          <div className='text-sm text-gray-500 mb-3'>
            {t('输入输入/输出 Token 数量后，这里会展示命中的文本分段规则与价格摘要。')}
          </div>

          <div
            style={{
              width: '100%',
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
              gap: 12,
            }}
          >
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('输入 token')}</div>
              <Input
                value={previewInput?.inputTokens || ''}
                placeholder={t('例如 8000')}
                onChange={(value) => onPreviewInputChange?.('inputTokens', value)}
              />
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('输出 token')}</div>
              <Input
                value={previewInput?.outputTokens || ''}
                placeholder={t('例如 2000')}
                onChange={(value) => onPreviewInputChange?.('outputTokens', value)}
              />
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('服务层级')}</div>
              <Input
                value={previewInput?.serviceTier || ''}
                placeholder={t('例如 standard / priority')}
                onChange={(value) => onPreviewInputChange?.('serviceTier', value)}
              />
            </div>
          </div>

          <Space wrap>
            <Tag color='blue'>
              {t('已启用规则 {{count}} 条', { count: enabledRuleCount })}
            </Tag>
            <Tag color={previewResult?.matchedRule ? 'green' : 'grey'}>
              {previewResult?.matchedRule ? t('已命中规则') : t('未命中规则')}
            </Tag>
            {previewResult?.matchedRule ? (
              <Tag color='violet'>
                {t('优先级 {{priority}}', {
                  priority: previewResult.matchedRule.priority || '-',
                })}
              </Tag>
            ) : null}
          </Space>

          {enabledRuleCount === 0 ? (
            <Empty
              title={t('暂无预览')}
              description={t('新增或编辑文本分段规则后，这里会显示命中结果与费用摘要。')}
            />
          ) : previewResult?.matchedRule ? (
            renderMatchedPreview(TEXT_SEGMENT_PREVIEW_PRICE_FIELDS)
          ) : (
            <Empty
              title={t('未命中规则')}
              description={t('调整输入或输出 token 数量后，这里会显示命中的文本分段规则。')}
            />
          )}
        </Space>
      ) : (
        <Space vertical align='start' style={{ width: '100%' }}>
          <div className='text-sm text-gray-500 mb-3'>
            {t('输入媒体任务条件后，这里会展示命中的 segment、计费公式、日志说明与 JSON 预览。')}
          </div>

          <Space wrap>
            <Tag color='blue'>
              {`${t('任务类型')}：${selectedAdvancedConfig?.taskType || '-'}`}
            </Tag>
            <Tag color='cyan'>
              {`${t('计费单位')}：${selectedAdvancedConfig?.billingUnit || '-'}`}
            </Tag>
            <Tag color='violet'>{`${t('规则数')}：${enabledRuleCount}`}</Tag>
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
              <div className='mb-1 font-medium text-gray-700'>{t('任务动作')}</div>
              <Input
                value={previewInput?.rawAction || ''}
                placeholder={t('如 generate / firstTailGenerate')}
                onChange={(value) => onPreviewInputChange?.('rawAction', value)}
              />
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('推理模式')}</div>
              <Input
                value={previewInput?.inferenceMode || ''}
                placeholder={t('如 fast / quality')}
                onChange={(value) => onPreviewInputChange?.('inferenceMode', value)}
              />
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('分辨率')}</div>
              <Input
                value={previewInput?.resolution || ''}
                placeholder={t('如 720p / 1080p')}
                onChange={(value) => onPreviewInputChange?.('resolution', value)}
              />
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('宽高比')}</div>
              <Input
                value={previewInput?.aspectRatio || ''}
                placeholder={t('如 16:9 / 9:16')}
                onChange={(value) => onPreviewInputChange?.('aspectRatio', value)}
              />
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('输出时长')}</div>
              <Input
                value={previewInput?.outputDuration || ''}
                placeholder={t('秒')}
                onChange={(value) => onPreviewInputChange?.('outputDuration', value)}
              />
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('输入视频时长')}</div>
              <Input
                value={previewInput?.inputVideoDuration || ''}
                placeholder={t('秒')}
                onChange={(value) =>
                  onPreviewInputChange?.('inputVideoDuration', value)
                }
              />
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>
                {t('本次上报 Token')}
              </div>
              <Input
                value={previewInput?.usageTotalTokens || ''}
                placeholder={t('如 2400')}
                onChange={(value) =>
                  onPreviewInputChange?.('usageTotalTokens', value)
                }
              />
            </div>
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-2 font-medium text-gray-700'>{t('音频')}</div>
            <RadioGroup
              type='button'
              value={previewInput?.audio || ''}
              onChange={(event) =>
                onPreviewInputChange?.('audio', event.target.value)
              }
            >
              {BOOLEAN_PREVIEW_OPTIONS.map((option) => (
                <Radio key={option.value || 'audio-any'} value={option.value}>
                  {t(option.label)}
                </Radio>
              ))}
            </RadioGroup>
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-2 font-medium text-gray-700'>{t('输入视频')}</div>
            <RadioGroup
              type='button'
              value={previewInput?.inputVideo || ''}
              onChange={(event) =>
                onPreviewInputChange?.('inputVideo', event.target.value)
              }
            >
              {BOOLEAN_PREVIEW_OPTIONS.map((option) => (
                <Radio key={option.value || 'video-any'} value={option.value}>
                  {t(option.label)}
                </Radio>
              ))}
            </RadioGroup>
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-2 font-medium text-gray-700'>{t('草稿模式')}</div>
            <RadioGroup
              type='button'
              value={previewInput?.draft || ''}
              onChange={(event) =>
                onPreviewInputChange?.('draft', event.target.value)
              }
            >
              {BOOLEAN_PREVIEW_OPTIONS.map((option) => (
                <Radio key={option.value || 'draft-any'} value={option.value}>
                  {t(option.label)}
                </Radio>
              ))}
            </RadioGroup>
          </div>

          <Space wrap>
            <Tag color={previewResult?.matchedRule ? 'green' : 'grey'}>
              {previewResult?.matchedRule ? t('已命中规则') : t('未命中规则')}
            </Tag>
            {previewResult?.matchedRule ? (
              <Tag color='violet'>
                {t('优先级 {{priority}}', {
                  priority: previewResult.matchedRule.priority || '-',
                })}
              </Tag>
            ) : null}
          </Space>

          {enabledRuleCount === 0 ? (
            <Empty
              title={t('暂无预览')}
              description={t('新增或编辑媒体任务规则后，这里会显示命中结果与计费摘要。')}
            />
          ) : previewResult?.matchedRule ? (
            renderMatchedPreview(MEDIA_TASK_PREVIEW_PRICE_FIELDS)
          ) : (
            <Empty
              title={t('未命中规则')}
              description={t('调整任务动作、时长、草稿模式等条件后，这里会显示命中的媒体任务规则。')}
            />
          )}
        </Space>
      )}
    </Card>
  );
}
