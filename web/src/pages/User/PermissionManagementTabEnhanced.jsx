import React, { useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';

const getEmptyDescription = (t, keyword, defaultText, filteredText) =>
  keyword.trim() ? filteredText : defaultText;

const PermissionManagementTabEnhanced = ({ t, canBindProfile }) => {
  const [loading, setLoading] = useState(true);
  const [users, setUsers] = useState([]);
  const [profiles, setProfiles] = useState([]);
  const [listError, setListError] = useState('');
  const [profilesError, setProfilesError] = useState('');
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [editingUser, setEditingUser] = useState(null);
  const [selectedProfileId, setSelectedProfileId] = useState(0);
  const [saving, setSaving] = useState(false);

  const loadProfiles = async () => {
    setProfilesError('');
    try {
      const res = await API.get('/api/admin/permission/profiles?p=1&page_size=100');
      const { success, message, data } = res.data;
      if (!success) {
        setProfiles([]);
        setProfilesError(message || t('加载权限模板失败'));
        showError(message);
        return false;
      }
      setProfiles(data.items || []);
      return true;
    } catch (error) {
      setProfiles([]);
      setProfilesError(t('加载权限模板失败，请稍后重试'));
      showError(error);
      return false;
    }
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
      const res = await API.get(`/api/admin/permission/users?${params.toString()}`);
      const { success, message, data } = res.data;
      if (!success) {
        setUsers([]);
        setTotal(0);
        setListError(message || t('加载权限用户列表失败'));
        showError(message);
        return false;
      }
      setUsers((data.items || []).map((item) => ({ ...item, key: item.id })));
      setTotal(data.total || 0);
      setPage(data.page || nextPage);
      return true;
    } catch (error) {
      setUsers([]);
      setTotal(0);
      setListError(t('加载权限用户列表失败，请稍后重试'));
      showError(error);
      return false;
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProfiles();
    loadUsers(1, pageSize, '');
  }, []);

  const openBindModal = (user) => {
    setEditingUser(user);
    setSelectedProfileId(user.profile_id || 0);
  };

  const closeBindModal = () => {
    setEditingUser(null);
    setSelectedProfileId(0);
  };

  const handleBindProfile = async () => {
    if (!editingUser) return;
    setSaving(true);
    try {
      const res = await API.put(`/api/admin/permission/users/${editingUser.id}`, {
        profile_id: selectedProfileId || 0,
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('权限配置已更新'));
      closeBindModal();
      await loadUsers(page, pageSize, keyword);
    } finally {
      setSaving(false);
    }
  };

  const columns = useMemo(
    () => [
      { title: t('用户名'), dataIndex: 'username' },
      { title: t('显示名称'), dataIndex: 'display_name' },
      {
        title: t('身份'),
        dataIndex: 'user_type',
        render: (value) => <Tag color='blue'>{value || '-'}</Tag>,
      },
      {
        title: t('权限模板'),
        dataIndex: 'profile_name',
        render: (_, record) =>
          record.profile_name
            ? `${record.profile_name} (${record.profile_type})`
            : t('未绑定'),
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
        render: (_, record) =>
          canBindProfile ? (
            <Button
              size='small'
              theme='outline'
              onClick={() => openBindModal(record)}
            >
              {t('配置权限')}
            </Button>
          ) : null,
      },
    ],
    [canBindProfile, t],
  );

  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input
          placeholder={t('搜索用户名或显示名称')}
          value={keyword}
          onChange={setKeyword}
          style={{ width: 280 }}
        />
        <Button theme='solid' onClick={() => loadUsers(1, pageSize, keyword)}>
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
              t('暂无权限配置数据'),
              t('没有匹配的权限用户'),
            )}
          />
        }
      />

      <Modal
        title={t('配置权限模板')}
        visible={!!editingUser}
        onCancel={closeBindModal}
        onOk={handleBindProfile}
        confirmLoading={saving}
      >
        <div style={{ marginBottom: 12 }}>
          {editingUser
            ? `${editingUser.username} / ${editingUser.display_name || '-'}`
            : ''}
        </div>
        {profilesError ? (
          <Banner
            type='warning'
            description={profilesError}
            closeIcon={null}
            style={{ marginBottom: 12 }}
          />
        ) : null}
        <Select
          value={selectedProfileId}
          onChange={setSelectedProfileId}
          style={{ width: '100%' }}
          optionList={[
            { label: t('清空绑定'), value: 0 },
            ...profiles.map((profile) => ({
              label: `${profile.profile_name} (${profile.profile_type})`,
              value: profile.id,
            })),
          ]}
        />
      </Modal>
    </>
  );
};

export default PermissionManagementTabEnhanced;
