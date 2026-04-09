import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Checkbox,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import CardPro from '../../components/common/ui/CardPro';
import { API, createCardProPagination, showError, showSuccess, timestamp2string } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import {
  ADMIN_PERMISSION_RESOURCES,
  PERMISSION_PROFILE_TYPE_OPTIONS,
} from '../AdminConsole/permissionCatalogUiClean';

const { Text } = Typography;

const emptyTemplateForm = {
  profile_name: '',
  profile_type: 'admin',
  description: '',
  status: 1,
  selectedActions: {},
};

const sectionStyle = {
  border: '1px solid var(--semi-color-border)',
  borderRadius: 12,
  padding: 16,
  background: 'var(--semi-color-bg-0)',
};

const actionLinkStyle = {
  paddingLeft: 0,
  paddingRight: 0,
};

const buildTemplateItemsPayload = (selectedActions) => {
  const items = [];
  Object.entries(selectedActions).forEach(([resourceKey, actionMap]) => {
    Object.entries(actionMap || {}).forEach(([actionKey, checked]) => {
      if (checked) {
        items.push({ resource_key: resourceKey, action_key: actionKey, allowed: true });
      }
    });
  });
  return items;
};

const buildSelectedActions = (items = []) => {
  const nextState = {};
  items.forEach((item) => {
    if (!nextState[item.resource_key]) {
      nextState[item.resource_key] = {};
    }
    nextState[item.resource_key][item.action_key] = item.allowed === true;
  });
  return nextState;
};

