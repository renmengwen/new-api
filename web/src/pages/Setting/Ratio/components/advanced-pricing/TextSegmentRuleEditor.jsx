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

import React, { useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Input,
  Radio,
  RadioGroup,
  SideSheet,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconPlus } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  buildTextSegmentConditionSummary,
  buildTextSegmentPreview,
  createEmptyTextSegmentRule,
  getTextSegmentRuleEditorMeta,
  normalizeTextSegmentRule,
  serializeAdvancedPricingConfig,
  serializeTextSegmentRule,
  sortTextSegmentRules,
  validateTextSegmentRules,
} from '../../hooks/advancedPricingRuleHelpers';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';

const { TextArea } = Input;
const { Text } = Typography;
const INTEGER_INPUT_REGEX = /^\d*$/;
const DECIMAL_INPUT_REGEX = /^(\d+(\.\d*)?|\.\d*)?$/;
const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

const TEXT_SEGMENT_FIELDS = [
  {
    field: 'priority',
    label: '优先级',
    placeholder: '越小越优先',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'inputMin',
    label: '输入最小值',
    placeholder: '留空表示不限',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'inputMax',
    label: '输入最大值',
    placeholder: '留空表示不限',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'outputMin',
    label: '输出最小值',
    placeholder: '留空表示不限',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'outputMax',
    label: '输出最大值',
    placeholder: '留空表示不限',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'serviceTier',
    label: '服务层级',
    placeholder: '例如：standard / premium',
  },
  {
    field: 'inputPrice',
    label: '输入单价',
    placeholder: '必填',
    regex: DECIMAL_INPUT_REGEX,
  },
  {
    field: 'outputPrice',
    label: '输出单价',
    placeholder: '可选',
    regex: DECIMAL_INPUT_REGEX,
  },
  {
    field: 'cacheReadPrice',
    label: '缓存读单价',
    placeholder: '可选',
    regex: DECIMAL_INPUT_REGEX,
  },
  {
    field: 'cacheWritePrice',
    label: '缓存写单价',
    placeholder: '可选',
    regex: DECIMAL_INPUT_REGEX,
  },
];

