import React, { useCallback, useMemo, useState } from 'react';
import {
  Button,
  Collapsible,
  Input,
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
  flattenGroupSpecialUsableRules,
  OP_ADD,
  OP_APPEND,
  OP_REMOVE,
  parseJSONSafe,
  serializeGroupSpecialUsableRules,
} from '../groupSettingsSerialization';

const { Text } = Typography;

let nextNewSpecialUsableCounter = 0;

const OP_TAG_MAP = {
  [OP_ADD]: {
    color: 'green',
    label: '添加 (+:)',
  },
  [OP_REMOVE]: {
    color: 'red',
    label: '移除 (-:)',
  },
  [OP_APPEND]: {
    color: 'blue',
    label: '追加',
  },
};

function UsableGroupSection({
  groupName,
  items,
  opOptions,
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
                value={item.op}
                optionList={opOptions}
                onChange={(value) => onUpdate(item._id, 'op', value)}
                style={{ width: 120 }}
                renderSelectedItem={(optionNode) => {
                  const current = OP_TAG_MAP[optionNode.value] || {};
                  return (
                    <Tag size='small' color={current.color}>
                      {optionNode.label}
                    </Tag>
                  );
                }}
              />
              <Input
                size='small'
                value={item.targetGroup}
                placeholder={t('分组名称')}
                onChange={(value) => onUpdate(item._id, 'targetGroup', value)}
                style={{ flex: 1 }}
              />
              <Input
                size='small'
                value={item.description}
                placeholder={t('分组描述')}
                onChange={(value) => onUpdate(item._id, 'description', value)}
                style={{ flex: 1 }}
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

export default function GroupSpecialUsableRules({
  value,
  groupNames = [],
  onChange,
}) {
  const { t } = useTranslation();
  const [rules, setRules] = useState(() =>
    flattenGroupSpecialUsableRules(parseJSONSafe(value, {})),
  );
  const [newGroupName, setNewGroupName] = useState('');

  const emitChange = useCallback(
    (nextRules) => {
      setRules(nextRules);
      onChange?.(serializeGroupSpecialUsableRules(nextRules));
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
      nextNewSpecialUsableCounter += 1;
      emitChange([
        ...rules,
        {
          _id: `gsu_new_${Date.now()}_${nextNewSpecialUsableCounter}`,
          userGroup,
          op: OP_APPEND,
          targetGroup: '',
          description: '',
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

  const opOptions = useMemo(
    () => [
      {
        value: OP_ADD,
        label: t('添加 (+:)'),
      },
      {
        value: OP_REMOVE,
        label: t('移除 (-:)'),
      },
      {
        value: OP_APPEND,
        label: t('追加'),
      },
    ],
    [t],
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
        <UsableGroupSection
          key={group.name}
          groupName={group.name}
          items={group.items}
          opOptions={opOptions}
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
