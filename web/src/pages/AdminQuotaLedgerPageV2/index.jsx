import React, { useEffect, useMemo, useState } from 'react';
import { Banner, Button, Empty, Input, Select, Table, Tag, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import CardPro from '../../components/common/ui/CardPro';
import { API, createCardProPagination, renderQuota, showError, timestamp2string } from '../../helpers';
import {
  QUOTA_LEDGER_ENTRY_TYPE_OPTIONS,
  getQuotaAccountName,
  getQuotaEntryTypeLabel,
  getQuotaOperatorName,
  getQuotaReasonLabel,
} from '../../helpers/quotaLedgerDisplay';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';

const { Text } = Typography;
const ADMIN_QUOTA_LEDGER_DIGITS = 6;

const AdminQuotaLedgerPageV2 = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const { loading: permissionLoading, hasActionPermission } = useUserPermissions();
  const canRead = hasActionPermission('quota_management', 'ledger_read');

  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState([]);
  const [listError, setListError] = useState('');
  const [userId, setUserId] = useState('');
  const [entryType, setEntryType] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  const loadLedger = async (nextPage = page, nextPageSize = pageSize, nextUserId = userId, nextEntryType = entryType) => {
    if (!canRead) {
      return;
    }

    setLoading(true);
    setListError('');
    try {
      const params = new URLSearchParams({
        p: String(nextPage),
        page_size: String(nextPageSize),
      });
      if (nextUserId.trim()) {
        params.set('user_id', nextUserId.trim());
      }
      if (nextEntryType) {
        params.set('entry_type', nextEntryType);
      }

      const res = await API.get(`/api/admin/quota/ledger?${params.toString()}`);
      if (!res.data.success) {
        setItems([]);
        setTotal(0);
        setListError(res.data.message || t('加载额度流水失败'));
        return;
      }

      const data = res.data.data || {};
      setItems((data.items || []).map((item) => ({ ...item, key: item.id })));
      setPage(data.page || nextPage);
      setPageSize(data.page_size || nextPageSize);
      setTotal(data.total || 0);
    } catch (error) {
      setItems([]);
      setTotal(0);
      setListError(t('加载额度流水失败，请稍后重试'));
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  const resetFilters = async () => {
    setUserId('');
    setEntryType('');
    await loadLedger(1, pageSize, '', '');
  };

  useEffect(() => {
    if (!permissionLoading && canRead) {
      loadLedger(1, pageSize, '', '');
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
            <Button size='small' type='tertiary' onClick={() => loadLedger(page, pageSize, userId, entryType)}>
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
              onChange={setUserId}
              style={{ width: isMobile ? '100%' : 200 }}
            />
            <Select
              value={entryType}
              onChange={setEntryType}
              optionList={QUOTA_LEDGER_ENTRY_TYPE_OPTIONS}
              style={{ width: isMobile ? '100%' : 220 }}
            />
            <Button size='small' type='tertiary' onClick={() => loadLedger(1, pageSize, userId, entryType)}>
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
            setPage(nextPage);
            loadLedger(nextPage, pageSize, userId, entryType);
          },
          onPageSizeChange: (nextSize) => {
            setPage(1);
            setPageSize(nextSize);
            loadLedger(1, nextSize, userId, entryType);
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
              <Button size='small' type='tertiary' onClick={() => loadLedger(page, pageSize, userId, entryType)}>
                {t('重新加载')}
              </Button>
            </div>
          </div>
        ) : null}
        <Table
          className='grid-bordered-table'
          size='small'
          bordered={true}
          columns={columns}
          dataSource={items}
          loading={loading}
          pagination={false}
          empty={<Empty description={userId || entryType ? t('没有匹配的额度流水') : t('暂无额度流水')} />}
        />
      </CardPro>
    </div>
  );
};

export default AdminQuotaLedgerPageV2;
