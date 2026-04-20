import React, { useCallback, useMemo, useState } from 'react';
import {
  Button,
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
  parseAutoGroups,
  serializeAutoGroups,
} from '../groupSettingsSerialization';

const { Text } = Typography;

let nextNewAutoGroupCounter = 0;

export default function AutoGroupList({ value, groupNames = [], onChange }) {
  const { t } = useTranslation();
  const [items, setItems] = useState(() => parseAutoGroups(value));

  const emitChange = useCallback(
    (nextItems) => {
      setItems(nextItems);
      onChange?.(serializeAutoGroups(nextItems));
    },
    [onChange],
  );

  const groupOptions = useMemo(
    () => groupNames.map((name) => ({ value: name, label: name })),
    [groupNames],
  );

  const addItem = useCallback(() => {
    nextNewAutoGroupCounter += 1;
    emitChange([
      ...items,
      {
        _id: `ag_new_${Date.now()}_${nextNewAutoGroupCounter}`,
        name: '',
      },
    ]);
  }, [emitChange, items]);

  const removeItem = useCallback(
    (itemId) => {
      emitChange(items.filter((item) => item._id !== itemId));
    },
    [emitChange, items],
  );

  const updateItem = useCallback(
    (itemId, name) => {
      emitChange(
        items.map((item) =>
          item._id === itemId
            ? {
                ...item,
                name,
              }
            : item,
        ),
      );
    },
    [emitChange, items],
  );

  const moveUp = useCallback(
    (index) => {
      if (index <= 0) {
        return;
      }

      const nextItems = [...items];
      [nextItems[index - 1], nextItems[index]] = [
        nextItems[index],
        nextItems[index - 1],
      ];
      emitChange(nextItems);
    },
    [emitChange, items],
  );

  const moveDown = useCallback(
    (index) => {
      if (index >= items.length - 1) {
        return;
      }

      const nextItems = [...items];
      [nextItems[index], nextItems[index + 1]] = [
        nextItems[index + 1],
        nextItems[index],
      ];
      emitChange(nextItems);
    },
    [emitChange, items],
  );

  if (items.length === 0) {
    return (
      <div>
        <Text type='tertiary' className='block py-4 text-center'>
          {t('暂无自动分组，点击下方按钮添加')}
        </Text>
        <div className='mt-2 flex justify-center'>
          <Button icon={<IconPlus />} theme='outline' onClick={addItem}>
            {t('添加分组')}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className='space-y-2'>
        {items.map((item, index) => (
          <div key={item._id} className='flex items-center gap-2'>
            <Tag size='small' color='blue' className='shrink-0'>
              {index + 1}
            </Tag>
            <Select
              size='small'
              filter
              allowCreate
              value={item.name || undefined}
              placeholder={t('选择分组')}
              optionList={groupOptions}
              onChange={(groupName) => updateItem(item._id, groupName)}
              style={{ flex: 1 }}
              position='bottomLeft'
            />
            <Button
              icon={<IconChevronUp />}
              theme='borderless'
              size='small'
              disabled={index === 0}
              onClick={() => moveUp(index)}
            />
            <Button
              icon={<IconChevronDown />}
              theme='borderless'
              size='small'
              disabled={index === items.length - 1}
              onClick={() => moveDown(index)}
            />
            <Popconfirm
              title={t('确认移除？')}
              onConfirm={() => removeItem(item._id)}
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
      <div className='mt-3 flex justify-center'>
        <Button icon={<IconPlus />} theme='outline' onClick={addItem}>
          {t('添加分组')}
        </Button>
      </div>
    </div>
  );
}
