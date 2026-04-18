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
  Modal,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconPlus } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  buildTextSegmentConditionSummary,
  createEmptyTextSegmentRule,
  normalizeTextSegmentRule,
  sortTextSegmentRules,
  validateTextSegmentRules,
} from '../../hooks/advancedPricingRuleHelpers';

const { Text } = Typography;

const INTEGER_INPUT_REGEX = /^\d*$/;
const DECIMAL_INPUT_REGEX = /^(\d+(\.\d*)?|\.\d*)?$/;

const FORM_FIELDS = [
  {
    field: 'priority',
    label: '优先级',
    placeholder: '越小越优先',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'inputMin',
    label: '输入最小值',
    placeholder: '可留空',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'inputMax',
    label: '输入最大值',
    placeholder: '可留空',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'outputMin',
    label: '输出最小值',
    placeholder: '可留空',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'outputMax',
    label: '输出最大值',
    placeholder: '可留空',
    regex: INTEGER_INPUT_REGEX,
  },
  {
    field: 'inputPrice',
    label: '输入价格',
    placeholder: '$/1M tokens',
    regex: DECIMAL_INPUT_REGEX,
  },
  {
    field: 'outputPrice',
    label: '输出价格',
    placeholder: '$/1M tokens',
    regex: DECIMAL_INPUT_REGEX,
  },
  {
    field: 'cacheReadPrice',
    label: '缓存读取价格',
    placeholder: '$/1M tokens',
    regex: DECIMAL_INPUT_REGEX,
  },
  {
    field: 'cacheWritePrice',
    label: '缓存创建价格',
    placeholder: '$/1M tokens',
    regex: DECIMAL_INPUT_REGEX,
  },
];

export default function TextSegmentRuleEditor({
  rules,
  validationErrors,
  onChange,
}) {
  const { t } = useTranslation();
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRuleId, setEditingRuleId] = useState('');
  const [draftRule, setDraftRule] = useState(createEmptyTextSegmentRule(1));
  const [draftErrors, setDraftErrors] = useState([]);

  const sortedRules = useMemo(() => sortTextSegmentRules(rules), [rules]);

  const resetDraftState = () => {
    setEditingRuleId('');
    setDraftRule(createEmptyTextSegmentRule(Date.now()));
    setDraftErrors([]);
  };

  const openCreateModal = () => {
    setEditingRuleId('');
    setDraftRule(createEmptyTextSegmentRule(Date.now()));
    setDraftErrors([]);
    setModalVisible(true);
  };

  const openEditModal = (rule) => {
    setEditingRuleId(rule.id);
    setDraftRule(normalizeTextSegmentRule(rule));
    setDraftErrors([]);
    setModalVisible(true);
  };

  const buildNextRules = (candidateRule) =>
    editingRuleId
      ? rules.map((rule) => (rule.id === editingRuleId ? candidateRule : rule))
      : [...rules, candidateRule];

  const resolveDraftErrors = (candidateRule) => {
    const nextRules = buildNextRules(candidateRule);
    return validateTextSegmentRules(nextRules).filter(
      (error) =>
        error.includes(candidateRule.id) ||
        (editingRuleId && error.includes(editingRuleId)),
    );
  };

  const handleDraftFieldChange = (field, value, regex) => {
    if (!regex.test(value)) {
      return;
    }

    const nextDraftRule = {
      ...draftRule,
      [field]: value,
    };

    setDraftRule(nextDraftRule);
    setDraftErrors(resolveDraftErrors(nextDraftRule));
  };

  const handleDraftEnabledChange = (checked) => {
    const nextDraftRule = {
      ...draftRule,
      enabled: checked,
    };
    setDraftRule(nextDraftRule);
    setDraftErrors(resolveDraftErrors(nextDraftRule));
  };

  const handleSaveDraft = () => {
    const candidateRule = normalizeTextSegmentRule(draftRule);
    const nextDraftErrors = resolveDraftErrors(candidateRule);
    if (nextDraftErrors.length > 0) {
      setDraftErrors(nextDraftErrors);
      return;
    }

    onChange(sortTextSegmentRules(buildNextRules(candidateRule)));
    setModalVisible(false);
    resetDraftState();
  };

  const handleDeleteRule = (ruleId) => {
    onChange(rules.filter((rule) => rule.id !== ruleId));
  };

  const handleInlineEnabledChange = (ruleId, checked) => {
    onChange(
      rules.map((rule) =>
        rule.id === ruleId ? { ...rule, enabled: checked } : rule,
      ),
    );
  };

  const columns = useMemo(
    () => [
      {
        title: t('启用'),
        dataIndex: 'enabled',
        key: 'enabled',
        render: (_, record) => (
          <Switch
            checked={record.enabled !== false}
            onChange={(checked) => handleInlineEnabledChange(record.id, checked)}
          />
        ),
      },
      {
        title: t('优先级'),
        dataIndex: 'priority',
        key: 'priority',
      },
      {
        title: t('条件摘要'),
        dataIndex: 'conditionSummary',
        key: 'conditionSummary',
        render: (_, record) => buildTextSegmentConditionSummary(record),
      },
      {
        title: t('价格摘要'),
        key: 'priceSummary',
        render: (_, record) => (
          <Space wrap>
            <Tag color='blue'>{t('输入')}: ${record.inputPrice || '-'}</Tag>
            <Tag color='green'>{t('输出')}: ${record.outputPrice || '-'}</Tag>
            <Tag color='cyan'>
              {t('缓存读')}: ${record.cacheReadPrice || '-'}
            </Tag>
            <Tag color='indigo'>
              {t('缓存写')}: ${record.cacheWritePrice || '-'}
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
              onClick={() => openEditModal(record)}
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
    [t, rules],
  );

  return (
    <>
      <Card
        title={t('文本分段规则编辑器')}
        headerExtraContent={
          <Button icon={<IconPlus />} onClick={openCreateModal}>
            {t('新增规则')}
          </Button>
        }
      >
        <div className='text-sm text-gray-500 mb-3'>
          {t('按优先级从小到大命中；建议保留一个兜底规则避免请求落空。')}
        </div>

        {validationErrors.length > 0 ? (
          <Banner
            type='warning'
            bordered
            fullMode={false}
            closeIcon={null}
            style={{ marginBottom: 16 }}
            title={t('当前规则存在待处理问题')}
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
      </Card>

      <Modal
        title={editingRuleId ? t('编辑文本规则') : t('新增文本规则')}
        visible={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          resetDraftState();
        }}
        onOk={handleSaveDraft}
        size='large'
      >
        <Space vertical align='start' style={{ width: '100%' }}>
          <div>
            <div className='mb-2 font-medium text-gray-700'>{t('启用状态')}</div>
            <Switch
              checked={draftRule.enabled !== false}
              onChange={handleDraftEnabledChange}
            />
          </div>

          <div
            style={{
              width: '100%',
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
              gap: 12,
            }}
          >
            {FORM_FIELDS.map((fieldMeta) => (
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

          <div>
            <div className='mb-1 font-medium text-gray-700'>{t('条件摘要')}</div>
            <Tag color='cyan'>{buildTextSegmentConditionSummary(draftRule)}</Tag>
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
        </Space>
      </Modal>
    </>
  );
}
