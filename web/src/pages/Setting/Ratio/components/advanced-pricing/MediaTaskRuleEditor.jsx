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
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconPlus } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import CollapsibleJsonBlock from './CollapsibleJsonBlock';
import {
  MEDIA_TASK_RULE_TYPE,
  buildMediaTaskConditionSummary,
  buildMediaTaskPreview,
  createEmptyMediaTaskRule,
  normalizeMediaTaskRule,
  serializeAdvancedPricingConfig,
  serializeMediaTaskRule,
  sortMediaTaskRules,
  validateMediaTaskConfig,
} from '../../hooks/advancedPricingRuleHelpers';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';

const { Text } = Typography;
const INTEGER_INPUT_REGEX = /^\d*$/;
const DECIMAL_INPUT_REGEX = /^(\d+(\.\d*)?|\.\d*)?$/;
const EMPTY_MEDIA_TASK_PREVIEW_INPUT = {
  rawAction: '',
  inferenceMode: '',
  usageTotalTokens: '',
  inputVideo: '',
  audio: '',
  draft: '',
  resolution: '',
  aspectRatio: '',
  outputDuration: '',
  inputVideoDuration: '',
};

const hasMatchingPriorityError = (error, priorityValue) => {
  const priority = String(priorityValue || '').trim();
  if (!priority) {
    return false;
  }

  return (
    error.startsWith(`priority ${priority} duplicated:`) ||
    error.startsWith(`优先级 ${priority} 重复:`)
  );
};

const updateConfigRuleSetField = (config, field, value) => ({
  ...config,
  ruleType: MEDIA_TASK_RULE_TYPE,
  [field]: value,
});

