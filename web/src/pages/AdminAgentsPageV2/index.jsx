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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Descriptions,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import ModalActionFooter from '../../components/common/modals/ModalActionFooter';
import CardPro from '../../components/common/ui/CardPro';
import { API, createCardProPagination, showError, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import { toGroupOptions } from '../../hooks/users/useUsersData.helpers';

const { Text, Title } = Typography;

const emptyFormState = {
  username: '',
  password: '',
  display_name: '',
  agent_name: '',
  company_name: '',
  contact_phone: '',
  remark: '',
  group: '',
};

const sectionStyle = {
  border: '1px solid var(--semi-color-border)',
  borderRadius: 12,
  padding: 16,
  background: 'var(--semi-color-bg-0)',
};

const mergedSectionStyle = {
  ...sectionStyle,
  display: 'flex',
  flexDirection: 'column',
  gap: 24,
};

const sectionBlockStyle = {
  borderTop: '1px solid var(--semi-color-border)',
  paddingTop: 20,
};

const fieldStyle = {
  display: 'flex',
  flexDirection: 'column',
  gap: 6,
};

const actionLinkStyle = {
  paddingLeft: 0,
  paddingRight: 0,
};

const AdminAgentsPageV2 = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const { loading: permissionLoading, hasActionPermission } = useUserPermissions();

  const canRead = hasActionPermission('agent_management', 'read');
  const canCreate = hasActionPermission('agent_management', 'create');
  const canUpdate = hasActionPermission('agent_management', 'update');
  const canUpdateStatus = hasActionPermission('agent_management', 'update_status');

  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [agents, setAgents] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [listError, setListError] = useState('');
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState('');
  const [detailData, setDetailData] = useState(null);
  const [detailAgentId, setDetailAgentId] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingAgent, setEditingAgent] = useState(null);
  const [formState, setFormState] = useState(emptyFormState);
  const [groupOptions, setGroupOptions] = useState([]);
  const [defaultGroup, setDefaultGroup] = useState('');

  const closeModal = () => {
    setModalVisible(false);
    setEditingAgent(null);
    setFormState({ ...emptyFormState, group: defaultGroup });
  };

  const fetchGroups = async () => {
    try {
      const res = await API.get('/api/group/');
      if (res?.data?.success !== true) {
        showError(res?.data?.message || t('加载分组列表失败'));
        return;
      }
      const options = toGroupOptions(res.data);
      setGroupOptions(options);
      const nextDefaultGroup = options[0]?.value || '';
      setDefaultGroup(nextDefaultGroup);
      setFormState((prev) => ({
        ...prev,
        group: prev.group || nextDefaultGroup,
      }));
    } catch (error) {
      showError(error);
    }
  };

  const loadAgents = async (nextPage = page, nextPageSize = pageSize, nextKeyword = keyword) => {
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
      if (nextKeyword.trim()) {
        params.set('keyword', nextKeyword.trim());
      }

      const res = await API.get(`/api/admin/agents?${params.toString()}`);
      if (!res.data.success) {
        setAgents([]);
        setTotal(0);
        setListError(res.data.message || t('加载代理商列表失败'));
        return;
      }

      const data = res.data.data || {};
      setAgents((data.items || []).map((item) => ({ ...item, key: item.id })));
      setPage(data.page || nextPage);
      setPageSize(data.page_size || nextPageSize);
      setTotal(data.total || 0);
    } catch (error) {
      setAgents([]);
      setTotal(0);
      setListError(t('加载代理商列表失败，请稍后重试'));
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  const loadAgentDetail = async (agentId) => {
    setDetailVisible(true);
    setDetailAgentId(agentId);
    setDetailLoading(true);
    setDetailError('');
    setDetailData(null);
    try {
      const res = await API.get(`/api/admin/agents/${agentId}`);
      if (!res.data.success) {
        setDetailError(res.data.message || t('加载代理商详情失败'));
        return;
      }

      setDetailData(res.data.data || null);
    } catch (error) {
      setDetailError(t('加载代理商详情失败，请稍后重试'));
      showError(error);
    } finally {
      setDetailLoading(false);
    }
  };

  const openCreateModal = () => {
    setEditingAgent(null);
    setFormState({ ...emptyFormState, group: defaultGroup });
    setModalVisible(true);
  };

  const openEditModal = async (record) => {
    setEditingAgent(record);
    setModalVisible(true);
    setFormState({
      username: record.username || '',
      password: '',
      display_name: record.display_name || '',
      agent_name: record.agent_name || '',
      company_name: record.company_name || '',
      contact_phone: record.contact_phone || '',
      remark: '',
      group: record.group || defaultGroup,
    });

    try {
      const res = await API.get(`/api/admin/agents/${record.id}`);
      if (!res.data.success) {
        showError(res.data.message || t('加载代理商详情失败'));
        return;
      }

      const data = res.data.data || {};
      setFormState((prev) => ({
        ...prev,
        display_name: data.display_name || '',
        agent_name: data.agent_name || '',
        company_name: data.company_name || '',
        contact_phone: data.contact_phone || '',
        remark: data.remark || '',
        group: data.group || defaultGroup,
      }));
    } catch (error) {
      showError(error);
    }
  };

  const handleSubmit = async () => {
    if (!editingAgent && (!formState.username.trim() || !formState.password.trim() || !formState.agent_name.trim())) {
      showError(t('请填写用户名、初始密码和代理商名称'));
      return;
    }
    if (editingAgent && !formState.agent_name.trim()) {
      showError(t('请填写代理商名称'));
      return;
    }

    if (!editingAgent && !formState.group) {
      showError(t('请选择分组'));
      return;
    }

    setSubmitting(true);
    try {
      const payload = editingAgent
        ? {
            display_name: formState.display_name.trim(),
            agent_name: formState.agent_name.trim(),
            company_name: formState.company_name.trim(),
            contact_phone: formState.contact_phone.trim(),
            remark: formState.remark.trim(),
          }
        : {
            username: formState.username.trim(),
            password: formState.password,
            agent_name: formState.agent_name.trim(),
            company_name: formState.company_name.trim(),
            contact_phone: formState.contact_phone.trim(),
            remark: formState.remark.trim(),
            group: formState.group,
          };

      const res = editingAgent
        ? await API.put(`/api/admin/agents/${editingAgent.id}`, payload)
        : await API.post('/api/admin/agents', payload);
      if (!res.data.success) {
        showError(res.data.message || t('保存代理商失败'));
        return;
      }

      const currentEditingId = editingAgent?.id;
      showSuccess(editingAgent ? t('代理商资料已更新') : t('代理商已创建'));
      closeModal();
      await loadAgents(page, pageSize, keyword);
      if (detailVisible && currentEditingId && detailAgentId === currentEditingId) {
        await loadAgentDetail(currentEditingId);
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleStatusUpdate = async (record, enabled) => {
    try {
      const res = await API.post(`/api/admin/agents/${record.id}/${enabled ? 'enable' : 'disable'}`);
      if (!res.data.success) {
        showError(res.data.message || t('更新代理商状态失败'));
        return;
      }

      showSuccess(enabled ? t('代理商已启用') : t('代理商已停用'));
      await loadAgents(page, pageSize, keyword);
      if (detailVisible && detailAgentId === record.id) {
        await loadAgentDetail(record.id);
      }
    } catch (error) {
      showError(error);
    }
  };

  const resetFilters = async () => {
    setKeyword('');
    await loadAgents(1, pageSize, '');
  };

  useEffect(() => {
    fetchGroups();
    if (!permissionLoading && canRead) {
      loadAgents(1, pageSize, '');
    }
  }, [permissionLoading, canRead]);

  const columns = useMemo(
    () => [
      {
        title: t('代理商'),
        dataIndex: 'username',
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text strong>{record.agent_name || record.display_name || record.username}</Text>
            <Text type='tertiary' size='small'>
              {record.username}
            </Text>
          </div>
        ),
      },
      {
        title: t('公司名称'),
        dataIndex: 'company_name',
        render: (value) => value || '-',
      },
      {
        title: t('联系电话'),
        dataIndex: 'contact_phone',
        width: 140,
        render: (value) => value || '-',
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 100,
        render: (value) => (
          <Tag color={value === 1 ? 'green' : 'red'} shape='circle'>
            {value === 1 ? t('启用') : t('停用')}
          </Tag>
        ),
      },
      {
        title: t('操作'),
        dataIndex: 'operate',
        width: 220,
        render: (_, record) => (
          <Space spacing={12}>
            <Button
              size='small'
              theme='borderless'
              type='tertiary'
              style={actionLinkStyle}
              onClick={() => loadAgentDetail(record.id)}
            >
              {t('详情')}
            </Button>
            {canUpdate ? (
              <Button
                size='small'
                theme='borderless'
                type='tertiary'
                style={actionLinkStyle}
                onClick={() => openEditModal(record)}
              >
                {t('编辑')}
              </Button>
            ) : null}
            {canUpdateStatus ? (
              <Button
                size='small'
                theme='borderless'
                type={record.status === 1 ? 'danger' : 'primary'}
                style={actionLinkStyle}
                onClick={() => handleStatusUpdate(record, record.status !== 1)}
              >
                {record.status === 1 ? t('停用') : t('启用')}
              </Button>
            ) : null}
          </Space>
        ),
      },
    ],
    [canUpdate, canUpdateStatus, t],
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
        <Banner type='warning' closeIcon={null} description={t('你没有代理商管理的查看权限')} />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <Modal
        title={editingAgent ? t('编辑代理商') : t('新增代理商')}
        visible={modalVisible}
        onCancel={closeModal}
        footer={
          <ModalActionFooter
            onConfirm={handleSubmit}
            onCancel={closeModal}
            confirmText={editingAgent ? t('保存变更') : t('确认创建')}
            cancelText={t('取消')}
            confirmLoading={submitting}
          />
        }
        width={760}
      >
        <div style={mergedSectionStyle}>
          <div>
            <div className='mb-3 flex flex-col gap-1'>
              <Text strong>{t('基础信息')}</Text>
              <Text type='tertiary'>{t('维护账号标识、代理商名称和联系信息。')}</Text>
            </div>
            <div className='grid gap-3 md:grid-cols-2'>
              {!editingAgent ? (
                <>
                  <div style={fieldStyle}>
                    <Text type='tertiary'>{t('登录用户名')}</Text>
                    <Input
                      placeholder={t('请输入登录用户名')}
                      value={formState.username}
                      onChange={(value) => setFormState((prev) => ({ ...prev, username: value }))}
                    />
                  </div>
                  <div style={fieldStyle}>
                    <Text type='tertiary'>{t('初始密码')}</Text>
                    <Input
                      mode='password'
                      placeholder={t('请输入初始密码')}
                      value={formState.password}
                      onChange={(value) => setFormState((prev) => ({ ...prev, password: value }))}
                    />
                  </div>
                </>
              ) : (
                <div style={fieldStyle}>
                  <Text type='tertiary'>{t('显示名称')}</Text>
                  <Input
                    placeholder={t('请输入显示名称')}
                    value={formState.display_name}
                    onChange={(value) => setFormState((prev) => ({ ...prev, display_name: value }))}
                  />
                </div>
              )}
              <div style={fieldStyle}>
                <Text type='tertiary'>{t('代理商名称')}</Text>
                <Input
                  placeholder={t('请输入代理商名称')}
                  value={formState.agent_name}
                  onChange={(value) => setFormState((prev) => ({ ...prev, agent_name: value }))}
                />
              </div>
              <div style={fieldStyle}>
                <Text type='tertiary'>{t('公司名称')}</Text>
                <Input
                  placeholder={t('请输入公司名称')}
                  value={formState.company_name}
                  onChange={(value) => setFormState((prev) => ({ ...prev, company_name: value }))}
                />
              </div>
              <div style={fieldStyle}>
                <Text type='tertiary'>{t('联系电话')}</Text>
                <Input
                  placeholder={t('请输入联系电话')}
                  value={formState.contact_phone}
                  onChange={(value) => setFormState((prev) => ({ ...prev, contact_phone: value }))}
                />
              </div>
              {!editingAgent ? (
                <div style={fieldStyle}>
                  <Text type='tertiary'>{t('分组')}</Text>
                  <Select
                    placeholder={t('请选择分组')}
                    optionList={groupOptions}
                    value={formState.group}
                    onChange={(value) =>
                      setFormState((prev) => ({ ...prev, group: value || '' }))
                    }
                  />
                </div>
              ) : null}
            </div>
          </div>
          <div style={sectionBlockStyle}>
            <div className='mb-3 flex flex-col gap-1'>
              <Text strong>{t('备注信息')}</Text>
              <Text type='tertiary'>{t('补充合作背景、负责人或特殊说明。')}</Text>
            </div>
            <TextArea
              rows={3}
              placeholder={t('请输入备注信息')}
              value={formState.remark}
              onChange={(value) => setFormState((prev) => ({ ...prev, remark: value }))}
            />
          </div>
        </div>
      </Modal>

      <Modal
        title={t('代理商详情')}
        visible={detailVisible}
        footer={null}
        width={720}
        onCancel={() => {
          setDetailVisible(false);
          setDetailData(null);
          setDetailError('');
          setDetailAgentId(0);
        }}
      >
        {detailLoading ? <Text>{t('加载中')}</Text> : null}
        {!detailLoading && detailError ? (
          <div className='flex flex-col gap-2'>
            <Banner type='warning' closeIcon={null} description={detailError} />
            <div>
              <Button size='small' type='tertiary' onClick={() => loadAgentDetail(detailAgentId)}>
                {t('重试')}
              </Button>
            </div>
          </div>
        ) : null}
        {!detailLoading && !detailError && detailData ? (
          <div className='flex flex-col gap-4'>
            <div style={sectionStyle}>
              <div className='flex items-start justify-between gap-4'>
                <div className='flex flex-col gap-1'>
                  <Title heading={6} style={{ margin: 0 }}>
                    {detailData.agent_name || detailData.display_name || detailData.username}
                  </Title>
                  <Text type='tertiary'>{detailData.username}</Text>
                </div>
                <Tag color={detailData.status === 1 ? 'green' : 'red'} shape='circle'>
                  {detailData.status === 1 ? t('启用') : t('停用')}
                </Tag>
              </div>
            </div>
            <div style={mergedSectionStyle}>
              <div className='mb-3 flex flex-col gap-1'>
                <Text strong>{t('账号资料')}</Text>
                <Text type='tertiary'>{t('查看代理商名称、公司信息和联系方式。')}</Text>
              </div>
              <Descriptions
                data={[
                  { key: t('显示名称'), value: detailData.display_name || '-' },
                  { key: t('代理商名称'), value: detailData.agent_name || '-' },
                  { key: t('公司名称'), value: detailData.company_name || '-' },
                  { key: t('联系电话'), value: detailData.contact_phone || '-' },
                ]}
                columns={2}
              />
              <div style={sectionBlockStyle}>
                <div className='mb-3 flex flex-col gap-1'>
                  <Text strong>{t('备注信息')}</Text>
                </div>
                <Text style={{ whiteSpace: 'pre-wrap' }}>{detailData.remark || '-'}</Text>
              </div>
              <div style={sectionBlockStyle}>
                <div className='mb-3 flex flex-col gap-1'>
                  <Text strong>{t('额度摘要')}</Text>
                  <Text type='tertiary'>{t('查看当前可用额度和冻结额度。')}</Text>
                </div>
                <Descriptions
                  data={[
                    { key: t('当前额度'), value: detailData.quota_summary?.balance ?? 0 },
                    { key: t('冻结额度'), value: detailData.quota_summary?.frozen_balance ?? 0 },
                  ]}
                  columns={2}
                />
              </div>
            </div>
          </div>
        ) : null}
      </Modal>

      <CardPro
        type='type3'
        descriptionArea={
          <div className='flex flex-col gap-1'>
            <Text strong>{t('代理商管理')}</Text>
            <Text type='tertiary'>{t('维护代理商账号、资料信息和启停状态。')}</Text>
          </div>
        }
        actionsArea={
          <div className='flex flex-wrap items-center gap-2'>
            {canCreate ? (
              <Button size='small' theme='light' type='primary' onClick={openCreateModal}>
                {t('新增代理商')}
              </Button>
            ) : null}
            <Button size='small' type='tertiary' onClick={() => loadAgents(page, pageSize, keyword)}>
              {t('刷新')}
            </Button>
          </div>
        }
        searchArea={
          <div className='flex flex-col md:flex-row items-center gap-2 w-full'>
            <Input
              size='small'
              placeholder={t('搜索用户名、显示名称或代理商名称')}
              value={keyword}
              onChange={setKeyword}
              style={{ width: isMobile ? '100%' : 320 }}
            />
            <Button size='small' type='tertiary' onClick={() => loadAgents(1, pageSize, keyword)}>
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
            loadAgents(nextPage, pageSize, keyword);
          },
          onPageSizeChange: (nextSize) => {
            setPage(1);
            setPageSize(nextSize);
            loadAgents(1, nextSize, keyword);
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
              <Button size='small' type='tertiary' onClick={() => loadAgents(page, pageSize, keyword)}>
                {t('重新加载')}
              </Button>
            </div>
          </div>
        ) : null}
        <Table
          className='grid-bordered-table'
          size='small'
          columns={columns}
          dataSource={agents}
          loading={loading}
          pagination={false}
          empty={<Empty description={keyword.trim() ? t('没有匹配的代理商') : t('暂无代理商数据')} />}
        />
      </CardPro>
    </div>
  );
};

export default AdminAgentsPageV2;
