import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Empty,
  Input,
  Modal,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import CardPro from '../../components/common/ui/CardPro';
import { API, createCardProPagination, showError, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import {
  ACTION_OVERRIDE_OPTIONS,
  ADMIN_DATA_SCOPE_RESOURCES,
  ADMIN_MENU_OPTIONS,
  ADMIN_PERMISSION_RESOURCES,
  DATA_SCOPE_OPTIONS,
  MENU_OVERRIDE_OPTIONS,
  USER_PERMISSION_TYPE_OPTIONS,
} from '../AdminConsole/permissionCatalogUiClean';

const { Text } = Typography;

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

const buildActionOverrideMap = (items = []) => {
  const nextMap = {};
  items.forEach((item) => {
    nextMap[`${item.resource_key}.${item.action_key}`] = item.effect;
  });
  return nextMap;
};

const buildMenuOverrideMap = (items = []) => {
  const nextMap = {};
  items.forEach((item) => {
    nextMap[`${item.section_key}.${item.module_key}`] = item.effect;
  });
  return nextMap;
};

const buildDataScopeOverrideMap = (items = []) => {
  const nextMap = {};
  items.forEach((item) => {
    nextMap[item.resource_key] = {
      scopeType: item.scope_type,
      scopeValue: Array.isArray(item.scope_value) ? item.scope_value.join(',') : '',
    };
  });
  return nextMap;
};

const buildActionOverridePayload = (overrideMap) =>
  Object.entries(overrideMap)
    .filter(([, effect]) => effect === 'allow' || effect === 'deny')
    .map(([key, effect]) => {
      const [resource_key, action_key] = key.split('.');
      return { resource_key, action_key, effect };
    });

const buildMenuOverridePayload = (overrideMap) =>
  Object.entries(overrideMap)
    .filter(([, effect]) => effect === 'show' || effect === 'hide')
    .map(([key, effect]) => {
      const [section_key, module_key] = key.split('.');
      return { section_key, module_key, effect };
    });

const buildDataScopeOverridePayload = (overrideMap) =>
  Object.entries(overrideMap)
    .filter(([, value]) => value?.scopeType && value.scopeType !== 'inherit')
    .map(([resource_key, value]) => ({
      resource_key,
      scope_type: value.scopeType,
      scope_value:
        value.scopeType === 'assigned'
          ? String(value.scopeValue || '')
              .split(',')
              .map((item) => Number(item.trim()))
              .filter((item) => Number.isInteger(item) && item > 0)
          : [],
    }));

const getEffectiveScopeLabel = (value, t) => {
  switch (value) {
    case 'all':
      return t('全部用户');
    case 'self':
      return t('仅自己');
    case 'agent_only':
      return t('仅绑定用户');
    case 'assigned':
      return t('指定用户');
    default:
      return t('默认范围');
  }
};

const AdminUserPermissionsPageV3 = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const { loading: permissionLoading, hasActionPermission } = useUserPermissions();

  const canRead = hasActionPermission('permission_management', 'read');
  const canBind = hasActionPermission('permission_management', 'bind_profile');

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [items, setItems] = useState([]);
  const [templates, setTemplates] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [userType, setUserType] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [listError, setListError] = useState('');
  const [modalVisible, setModalVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState('');
  const [editingUser, setEditingUser] = useState(null);
  const [selectedProfileId, setSelectedProfileId] = useState(0);
  const [actionOverrideMap, setActionOverrideMap] = useState({});
  const [menuOverrideMap, setMenuOverrideMap] = useState({});
  const [dataScopeOverrideMap, setDataScopeOverrideMap] = useState({});
  const [effectiveDataScopes, setEffectiveDataScopes] = useState({});

  const closeModal = () => {
    setModalVisible(false);
    setEditingUser(null);
    setDetailError('');
    setSelectedProfileId(0);
    setActionOverrideMap({});
    setMenuOverrideMap({});
    setDataScopeOverrideMap({});
    setEffectiveDataScopes({});
  };

  const loadTemplates = async () => {
    try {
      const res = await API.get('/api/admin/permission-templates?p=1&page_size=100');
      if (res.data.success) {
        setTemplates(res.data.data?.items || []);
      }
    } catch (error) {
      showError(error);
    }
  };

  const loadTargets = async (nextPage = page, nextPageSize = pageSize, nextKeyword = keyword, nextUserType = userType) => {
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
      if (nextUserType) {
        params.set('user_type', nextUserType);
      }

      const res = await API.get(`/api/admin/user-permissions/users?${params.toString()}`);
      if (!res.data.success) {
        setItems([]);
        setTotal(0);
        setListError(res.data.message || t('加载用户权限列表失败'));
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
      setListError(t('加载用户权限列表失败，请稍后重试'));
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  const openModal = async (record) => {
    setModalVisible(true);
    setEditingUser(record);
    setDetailLoading(true);
    setDetailError('');
    try {
      const res = await API.get(`/api/admin/user-permissions/users/${record.id}`);
      if (!res.data.success) {
        setDetailError(res.data.message || t('加载权限详情失败'));
        return;
      }

      const data = res.data.data || {};
      const resolvedUserType = data.user?.user_type || record.user_type || '';
      setEditingUser((prev) =>
        prev
          ? {
              ...prev,
              user_type: resolvedUserType,
            }
          : record,
      );
      setSelectedProfileId(data.binding?.profile_id || 0);
      setActionOverrideMap(buildActionOverrideMap(data.action_overrides || []));
      setMenuOverrideMap(buildMenuOverrideMap(data.menu_overrides || []));
      setDataScopeOverrideMap(buildDataScopeOverrideMap(data.data_scope_overrides || []));
      setEffectiveDataScopes(data.effective_data_scopes || {});
    } catch (error) {
      setDetailError(t('加载权限详情失败，请稍后重试'));
      showError(error);
    } finally {
      setDetailLoading(false);
    }
  };

  const handleSave = async () => {
    if (!editingUser) {
      return;
    }

    setSaving(true);
    try {
      const bindRes = await API.put(`/api/admin/user-permissions/users/${editingUser.id}/template`, {
        profile_id: selectedProfileId || 0,
      });
      if (!bindRes.data.success) {
        showError(bindRes.data.message || t('保存模板绑定失败'));
        return;
      }

      const overrideRes = await API.put(`/api/admin/user-permissions/users/${editingUser.id}/overrides`, {
        action_overrides: buildActionOverridePayload(actionOverrideMap),
        menu_overrides: buildMenuOverridePayload(menuOverrideMap),
        data_scope_overrides: buildDataScopeOverridePayload(dataScopeOverrideMap),
      });
      if (!overrideRes.data.success) {
        showError(overrideRes.data.message || t('保存权限覆盖失败'));
        return;
      }

      showSuccess(t('用户权限已更新'));
      closeModal();
      await loadTargets(page, pageSize, keyword, userType);
    } catch (error) {
      showError(error);
    } finally {
      setSaving(false);
    }
  };

  const resetFilters = async () => {
    setKeyword('');
    setUserType('');
    await loadTargets(1, pageSize, '', '');
  };

  useEffect(() => {
    if (!permissionLoading && canRead) {
      loadTemplates();
      loadTargets(1, pageSize, '', '');
    }
  }, [permissionLoading, canRead]);

  const columns = useMemo(
    () => [
      {
        title: t('对象'),
        dataIndex: 'username',
        render: (_, record) => (
          <div className='flex flex-col gap-1'>
            <Text strong>{record.display_name || record.username}</Text>
            <Text type='tertiary' size='small'>
              {record.username}
            </Text>
          </div>
        ),
      },
      {
        title: t('类型'),
        dataIndex: 'user_type',
        width: 120,
        render: (value) => <Tag color='blue'>{value || '-'}</Tag>,
      },
      {
        title: t('绑定模板'),
        dataIndex: 'profile_name',
        render: (_, record) =>
          record.profile_name ? `${record.profile_name} (${record.profile_type})` : t('未绑定模板'),
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
        width: 110,
        render: (_, record) =>
          canBind ? (
            <Button
              size='small'
              theme='borderless'
              type='tertiary'
              style={actionLinkStyle}
              onClick={() => openModal(record)}
            >
              {t('配置权限')}
            </Button>
          ) : null,
      },
    ],
    [canBind, t],
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
        <Banner type='warning' closeIcon={null} description={t('你没有用户权限管理的查看权限')} />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <Modal
        title={editingUser ? `${t('用户权限管理')} · ${editingUser.display_name || editingUser.username}` : t('用户权限管理')}
        visible={modalVisible}
        onCancel={closeModal}
        onOk={handleSave}
        okText={t('保存权限')}
        cancelText={t('取消')}
        confirmLoading={saving}
        width={980}
      >
        {detailLoading ? <Text>{t('加载中')}</Text> : null}
        {!detailLoading && detailError ? <Banner type='warning' closeIcon={null} description={detailError} /> : null}
        {!detailLoading && !detailError ? (
          <Space vertical spacing='loose' style={{ width: '100%' }}>
            <div style={sectionStyle}>
              <div className='mb-3 flex flex-col gap-1'>
                <Text strong>{t('模板绑定')}</Text>
                <Text type='tertiary'>{t('先绑定一个权限模板，再对该账号做单独覆盖。')}</Text>
              </div>
              <Select
                value={selectedProfileId}
                optionList={[
                  { label: t('不绑定模板'), value: 0 },
                  ...templates
                    .filter(
                      (template) =>
                        !editingUser ||
                        template.profile_type === editingUser.user_type ||
                        (editingUser.user_type === 'root' && template.profile_type === 'admin'),
                    )
                    .map((template) => ({
                      label: `${template.profile_name} (${template.profile_type})`,
                      value: template.id,
                    })),
                ]}
                onChange={setSelectedProfileId}
                style={{ width: '100%' }}
              />
            </div>

            <div style={sectionStyle}>
              <div className='mb-3 flex flex-col gap-1'>
                <Text strong>{t('动作权限覆盖')}</Text>
                <Text type='tertiary'>{t('对模板能力做单账号级别的允许或禁止。')}</Text>
              </div>
              <div className='flex flex-col gap-3'>
                {ADMIN_PERMISSION_RESOURCES.map((resource) => (
                  <div key={resource.resourceKey} className='rounded-xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3'>
                    <div className='mb-2'>
                      <Text strong>{t(resource.label)}</Text>
                    </div>
                    <div className='flex flex-col gap-2'>
                      {resource.actions.map((action) => {
                        const key = `${resource.resourceKey}.${action.actionKey}`;
                        return (
                          <div key={key} className='flex flex-col gap-2 md:flex-row md:items-center md:justify-between'>
                            <Text>{t(action.label)}</Text>
                            <Radio.Group
                              type='button'
                              value={actionOverrideMap[key] || 'inherit'}
                              onChange={(event) =>
                                setActionOverrideMap((prev) => ({
                                  ...prev,
                                  [key]: event.target.value,
                                }))
                              }
                              options={ACTION_OVERRIDE_OPTIONS.map((option) => ({
                                label: t(option.label),
                                value: option.value,
                              }))}
                            />
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div style={sectionStyle}>
              <div className='mb-3 flex flex-col gap-1'>
                <Text strong>{t('菜单可见性覆盖')}</Text>
                <Text type='tertiary'>{t('决定该账号在左侧导航中是否看得到对应菜单。')}</Text>
              </div>
              <div className='flex flex-col gap-3'>
                {ADMIN_MENU_OPTIONS.map((menu) => {
                  const key = `${menu.sectionKey}.${menu.moduleKey}`;
                  return (
                    <div key={key} className='flex flex-col gap-2 rounded-xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3 md:flex-row md:items-center md:justify-between'>
                      <Text>{t(menu.label)}</Text>
                      <Radio.Group
                        type='button'
                        value={menuOverrideMap[key] || 'inherit'}
                        onChange={(event) =>
                          setMenuOverrideMap((prev) => ({
                            ...prev,
                            [key]: event.target.value,
                          }))
                        }
                        options={MENU_OVERRIDE_OPTIONS.map((option) => ({
                          label: t(option.label),
                          value: option.value,
                        }))}
                      />
                    </div>
                  );
                })}
              </div>
            </div>

            <div style={sectionStyle}>
              <div className='mb-3 flex flex-col gap-1'>
                <Text strong>{t('数据范围覆盖')}</Text>
                <Text type='tertiary'>{t('控制该账号能够查看或操作哪些用户数据。')}</Text>
              </div>
              <div className='flex flex-col gap-3'>
                {ADMIN_DATA_SCOPE_RESOURCES.map((resource) => {
                  const currentScope = dataScopeOverrideMap[resource.resourceKey] || {
                    scopeType: 'inherit',
                    scopeValue: '',
                  };
                  return (
                    <div key={resource.resourceKey} className='rounded-xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3'>
                      <div className='mb-2 flex flex-col gap-1 md:flex-row md:items-center md:justify-between'>
                        <Text strong>{t(resource.label)}</Text>
                        <Tag color='grey'>
                          {t('当前生效')}：{getEffectiveScopeLabel(effectiveDataScopes[resource.resourceKey], t)}
                        </Tag>
                      </div>
                      <div className='flex flex-col gap-2'>
                        <Radio.Group
                          type='button'
                          value={currentScope.scopeType || 'inherit'}
                          onChange={(event) =>
                            setDataScopeOverrideMap((prev) => ({
                              ...prev,
                              [resource.resourceKey]: {
                                scopeType: event.target.value,
                                scopeValue:
                                  event.target.value === 'assigned'
                                    ? prev[resource.resourceKey]?.scopeValue || ''
                                    : '',
                              },
                            }))
                          }
                          options={DATA_SCOPE_OPTIONS.map((option) => ({
                            label: t(option.label),
                            value: option.value,
                          }))}
                        />
                        {currentScope.scopeType === 'assigned' ? (
                          <Input
                            placeholder={t('请输入用户 ID，多个用户请使用英文逗号分隔')}
                            value={currentScope.scopeValue || ''}
                            onChange={(value) =>
                              setDataScopeOverrideMap((prev) => ({
                                ...prev,
                                [resource.resourceKey]: {
                                  scopeType: 'assigned',
                                  scopeValue: value,
                                },
                              }))
                            }
                          />
                        ) : null}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </Space>
        ) : null}
      </Modal>

      <CardPro
        type='type3'
        descriptionArea={
          <div className='flex flex-col gap-1'>
            <Text strong>{t('用户权限管理')}</Text>
            <Text type='tertiary'>{t('对账号做模板绑定、动作覆盖、菜单覆盖和数据范围覆盖。')}</Text>
          </div>
        }
        actionsArea={
          <div className='flex flex-wrap items-center gap-2'>
            <Button size='small' type='tertiary' onClick={() => loadTargets(page, pageSize, keyword, userType)}>
              {t('刷新')}
            </Button>
          </div>
        }
        searchArea={
          <div className='flex flex-col md:flex-row items-center gap-2 w-full'>
            <Input
              size='small'
              placeholder={t('搜索用户名、显示名称或邮箱')}
              value={keyword}
              onChange={setKeyword}
              style={{ width: isMobile ? '100%' : 280 }}
            />
            <Select
              value={userType}
              optionList={USER_PERMISSION_TYPE_OPTIONS}
              onChange={setUserType}
              style={{ width: isMobile ? '100%' : 180 }}
            />
            <Button size='small' type='tertiary' onClick={() => loadTargets(1, pageSize, keyword, userType)}>
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
            loadTargets(nextPage, pageSize, keyword, userType);
          },
          onPageSizeChange: (nextSize) => {
            setPage(1);
            setPageSize(nextSize);
            loadTargets(1, nextSize, keyword, userType);
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
              <Button size='small' type='tertiary' onClick={() => loadTargets(page, pageSize, keyword, userType)}>
                {t('重新加载')}
              </Button>
            </div>
          </div>
        ) : null}
        <Table
          size='small'
          columns={columns}
          dataSource={items}
          loading={loading}
          pagination={false}
          empty={<Empty description={keyword || userType ? t('没有匹配的权限对象') : t('暂无权限对象')} />}
        />
      </CardPro>
    </div>
  );
};

export default AdminUserPermissionsPageV3;