export default function MediaTaskRuleEditor({
  config,
  validationErrors = [],
  onChange,
}) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const tristateOptions = useMemo(
    () => [
      { value: '', label: t('不限') },
      { value: 'true', label: t('是') },
      { value: 'false', label: t('否') },
    ],
    [t],
  );
  const draftOptions = useMemo(
    () => [
      { value: '', label: t('不限') },
      { value: 'true', label: t('草稿') },
      { value: 'false', label: t('非草稿') },
    ],
    [t],
  );
  const numericFields = useMemo(
    () => [
      {
        field: 'priority',
        label: t('优先级'),
        placeholder: t('越小越优先'),
        regex: INTEGER_INPUT_REGEX,
      },
      {
        field: 'rawAction',
        label: t('任务动作'),
        placeholder: t('如 generate / firstTailGenerate'),
      },
      {
        field: 'inferenceMode',
        label: t('推理模式'),
        placeholder: t('如 fast / quality'),
      },
      {
        field: 'inputModality',
        label: t('输入模态'),
        placeholder: t('例如 image / audio'),
      },
      {
        field: 'outputModality',
        label: t('输出模态'),
        placeholder: t('例如 image / video'),
      },
      {
        field: 'billingUnit',
        label: t('计费单位'),
        placeholder: t('例如 per_image'),
      },
      {
        field: 'resolution',
        label: t('分辨率'),
        placeholder: t('如 720p / 1080p'),
      },
      {
        field: 'aspectRatio',
        label: t('宽高比'),
        placeholder: t('如 16:9 / 9:16'),
      },
      {
        field: 'outputDurationMin',
        label: t('输出时长最小值'),
        placeholder: t('秒'),
        regex: INTEGER_INPUT_REGEX,
      },
      {
        field: 'outputDurationMax',
        label: t('输出时长最大值'),
        placeholder: t('秒'),
        regex: INTEGER_INPUT_REGEX,
      },
      {
        field: 'inputVideoDurationMin',
        label: t('输入视频时长最小值'),
        placeholder: t('秒'),
        regex: INTEGER_INPUT_REGEX,
      },
      {
        field: 'inputVideoDurationMax',
        label: t('输入视频时长最大值'),
        placeholder: t('秒'),
        regex: INTEGER_INPUT_REGEX,
      },
      {
        field: 'unitPrice',
        label: t('单价'),
        placeholder: t('必填'),
        regex: DECIMAL_INPUT_REGEX,
      },
      {
        field: 'minTokens',
        label: t('最低结算 Token'),
        placeholder: t('可选'),
        regex: INTEGER_INPUT_REGEX,
      },
      {
        field: 'draftCoefficient',
        label: t('草稿系数'),
        placeholder: t('如 0.5'),
        regex: DECIMAL_INPUT_REGEX,
      },
      {
        field: 'imageSizeTier',
        label: t('图像档位'),
        placeholder: t('例如 1k / 2k / 4k'),
      },
      {
        field: 'toolUsageType',
        label: t('工具调用类型'),
        placeholder: t('例如 google_search'),
      },
      {
        field: 'toolUsageCount',
        label: t('工具调用次数'),
        placeholder: t('可选'),
        regex: INTEGER_INPUT_REGEX,
      },
      {
        field: 'toolOveragePrice',
        label: t('超额单价'),
        placeholder: t('选填'),
        regex: DECIMAL_INPUT_REGEX,
      },
      {
        field: 'freeQuota',
        label: t('免费额度'),
        placeholder: t('可选'),
        regex: INTEGER_INPUT_REGEX,
      },
      {
        field: 'overageThreshold',
        label: t('超额阈值'),
        placeholder: t('可选'),
        regex: INTEGER_INPUT_REGEX,
      },
    ],
    [t],
  );
  const [sideSheetVisible, setSideSheetVisible] = useState(false);
  const [editingRuleId, setEditingRuleId] = useState('');
  const [draftRule, setDraftRule] = useState(createEmptyMediaTaskRule(1));
  const [draftErrors, setDraftErrors] = useState([]);
  const [sheetPreviewInput, setSheetPreviewInput] = useState(
    EMPTY_MEDIA_TASK_PREVIEW_INPUT,
  );

  const rules = Array.isArray(config?.rules) ? config.rules : [];
  const sortedRules = useMemo(() => sortMediaTaskRules(rules), [rules]);
  const previewInput = sheetPreviewInput;
  const ruleMeta = useMemo(
    () => ({
      ruleType: MEDIA_TASK_RULE_TYPE,
      totalRules: sortedRules.length,
      enabledRules: sortedRules.filter((rule) => rule?.enabled !== false)
        .length,
    }),
    [sortedRules],
  );
  const serializedConfig = useMemo(
    () => serializeAdvancedPricingConfig(config),
    [config],
  );

  const resetDraftState = () => {
    setEditingRuleId('');
    setDraftRule(createEmptyMediaTaskRule(Date.now()));
    setDraftErrors([]);
    setSheetPreviewInput({ ...EMPTY_MEDIA_TASK_PREVIEW_INPUT });
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
    setDraftRule(normalizeMediaTaskRule(rule));
    setDraftErrors([]);
    setSideSheetVisible(true);
  };

  const buildNextRules = (candidateRule) =>
    editingRuleId
      ? rules.map((rule) => (rule.id === editingRuleId ? candidateRule : rule))
      : [...rules, candidateRule];

  const candidatePreviewRules = useMemo(
    () => sortMediaTaskRules(buildNextRules(normalizeMediaTaskRule(draftRule))),
    [draftRule, editingRuleId, rules],
  );
  const candidateEnabledRuleCount = useMemo(
    () =>
      candidatePreviewRules.filter((rule) => rule?.enabled !== false).length,
    [candidatePreviewRules],
  );
  const sheetPreviewResult = useMemo(
    () => buildMediaTaskPreview(candidatePreviewRules, previewInput),
    [candidatePreviewRules, previewInput],
  );
  const previewResult = sheetPreviewResult;

  const resolveDraftErrors = (candidateRule) => {
    const nextConfig = {
      ...config,
      ruleType: MEDIA_TASK_RULE_TYPE,
      rules: buildNextRules(candidateRule),
    };

    return validateMediaTaskConfig(nextConfig).filter(
      (error) =>
        error.includes(candidateRule.id) ||
        hasMatchingPriorityError(error, candidateRule.priority),
    );
  };

  const handleRuleSetFieldChange = (field, value) => {
    onChange(updateConfigRuleSetField(config, field, value));
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

  const handlePreviewInputChange = (field, value, regex) => {
    if (regex && !regex.test(value)) {
      return;
    }

    setSheetPreviewInput((currentValue) => ({
      ...currentValue,
      [field]: value,
    }));
  };

  const handleSaveDraft = () => {
    const candidateRule = normalizeMediaTaskRule(draftRule);
    const nextDraftErrors = resolveDraftErrors(candidateRule);

    if (nextDraftErrors.length > 0) {
      setDraftErrors(nextDraftErrors);
      return;
    }

    onChange({
      ...config,
      ruleType: MEDIA_TASK_RULE_TYPE,
      rules: sortMediaTaskRules(buildNextRules(candidateRule)),
    });
    handleCloseSideSheet();
  };

  const handleDeleteRule = (ruleId) => {
    onChange({
      ...config,
      ruleType: MEDIA_TASK_RULE_TYPE,
      rules: rules.filter((rule) => rule.id !== ruleId),
    });
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
        title: t('条件摘要'),
        key: 'conditionSummary',
        render: (_, record) => (
          <Tag color='cyan'>{buildMediaTaskConditionSummary(record)}</Tag>
        ),
      },
      {
        title: t('计费摘要'),
        key: 'billingSummary',
        render: (_, record) => (
          <Space wrap>
            <Tag color='blue'>{`${t('单价')}：${record.unitPrice || '-'}`}</Tag>
            <Tag color='green'>
              {`${t('最低结算 Token')}：${record.minTokens || '-'}`}
            </Tag>
            <Tag color='violet'>
              {`${t('草稿系数')}：${record.draftCoefficient || '-'}`}
            </Tag>
          </Space>
        ),
      },
      {
        title: t('备注'),
        dataIndex: 'remark',
        key: 'remark',
        render: (value) => value || '-',
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
    [config, rules, t],
  );

  return (
    <>
      <Card
        title={t('媒体任务规则编辑器')}
        headerExtraContent={
          <Button icon={<IconPlus />} onClick={openCreateSideSheet}>
            {t('新增规则')}
          </Button>
        }
      >
        <div className='text-sm text-gray-500 mb-3'>
          {t(
            '在不改变页面布局的前提下，这里维护媒体任务规则集与规则条件矩阵；保存后会生成后端可接受的标准规则 JSON。',
          )}
        </div>

        <Card
          bodyStyle={{ padding: 16 }}
          style={{
            marginBottom: 16,
            background: 'var(--semi-color-fill-0)',
          }}
          title={t('规则摘要')}
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
              <Tag color='blue'>{ruleMeta.ruleType}</Tag>
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('规则数量')}</div>
              <Space wrap>
                <Tag color='cyan'>{`${t('规则总数')}：${ruleMeta.totalRules}`}</Tag>
                <Tag color='green'>{`${t('启用规则')}：${ruleMeta.enabledRules}`}</Tag>
              </Space>
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>
                {t('显示名称')}
              </div>
              <Text>{config?.displayName || '-'}</Text>
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>{t('任务类型')}</div>
              <Text>{config?.taskType || '-'}</Text>
            </div>
            <div>
              <div className='mb-1 font-medium text-gray-700'>
                {t('计费单位')}
              </div>
              <Text>{config?.billingUnit || '-'}</Text>
            </div>
            <div style={{ gridColumn: '1 / -1' }}>
              <div className='mb-1 font-medium text-gray-700'>{t('备注')}</div>
              <Text>{config?.note || '-'}</Text>
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
              placeholder={t('如 即梦视频任务')}
              onChange={(value) => handleRuleSetFieldChange('displayName', value)}
            />
          </div>
          <div>
            <div className='mb-1 font-medium text-gray-700'>{t('任务类型')}</div>
            <Input
              value={config?.taskType || ''}
              placeholder={t('如 video_generation')}
              onChange={(value) => handleRuleSetFieldChange('taskType', value)}
            />
          </div>
          <div>
            <div className='mb-1 font-medium text-gray-700'>
              {t('计费单位')}
            </div>
            <Input
              value={config?.billingUnit || ''}
              placeholder={t('如 total_tokens')}
              onChange={(value) => handleRuleSetFieldChange('billingUnit', value)}
            />
          </div>
          <div style={{ gridColumn: '1 / -1' }}>
            <div className='mb-1 font-medium text-gray-700'>{t('备注')}</div>
            <TextArea
              value={config?.note || ''}
              placeholder={t('填写给运营看的规则说明')}
              onChange={(value) => handleRuleSetFieldChange('note', value)}
              autosize={{ minRows: 2, maxRows: 4 }}
            />
          </div>
        </div>

        {validationErrors.length > 0 ? (
          <Banner
            type='warning'
            bordered
            fullMode={false}
            closeIcon={null}
            style={{ marginBottom: 16 }}
            title={t('当前媒体任务规则存在待处理问题')}
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
              {t('暂无媒体任务规则，点击右上角新增规则')}
            </div>
          }
        />

        <Card
          bodyStyle={{ padding: 16 }}
          style={{
            marginTop: 16,
            background: 'var(--semi-color-fill-0)',
          }}
        >
          <CollapsibleJsonBlock title={t('保存后规则 JSON')}>
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
              {JSON.stringify(serializedConfig.segments || [], null, 2)}
            </pre>
          </CollapsibleJsonBlock>
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
              {editingRuleId ? t('编辑媒体任务规则') : t('新增媒体任务规则')}
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
            {numericFields.map((fieldMeta) => (
              <div key={fieldMeta.field}>
                <div className='mb-1 font-medium text-gray-700'>{fieldMeta.label}</div>
                <Input
                  value={draftRule[fieldMeta.field]}
                  placeholder={fieldMeta.placeholder}
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
            <div className='mb-2 font-medium text-gray-700'>{t('音频条件')}</div>
            <RadioGroup
              type='button'
              value={draftRule.audio}
              onChange={(event) =>
                handleDraftFieldChange('audio', event.target.value)
              }
            >
              {tristateOptions.map((option) => (
                <Radio key={option.value || 'any'} value={option.value}>
                  {option.label}
                </Radio>
              ))}
            </RadioGroup>
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-2 font-medium text-gray-700'>
              {t('输入视频条件')}
            </div>
            <RadioGroup
              type='button'
              value={draftRule.inputVideo}
              onChange={(event) =>
                handleDraftFieldChange('inputVideo', event.target.value)
              }
            >
              {tristateOptions.map((option) => (
                <Radio key={option.value || 'any'} value={option.value}>
                  {option.label}
                </Radio>
              ))}
            </RadioGroup>
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-2 font-medium text-gray-700'>{t('草稿模式')}</div>
            <RadioGroup
              type='button'
              value={draftRule.draft}
              onChange={(event) =>
                handleDraftFieldChange('draft', event.target.value)
              }
            >
              {draftOptions.map((option) => (
                <Radio key={option.value || 'any'} value={option.value}>
                  {option.label}
                </Radio>
              ))}
            </RadioGroup>
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-1 font-medium text-gray-700'>{t('备注')}</div>
            <TextArea
              value={draftRule.remark}
              placeholder={t('如 首帧图生视频 / 草稿模式')}
              onChange={(value) => handleDraftFieldChange('remark', value)}
              autosize={{ minRows: 2, maxRows: 4 }}
            />
          </div>

          <div style={{ width: '100%' }}>
            <div className='mb-1 font-medium text-gray-700'>{t('条件摘要')}</div>
            <Tag color='cyan'>{buildMediaTaskConditionSummary(draftRule)}</Tag>
          </div>

          <div style={{ width: '100%' }}>
            <CollapsibleJsonBlock title={t('生成的规则 JSON')}>
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
                {JSON.stringify(serializeMediaTaskRule(draftRule), null, 2)}
              </pre>
            </CollapsibleJsonBlock>
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
            title={t('任务计费预览')}
          >
            <Space vertical align='start' style={{ width: '100%' }}>
              <div className='text-sm text-gray-500'>
                {t('预览当前草稿规则集的命中结果与计费摘要。')}
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
                    {t('任务动作')}
                  </div>
                  <Input
                    value={previewInput?.rawAction || ''}
                    placeholder={t('如 generate / firstTailGenerate')}
                    onChange={(value) =>
                      handlePreviewInputChange('rawAction', value)
                    }
                  />
                </div>
                <div>
                  <div className='mb-1 font-medium text-gray-700'>
                    {t('本次上报 Token')}
                  </div>
                  <Input
                    value={previewInput?.usageTotalTokens || ''}
                    placeholder={t('如 1200')}
                    onChange={(value) =>
                      handlePreviewInputChange(
                        'usageTotalTokens',
                        value,
                        INTEGER_INPUT_REGEX,
                      )
                    }
                  />
                </div>
                <div>
                  <div className='mb-1 font-medium text-gray-700'>
                    {t('分辨率')}
                  </div>
                  <Input
                    value={previewInput?.resolution || ''}
                    placeholder={t('如 1080p')}
                    onChange={(value) =>
                      handlePreviewInputChange('resolution', value)
                    }
                  />
                </div>
                <div>
                  <div className='mb-1 font-medium text-gray-700'>
                    {t('宽高比')}
                  </div>
                  <Input
                    value={previewInput?.aspectRatio || ''}
                    placeholder={t('如 16:9')}
                    onChange={(value) =>
                      handlePreviewInputChange('aspectRatio', value)
                    }
                  />
                </div>
                <div>
                  <div className='mb-1 font-medium text-gray-700'>
                    {t('输出时长')}
                  </div>
                  <Input
                    value={previewInput?.outputDuration || ''}
                    placeholder={t('如 5')}
                    onChange={(value) =>
                      handlePreviewInputChange(
                        'outputDuration',
                        value,
                        INTEGER_INPUT_REGEX,
                      )
                    }
                  />
                </div>
                <div>
                  <div className='mb-1 font-medium text-gray-700'>
                    {t('输入视频时长')}
                  </div>
                  <Input
                    value={previewInput?.inputVideoDuration || ''}
                    placeholder={t('如 12')}
                    onChange={(value) =>
                      handlePreviewInputChange(
                        'inputVideoDuration',
                        value,
                        INTEGER_INPUT_REGEX,
                      )
                    }
                  />
                </div>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('推理模式')}
                </div>
                <Input
                  value={previewInput?.inferenceMode || ''}
                  placeholder={t('如 fast / quality')}
                  onChange={(value) =>
                    handlePreviewInputChange('inferenceMode', value)
                  }
                />
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-2 font-medium text-gray-700'>{t('音频')}</div>
                <RadioGroup
                  type='button'
                  value={previewInput?.audio || ''}
                  onChange={(event) =>
                    handlePreviewInputChange('audio', event.target.value)
                  }
                >
                  {tristateOptions.map((option) => (
                    <Radio
                      key={option.value || 'preview-audio-any'}
                      value={option.value}
                    >
                      {option.label}
                    </Radio>
                  ))}
                </RadioGroup>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-2 font-medium text-gray-700'>
                  {t('输入视频')}
                </div>
                <RadioGroup
                  type='button'
                  value={previewInput?.inputVideo || ''}
                  onChange={(event) =>
                    handlePreviewInputChange('inputVideo', event.target.value)
                  }
                >
                  {tristateOptions.map((option) => (
                    <Radio
                      key={option.value || 'preview-video-any'}
                      value={option.value}
                    >
                      {option.label}
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
                    handlePreviewInputChange('draft', event.target.value)
                  }
                >
                  {draftOptions.map((option) => (
                    <Radio
                      key={option.value || 'preview-draft-any'}
                      value={option.value}
                    >
                      {option.label}
                    </Radio>
                  ))}
                </RadioGroup>
              </div>

              <Space wrap>
                <Tag color='blue'>
                  {`${t('规则总数')}：${candidatePreviewRules.length}`}
                </Tag>
                <Tag color='cyan'>
                  {`${t('启用规则')}：${candidateEnabledRuleCount}`}
                </Tag>
                <Tag color={sheetPreviewResult?.matchedRule ? 'green' : 'grey'}>
                  {sheetPreviewResult?.matchedRule
                    ? t('已命中规则')
                    : t('未命中规则')}
                </Tag>
                {previewResult?.matchedRule ? (
                  <Tag color='violet'>
                    {`${t('优先级')}：${previewResult.matchedRule.priority || '-'}`}
                  </Tag>
                ) : null}
              </Space>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('命中条件摘要')}
                </div>
                <Tag color='cyan'>{previewResult?.conditionSummary || '-'}</Tag>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('预计计费公式')}
                </div>
                <Tag color='blue'>{previewResult?.formulaSummary || '-'}</Tag>
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
                    {t('本次上报 Token')}
                  </div>
                  <div className='font-medium'>
                    {previewResult?.priceSummary?.usageTotalTokens || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('结算 Token')}
                  </div>
                  <div className='font-medium'>
                    {previewResult?.priceSummary?.billableTokens || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('最低结算 Token')}
                  </div>
                  <div className='font-medium'>
                    {previewResult?.priceSummary?.minTokens || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('单价')}
                  </div>
                  <div className='font-medium'>
                    {previewResult?.priceSummary?.unitPrice || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('草稿系数')}
                  </div>
                  <div className='font-medium'>
                    {previewResult?.priceSummary?.draftCoefficient || '-'}
                  </div>
                </div>
                <div>
                  <div className='text-xs text-gray-500 mb-1'>
                    {t('预估费用')}
                  </div>
                  <div className='font-medium'>
                    {sheetPreviewResult?.priceSummary?.estimatedCost || '-'}
                  </div>
                </div>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('日志详情')}
                </div>
                <div>{previewResult?.logPreview?.detailSummary || '-'}</div>
              </div>

              <div style={{ width: '100%' }}>
                <div className='mb-1 font-medium text-gray-700'>
                  {t('计费过程')}
                </div>
                <div>{previewResult?.logPreview?.processSummary || '-'}</div>
              </div>

              <div style={{ width: '100%' }}>
                <CollapsibleJsonBlock title={t('命中规则 JSON')}>
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
                      previewResult?.matchedRule
                        ? serializeMediaTaskRule(previewResult.matchedRule)
                        : null,
                      null,
                      2,
                    )}
                  </pre>
                </CollapsibleJsonBlock>
              </div>
            </Space>
          </Card>
        </Space>
      </SideSheet>
    </>
  );
}
