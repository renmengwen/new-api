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
import { useEffect, useState } from 'react';
import { API } from '../../helpers';
import { hasPermissionAction, isRootPermissionUser } from './permissionAccess.js';
import {
  isPermissionSidebarModuleAllowed,
  isPermissionSidebarSectionAllowed,
} from './sidebarPermissionSnapshot.js';

/**
 * 用户权限 Hook。
 * 从后端读取动作权限和侧边栏权限，避免仅依赖前端角色判断。
 */
export const useUserPermissions = () => {
  const [permissions, setPermissions] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const loadPermissions = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await API.get('/api/user/self');
      if (res.data.success) {
        setPermissions(res.data.data.permissions);
      } else {
        setError(res.data.message || '获取权限失败');
        console.error('获取权限失败:', res.data.message);
      }
    } catch (requestError) {
      setError('网络错误，请重试');
      console.error('加载用户权限异常:', requestError);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPermissions();
  }, []);

  const storedUser = (() => {
    try {
      const raw = localStorage.getItem('user');
      return raw ? JSON.parse(raw) : null;
    } catch (error) {
      return null;
    }
  })();

  const rootFallbackUser = permissions || storedUser
    ? {
        role: storedUser?.role,
        user_type: storedUser?.user_type,
      }
    : null;

  const hasSidebarSettingsPermission = () => {
    return permissions?.sidebar_settings === true || isRootPermissionUser(permissions, rootFallbackUser);
  };

  const hasActionPermission = (resourceKey, actionKey) => {
    return hasPermissionAction(permissions, resourceKey, actionKey, rootFallbackUser);
  };

  const hasAnyActionPermission = (requiredActions = []) => {
    return requiredActions.some(({ resource, action }) =>
      hasActionPermission(resource, action),
    );
  };

  const isSidebarSectionAllowed = (sectionKey) => {
    return isPermissionSidebarSectionAllowed(
      permissions?.sidebar_modules,
      sectionKey,
    );
  };

  const isSidebarModuleAllowed = (sectionKey, moduleKey) => {
    return isPermissionSidebarModuleAllowed(
      permissions?.sidebar_modules,
      sectionKey,
      moduleKey,
    );
  };

  const getAllowedSidebarSections = () => {
    if (!permissions?.sidebar_modules) return [];

    return Object.keys(permissions.sidebar_modules).filter((sectionKey) =>
      isSidebarSectionAllowed(sectionKey),
    );
  };

  const getAllowedSidebarModules = (sectionKey) => {
    if (!permissions?.sidebar_modules) return [];
    const sectionPerms = permissions.sidebar_modules[sectionKey];

    if (sectionPerms === false) return [];
    if (!sectionPerms || typeof sectionPerms !== 'object') return [];

    return Object.keys(sectionPerms).filter(
      (moduleKey) =>
        moduleKey !== 'enabled' && sectionPerms[moduleKey] === true,
    );
  };

  return {
    permissions,
    loading,
    error,
    actionPermissions: permissions?.actions || {},
    loadPermissions,
    hasSidebarSettingsPermission,
    hasActionPermission,
    hasAnyActionPermission,
    isSidebarSectionAllowed,
    isSidebarModuleAllowed,
    getAllowedSidebarSections,
    getAllowedSidebarModules,
  };
};

export default useUserPermissions;
