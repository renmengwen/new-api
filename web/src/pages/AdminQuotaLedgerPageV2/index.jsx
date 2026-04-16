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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Banner, Button, Empty, Input, Modal, Select, Table, Tag, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import CardPro from '../../components/common/ui/CardPro';
import {
  API,
  createCardProPagination,
  MAX_EXCEL_EXPORT_ROWS,
  downloadExcelBlob,
  renderQuota,
  showError,
  showInfo,
  timestamp2string,
} from '../../helpers';
import {
  QUOTA_LEDGER_ENTRY_TYPE_OPTIONS,
  getQuotaAccountName,
  getQuotaEntryTypeLabel,
  getQuotaOperatorName,
  getQuotaReasonLabel,
} from '../../helpers/quotaLedgerDisplay';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import {
  changeCommittedPage,
  changeCommittedPageSize,
  commitQuotaLedgerFilters,
  createQuotaLedgerQueryState,
  createRequestSequenceTracker,
  getRefreshRequestState,
  resetDraftAndCommittedFilters,
  updateDraftFilters,
} from './requestState';

const { Text } = Typography;
const ADMIN_QUOTA_LEDGER_DIGITS = 6;

const parseOptionalInteger = (value) => {
  const normalizedValue = String(value ?? '').trim();
  if (!/^[+-]?\d+$/.test(normalizedValue)) {
    return 0;
  }

  return Number.parseInt(normalizedValue, 10);
};

