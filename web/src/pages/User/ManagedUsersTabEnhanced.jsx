import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Empty,
  Input,
  InputNumber,
  Modal,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  API,
  showError,
  showSuccess,
  showWarning,
  timestamp2string,
} from '../../helpers';
import { IconClose } from '@douyinfe/semi-icons';
import ModalActionFooter from '../../components/common/modals/ModalActionFooter';

const { Text } = Typography;

const getEmptyDescription = (t, keyword, defaultText, filteredText) =>
  keyword.trim() ? filteredText : defaultText;

const ManagedUsersTabEnhanced = ({
  t,
  canUpdateUserStatus,
  canAdjustQuota,
}) => {
  const [loading, setLoading] = useState(true);
  const [users, setUsers] = useState([]);
  const [listError, setListError] = useState('');
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [selectedRows, setSelectedRows] = useState([]);
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState('');
  const [detailData, setDetailData] = useState(null);
  const [detailUserId, setDetailUserId] = useState(0);
  const [adjustVisible, setAdjustVisible] = useState(false);
  const [adjustSubmitting, setAdjustSubmitting] = useState(false);
  const [adjustingUser, setAdjustingUser] = useState(null);
  const [adjustForm, setAdjustForm] = useState({
    delta: 0,
    reason: '',
    remark: '',
  });
  const [batchAdjustVisible, setBatchAdjustVisible] = useState(false);
  const [batchAdjustSubmitting, setBatchAdjustSubmitting] = useState(false);
  const [batchAdjustResult, setBatchAdjustResult] = useState(null);
  const [batchAdjustForm, setBatchAdjustForm] = useState({
    delta: 0,
    reason: '',
    remark: '',
  });

  const clearSelection = () => {
    setSelectedRowKeys([]);
    setSelectedRows([]);
  };

  const loadUsers = async (
    nextPage = page,
    nextPageSize = pageSize,
    nextKeyword = keyword,
  ) => {
    setLoading(true);
    setListError('');
    try {
      const params = new URLSearchParams({
        p: String(nextPage),
        page_size: String(nextPageSize),
      });
      if (nextKeyword.trim()) {
        params.set('keyword', nextKeyword.trim());
      }
      const res = await API.get(`/api/admin/users?${params.toString()}`);
      const { success, message, data } = res.data;
      if (!success) {
        setUsers([]);
        setTotal(0);
        setListError(message || t('加载用户列表失败'));
        showError(message);
        return false;
      }
      setUsers((data.items || []).map((item) => ({ ...item, key: item.id })));
      setTotal(data.total || 0);
      setPage(data.page || nextPage);
      clearSelection();
      return true;
    } catch (error) {
      setUsers([]);
      setTotal(0);
      setListError(t('加载用户列表失败，请稍后重试'));
      showError(error);
      return false;
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadUsers(1, pageSize, '');
  }, []);

  const loadUserDetail = async (userId, options = {}) => {
    const { openModal = true } = options;
    if (openModal) {
      setDetailVisible(true);
    }
    setDetailUserId(userId);
    setDetailLoading(true);
    setDetailError('');
    try {
      const res = await API.get(`/api/admin/users/${userId}`);
      const { success, message, data } = res.data;
      if (!success) {
        setDetailData(null);
        setDetailError(message || t('加载用户详情失败'));
        showError(message);
        return false;
      }
      setDetailData(data);
      return true;
    } catch (error) {
      setDetailData(null);
      setDetailError(t('加载用户详情失败，请稍后重试'));
      showError(error);
      return false;
    } finally {
      setDetailLoading(false);
    }
  };

  const refreshDetailIfNeeded = async (userIds = []) => {
    if (!detailVisible || !detailUserId) {
      return;
    }
    if (userIds.length === 0 || userIds.includes(detailUserId)) {
      await loadUserDetail(detailUserId, { openModal: false });
    }
  };

  const handleStatusUpdate = async (user, enabled) => {
    const res = await API.post(
      `/api/admin/users/${user.id}/${enabled ? 'enable' : 'disable'}`,
    );
    const { success, message } = res.data;
    if (!success) {
      showError(message);
      return;
    }
    showSuccess(enabled ? t('用户已启用') : t('用户已停用'));
    await loadUsers(page, pageSize, keyword);
    await refreshDetailIfNeeded([user.id]);
  };

  const openAdjustModal = (user) => {
    setAdjustingUser(user);
    setAdjustForm({ delta: 0, reason: '', remark: '' });
    setAdjustVisible(true);
  };

  const handleAdjustQuota = async () => {
    if (!adjustingUser) return;
    setAdjustSubmitting(true);
    try {
      const res = await API.post('/api/admin/quota/adjust', {
        target_user_id: adjustingUser.id,
        delta: Number(adjustForm.delta || 0),
        reason: adjustForm.reason,
        remark: adjustForm.remark,
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      const currentUserId = adjustingUser.id;
      showSuccess(t('额度已调整'));
      setAdjustVisible(false);
      setAdjustingUser(null);
      await loadUsers(page, pageSize, keyword);
      await refreshDetailIfNeeded([currentUserId]);
    } finally {
      setAdjustSubmitting(false);
    }
  };

  const openBatchAdjustModal = () => {
    if (selectedRows.length === 0) {
      showWarning(t('请先勾选要调整额度的用户'));
      return;
    }
    setBatchAdjustForm({ delta: 0, reason: '', remark: '' });
    setBatchAdjustResult(null);
    setBatchAdjustVisible(true);
  };

  const closeBatchAdjustModal = () => {
    setBatchAdjustVisible(false);
    setBatchAdjustResult(null);
    setBatchAdjustForm({ delta: 0, reason: '', remark: '' });
  };

  const handleBatchAdjustQuota = async () => {
    if (selectedRows.length === 0) {
      showWarning(t('请先勾选要调整额度的用户'));
      return;
    }
    setBatchAdjustSubmitting(true);
    try {
      const res = await API.post('/api/admin/quota/adjust/batch', {
        target_user_ids: selectedRows.map((item) => item.id),
        delta: Number(batchAdjustForm.delta || 0),
        reason: batchAdjustForm.reason,
        remark: batchAdjustForm.remark,
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setBatchAdjustResult(data || {});
      if ((data?.success_count || 0) > 0) {
        showSuccess(
          `${t('批量调额完成')}，${t('成功')} ${data.success_count}，${t('失败')} ${data.failed_count || 0}`,
        );
      } else {
        showWarning(
          `${t('批量调额未成功执行')}，${t('失败')} ${data?.failed_count || 0}`,
        );
      }
      await loadUsers(page, pageSize, keyword);
      const successUserIds = (data?.success_user_ids || []).map((id) =>
        Number(id),
      );
      await refreshDetailIfNeeded(successUserIds);
    } finally {
      setBatchAdjustSubmitting(false);
    }
  };

  const rowSelection = useMemo(
    () =>
      canAdjustQuota
        ? {
            selectedRowKeys,
            onChange: (nextKeys, nextRows) => {
              setSelectedRowKeys(nextKeys);
              setSelectedRows(nextRows);
            },
          }
        : undefined,
    [canAdjustQuota, selectedRowKeys],
  );

  const columns = useMemo(
    () => [
      { title: t('用户名'), dataIndex: 'username' },
      { title: t('显示名称'), dataIndex: 'display_name' },
      {
        title: t('余额'),
        dataIndex: 'quota',
        render: (value) => value ?? 0,
      },
      {
        title: t('所属代理商'),
        dataIndex: 'parent_agent_id',
        render: (value) => value || '-',
      },
      {
        title: t('最近活跃'),
        dataIndex: 'last_active_at',
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        render: (value) => (
          <Tag color={value === 1 ? 'green' : 'red'}>
            {value === 1 ? t('启用') : t('禁用')}
          </Tag>
        ),
      },
      {
        title: t('操作'),
        dataIndex: 'operate',
        render: (_, record) => (
          <Space>
            <Button
              size='small'
              theme='outline'
              onClick={() => loadUserDetail(record.id)}
            >
              {t('详情')}
            </Button>
            {canAdjustQuota ? (
              <Button
                size='small'
                theme='solid'
                onClick={() => openAdjustModal(record)}
              >
                {t('调额')}
              </Button>
            ) : null}
            {canUpdateUserStatus ? (
              <Button
                size='small'
                theme='light'
                type={record.status === 1 ? 'danger' : 'primary'}
                onClick={() => handleStatusUpdate(record, record.status !== 1)}
              >
                {record.status === 1 ? t('停用') : t('启用')}
              </Button>
            ) : null}
          </Space>
        ),
      },
    ],
    [canAdjustQuota, canUpdateUserStatus, t],
  );

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input
          placeholder={t('搜索用户')}
          value={keyword}
          onChange={setKeyword}
          style={{ width: 280 }}
        />
        <Button onClick={() => loadUsers(1, pageSize, keyword)}>
          {t('查询')}
        </Button>
        {canAdjustQuota ? (
          <>
            <Button
              theme='solid'
              type='primary'
              disabled={selectedRowKeys.length === 0}
              onClick={openBatchAdjustModal}
            >
              {t('批量调额')}
              {selectedRowKeys.length > 0 ? ` (${selectedRowKeys.length})` : ''}
            </Button>
            {selectedRowKeys.length > 0 ? (
              <Button theme='borderless' onClick={clearSelection}>
                {t('清空勾选')}
              </Button>
            ) : null}
          </>
        ) : null}
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
        rowSelection={rowSelection}
        columns={columns}
        dataSource={users}
        loading={loading}
        pagination={{
          currentPage: page,
          pageSize,
          total,
          pageSizeOpts: [10, 20, 50],
          showSizeChanger: true,
          onPageChange: (nextPage) => {
            setPage(nextPage);
            loadUsers(nextPage, pageSize, keyword);
          },
          onPageSizeChange: (nextSize) => {
            setPageSize(nextSize);
            setPage(1);
            loadUsers(1, nextSize, keyword);
          },
        }}
        empty={
          <Empty
            description={getEmptyDescription(
              t,
              keyword,
              t('暂无用户数据'),
              t('没有匹配的用户'),
            )}
          />
        }
      />

      <Modal
        title={t('用户详情')}
        visible={detailVisible}
        footer={null}
        onCancel={() => {
          setDetailVisible(false);
          setDetailData(null);
          setDetailError('');
          setDetailUserId(0);
        }}
      >
        {detailLoading ? (
          <Text>{t('加载中')}</Text>
        ) : detailError ? (
          <Space vertical align='start' style={{ width: '100%' }}>
            <Banner type='warning' description={detailError} closeIcon={null} />
            <Button
              onClick={() => loadUserDetail(detailUserId, { openModal: false })}
            >
              {t('重试')}
            </Button>
          </Space>
        ) : detailData ? (
          <Space vertical align='start'>
            <Text>{`${t('用户名')}：${detailData.username}`}</Text>
            <Text>{`${t('显示名称')}：${detailData.display_name || '-'}`}</Text>
            <Text>{`${t('状态')}：${detailData.status === 1 ? t('启用') : t('禁用')}`}</Text>
            <Text>{`${t('邮箱')}：${detailData.email || '-'}`}</Text>
            <Text>{`${t('余额')}：${detailData.quota_summary?.balance ?? 0}`}</Text>
            <Text>{`${t('冻结余额')}：${detailData.quota_summary?.frozen_balance ?? 0}`}</Text>
            <Text>{`${t('累计调增')}：${detailData.quota_summary?.total_adjusted_in ?? 0}`}</Text>
            <Text>{`${t('累计调减')}：${detailData.quota_summary?.total_adjusted_out ?? 0}`}</Text>
          </Space>
        ) : (
          <Empty description={t('暂无详情')} />
        )}
      </Modal>

      <Modal
        title={t('调整额度')}
        visible={adjustVisible}
        onCancel={() => {
          setAdjustVisible(false);
          setAdjustingUser(null);
        }}
        footer={
          <ModalActionFooter
            onConfirm={handleAdjustQuota}
            onCancel={() => {
              setAdjustVisible(false);
              setAdjustingUser(null);
            }}
            confirmText={t('提交')}
            cancelText={t('取消')}
            confirmLoading={adjustSubmitting}
          />
        }
      >
        <Space vertical align='start' style={{ width: '100%' }}>
          <Text>
            {adjustingUser ? `${t('目标用户')}：${adjustingUser.username}` : ''}
          </Text>
          <InputNumber
            value={adjustForm.delta}
            onChange={(value) =>
              setAdjustForm((prev) => ({ ...prev, delta: value }))
            }
            style={{ width: '100%' }}
            placeholder={t('输入正数为增加，负数为减少')}
          />
          <Input
            value={adjustForm.reason}
            onChange={(value) =>
              setAdjustForm((prev) => ({ ...prev, reason: value }))
            }
            placeholder={t('调整原因')}
          />
          <Input
            value={adjustForm.remark}
            onChange={(value) =>
              setAdjustForm((prev) => ({ ...prev, remark: value }))
            }
            placeholder={t('备注')}
          />
        </Space>
      </Modal>

      <Modal
        title={batchAdjustResult ? t('批量调额结果') : t('批量调额')}
        visible={batchAdjustVisible}
        onCancel={closeBatchAdjustModal}
        footer={
          <ModalActionFooter
            onConfirm={batchAdjustResult ? closeBatchAdjustModal : handleBatchAdjustQuota}
            onCancel={closeBatchAdjustModal}
            confirmText={batchAdjustResult ? t('关闭') : t('提交')}
            cancelText={t('取消')}
            confirmLoading={batchAdjustSubmitting}
            confirmIcon={batchAdjustResult ? <IconClose /> : undefined}
            showCancel={!batchAdjustResult}
          />
        }
      >
        {batchAdjustResult ? (
          <Space vertical align='start' style={{ width: '100%' }}>
            <Text>{`${t('目标数量')}：${batchAdjustResult.target_count || 0}`}</Text>
            <Text>{`${t('成功数量')}：${batchAdjustResult.success_count || 0}`}</Text>
            <Text>{`${t('失败数量')}：${batchAdjustResult.failed_count || 0}`}</Text>
            {(batchAdjustResult.failed_items || []).length > 0 ? (
              <Space vertical align='start' style={{ width: '100%' }}>
                <Text strong>{t('失败明细')}</Text>
                {(batchAdjustResult.failed_items || []).map((item, index) => (
                  <Banner
                    key={`${item.target_user_id || index}-${index}`}
                    type='warning'
                    closeIcon={null}
                    description={`${item.username || '-'} / ID ${item.target_user_id}: ${item.error_message}`}
                  />
                ))}
              </Space>
            ) : (
              <Empty description={t('本次批量调额没有失败项')} />
            )}
          </Space>
        ) : (
          <Space vertical align='start' style={{ width: '100%' }}>
            <Text>{`${t('已勾选用户')}：${selectedRows.length}`}</Text>
            <Text>
              {selectedRows
                .slice(0, 5)
                .map((item) => item.username)
                .join('、')}
              {selectedRows.length > 5 ? ' ...' : ''}
            </Text>
            <InputNumber
              value={batchAdjustForm.delta}
              onChange={(value) =>
                setBatchAdjustForm((prev) => ({ ...prev, delta: value }))
              }
              style={{ width: '100%' }}
              placeholder={t('输入正数为增加，负数为减少')}
            />
            <Input
              value={batchAdjustForm.reason}
              onChange={(value) =>
                setBatchAdjustForm((prev) => ({ ...prev, reason: value }))
              }
              placeholder={t('调整原因')}
            />
            <Input
              value={batchAdjustForm.remark}
              onChange={(value) =>
                setBatchAdjustForm((prev) => ({ ...prev, remark: value }))
              }
              placeholder={t('备注')}
            />
          </Space>
        )}
      </Modal>
    </>
  );
};

export default ManagedUsersTabEnhanced;
