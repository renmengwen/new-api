import React, { useCallback, useMemo, useRef, useState } from 'react';
import {
  Button,
  Checkbox,
  Input,
  InputNumber,
  Popconfirm,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconPlus } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import CardTable from '../../../../components/common/ui/CardTable';
import {
  buildGroupTableRows,
  serializeGroupTableRows,
} from '../groupSettingsSerialization';

const { Text } = Typography;

let nextNewGroupCounter = 1;

export default function GroupTable({
  groupRatio,
  userUsableGroups,
  onChange,
}) {
  const { t } = useTranslation();
  const [rows, setRows] = useState(() =>
    buildGroupTableRows(groupRatio, userUsableGroups),
  );
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const emitChange = useCallback((updater) => {
    setRows((previousRows) => {
      const nextRows =
        typeof updater === 'function' ? updater(previousRows) : updater;
      onChangeRef.current?.(serializeGroupTableRows(nextRows));
      return nextRows;
    });
  }, []);

  const updateRow = useCallback(
    (rowId, field, value) => {
      emitChange((previousRows) =>
        previousRows.map((row) => {
          if (row._id !== rowId) {
            return row;
          }

          const nextRow = {
            ...row,
            [field]: value,
          };

          if (field === 'ratio') {
            nextRow.ratioMissing = false;
          }

          return nextRow;
        }),
      );
    },
    [emitChange],
  );

  const addRow = useCallback(() => {
    emitChange((previousRows) => {
      const existingNames = new Set(previousRows.map((row) => row.name));
      let nextName = `group_${nextNewGroupCounter}`;
      while (existingNames.has(nextName)) {
        nextNewGroupCounter += 1;
        nextName = `group_${nextNewGroupCounter}`;
      }
      nextNewGroupCounter += 1;

      return [
        ...previousRows,
        {
          _id: `gr_new_${Date.now()}_${nextNewGroupCounter}`,
          name: nextName,
          ratio: 1,
          selectable: true,
          description: '',
          ratioMissing: false,
        },
      ];
    });
  }, [emitChange]);

  const removeRow = useCallback(
    (rowId) => {
      emitChange((previousRows) =>
        previousRows.filter((row) => row._id !== rowId),
      );
    },
    [emitChange],
  );

  const duplicateNames = useMemo(() => {
    const counts = {};
    rows.forEach((row) => {
      if (!row.name) {
        return;
      }
      counts[row.name] = (counts[row.name] || 0) + 1;
    });
    return new Set(
      Object.keys(counts).filter((groupName) => counts[groupName] > 1),
    );
  }, [rows]);

  const duplicateNamesRef = useRef(duplicateNames);
  duplicateNamesRef.current = duplicateNames;

  const columns = useMemo(
    () => [
      {
        title: t('分组名称'),
        dataIndex: 'name',
        key: 'name',
        width: 180,
        render: (_, record) => (
          <Input
            size='small'
            value={record.name}
            status={
              duplicateNamesRef.current.has(record.name) ? 'warning' : undefined
            }
            onChange={(value) => updateRow(record._id, 'name', value)}
          />
        ),
      },
      {
        title: t('倍率'),
        dataIndex: 'ratio',
        key: 'ratio',
        width: 120,
        render: (_, record) => (
          <InputNumber
            size='small'
            min={0}
            step={0.1}
            value={record.ratio}
            style={{ width: '100%' }}
            onChange={(value) => updateRow(record._id, 'ratio', value ?? 0)}
          />
        ),
      },
      {
        title: t('用户可选'),
        dataIndex: 'selectable',
        key: 'selectable',
        width: 100,
        align: 'center',
        render: (_, record) => (
          <Checkbox
            checked={record.selectable}
            onChange={(event) =>
              updateRow(record._id, 'selectable', event.target.checked)
            }
          />
        ),
      },
      {
        title: t('描述'),
        dataIndex: 'description',
        key: 'description',
        render: (_, record) =>
          record.selectable ? (
            <Input
              size='small'
              value={record.description}
              placeholder={t('分组描述')}
              onChange={(value) => updateRow(record._id, 'description', value)}
            />
          ) : (
            <Text type='tertiary' size='small'>
              -
            </Text>
          ),
      },
      {
        title: '',
        key: 'actions',
        width: 50,
        render: (_, record) => (
          <Popconfirm
            title={t('确认删除该分组？')}
            onConfirm={() => removeRow(record._id)}
            position='left'
          >
            <Button
              icon={<IconDelete />}
              type='danger'
              theme='borderless'
              size='small'
            />
          </Popconfirm>
        ),
      },
    ],
    [removeRow, t, updateRow],
  );

  return (
    <div>
      <CardTable
        columns={columns}
        dataSource={rows}
        rowKey='_id'
        hidePagination
        size='small'
        empty={
          <Text type='tertiary'>
            {t('暂无分组，点击下方按钮添加')}
          </Text>
        }
      />
      <div className='mt-3 flex justify-center'>
        <Button icon={<IconPlus />} theme='outline' onClick={addRow}>
          {t('添加分组')}
        </Button>
      </div>
      {duplicateNames.size > 0 ? (
        <Text type='warning' size='small' className='mt-2 block'>
          {t('存在重复的分组名称：')}
          {Array.from(duplicateNames).join(', ')}
        </Text>
      ) : null}
    </div>
  );
}
