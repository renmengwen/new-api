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

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';
import { normalizeUserPageData, toGroupOptions } from './useUsersData.helpers';
import { isUserDeleted } from '../../components/table/users/statusHelpers';

export const useUsersData = (mode = 'legacy') => {
  const { t } = useTranslation();
  const [compactMode, setCompactMode] = useTableCompactMode('users');
  const isManagedMode = mode === 'managed';

  // State management
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searching, setSearching] = useState(false);
  const [groupOptions, setGroupOptions] = useState([]);
  const [userCount, setUserCount] = useState(0);

  // Modal states
  const [showAddUser, setShowAddUser] = useState(false);
  const [showEditUser, setShowEditUser] = useState(false);
  const [editingUser, setEditingUser] = useState({
    id: undefined,
  });

  // Form initial values
  const formInitValues = {
    searchKeyword: '',
    searchGroup: '',
  };

  // Form API reference
  const [formApi, setFormApi] = useState(null);

  // Get form values helper function
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      searchGroup: formValues.searchGroup || '',
    };
  };

  // Set user format with key field
  const setUserFormat = (users) => {
    if (!Array.isArray(users)) {
      setUsers([]);
      return;
    }
    for (let i = 0; i < users.length; i++) {
      users[i].key = users[i].id;
    }
    setUsers(users);
  };

  // Load users data
  const loadUsers = async (startIdx, pageSize) => {
    setLoading(true);
    const endpoint = isManagedMode
      ? `/api/admin/users?p=${startIdx}&page_size=${pageSize}`
      : `/api/user/?p=${startIdx}&page_size=${pageSize}`;
    const res = await API.get(endpoint);
    const { success, message, data } = res.data;
    if (success) {
      const normalizedPageData = normalizeUserPageData(data, startIdx);
      setActivePage(normalizedPageData.page);
      setUserCount(normalizedPageData.total);
      setUserFormat(normalizedPageData.items);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  // Search users with keyword and group
  const searchUsers = async (
    startIdx,
    pageSize,
    searchKeyword = null,
    searchGroup = null,
  ) => {
    // If no parameters passed, get values from form
    if (searchKeyword === null || searchGroup === null) {
      const formValues = getFormValues();
      searchKeyword = formValues.searchKeyword;
      searchGroup = formValues.searchGroup;
    }

    if (searchKeyword === '' && searchGroup === '') {
      // If keyword is blank, load files instead
      await loadUsers(startIdx, pageSize);
      return;
    }
    setSearching(true);
    const endpoint = isManagedMode
      ? `/api/admin/users?keyword=${encodeURIComponent(searchKeyword)}&p=${startIdx}&page_size=${pageSize}`
      : `/api/user/search?keyword=${encodeURIComponent(searchKeyword)}&group=${encodeURIComponent(searchGroup)}&p=${startIdx}&page_size=${pageSize}`;
    const res = await API.get(endpoint);
    const { success, message, data } = res.data;
    if (success) {
      const normalizedPageData = normalizeUserPageData(data, startIdx);
      setActivePage(normalizedPageData.page);
      setUserCount(normalizedPageData.total);
      setUserFormat(normalizedPageData.items);
    } else {
      showError(message);
    }
    setSearching(false);
  };

  // Manage user operations (promote, demote, enable, disable, delete)
  const manageUser = async (userId, action) => {
    // Trigger loading state to force table re-render
    setLoading(true);

    if (isManagedMode) {
      let managedRes;
      if (action === 'enable' || action === 'disable') {
        managedRes = await API.post(`/api/admin/users/${userId}/${action}`);
      } else if (action === 'delete') {
        managedRes = await API.delete(`/api/admin/users/${userId}`);
      } else {
        showError(t('当前模式暂不支持此操作'));
        setLoading(false);
        return;
      }

      const { success, message, data } = managedRes.data;
      if (success) {
        showSuccess(t('操作已完成'));
        if (action === 'delete') {
          setUsers((prev) => prev.filter((u) => u.id !== userId));
          setUserCount((count) => Math.max(count - 1, 0));
        } else {
          setUsers((prev) =>
            prev.map((u) =>
              u.id === userId ? { ...u, status: data?.status ?? u.status } : u,
            ),
          );
        }
      } else {
        showError(message);
      }

      setLoading(false);
      return;
    }

    const res = await API.post('/api/user/manage', {
      id: userId,
      action,
    });

    const { success, message } = res.data;
    if (success) {
      showSuccess(t('操作已完成'));
      const user = res.data.data;

      // Create a new array and new object to ensure React detects changes
      const newUsers = users.map((u) => {
        if (u.id === userId) {
          if (action === 'delete') {
            return { ...u, DeletedAt: new Date() };
          }
          return { ...u, status: user.status, role: user.role };
        }
        return u;
      });

      setUsers(newUsers);
    } else {
      showError(message);
    }

    setLoading(false);
  };

  const resetUserPasskey = async (user) => {
    if (!user) {
      return;
    }
    if (isManagedMode) {
      showError(t('当前模式暂不支持此操作'));
      return;
    }
    try {
      const res = await API.delete(`/api/user/${user.id}/reset_passkey`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('Passkey 已重置'));
      } else {
        showError(message || t('鎿嶄綔澶辫触锛岃閲嶈瘯'));
      }
    } catch (error) {
      showError(t('鎿嶄綔澶辫触锛岃閲嶈瘯'));
    }
  };

  const resetUserTwoFA = async (user) => {
    if (!user) {
      return;
    }
    if (isManagedMode) {
      showError(t('当前模式暂不支持此操作'));
      return;
    }
    try {
      const res = await API.delete(`/api/user/${user.id}/2fa`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('2FA 已重置'));
      } else {
        showError(message || t('鎿嶄綔澶辫触锛岃閲嶈瘯'));
      }
    } catch (error) {
      showError(t('鎿嶄綔澶辫触锛岃閲嶈瘯'));
    }
  };

  // Handle page change
  const handlePageChange = (page) => {
    setActivePage(page);
    const { searchKeyword, searchGroup } = getFormValues();
    if (searchKeyword === '' && searchGroup === '') {
      loadUsers(page, pageSize).then();
    } else {
      searchUsers(page, pageSize, searchKeyword, searchGroup).then();
    }
  };

  // Handle page size change
  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    loadUsers(activePage, size)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  // Handle table row styling for disabled/deleted users
  const handleRow = (record, index) => {
    if (isUserDeleted(record) || record.status !== 1) {
      return {
        style: {
          background: 'var(--semi-color-disabled-border)',
        },
      };
    } else {
      return {};
    }
  };

  // Refresh data
  const refresh = async (page = activePage) => {
    const { searchKeyword, searchGroup } = getFormValues();
    if (searchKeyword === '' && searchGroup === '') {
      await loadUsers(page, pageSize);
    } else {
      await searchUsers(page, pageSize, searchKeyword, searchGroup);
    }
  };

  // Fetch groups data
  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      if (res === undefined) {
        return;
      }
      setGroupOptions(toGroupOptions(res.data));
    } catch (error) {
      showError(error.message);
    }
  };

  // Modal control functions
  const closeAddUser = () => {
    setShowAddUser(false);
  };

  const closeEditUser = () => {
    setShowEditUser(false);
    setEditingUser({
      id: undefined,
    });
  };

  const createUser = async (values) => {
    const endpoint = isManagedMode ? '/api/admin/users' : '/api/user/';
    const res = await API.post(endpoint, values);
    return res.data;
  };

  const loadUserDetail = async (userId) => {
    const endpoint = isManagedMode
      ? `/api/admin/users/${userId}`
      : `/api/user/${userId}`;
    const res = await API.get(endpoint);
    return res.data;
  };

  const updateUser = async (userId, values) => {
    if (isManagedMode) {
      const res = await API.put(`/api/admin/users/${userId}`, values);
      return res.data;
    }
    const payload = userId ? { ...values, id: parseInt(userId) } : values;
    const endpoint = userId ? '/api/user/' : '/api/user/self';
    const res = await API.put(endpoint, payload);
    return res.data;
  };

  // Initialize data on component mount
  useEffect(() => {
    loadUsers(1, pageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
    fetchGroups().then();
  }, [mode]);

  return {
    // Data state
    users,
    loading,
    activePage,
    pageSize,
    userCount,
    searching,
    groupOptions,

    // Modal state
    showAddUser,
    showEditUser,
    editingUser,
    setShowAddUser,
    setShowEditUser,
    setEditingUser,

    // Form state
    formInitValues,
    formApi,
    setFormApi,

    // UI state
    compactMode,
    setCompactMode,

    // Actions
    loadUsers,
    searchUsers,
    manageUser,
    resetUserPasskey,
    resetUserTwoFA,
    handlePageChange,
    handlePageSizeChange,
    handleRow,
    refresh,
    closeAddUser,
    closeEditUser,
    getFormValues,
    createUser,
    loadUserDetail,
    updateUser,
    mode,
    isManagedMode,

    // Translation
    t,
  };
};
