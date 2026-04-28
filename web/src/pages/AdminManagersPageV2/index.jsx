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
import {
  API,
  createCardProPagination,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import { toGroupOptions } from '../../hooks/users/useUsersData.helpers';

const { Text, Title } = Typography;

const emptyFormState = {
  username: '',
  password: '',
  display_name: '',
  email: '',
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

const isValidEmail = (value) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);

const AdminManagersPageV2 = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const { loading: permissionLoading, hasActionPermission } =
    useUserPermissions();

  const canRead = hasActionPermission('admin_management', 'read');
  const canCreate = hasActionPermission('admin_management', 'create');
  const canUpdate = hasActionPermission('admin_management', 'update');
  const canUpdateStatus = hasActionPermission(
    'admin_management',
    'update_status',
  );

  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [items, setItems] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [listError, setListError] = useState('');

  const [modalVisible, setModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState(null);
  const [formState, setFormState] = useState(emptyFormState);
  const [groupOptions, setGroupOptions] = useState([]);
  const [defaultGroup, setDefaultGroup] = useState('');

  const [detailVisible, setDetailVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState('');
  const [detailData, setDetailData] = useState(null);
  const [detailId, setDetailId] = useState(0);

  const closeModal = () => {
    setModalVisible(false);
    setEditingRecord(null);
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

  const loadManagers = async (
    nextPage = page,
    nextPageSize = pageSize,
    nextKeyword = keyword,
  ) => {
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

      const res = await API.get(`/api/admin/admin-users?${params.toString()}`);
      if (!res.data.success) {
        setItems([]);
        setTotal(0);
        setListError(res.data.message || t('加载管理员列表失败'));
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
      setListError(t('加载管理员列表失败，请稍后重试'));
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  const loadDetail = async (id) => {
    setDetailVisible(true);
    setDetailId(id);
    setDetailLoading(true);
    setDetailError('');
    setDetailData(null);
    try {
      const res = await API.get(`/api/admin/admin-users/${id}`);
      if (!res.data.success) {
        setDetailError(res.data.message || t('加载管理员详情失败'));
        return;
      }
      setDetailData(res.data.data || null);
    } catch (error) {
      setDetailError(t('加载管理员详情失败，请稍后重试'));
      showError(error);
    } finally {
      setDetailLoading(false);
    }
  };

  const openCreateModal = () => {
    setEditingRecord(null);
    setFormState({ ...emptyFormState, group: defaultGroup });
    setModalVisible(true);
  };

  const openEditModal = async (record) => {
    setEditingRecord(record);
    setModalVisible(true);
    setFormState({
      username: record.username || '',
      password: '',
      display_name: record.display_name || '',
      email: record.email || '',
      remark: record.remark || '',
      group: record.group || defaultGroup,
    });

    try {
      const res = await API.get(`/api/admin/admin-users/${record.id}`);
      if (!res.data.success) {
        showError(res.data.message || t('加载管理员详情失败'));
        return;
      }
      const data = res.data.data || {};
      setFormState({
        username: data.username || '',
        password: '',
        display_name: data.display_name || '',
        email: data.email || '',
        remark: data.remark || '',
        group: data.group || defaultGroup,
      });
    } catch (error) {
      showError(error);
    }
  };

  const handleSubmit = async () => {
    if (
      !editingRecord &&
      (!formState.username.trim() ||
        !formState.password.trim() ||
        !formState.email.trim())
    ) {
      showError(t('请填写用户名、初始密码和邮箱地址'));
      return;
    }
    if (editingRecord && !formState.email.trim()) {
      showError(t('请输入邮箱地址'));
      return;
    }
    if (!isValidEmail(formState.email.trim())) {
      showError(t('无效的邮箱地址'));
      return;
    }

    if (!editingRecord && !formState.group) {
      showError(t('请选择分组'));
      return;
    }

    setSubmitting(true);
    try {
      const payload = {
        username: formState.username.trim(),
        password: formState.password,
        display_name: formState.display_name.trim(),
        email: formState.email.trim(),
        remark: formState.remark.trim(),
        group: formState.group,
      };

      const res = editingRecord
        ? await API.put(`/api/admin/admin-users/${editingRecord.id}`, payload)
        : await API.post('/api/admin/admin-users', payload);

      if (!res.data.success) {
        showError(res.data.message || t('保存管理员失败'));
        return;
      }

      const currentEditingId = editingRecord?.id;
      showSuccess(editingRecord ? t('管理员资料已更新') : t('管理员已创建'));
      closeModal();
      await loadManagers(page, pageSize, keyword);
      if (detailVisible && currentEditingId && detailId === currentEditingId) {
        await loadDetail(currentEditingId);
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleStatusUpdate = async (record, enabled) => {
    try {
      const res = await API.post(
        `/api/admin/admin-users/${record.id}/${enabled ? 'enable' : 'disable'}`,
      );
      if (!res.data.success) {
        showError(res.data.message || t('更新管理员状态失败'));
        return;
      }

      showSuccess(enabled ? t('管理员已启用') : t('管理员已停用'));
      await loadManagers(page, pageSize, keyword);
      if (detailVisible && detailId === record.id) {
        await loadDetail(record.id);
      }
    } catch (error) {
      showError(error);
    }
  };

  const resetFilters = async () => {
    setKeyword('');
    await loadManagers(1, pageSize, '');
  };

  useEffect(() => {
    fetchGroups();
    if (!permissionLoading && canRead) {
      loadManagers(1, pageSize, '');
    }
  }, [permissionLoading, canRead]);

  const columns = useMemo(
    () => [
      {
        title: t('管理员'),
        dataIndex: 'username',
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text strong>{record.display_name || record.username}</Text>
            <Text type='tertiary' size='small'>
              {record.username}
            </Text>
            <Text type='tertiary' size='small'>
              {record.email || '-'}
            </Text>
          </div>
        ),
      },
      {
        title: t('备注'),
        dataIndex: 'remark',
        render: (value) => value || '-',
      },
      {
        title: t('最后活跃'),
        dataIndex: 'last_active_at',
        width: 180,
        render: (value) => (value ? timestamp2string(value) : '-'),
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
          <Space>
            <Button
              size='small'
              type='tertiary'
              onClick={() => loadDetail(record.id)}
            >
              {t('详情')}
            </Button>
            {canUpdate ? (
              <Button
                size='small'
                type='tertiary'
                onClick={() => openEditModal(record)}
              >
                {t('编辑')}
              </Button>
            ) : null}
            {canUpdateStatus ? (
              <Button
                size='small'
                theme={record.status === 1 ? 'solid' : 'light'}
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
        <Banner
          type='warning'
          closeIcon={null}
          description={t('你没有管理员管理的查看权限')}
        />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <Modal
        title={editingRecord ? t('编辑管理员') : t('新增管理员')}
        visible={modalVisible}
        onCancel={closeModal}
        footer={
          <ModalActionFooter
            onConfirm={handleSubmit}
            onCancel={closeModal}
            confirmText={editingRecord ? t('保存变更') : t('确认创建')}
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
              <Text type='tertiary'>
                {t('维护管理员账号标识、显示名称和密码信息。')}
              </Text>
            </div>
            <div className='grid gap-3 md:grid-cols-2'>
              <div style={fieldStyle}>
                <Text type='tertiary'>{t('登录用户名')}</Text>
                <Input
                  disabled={Boolean(editingRecord)}
                  placeholder={t('请输入登录用户名')}
                  value={formState.username}
                  onChange={(value) =>
                    setFormState((prev) => ({ ...prev, username: value }))
                  }
                />
              </div>
              <div style={fieldStyle}>
                <Text type='tertiary'>
                  {editingRecord ? t('重置密码') : t('初始密码')}
                </Text>
                <Input
                  mode='password'
                  placeholder={
                    editingRecord
                      ? t('如需重置密码请填写，留空则不修改')
                      : t('请输入初始密码，长度 8 到 20 位')
                  }
                  value={formState.password}
                  onChange={(value) =>
                    setFormState((prev) => ({ ...prev, password: value }))
                  }
                />
              </div>
              <div style={fieldStyle}>
                <Text type='tertiary'>{t('显示名称')}</Text>
                <Input
                  placeholder={t('请输入显示名称')}
                  value={formState.display_name}
                  onChange={(value) =>
                    setFormState((prev) => ({ ...prev, display_name: value }))
                  }
                />
              </div>
              <div style={fieldStyle}>
                <Text type='tertiary'>{t('邮箱地址')}</Text>
                <Input
                  type='email'
                  placeholder={t('请输入邮箱地址')}
                  value={formState.email}
                  onChange={(value) =>
                    setFormState((prev) => ({ ...prev, email: value }))
                  }
                />
              </div>
              {!editingRecord ? (
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
              <Text type='tertiary'>
                {t('补充岗位职责、权限说明或交接备注，仅后台可见。')}
              </Text>
            </div>
            <TextArea
              rows={3}
              placeholder={t('请输入备注信息')}
              value={formState.remark}
              onChange={(value) =>
                setFormState((prev) => ({ ...prev, remark: value }))
              }
            />
          </div>
        </div>
      </Modal>

      <Modal
        title={t('管理员详情')}
        visible={detailVisible}
        footer={null}
        width={720}
        onCancel={() => {
          setDetailVisible(false);
          setDetailData(null);
          setDetailError('');
          setDetailId(0);
        }}
      >
        {detailLoading ? <Text>{t('加载中')}</Text> : null}
        {!detailLoading && detailError ? (
          <div className='flex flex-col gap-2'>
            <Banner type='warning' closeIcon={null} description={detailError} />
            <div>
              <Button
                size='small'
                type='tertiary'
                onClick={() => loadDetail(detailId)}
              >
                {t('重试')}
              </Button>
            </div>
          </div>
        ) : null}
        {!detailLoading && !detailError && !detailData ? (
          <Empty description={t('暂无详情数据')} />
        ) : null}
        {!detailLoading && !detailError && detailData ? (
          <div className='flex flex-col gap-4'>
            <div style={sectionStyle}>
              <div className='flex items-start justify-between gap-4'>
                <div className='flex flex-col gap-1'>
                  <Title heading={6} style={{ margin: 0 }}>
                    {detailData.display_name || detailData.username}
                  </Title>
                  <Text type='tertiary'>{detailData.username}</Text>
                </div>
                <Tag
                  color={detailData.status === 1 ? 'green' : 'red'}
                  shape='circle'
                >
                  {detailData.status === 1 ? t('启用') : t('停用')}
                </Tag>
              </div>
            </div>
            <div style={mergedSectionStyle}>
              <div className='mb-3 flex flex-col gap-1'>
                <Text strong>{t('账号资料')}</Text>
                <Text type='tertiary'>
                  {t('查看管理员账号的显示名称、最近活跃时间和备注信息。')}
                </Text>
              </div>
              <Descriptions
                data={[
                  { key: t('登录用户名'), value: detailData.username || '-' },
                  { key: t('显示名称'), value: detailData.display_name || '-' },
                  { key: t('邮箱地址'), value: detailData.email || '-' },
                  {
                    key: t('最后活跃'),
                    value: detailData.last_active_at
                      ? timestamp2string(detailData.last_active_at)
                      : '-',
                  },
                ]}
                columns={2}
              />
              <div style={sectionBlockStyle}>
                <div className='mb-3 flex flex-col gap-1'>
                  <Text strong>{t('备注信息')}</Text>
                </div>
                <Text style={{ whiteSpace: 'pre-wrap' }}>
                  {detailData.remark || '-'}
                </Text>
              </div>
            </div>
          </div>
        ) : null}
      </Modal>

      <CardPro
        type='type3'
        descriptionArea={
          <div className='flex flex-col gap-1'>
            <Text strong>{t('管理员管理')}</Text>
            <Text type='tertiary'>
              {t(
                '维护后台管理员账号、状态和基础资料，权限配置请前往用户权限管理。',
              )}
            </Text>
          </div>
        }
        actionsArea={
          <div className='flex flex-wrap items-center gap-2'>
            {canCreate ? (
              <Button
                size='small'
                theme='light'
                type='primary'
                onClick={openCreateModal}
              >
                {t('新增管理员')}
              </Button>
            ) : null}
            <Button
              size='small'
              type='tertiary'
              onClick={() => loadManagers(page, pageSize, keyword)}
            >
              {t('刷新')}
            </Button>
          </div>
        }
        searchArea={
          <div className='flex flex-col md:flex-row items-center gap-2 w-full'>
            <Input
              size='small'
              placeholder={t('搜索用户名、显示名称、邮箱或备注')}
              value={keyword}
              onChange={setKeyword}
              style={{ width: isMobile ? '100%' : 320 }}
            />
            <Button
              size='small'
              type='tertiary'
              onClick={() => loadManagers(1, pageSize, keyword)}
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
            setPage(nextPage);
            loadManagers(nextPage, pageSize, keyword);
          },
          onPageSizeChange: (nextSize) => {
            setPage(1);
            setPageSize(nextSize);
            loadManagers(1, nextSize, keyword);
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
              <Button
                size='small'
                type='tertiary'
                onClick={() => loadManagers(page, pageSize, keyword)}
              >
                {t('重新加载')}
              </Button>
            </div>
          </div>
        ) : null}
        <Table
          className='grid-bordered-table'
          size='small'
          loading={loading}
          columns={columns}
          dataSource={items}
          pagination={false}
          empty={
            <Empty
              description={
                keyword.trim() ? t('没有匹配的管理员') : t('暂无管理员数据')
              }
            />
          }
        />
      </CardPro>
    </div>
  );
};

export default AdminManagersPageV2;
