import React, { useCallback, useMemo, useState } from 'react';
import {
  Button,
  Collapsible,
  InputNumber,
  Popconfirm,
  Select,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconChevronDown,
  IconChevronUp,
  IconDelete,
  IconPlus,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  flattenGroupGroupRatioRules,
  parseJSONSafe,
  serializeGroupGroupRatioRules,
} from '../groupSettingsSerialization';

const { Text } = Typography;

let nextNewGroupGroupRatioCounter = 0;

function GroupSection({
  groupName,
  groupOptions,
  items,
  onAdd,
  onRemove,
  onUpdate,
  t,
}) {
  const [open, setOpen] = useState(false);

  return (
    <div
      style={{
        border: '1px solid var(--semi-color-border)',
        borderRadius: 8,
        overflow: 'hidden',
      }}
    >
      <div
        className='flex cursor-pointer items-center justify-between'
        style={{
          background: 'var(--semi-color-fill-0)',
          padding: '8px 12px',
        }}
        onClick={() => setOpen(!open)}
      >
        <div className='flex items-center gap-2'>
          {open ? <IconChevronUp size='small' /> : <IconChevronDown size='small' />}
          <Text strong>{groupName}</Text>
          <Tag size='small' color='blue'>
            {items.length} {t('条规则')}
          </Tag>
        </div>
        <div className='flex items-center gap-1' onClick={(event) => event.stopPropagation()}>
          <Button
            icon={<IconPlus />}
            size='small'
            theme='borderless'
            onClick={() => onAdd(groupName)}
          />
          <Popconfirm
            title={t('确认删除该分组的所有规则？')}
            onConfirm={() => items.forEach((item) => onRemove(item._id))}
            position='left'
          >
            <Button
              icon={<IconDelete />}
              size='small'
              type='danger'
              theme='borderless'
            />
          </Popconfirm>
        </div>
      </div>
      <Collapsible isOpen={open} keepDOM>
        <div style={{ padding: '8px 12px' }}>
          {items.map((item) => (
            <div
              key={item._id}
              className='flex items-center gap-2'
              style={{ marginBottom: 6 }}
            >
              <Select
                size='small'
                filter
                allowCreate
                value={item.usingGroup || undefined}
                placeholder={t('选择使用分组')}
                optionList={groupOptions}
                onChange={(value) => onUpdate(item._id, 'usingGroup', value)}
                style={{ flex: 1 }}
                position='bottomLeft'
              />
              <InputNumber
                size='small'
                min={0}
                step={0.1}
                value={item.ratio}
                style={{ width: 100 }}
                onChange={(value) => onUpdate(item._id, 'ratio', value ?? 0)}
              />
              <Popconfirm
                title={t('确认删除该规则？')}
                onConfirm={() => onRemove(item._id)}
                position='left'
              >
                <Button
                  icon={<IconDelete />}
                  type='danger'
                  theme='borderless'
                  size='small'
                />
              </Popconfirm>
            </div>
          ))}
        </div>
      </Collapsible>
    </div>
  );
}

export default function GroupGroupRatioRules({
  value,
  groupNames = [],
  onChange,
}) {
  const { t } = useTranslation();
  const [rules, setRules] = useState(() =>
    flattenGroupGroupRatioRules(parseJSONSafe(value, {})),
  );
  const [newGroupName, setNewGroupName] = useState('');

  const emitChange = useCallback(
    (nextRules) => {
      setRules(nextRules);
      onChange?.(serializeGroupGroupRatioRules(nextRules));
    },
    [onChange],
  );

  const updateRule = useCallback(
    (ruleId, field, fieldValue) => {
      emitChange(
        rules.map((rule) =>
          rule._id === ruleId
            ? {
                ...rule,
                [field]: fieldValue,
              }
            : rule,
        ),
      );
    },
    [emitChange, rules],
  );

  const removeRule = useCallback(
    (ruleId) => {
      emitChange(rules.filter((rule) => rule._id !== ruleId));
    },
    [emitChange, rules],
  );

  const addRuleToGroup = useCallback(
    (userGroup) => {
      nextNewGroupGroupRatioCounter += 1;
      emitChange([
        ...rules,
        {
          _id: `ggr_new_${Date.now()}_${nextNewGroupGroupRatioCounter}`,
          userGroup,
          usingGroup: '',
          ratio: 1,
        },
      ]);
    },
    [emitChange, rules],
  );

  const addNewGroup = useCallback(() => {
    if (!newGroupName.trim()) {
      return;
    }

    addRuleToGroup(newGroupName.trim());
    setNewGroupName('');
  }, [addRuleToGroup, newGroupName]);

  const groupOptions = useMemo(
    () => groupNames.map((groupName) => ({ value: groupName, label: groupName })),
    [groupNames],
  );

  const groupedRules = useMemo(() => {
    const groupedMap = {};
    const order = [];

    rules.forEach((rule) => {
      if (!rule.userGroup) {
        return;
      }

      if (!groupedMap[rule.userGroup]) {
        groupedMap[rule.userGroup] = [];
        order.push(rule.userGroup);
      }

      groupedMap[rule.userGroup].push(rule);
    });

    return order.map((groupName) => ({
      name: groupName,
      items: groupedMap[groupName],
    }));
  }, [rules]);

  if (groupedRules.length === 0 && rules.length === 0) {
    return (
      <div>
        <Text type='tertiary' className='block py-4 text-center'>
          {t('暂无规则，点击下方按钮添加')}
        </Text>
        <div className='mt-2 flex justify-center gap-2'>
          <Select
            size='small'
            filter
            allowCreate
            placeholder={t('选择用户分组')}
            optionList={groupOptions}
            value={newGroupName || undefined}
            onChange={setNewGroupName}
            style={{ width: 200 }}
            position='bottomLeft'
          />
          <Button icon={<IconPlus />} theme='outline' onClick={addNewGroup}>
            {t('添加分组规则')}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className='space-y-2'>
      {groupedRules.map((group) => (
        <GroupSection
          key={group.name}
          groupName={group.name}
          groupOptions={groupOptions}
          items={group.items}
          onAdd={addRuleToGroup}
          onRemove={removeRule}
          onUpdate={updateRule}
          t={t}
        />
      ))}
      <div className='mt-3 flex justify-center gap-2'>
        <Select
          size='small'
          filter
          allowCreate
          placeholder={t('选择用户分组')}
          optionList={groupOptions}
          value={newGroupName || undefined}
          onChange={setNewGroupName}
          style={{ width: 200 }}
          position='bottomLeft'
        />
        <Button icon={<IconPlus />} theme='outline' onClick={addNewGroup}>
          {t('添加分组规则')}
        </Button>
      </div>
    </div>
  );
}