const AdminPermissionTemplatesPageV2 = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const { loading: permissionLoading, hasActionPermission } = useUserPermissions();

  const canRead = hasActionPermission('permission_management', 'read');
  const canEdit = hasActionPermission('permission_management', 'bind_profile');

  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [templates, setTemplates] = useState([]);
  const [profileTypeFilter, setProfileTypeFilter] = useState('');
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [listError, setListError] = useState('');
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState(null);
  const [formState, setFormState] = useState(emptyTemplateForm);

  const closeModal = () => {
    setModalVisible(false);
    setEditingTemplate(null);
    setFormState(emptyTemplateForm);
  };

  const loadTemplates = async (
    nextPage = page,
    nextPageSize = pageSize,
    nextProfileType = profileTypeFilter,
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
      if (nextProfileType) {
        params.set('profile_type', nextProfileType);
      }
      if (nextKeyword.trim()) {
        params.set('keyword', nextKeyword.trim());
      }

      const res = await API.get(`/api/admin/permission-templates?${params.toString()}`);
      if (!res.data.success) {
        setTemplates([]);
        setTotal(0);
        setListError(res.data.message || t('加载权限模板失败'));
        return;
      }

      const data = res.data.data || {};
      setTemplates((data.items || []).map((item) => ({ ...item, key: item.id })));
      setPage(data.page || nextPage);
      setPageSize(data.page_size || nextPageSize);
      setTotal(data.total || 0);
    } catch (error) {
      setTemplates([]);
      setTotal(0);
      setListError(t('加载权限模板失败，请稍后重试'));
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  const openCreateModal = () => {
    setEditingTemplate(null);
    setFormState(emptyTemplateForm);
    setModalVisible(true);
  };

  const openEditModal = async (record) => {
    setEditingTemplate(record);
    setModalVisible(true);
    setSubmitting(true);
    try {
      const res = await API.get(`/api/admin/permission-templates/${record.id}`);
      if (!res.data.success) {
        showError(res.data.message || t('加载模板详情失败'));
        return;
      }

      const data = res.data.data || {};
      setFormState({
        profile_name: data.profile?.profile_name || '',
        profile_type: data.profile?.profile_type || 'admin',
        description: data.profile?.description || '',
        status: data.profile?.status ?? 1,
        selectedActions: buildSelectedActions(data.items || []),
      });
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleActionToggle = (resourceKey, actionKey, checked) => {
    setFormState((prev) => ({
      ...prev,
      selectedActions: {
        ...prev.selectedActions,
        [resourceKey]: {
          ...(prev.selectedActions[resourceKey] || {}),
          [actionKey]: checked,
        },
      },
    }));
  };

  const handleSubmit = async () => {
    if (!formState.profile_name.trim()) {
      showError(t('请输入模板名称'));
      return;
    }

    setSubmitting(true);
    try {
      const payload = {
        profile_name: formState.profile_name.trim(),
        profile_type: formState.profile_type,
        description: formState.description.trim(),
        status: formState.status,
        items: buildTemplateItemsPayload(formState.selectedActions),
      };

      const res = editingTemplate
        ? await API.put(`/api/admin/permission-templates/${editingTemplate.id}`, payload)
        : await API.post('/api/admin/permission-templates', payload);
      if (!res.data.success) {
        showError(res.data.message || t('保存权限模板失败'));
        return;
      }

      showSuccess(editingTemplate ? t('权限模板已更新') : t('权限模板已创建'));
      closeModal();
      await loadTemplates(page, pageSize, profileTypeFilter, keyword);
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const resetFilters = async () => {
    setKeyword('');
    setProfileTypeFilter('');
    await loadTemplates(1, pageSize, '', '');
  };

  useEffect(() => {
    if (!permissionLoading && canRead) {
      loadTemplates(1, pageSize, '', '');
    }
  }, [permissionLoading, canRead]);

  const columns = useMemo(
    () => [
      {
        title: t('模板名称'),
        dataIndex: 'profile_name',
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text strong>{record.profile_name}</Text>
            <Text type='tertiary' size='small'>
              {record.description || '-'}
            </Text>
          </div>
        ),
      },
      {
        title: t('适用对象'),
        dataIndex: 'profile_type',
        width: 120,
        render: (value) => <Tag color='blue'>{value}</Tag>,
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
        title: t('更新时间'),
        dataIndex: 'updated_at',
        width: 180,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('操作'),
        dataIndex: 'operate',
        width: 100,
        render: (_, record) =>
          canEdit ? (
            <Button
              size='small'
              theme='borderless'
              type='tertiary'
              style={actionLinkStyle}
              onClick={() => openEditModal(record)}
            >
              {t('编辑')}
            </Button>
          ) : null,
      },
    ],
    [canEdit, t],
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
        <Banner type='warning' closeIcon={null} description={t('你没有权限模板管理的查看权限')} />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <Modal
        title={editingTemplate ? t('编辑权限模板') : t('新增权限模板')}
        visible={modalVisible}
        onCancel={closeModal}
        onOk={handleSubmit}
        okText={editingTemplate ? t('保存变更') : t('确认创建')}
        cancelText={t('取消')}
        confirmLoading={submitting}
        width={880}
      >
        <Space vertical spacing='loose' style={{ width: '100%' }}>
          <div style={sectionStyle}>
            <div className='mb-3 flex flex-col gap-1'>
              <Text strong>{t('模板信息')}</Text>
              <Text type='tertiary'>{t('定义模板名称、适用对象和启停状态。')}</Text>
            </div>
            <div className='grid gap-3 md:grid-cols-2'>
              <Input
                placeholder={t('请输入模板名称')}
                value={formState.profile_name}
                onChange={(value) => setFormState((prev) => ({ ...prev, profile_name: value }))}
              />
              <Select
                value={formState.profile_type}
                optionList={PERMISSION_PROFILE_TYPE_OPTIONS}
                onChange={(value) => setFormState((prev) => ({ ...prev, profile_type: value }))}
              />
              <Select
                value={formState.status}
                optionList={[
                  { label: t('启用'), value: 1 },
                  { label: t('停用'), value: 0 },
                ]}
                onChange={(value) => setFormState((prev) => ({ ...prev, status: value }))}
              />
              <Input
                placeholder={t('请输入模板说明')}
                value={formState.description}
                onChange={(value) => setFormState((prev) => ({ ...prev, description: value }))}
              />
            </div>
          </div>
          <div style={sectionStyle}>
            <div className='mb-3 flex flex-col gap-1'>
              <Text strong>{t('权限矩阵')}</Text>
              <Text type='tertiary'>{t('勾选模板默认拥有的动作权限，作为账号绑定后的基线权限。')}</Text>
            </div>
            <div className='flex flex-col gap-3'>
              {ADMIN_PERMISSION_RESOURCES.map((resource) => (
                <div key={resource.resourceKey} className='rounded-xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3'>
                  <div className='mb-2 flex flex-col gap-1 md:flex-row md:items-center md:justify-between'>
                    <Text strong>{t(resource.label)}</Text>
                    <Text type='tertiary' size='small'>
                      {t('共 {{count}} 个动作', { count: resource.actions.length })}
                    </Text>
                  </div>
                  <Space wrap spacing={16}>
                    {resource.actions.map((action) => (
                      <Checkbox
                        key={`${resource.resourceKey}.${action.actionKey}`}
                        checked={Boolean(formState.selectedActions[resource.resourceKey]?.[action.actionKey])}
                        onChange={(event) =>
                          handleActionToggle(resource.resourceKey, action.actionKey, event.target.checked)
                        }
                      >
                        {t(action.label)}
                      </Checkbox>
                    ))}
                  </Space>
                </div>
              ))}
            </div>
          </div>
        </Space>
      </Modal>

      <CardPro
        type='type3'
        descriptionArea={
          <div className='flex flex-col gap-1'>
            <Text strong>{t('权限模板管理')}</Text>
            <Text type='tertiary'>{t('维护可复用的权限模板，供管理员、代理商和普通用户快速绑定。')}</Text>
          </div>
        }
        actionsArea={
          <div className='flex flex-wrap items-center gap-2'>
            {canEdit ? (
              <Button size='small' theme='light' type='primary' onClick={openCreateModal}>
                {t('新增模板')}
              </Button>
            ) : null}
            <Button size='small' type='tertiary' onClick={() => loadTemplates(page, pageSize, profileTypeFilter, keyword)}>
              {t('刷新')}
            </Button>
          </div>
        }
        searchArea={
          <div className='flex flex-col md:flex-row items-center gap-2 w-full'>
            <Input
              size='small'
              placeholder={t('搜索模板名称或说明')}
              value={keyword}
              onChange={setKeyword}
              style={{ width: isMobile ? '100%' : 260 }}
            />
            <Select
              value={profileTypeFilter}
              optionList={[{ label: t('全部对象'), value: '' }, ...PERMISSION_PROFILE_TYPE_OPTIONS]}
              onChange={setProfileTypeFilter}
              style={{ width: isMobile ? '100%' : 180 }}
            />
            <Button size='small' type='tertiary' onClick={() => loadTemplates(1, pageSize, profileTypeFilter, keyword)}>
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
            loadTemplates(nextPage, pageSize, profileTypeFilter, keyword);
          },
          onPageSizeChange: (nextSize) => {
            setPage(1);
            setPageSize(nextSize);
            loadTemplates(1, nextSize, profileTypeFilter, keyword);
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
              <Button size='small' type='tertiary' onClick={() => loadTemplates(page, pageSize, profileTypeFilter, keyword)}>
                {t('重新加载')}
              </Button>
            </div>
          </div>
        ) : null}
        <Table
          size='small'
          columns={columns}
          dataSource={templates}
          loading={loading}
          pagination={false}
          empty={<Empty description={profileTypeFilter || keyword ? t('没有匹配的权限模板') : t('暂无权限模板')} />}
        />
      </CardPro>
    </div>
  );
};

export default AdminPermissionTemplatesPageV2;