function TextSegmentRulesEditor({ rules, validationErrors, onChange }) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [sideSheetVisible, setSideSheetVisible] = useState(false);
  const [editingRuleId, setEditingRuleId] = useState('');
  const [draftRule, setDraftRule] = useState(createEmptyTextSegmentRule(1));
  const [draftErrors, setDraftErrors] = useState([]);
  const [sheetPreviewInput, setSheetPreviewInput] = useState({
    inputTokens: '',
    outputTokens: '',
    serviceTier: '',
  });

  const sortedRules = useMemo(() => sortTextSegmentRules(rules || []), [rules]);
  const previewInput = sheetPreviewInput;

  const resetDraftState = () => {
    setEditingRuleId('');
    setDraftRule(createEmptyTextSegmentRule(Date.now()));
    setDraftErrors([]);
    setSheetPreviewInput({
      inputTokens: '',
      outputTokens: '',
      serviceTier: '',
    });
  };

  const handleCloseSideSheet = () => {
    setSideSheetVisible(false);
    resetDraftState();
  };

  const openCreateSideSheet = () => {
    resetDraftState();
    setSideSheetVisible(true);
  };

  const openEditSideSheet = (rule) => {
    setEditingRuleId(rule.id);
    setDraftRule(normalizeTextSegmentRule(rule));
    setDraftErrors([]);
    setSideSheetVisible(true);
  };

  const buildNextRules = (candidateRule) =>
    editingRuleId
      ? sortedRules.map((rule) =>
          rule.id === editingRuleId ? candidateRule : rule,
        )
      : [...sortedRules, candidateRule];

  const candidatePreviewRules = useMemo(
    () => buildNextRules(normalizeTextSegmentRule(draftRule)),
    [draftRule, editingRuleId, sortedRules],
  );
  const sheetPreviewResult = useMemo(
    () => buildTextSegmentPreview(candidatePreviewRules, previewInput),
    [candidatePreviewRules, previewInput],
  );

  const resolveDraftErrors = (candidateRule) => {
    const candidateRuleIdPattern = candidateRule.id
      ? new RegExp(`(^|[\\s:/])${escapeRegExp(candidateRule.id)}(?=$|[\\s:/])`)
      : null;

    return validateTextSegmentRules(buildNextRules(candidateRule)).filter(
      (error) =>
        candidateRuleIdPattern &&
        error.includes(candidateRule.id) &&
        candidateRuleIdPattern.test(error),
    );
  };

  const handleDraftFieldChange = (field, value, regex) => {
    if (regex && !regex.test(value)) {
      return;
    }

    const nextDraftRule = {
      ...draftRule,
      [field]: value,
    };

    setDraftRule(nextDraftRule);
    setDraftErrors(resolveDraftErrors(nextDraftRule));
  };

  const handleEnabledChange = (value) => {
    const nextDraftRule = {
      ...draftRule,
      enabled: value === 'true',
    };

    setDraftRule(nextDraftRule);
    setDraftErrors(resolveDraftErrors(nextDraftRule));
  };

  const handlePreviewInputChange = (field, value) => {
    if (field !== 'serviceTier' && !INTEGER_INPUT_REGEX.test(value)) {
      return;
    }

    setSheetPreviewInput((currentValue) => ({
      ...currentValue,
      [field]: value,
    }));
  };

  const handleSaveDraft = () => {
    const candidateRule = normalizeTextSegmentRule(draftRule);
    const nextDraftErrors = resolveDraftErrors(candidateRule);

    if (nextDraftErrors.length > 0) {
      setDraftErrors(nextDraftErrors);
      return;
    }

    onChange(sortTextSegmentRules(buildNextRules(candidateRule)));
    handleCloseSideSheet();
  };

  const handleDeleteRule = (ruleId) => {
    onChange(sortedRules.filter((rule) => rule.id !== ruleId));
  };

  const columns = useMemo(
    () => [
      {
        title: t('优先级'),
        dataIndex: 'priority',
        key: 'priority',
        render: (value) => value || '-',
      },
      {
        title: t('状态'),
        dataIndex: 'enabled',
        key: 'enabled',
        render: (value) => (
          <Tag color={value === false ? 'grey' : 'green'}>
            {value === false ? t('停用') : t('启用')}
          </Tag>
        ),
      },
      {
        title: t('条件摘要'),
        key: 'conditionSummary',
        render: (_, record) => (
          <Tag color='cyan'>{buildTextSegmentConditionSummary(record)}</Tag>
        ),
      },
      {
        title: t('服务层级'),
        dataIndex: 'serviceTier',
        key: 'serviceTier',
        render: (value) => value || '-',
      },
      {
        title: t('计费摘要'),
        key: 'billingSummary',
        render: (_, record) => (
          <Space wrap>
            <Tag color='blue'>{`${t('输入单价')}：${record.inputPrice || '-'}`}</Tag>
            <Tag color='green'>{`${t('输出单价')}：${record.outputPrice || '-'}`}</Tag>
            <Tag color='cyan'>
              {`${t('缓存读单价')}：${record.cacheReadPrice || '-'}`}
            </Tag>
            <Tag color='violet'>
              {`${t('缓存写单价')}：${record.cacheWritePrice || '-'}`}
            </Tag>
          </Space>
        ),
      },
      {
        title: t('操作'),
        key: 'action',
        render: (_, record) => (
          <Space>
            <Button
              size='small'
              type='tertiary'
              icon={<IconEdit />}
              onClick={() => openEditSideSheet(record)}
            >
              {t('编辑')}
            </Button>
            <Button
              size='small'
              type='danger'
              icon={<IconDelete />}
              onClick={() => handleDeleteRule(record.id)}
            >
              {t('删除')}
            </Button>
          </Space>
        ),
      },
    ],
    [t],
  );

  return (
    <>
      <Card
        title={t('规则列表')}
        headerExtraContent={
          <Button icon={<IconPlus />} onClick={openCreateSideSheet}>
            {t('新增规则')}
          </Button>
        }
      >
        <div className='text-sm text-gray-500 mb-3'>
          {t(
            '在这里维护文本请求的分段区间、优先级与价格；左侧列表用于快速查看，右侧抽屉用于新增和编辑规则。',
          )}
        </div>

        {validationErrors.length > 0 ? (
          <Banner
            type='warning'
            bordered
            fullMode={false}
            closeIcon={null}
            style={{ marginBottom: 16 }}
            title={t('当前文本分段规则存在待处理问题')}
            description={
              <Space vertical align='start'>
                {validationErrors.map((error) => (
                  <Text key={error}>{error}</Text>
                ))}
              </Space>
            }
          />
        ) : null}

        <Table
          columns={columns}
          dataSource={sortedRules}
          rowKey='id'
          pagination={false}
          empty={
            <div style={{ textAlign: 'center', padding: 16 }}>
              {t('暂无文本规则，点击右上角新增规则')}
            </div>
          }
        />

        <Card
          bodyStyle={{ padding: 16 }}
          style={{
            marginTop: 16,
            background: 'var(--semi-color-fill-0)',
          }}
          title={t('当前规则 JSON')}
        >
          <pre
            style={{
              margin: 0,
              padding: 12,
              borderRadius: 8,
              background: 'var(--semi-color-fill-1)',
              width: '100%',
              overflowX: 'auto',
            }}
          >
            {JSON.stringify(
              sortedRules
                .filter((rule) => rule?.enabled !== false)
                .map((rule) => serializeTextSegmentRule(rule)),
              null,
              2,
            )}
          </pre>
        </Card>
      </Card>

      <SideSheet
        placement='right'
        title={
          <Space>
            <Tag color='blue' shape='circle'>
              {editingRuleId ? t('编辑') : t('新增')}
            </Tag>
            <Text strong>
              {editingRuleId ? t('编辑文本规则') : t('新增文本规则')}
            </Text>
          </Space>
        }
        visible={sideSheetVisible}
        onCancel={handleCloseSideSheet}
        width={isMobile ? '100%' : 720}
        bodyStyle={{ padding: 16 }}
        footer={
          <Space>
            <Button onClick={handleCloseSideSheet}>{t('取消')}</Button>
            <Button theme='solid' type='primary' onClick={handleSaveDraft}>
              {editingRuleId ? t('保存规则') : t('新增规则')}
            </Button>
          </Space>
        }
        closeIcon={null}
      >
        <Space vertical align='start' style={{ width: '100%' }}>
          <div
            style={{
              width: '100%',
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
              gap: 12,
            }}
          >
            {TEXT_SEGMENT_FIELDS.map((fieldMeta) => (
              <div key={fieldMeta.field}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t(fieldMeta.label)}
                </div>
                <Input
                  value={draftRule[fieldMeta.field]}
                  placeholder={t(fieldMeta.placeholder)}
                  onChange={(value) =>
                    handleDraftFieldChange(
                      fieldMeta.field,
                      value,
                      fieldMeta.regex,
                    )
                  }
                />
              </div>
            ))}
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-2 font-medium text-gray-700'>{t('状态')}</div>
            <RadioGroup
              type='button'
              value={draftRule.enabled === false ? 'false' : 'true'}
              onChange={(event) => handleEnabledChange(event.target.value)}
            >
              <Radio value='true'>{t('启用')}</Radio>
              <Radio value='false'>{t('停用')}</Radio>
            </RadioGroup>
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-1 font-medium text-gray-700'>{t('条件摘要')}</div>
            <Tag color={draftRule.enabled === false ? 'grey' : 'cyan'}>
              {buildTextSegmentConditionSummary(draftRule)}
            </Tag>
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-1 font-medium text-gray-700'>
              {t('生成的规则 JSON')}
            </div>
            <pre
              style={{
                margin: 0,
                padding: 12,
                borderRadius: 8,
                background: 'var(--semi-color-fill-1)',
                width: '100%',
                overflowX: 'auto',
              }}
            >
              {JSON.stringify(serializeTextSegmentRule(draftRule), null, 2)}
            </pre>
          </div>

          {draftErrors.length > 0 ? (
            <Banner
              type='danger'
              bordered
              fullMode={false}
              closeIcon={null}
              title={t('请先修正以下问题')}
              description={
                <Space vertical align='start'>
                  {draftErrors.map((error) => (
                    <Text key={error}>{error}</Text>
                  ))}
                </Space>
              }
            />
          ) : null}

          <Card
            bodyStyle={{ padding: 16 }}
            style={{
              width: '100%',
              background: 'var(--semi-color-fill-0)',
            }}
            title={t('规则命中预览')}
          >
            <Space vertical align='start' style={{ width: '100%' }}>
              <div className='text-sm text-gray-500'>
                {t('输入预览参数后，这里会展示保存当前草稿后规则集的命中结果。')}
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
                  <div className='mb-1 font-medium text-gray-700'>
                    {t('输入 token')}
                  </div>
                  <Input
                    value={previewInput?.inputTokens || ''}
                    placeholder={t('例如 8000')}
                    onChange={(value) =>
                      handlePreviewInputChange('inputTokens', value)
                    }
                  />
                </div>
                <div>
                  <div className='mb-1 font-medium text-gray-700'>
                    {t('输出 token')}
                  </div>
                  <Input
                    value={previewInput?.outputTokens || ''}
                    placeholder={t('例如 2000')}
                    onChange={(value) =>
                      handlePreviewInputChange('outputTokens', value)
                    }
                  />
                </div>
                <div>
                  <div className='mb-1 font-medium text-gray-700'>
                    {t('服务层级')}
                  </div>
                  <Input
                    value={previewInput?.serviceTier || ''}
                    placeholder={t('例如 standard / priority')}
                    onChange={(value) =>
                      handlePreviewInputChange('serviceTier', value)
                    }
                  />
                </div>
              </div>

              <Space wrap>
                <Tag color='blue'>
                  {`${t('规则总数')}：${candidatePreviewRules.length}`}
                </Tag>
                <Tag color={sheetPreviewResult?.matchedRule ? 'green' : 'grey'}>
                  {sheetPreviewResult?.matchedRule
                    ? t('已命中规则')
                    : t('未命中规则')}
                </Tag>
                {sheetPreviewResult?.matchedRule ? (
                  <Tag color='violet'>
                    {`${t('优先级')}：${sheetPreviewResult.matchedRule.priority || '-'}`}
                  </Tag>
                ) : null}
              </Space>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('匹配条件')}
                </div>
                <Tag color='cyan'>
                  {sheetPreviewResult?.conditionSummary || '-'}
                </Tag>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('预计计费公式')}
                </div>
                <Tag color='blue'>
                  {sheetPreviewResult?.formulaSummary || '-'}
                </Tag>
              </div>

              <div
                style={{
                  width: '100%',
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
                  gap: 12,
                }}
              >
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('输入费用')}
                  </div>
                  <div className='font-medium'>
                    {sheetPreviewResult?.priceSummary?.inputCost || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('输出费用')}
                  </div>
                  <div className='font-medium'>
                    {sheetPreviewResult?.priceSummary?.outputCost || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('总费用')}
                  </div>
                  <div className='font-medium'>
                    {sheetPreviewResult?.priceSummary?.totalCost || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('缓存读单价')}
                  </div>
                  <div className='font-medium'>
                    {sheetPreviewResult?.priceSummary?.cacheReadPrice || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('缓存写单价')}
                  </div>
                  <div className='font-medium'>
                    {sheetPreviewResult?.priceSummary?.cacheWritePrice || '-'}
                  </div>
                </div>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('日志详情')}
                </div>
                <div>{sheetPreviewResult?.logPreview?.detailSummary || '-'}</div>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('计费过程')}
                </div>
                <div>{sheetPreviewResult?.logPreview?.processSummary || '-'}</div>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('命中规则 JSON')}
                </div>
                <pre
                  style={{
                    margin: 0,
                    padding: 12,
                    borderRadius: 8,
                    background: 'var(--semi-color-fill-1)',
                    width: '100%',
                    overflowX: 'auto',
                  }}
                >
                  {JSON.stringify(
                    sheetPreviewResult?.matchedRule
                      ? serializeTextSegmentRule(sheetPreviewResult.matchedRule)
                      : null,
                    null,
                    2,
                  )}
                </pre>
              </div>
            </Space>
          </Card>
        </Space>
      </SideSheet>
    </>
  );
}

