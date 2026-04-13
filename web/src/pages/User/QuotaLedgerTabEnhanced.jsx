import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Empty,
  Input,
  Select,
  Space,
  Table,
  Tag,
} from '@douyinfe/semi-ui';
import { API, showError, timestamp2string } from '../../helpers';
import {
  QUOTA_LEDGER_ENTRY_TYPE_OPTIONS,
  getQuotaAccountName,
  getQuotaEntryTypeLabel,
  getQuotaOperatorName,
} from '../../helpers/quotaLedgerDisplay';

const getEmptyDescription = (t, userId, entryType) =>
  userId || entryType ? t('没有匹配的额度流水') : t('暂无额度流水');

const QuotaLedgerTabEnhanced = ({ t }) => {
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState([]);
  const [listError, setListError] = useState('');
  const [userId, setUserId] = useState('');
  const [entryType, setEntryType] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  const loadLedger = async (
    nextPage = page,
    nextPageSize = pageSize,
    nextUserId = userId,
    nextEntryType = entryType,
  ) => {
    setLoading(true);
    setListError('');
    try {
      const params = new URLSearchParams({
        p: String(nextPage),
        page_size: String(nextPageSize),
      });
      if (nextUserId) {
        params.set('user_id', nextUserId);
      }
      if (nextEntryType) {
        params.set('entry_type', nextEntryType);
      }
      const res = await API.get(`/api/admin/quota/ledger?${params.toString()}`);
      const { success, message, data } = res.data;
      if (!success) {
        setItems([]);
        setTotal(0);
        setListError(message || t('加载额度流水失败'));
        showError(message);
        return false;
      }
      setItems((data.items || []).map((item) => ({ ...item, key: item.id })));
      setTotal(data.total || 0);
      setPage(data.page || nextPage);
      setPageSize(data.page_size || nextPageSize);
      return true;
    } catch (error) {
      setItems([]);
      setTotal(0);
      setListError(t('加载额度流水失败，请稍后重试'));
      showError(error);
      return false;
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadLedger(1, pageSize, '', '');
  }, []);

  const columns = useMemo(
    () => [
      { title: t('流水号'), dataIndex: 'biz_no', width: 180 },
      { title: t('账户ID'), dataIndex: 'account_id', width: 100 },
      {
        title: t('被操作账号'),
        dataIndex: 'account_username',
        width: 140,
        render: (_, record) => getQuotaAccountName(record),
      },
      {
        title: t('类型'),
        dataIndex: 'entry_type',
        render: (value) => <Tag color='blue'>{getQuotaEntryTypeLabel(value)}</Tag>,
      },
      {
        title: t('方向'),
        dataIndex: 'direction',
        render: (value) => (
          <Tag color={value === 'in' ? 'green' : 'red'}>{value}</Tag>
        ),
      },
      { title: t('金额'), dataIndex: 'amount' },
      { title: t('变动前'), dataIndex: 'balance_before' },
      { title: t('变动后'), dataIndex: 'balance_after' },
      {
        title: t('操作人'),
        dataIndex: 'operator_username',
        render: (_, record) => getQuotaOperatorName(record),
      },
      { title: t('原因'), dataIndex: 'reason', ellipsis: true },
      {
        title: t('时间'),
        dataIndex: 'created_at',
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
    ],
    [t],
  );

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input
          placeholder={t('按用户ID筛选')}
          value={userId}
          onChange={setUserId}
          style={{ width: 220 }}
        />
        <Select
          value={entryType}
          onChange={setEntryType}
          optionList={QUOTA_LEDGER_ENTRY_TYPE_OPTIONS}
          style={{ width: 220 }}
        />
        <Button onClick={() => loadLedger(1, pageSize, userId, entryType)}>
          {t('查询')}
        </Button>
      </Space>

      {listError ? (
        <Banner
          type='warning'
          description={listError}
          closeIcon={null}
          style={{ marginBottom: 16 }}
        />
      ) : null}

      <Table
        columns={columns}
        dataSource={items}
        loading={loading}
        pagination={{
          currentPage: page,
          pageSize,
          total,
          pageSizeOpts: [10, 20, 50],
          showSizeChanger: true,
          onPageChange: (nextPage) => {
            setPage(nextPage);
            loadLedger(nextPage, pageSize, userId, entryType);
          },
          onPageSizeChange: (nextSize) => {
            setPageSize(nextSize);
            setPage(1);
            loadLedger(1, nextSize, userId, entryType);
          },
        }}
        empty={<Empty description={getEmptyDescription(t, userId, entryType)} />}
      />
    </>
  );
};

export default QuotaLedgerTabEnhanced;