const AdminQuotaLedgerPageV2 = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const { loading: permissionLoading, hasActionPermission } = useUserPermissions();
  const canRead = hasActionPermission('quota_management', 'ledger_read');

  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState([]);
  const [listError, setListError] = useState('');
  const [queryState, setQueryState] = useState(() => createQuotaLedgerQueryState());
  const [total, setTotal] = useState(0);
  const requestTrackerRef = useRef(null);

  if (!requestTrackerRef.current) {
    requestTrackerRef.current = createRequestSequenceTracker();
  }

  const { draftFilters, committedRequest } = queryState;
  const { userId, entryType } = draftFilters;
  const { page, pageSize } = committedRequest;

  const loadLedger = async (nextQueryState = queryState) => {
    if (!canRead) {
      return;
    }

    const requestState = getRefreshRequestState(nextQueryState);
    const requestId = requestTrackerRef.current.issue();

    setLoading(true);
    setListError('');
    try {
      const params = new URLSearchParams({
        p: String(requestState.page),
        page_size: String(requestState.pageSize),
      });
      if (requestState.userId.trim()) {
        params.set('user_id', requestState.userId.trim());
      }
      if (requestState.entryType) {
        params.set('entry_type', requestState.entryType);
      }

      const res = await API.get(`/api/admin/quota/ledger?${params.toString()}`);
      if (!requestTrackerRef.current.shouldAccept(requestId)) {
        return;
      }
      if (!res.data.success) {
        setItems([]);
        setTotal(0);
        setListError(res.data.message || t('加载额度流水失败'));
        return;
      }

      const data = res.data.data || {};
      setItems((data.items || []).map((item) => ({ ...item, key: item.id })));
      setQueryState((currentState) => ({
        ...currentState,
        committedRequest: {
          ...requestState,
          page: data.page || requestState.page,
          pageSize: data.page_size || requestState.pageSize,
        },
      }));
      setTotal(data.total || 0);
    } catch (error) {
      if (!requestTrackerRef.current.shouldAccept(requestId)) {
        return;
      }
      setItems([]);
      setTotal(0);
      setListError(t('加载额度流水失败，请稍后重试'));
      showError(error);
    } finally {
      if (requestTrackerRef.current.shouldAccept(requestId)) {
        setLoading(false);
      }
    }
  };

  const resetFilters = async () => {
    const nextQueryState = resetDraftAndCommittedFilters(queryState);
    setQueryState(nextQueryState);
    await loadLedger(nextQueryState);
  };

  const runExport = async () => {
    try {
      await downloadExcelBlob({
        url: '/api/admin/quota/ledger/export',
        payload: {
          user_id: parseOptionalInteger(committedRequest.userId),
          entry_type: committedRequest.entryType,
          limit: MAX_EXCEL_EXPORT_ROWS,
        },
        fallbackFileName: 'quota-ledger.xlsx',
      });
    } catch (error) {
      showError(error);
    }
  };

  const exportLedger = async () => {
    if (loading) {
      return;
    }

    if (!total) {
      showInfo(t('无可导出数据'));
      return;
    }

    if (total > MAX_EXCEL_EXPORT_ROWS) {
      Modal.confirm({
        title: t('导出 Excel'),
        content: t('当前筛选结果超过 2000 条，将仅导出前 2000 条记录，是否继续？'),
        okText: t('继续导出'),
        cancelText: t('取消'),
        onOk: runExport,
      });
      return;
    }

    await runExport();
  };

  useEffect(() => {
    if (!permissionLoading && canRead) {
      loadLedger(createQuotaLedgerQueryState());
    }
  }, [permissionLoading, canRead]);

  const columns = useMemo(
    () => [
      {
        title: t('流水号'),
        dataIndex: 'biz_no',
        width: 180,
      },
      {
        title: t('账户 ID'),
        dataIndex: 'account_id',
        width: 96,
      },
      {
        title: t('被操作账号'),
        dataIndex: 'account_username',
        width: 140,
        render: (_, record) => getQuotaAccountName(record),
      },
      {
        title: t('类型'),
        dataIndex: 'entry_type',
        width: 110,
        render: (value) => <Tag color='blue'>{getQuotaEntryTypeLabel(value)}</Tag>,
      },
      {
        title: t('方向'),
        dataIndex: 'direction',
        width: 88,
        render: (value) => (
          <Tag color={value === 'in' ? 'green' : 'red'} shape='circle'>
            {value === 'in' ? t('入账') : t('出账')}
          </Tag>
        ),
      },
      {
        title: t('金额'),
        dataIndex: 'amount',
        width: 132,
        render: (value) => renderQuota(value, ADMIN_QUOTA_LEDGER_DIGITS),
      },
      {
        title: t('变动前'),
        dataIndex: 'balance_before',
        width: 132,
        render: (value) => renderQuota(value, ADMIN_QUOTA_LEDGER_DIGITS),
      },
      {
        title: t('变动后'),
        dataIndex: 'balance_after',
        width: 132,
        render: (value) => renderQuota(value, ADMIN_QUOTA_LEDGER_DIGITS),
      },
      {
        title: t('模型名称'),
        dataIndex: 'model_name',
        width: 180,
        render: (value) => value || '-',
      },
      {
        title: t('操作人'),
        dataIndex: 'operator_username',
        width: 120,
        render: (_, record) => getQuotaOperatorName(record),
      },
      {
        title: t('原因'),
        dataIndex: 'reason',
        ellipsis: true,
        render: (value) => getQuotaReasonLabel(value),
      },
      {
        title: t('时间'),
        dataIndex: 'created_at',
        width: 170,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
    ],
    [t],
  );

  if (permissionLoading) {
    return (
      <div className='mt-[60px] px-2'>
        <Text>{t('加载中')}</Text>
      </div>
    );
  }

  if (!canRead) {
    return (
      <div className='mt-[60px] px-2'>
        <Banner type='warning' closeIcon={null} description={t('你没有额度流水的查看权限')} />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <CardPro
        type='type3'
        descriptionArea={
          <div className='flex flex-col gap-1'>
            <Text strong>{t('额度流水')}</Text>
            <Text type='tertiary'>{t('查看调额、充值、回收、消耗等额度账本记录。')}</Text>
          </div>
        }
        actionsArea={
          <div className='flex flex-wrap items-center gap-2'>
            <Button size='small' type='tertiary' onClick={exportLedger} disabled={loading}>
              {t('导出 Excel')}
            </Button>
            <Button size='small' type='tertiary' onClick={() => loadLedger(queryState)}>
              {t('刷新')}
            </Button>
          </div>
        }
        searchArea={
          <div className='flex flex-col md:flex-row items-center gap-2 w-full'>
            <Input
              size='small'
              placeholder={t('按用户 ID 筛选')}
              value={userId}
              onChange={(value) => setQueryState((currentState) => updateDraftFilters(currentState, { userId: value }))}
              style={{ width: isMobile ? '100%' : 200 }}
            />
            <Select
              value={entryType}
              onChange={(value) => setQueryState((currentState) => updateDraftFilters(currentState, { entryType: value }))}
              optionList={QUOTA_LEDGER_ENTRY_TYPE_OPTIONS}
              style={{ width: isMobile ? '100%' : 220 }}
            />
            <Button
              size='small'
              type='tertiary'
              onClick={() => {
                const nextQueryState = commitQuotaLedgerFilters(queryState);
                setQueryState(nextQueryState);
                loadLedger(nextQueryState);
              }}
            >
              {t('查询')}
            </Button>
            <Button size='small' type='tertiary' onClick={resetFilters}>
              {t('重置')}
            </Button>
          </div>
        }
        paginationArea={createCardProPagination({
          currentPage: page,
          pageSize,
          total,
          onPageChange: (nextPage) => {
            const nextQueryState = changeCommittedPage(queryState, nextPage);
            setQueryState(nextQueryState);
            loadLedger(nextQueryState);
          },
          onPageSizeChange: (nextSize) => {
            const nextQueryState = changeCommittedPageSize(queryState, nextSize);
            setQueryState(nextQueryState);
            loadLedger(nextQueryState);
          },
          isMobile,
          t,
        })}
        t={t}
      >
        {listError ? (
          <div className='mb-3 flex flex-col gap-2'>
            <Banner type='warning' closeIcon={null} description={listError} />
            <div>
              <Button size='small' type='tertiary' onClick={() => loadLedger(queryState)}>
                {t('重新加载')}
              </Button>
            </div>
          </div>
        ) : null}
        <Table
          className='grid-bordered-table'
          size='default'
          bordered={true}
          columns={columns}
          dataSource={items}
          loading={loading}
          pagination={false}
          empty={
            <Empty
              description={
                committedRequest.userId || committedRequest.entryType
                  ? t('没有匹配的额度流水')
                  : t('暂无额度流水')
              }
            />
          }
        />
      </CardPro>
    </div>
  );
};

export default AdminQuotaLedgerPageV2;