export default function TextSegmentRuleEditor({
  config,
  rules,
  validationErrors = [],
  onChange,
  onConfigChange,
}) {
  const { t } = useTranslation();
  const ruleMeta = useMemo(
    () => getTextSegmentRuleEditorMeta(config, rules),
    [config, rules],
  );
  const serializedConfig = useMemo(
    () =>
      serializeAdvancedPricingConfig({
        ...(config || {}),
        rules,
      }),
    [config, rules],
  );
  const priorityHint = t(
    '按优先级从小到大依次匹配，命中第一条启用规则后停止；停用规则不会参与命中预览和最终保存。',
  );

  const handleConfigFieldChange = (field, value) => {
    onConfigChange({
      ...(config || {}),
      rules,
      [field]: value,
    });
  };

  return (
    <Space vertical align='start' style={{ width: '100%' }}>
      <Card title={t('文本分段规则编辑器')} style={{ width: '100%' }}>
        <Card
          bodyStyle={{ padding: 16 }}
          style={{
            marginBottom: 16,
            background: 'var(--semi-color-fill-0)',
          }}
          title={t('文本分段规则摘要')}
        >
          <div
            style={{
              width: '100%',
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
              gap: 12,
            }}
          >
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('规则类型')}</div>
              <Tag color='blue'>{t('文本分段')}</Tag>
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('规则总数')}</div>
              <Space wrap>
                <Tag color='cyan'>{`${ruleMeta.totalRules} ${t('条规则')}`}</Tag>
                <Tag color='green'>{`${ruleMeta.enabledRules} ${t('条启用')}`}</Tag>
              </Space>
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>
                {t('默认兜底价格')}
              </div>
              <Space wrap>
                <Tag color={ruleMeta.hasDefaultPrice ? 'green' : 'orange'}>
                  {ruleMeta.hasDefaultPrice ? t('已设置') : t('未设置')}
                </Tag>
                <Text>
                  {ruleMeta.hasDefaultPrice
                    ? ruleMeta.defaultPrice
                    : t('未命中规则时不回退默认价格')}
                </Text>
              </Space>
            </div>
            <div style={{ gridColumn: '1 / -1' }}>
              <div className='mb-1 font-medium text-gray-700'>{t('命中顺序')}</div>
              <Text>{priorityHint}</Text>
            </div>
          </div>
        </Card>

        <div
          style={{
            width: '100%',
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
            gap: 12,
            marginBottom: 16,
          }}
        >
          <div>
            <div className='mb-1 font-medium text-gray-700'>{t('显示名称')}</div>
            <Input
              value={config?.displayName || ''}
              placeholder={t('例如：Gemini 长上下文分层')}
              onChange={(value) => handleConfigFieldChange('displayName', value)}
            />
          </div>
          <div>
            <div className='mb-1 font-medium text-gray-700'>{t('分段依据')}</div>
            <Input
              value={config?.segmentBasis || ''}
              placeholder={t('例如：input_tokens')}
              onChange={(value) => handleConfigFieldChange('segmentBasis', value)}
            />
          </div>
          <div>
            <div className='mb-1 font-medium text-gray-700'>{t('计费单位')}</div>
            <Input
              value={config?.billingUnit || ''}
              placeholder={t('例如：1M tokens')}
              onChange={(value) => handleConfigFieldChange('billingUnit', value)}
            />
          </div>
          <div>
            <div className='mb-1 font-medium text-gray-700'>{t('默认单价')}</div>
            <Input
              value={config?.defaultPrice || ''}
              placeholder={t('未命中规则时可选')}
              onChange={(value) => handleConfigFieldChange('defaultPrice', value)}
            />
          </div>
          <div style={{ gridColumn: '1 / -1' }}>
            <div className='mb-1 font-medium text-gray-700'>{t('备注')}</div>
            <TextArea
              value={config?.note || ''}
              rows={3}
              placeholder={t('补充当前文本分段规则的适用说明')}
              onChange={(value) => handleConfigFieldChange('note', value)}
            />
          </div>
        </div>

        <TextSegmentRulesEditor
          rules={rules}
          validationErrors={validationErrors}
          onChange={onChange}
        />

        <Card
          bodyStyle={{ padding: 16 }}
          style={{
            marginTop: 16,
            background: 'var(--semi-color-fill-0)',
          }}
          title={t('保存后配置 JSON')}
        >
          <pre
            style={{
              margin: 0,
              padding: 12,
              borderRadius: 8,
              background: 'var(--semi-color-fill-1)',
              width: '100%',
              overflowX: 'auto',
            }}
          >
            {JSON.stringify(serializedConfig, null, 2)}
          </pre>
        </Card>
      </Card>
    </Space>
  );
}
