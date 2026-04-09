import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Empty,
  Input,
  Modal,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const getEmptyDescription = (t, keyword, defaultText, filteredText) =>
  keyword.trim() ? filteredText : defaultText;

const AgentManagementTabEnhanced = ({
  t,
  canCreateAgent,
  canUpdateAgentStatus,
}) => {
  const [loading, setLoading] = useState(true);
  const [agents, setAgents] = useState([]);
  const [listError, setListError] = useState('');
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [creating, setCreating] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState('');
  const [detailAgentId, setDetailAgentId] = useState(0);
  const [detailData, setDetailData] = useState(null);
  const [formState, setFormState] = useState({
    username: '',
    password: '',
    agent_name: '',
    company_name: '',
    contact_phone: '',
    remark: '',
  });

  const loadAgents = async (
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
      const res = await API.get(`/api/admin/agents?${params.toString()}`);
      const { success, message, data } = res.data;
      if (!success) {
        setAgents([]);
        setTotal(0);
        setListError(message || t('加载代理商列表失败'));
        showError(message);
        return false;
      }
      setAgents((data.items || []).map((item) => ({ ...item, key: item.id })));
      setTotal(data.total || 0);
      setPage(data.page || nextPage);
      return true;
    } catch (error) {
      setAgents([]);
      setTotal(0);
      setListError(t('加载代理商列表失败，请稍后重试'));
      showError(error);
      return false;
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAgents(1, pageSize, '');
  }, []);

  const resetCreateForm = () => {
    setFormState({
      username: '',
      password: '',
      agent_name: '',
      company_name: '',
      contact_phone: '',
      remark: '',
    });
  };

  const loadAgentDetail = async (agentId, options = {}) => {
    const { openModal = true } = options;
    if (openModal) {
      setDetailVisible(true);
    }
    setDetailAgentId(agentId);
    setDetailLoading(true);
    setDetailError('');
    try {
      const res = await API.get(`/api/admin/agents/${agentId}`);
      const { success, message, data } = res.data;
      if (!success) {
        setDetailData(null);
        setDetailError(message || t('加载代理商详情失败'));
        showError(message);
        return false;
      }
      setDetailData(data);
      return true;
    } catch (error) {
      setDetailData(null);
      setDetailError(t('加载代理商详情失败，请稍后重试'));
      showError(error);
      return false;
    } finally {
      setDetailLoading(false);
    }
  };

  const refreshAgentDetailIfNeeded = async (agentId) => {
    if (detailVisible && detailAgentId === agentId) {
      await loadAgentDetail(agentId, { openModal: false });
    }
  };

  const handleCreateAgent = async () => {
    setCreating(true);
    try {
      const res = await API.post('/api/admin/agents', formState);
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('代理商已创建'));
      setShowCreateModal(false);
      resetCreateForm();
      await loadAgents(page, pageSize, keyword);
    } finally {
      setCreating(false);
    }
  };

  const handleStatusUpdate = async (agent, enabled) => {
    const res = await API.post(
      `/api/admin/agents/${agent.id}/${enabled ? 'enable' : 'disable'}`,
    );
    const { success, message } = res.data;
    if (!success) {
      showError(message);
      return;
    }
    showSuccess(enabled ? t('代理商已启用') : t('代理商已停用'));
    await loadAgents(page, pageSize, keyword);
    await refreshAgentDetailIfNeeded(agent.id);
  };

  const columns = useMemo(
    () => [
      { title: t('用户名'), dataIndex: 'username' },
      { title: t('代理商名称'), dataIndex: 'agent_name' },
      { title: t('公司名称'), dataIndex: 'company_name' },
      { title: t('联系电话'), dataIndex: 'contact_phone' },
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
              onClick={() => loadAgentDetail(record.id)}
            >
              {t('详情')}
            </Button>
            {canUpdateAgentStatus ? (
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
    [canUpdateAgentStatus, t],
  );

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input
          placeholder={t('搜索代理商名称')}
          value={keyword}
          onChange={setKeyword}
          style={{ width: 280 }}
        />
        <Button onClick={() => loadAgents(1, pageSize, keyword)}>
          {t('查询')}
        </Button>
        {canCreateAgent ? (
          <Button
            theme='solid'
            type='primary'
            onClick={() => setShowCreateModal(true)}
          >
            {t('新增代理商')}
          </Button>
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
        columns={columns}
        dataSource={agents}
        loading={loading}
        pagination={{
          currentPage: page,
          pageSize,
          total,
          pageSizeOpts: [10, 20, 50],
          showSizeChanger: true,
          onPageChange: (nextPage) => {
            setPage(nextPage);
            loadAgents(nextPage, pageSize, keyword);
          },
          onPageSizeChange: (nextSize) => {
            setPageSize(nextSize);
            setPage(1);
            loadAgents(1, nextSize, keyword);
          },
        }}
        empty={
          <Empty
            description={getEmptyDescription(
              t,
              keyword,
              t('暂无代理商数据'),
              t('没有匹配的代理商'),
            )}
          />
        }
      />

      <Modal
        title={t('新增代理商')}
        visible={showCreateModal}
        onCancel={() => {
          setShowCreateModal(false);
          resetCreateForm();
        }}
        onOk={handleCreateAgent}
        confirmLoading={creating}
      >
        <Space vertical align='start' style={{ width: '100%' }}>
          <Input
            placeholder={t('用户名')}
            value={formState.username}
            onChange={(value) =>
              setFormState((prev) => ({ ...prev, username: value }))
            }
          />
          <Input
            mode='password'
            placeholder={t('密码')}
            value={formState.password}
            onChange={(value) =>
              setFormState((prev) => ({ ...prev, password: value }))
            }
          />
          <Input
            placeholder={t('代理商名称')}
            value={formState.agent_name}
            onChange={(value) =>
              setFormState((prev) => ({ ...prev, agent_name: value }))
            }
          />
          <Input
            placeholder={t('公司名称')}
            value={formState.company_name}
            onChange={(value) =>
              setFormState((prev) => ({ ...prev, company_name: value }))
            }
          />
          <Input
            placeholder={t('联系电话')}
            value={formState.contact_phone}
            onChange={(value) =>
              setFormState((prev) => ({ ...prev, contact_phone: value }))
            }
          />
          <Input
            placeholder={t('备注')}
            value={formState.remark}
            onChange={(value) =>
              setFormState((prev) => ({ ...prev, remark: value }))
            }
          />
        </Space>
      </Modal>

      <Modal
        title={t('代理商详情')}
        visible={detailVisible}
        footer={null}
        onCancel={() => {
          setDetailVisible(false);
          setDetailData(null);
          setDetailError('');
          setDetailAgentId(0);
        }}
      >
        {detailLoading ? (
          <Text>{t('加载中')}</Text>
        ) : detailError ? (
          <Space vertical align='start' style={{ width: '100%' }}>
            <Banner type='warning' description={detailError} closeIcon={null} />
            <Button
              onClick={() => loadAgentDetail(detailAgentId, { openModal: false })}
            >
              {t('重试')}
            </Button>
          </Space>
        ) : detailData ? (
          <Space vertical align='start'>
            <Text>{`${t('用户名')}：${detailData.username}`}</Text>
            <Text>{`${t('代理商名称')}：${detailData.agent_name || '-'}`}</Text>
            <Text>{`${t('公司名称')}：${detailData.company_name || '-'}`}</Text>
            <Text>{`${t('联系电话')}：${detailData.contact_phone || '-'}`}</Text>
            <Text>{`${t('状态')}：${detailData.status === 1 ? t('启用') : t('禁用')}`}</Text>
            <Text>{`${t('余额')}：${detailData.quota_summary?.balance ?? 0}`}</Text>
            <Text>{`${t('冻结余额')}：${detailData.quota_summary?.frozen_balance ?? 0}`}</Text>
            <Text>{`${t('备注')}：${detailData.remark || '-'}`}</Text>
          </Space>
        ) : (
          <Empty description={t('暂无详情')} />
        )}
      </Modal>
    </>
  );
};

export default AgentManagementTabEnhanced;
