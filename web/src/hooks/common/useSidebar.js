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

import { useState, useEffect, useMemo, useContext, useRef } from 'react';
import { StatusContext } from '../../context/Status';
import { API } from '../../helpers';
import { buildFallbackUserSidebarConfig } from './sidebarFallback';
import {
  buildFinalSidebarConfig,
  normalizeUserSidebarConfig,
} from './sidebarPermissionSnapshot';

// 鍒涘缓涓€涓叏灞€浜嬩欢绯荤粺鏉ュ悓姝ユ墍鏈塽seSidebar瀹炰緥
const sidebarEventTarget = new EventTarget();
const SIDEBAR_REFRESH_EVENT = 'sidebar-refresh';

export const DEFAULT_ADMIN_CONFIG = {
  chat: {
    enabled: true,
    playground: true,
    chat: true,
  },
  console: {
    enabled: true,
    detail: true,
    token: true,
    log: true,
    midjourney: true,
    task: true,
  },
  personal: {
    enabled: true,
    topup: true,
    personal: true,
  },
  admin: {
    enabled: true,
    channel: true,
    models: true,
    deployment: true,
    redemption: true,
    user: true,
    'admin-users': true,
    agents: true,
    'permission-templates': true,
    'user-permissions': true,
    'quota-ledger': true,
    'audit-logs': true,
    subscription: true,
    setting: true,
  },
};

const deepClone = (value) => JSON.parse(JSON.stringify(value));

const getStoredUser = () => {
  try {
    const raw = localStorage.getItem('user');
    return raw ? JSON.parse(raw) : {};
  } catch (error) {
    return {};
  }
};

export const mergeAdminConfig = (savedConfig) => {
  const merged = deepClone(DEFAULT_ADMIN_CONFIG);
  if (!savedConfig || typeof savedConfig !== 'object') return merged;

  for (const [sectionKey, sectionConfig] of Object.entries(savedConfig)) {
    if (!sectionConfig || typeof sectionConfig !== 'object') continue;

    if (!merged[sectionKey]) {
      merged[sectionKey] = { ...sectionConfig };
      continue;
    }

    merged[sectionKey] = { ...merged[sectionKey], ...sectionConfig };
  }

  return merged;
};

export const useSidebar = () => {
  const [statusState] = useContext(StatusContext);
  const [userConfig, setUserConfig] = useState(null);
  const [strictPermissionSnapshot, setStrictPermissionSnapshot] = useState(false);
  const [loading, setLoading] = useState(true);
  const instanceIdRef = useRef(null);
  const hasLoadedOnceRef = useRef(false);

  if (!instanceIdRef.current) {
    const randomPart = Math.random().toString(16).slice(2);
    instanceIdRef.current = `sidebar-${Date.now()}-${randomPart}`;
  }

  // 鑾峰彇绠＄悊鍛橀厤缃?
  const adminConfig = useMemo(() => {
    if (statusState?.status?.SidebarModulesAdmin) {
      try {
        const config = JSON.parse(statusState.status.SidebarModulesAdmin);
        return mergeAdminConfig(config);
      } catch (error) {
        return mergeAdminConfig(null);
      }
    }
    return mergeAdminConfig(null);
  }, [statusState?.status?.SidebarModulesAdmin]);

  // 鍔犺浇鐢ㄦ埛閰嶇疆鐨勯€氱敤鏂规硶
  const loadUserConfig = async ({ withLoading } = {}) => {
    const shouldShowLoader =
      typeof withLoading === 'boolean'
        ? withLoading
        : !hasLoadedOnceRef.current;

    try {
      if (shouldShowLoader) {
        setLoading(true);
      }

      const res = await API.get('/api/user/self');
      const responseData = res.data?.data ?? {};
      const permissionsSidebarModules = Object.prototype.hasOwnProperty.call(
        responseData?.permissions ?? {},
        'sidebar_modules',
      )
        ? responseData.permissions.sidebar_modules
        : undefined;
      const rawSidebarModules =
        permissionsSidebarModules ?? responseData.sidebar_modules;
      const shouldUseFallbackConfig =
        rawSidebarModules === false ||
        rawSidebarModules === undefined ||
        rawSidebarModules === null ||
        (typeof rawSidebarModules === 'object' &&
          !Array.isArray(rawSidebarModules) &&
          Object.keys(rawSidebarModules).length === 0);
      if (res.data.success && !shouldUseFallbackConfig) {
        let config;
        // 妫€鏌idebar_modules鏄瓧绗︿覆杩樻槸瀵硅薄
        if (typeof rawSidebarModules === 'string') {
          config = JSON.parse(rawSidebarModules);
        } else {
          config = rawSidebarModules;
        }
        setStrictPermissionSnapshot(permissionsSidebarModules !== undefined);
        setUserConfig(normalizeUserSidebarConfig(config));
      } else {
        // 褰撶敤鎴锋病鏈夐厤缃椂锛岀敓鎴愪竴涓熀浜庣鐞嗗憳閰嶇疆鐨勯粯璁ょ敤鎴烽厤缃?        // 杩欐牱鍙互纭繚鏉冮檺鎺у埗姝ｇ‘鐢熸晥
        const defaultUserConfig = buildFallbackUserSidebarConfig(
          adminConfig,
          responseData,
        );
        Object.keys({}).forEach((sectionKey) => {
          if (adminConfig[sectionKey]?.enabled) {
            defaultUserConfig[sectionKey] = { enabled: true };
            // 涓烘瘡涓鐞嗗憳鍏佽鐨勬ā鍧楄缃粯璁ゅ€间负true
            Object.keys(adminConfig[sectionKey]).forEach((moduleKey) => {
              if (
                moduleKey !== 'enabled' &&
                adminConfig[sectionKey][moduleKey]
              ) {
                defaultUserConfig[sectionKey][moduleKey] = true;
              }
            });
          }
        });
        setStrictPermissionSnapshot(false);
        setUserConfig(normalizeUserSidebarConfig(defaultUserConfig));
      }
    } catch (error) {
      // 鍑洪敊鏃朵篃鐢熸垚榛樿閰嶇疆锛岃€屼笉鏄缃负绌哄璞?
      const defaultUserConfig = buildFallbackUserSidebarConfig(
        adminConfig,
        getStoredUser(),
      );
      Object.keys({}).forEach((sectionKey) => {
        if (adminConfig[sectionKey]?.enabled) {
          defaultUserConfig[sectionKey] = { enabled: true };
          Object.keys(adminConfig[sectionKey]).forEach((moduleKey) => {
            if (moduleKey !== 'enabled' && adminConfig[sectionKey][moduleKey]) {
              defaultUserConfig[sectionKey][moduleKey] = true;
            }
          });
        }
      });
      setStrictPermissionSnapshot(false);
      setUserConfig(normalizeUserSidebarConfig(defaultUserConfig));
    } finally {
      if (shouldShowLoader) {
        setLoading(false);
      }
      hasLoadedOnceRef.current = true;
    }
  };

  // 鍒锋柊鐢ㄦ埛閰嶇疆鐨勬柟娉曪紙渚涘閮ㄨ皟鐢級
  const refreshUserConfig = async () => {
    if (Object.keys(adminConfig).length > 0) {
      await loadUserConfig({ withLoading: false });
    }

    // 瑙﹀彂鍏ㄥ眬鍒锋柊浜嬩欢锛岄€氱煡鎵€鏈塽seSidebar瀹炰緥鏇存柊
    sidebarEventTarget.dispatchEvent(
      new CustomEvent(SIDEBAR_REFRESH_EVENT, {
        detail: { sourceId: instanceIdRef.current, skipLoader: true },
      }),
    );
  };

  // 鍔犺浇鐢ㄦ埛閰嶇疆
  useEffect(() => {
    // 鍙湁褰撶鐞嗗憳閰嶇疆鍔犺浇瀹屾垚鍚庢墠鍔犺浇鐢ㄦ埛閰嶇疆
    if (Object.keys(adminConfig).length > 0) {
      loadUserConfig();
    }
  }, [adminConfig]);

  // 鐩戝惉鍏ㄥ眬鍒锋柊浜嬩欢
  useEffect(() => {
    const handleRefresh = (event) => {
      if (event?.detail?.sourceId === instanceIdRef.current) {
        return;
      }

      if (Object.keys(adminConfig).length > 0) {
        loadUserConfig({
          withLoading: event?.detail?.skipLoader ? false : undefined,
        });
      }
    };

    sidebarEventTarget.addEventListener(SIDEBAR_REFRESH_EVENT, handleRefresh);

    return () => {
      sidebarEventTarget.removeEventListener(
        SIDEBAR_REFRESH_EVENT,
        handleRefresh,
      );
    };
  }, [adminConfig]);

  // 璁＄畻鏈€缁堢殑鏄剧ず閰嶇疆
  const finalConfig = useMemo(
    () =>
      buildFinalSidebarConfig(adminConfig, userConfig, {
        strictSnapshot: strictPermissionSnapshot,
      }),
    [adminConfig, strictPermissionSnapshot, userConfig],
  );

  // 妫€鏌ョ壒瀹氬姛鑳芥槸鍚﹀簲璇ユ樉绀?
  const isModuleVisible = (sectionKey, moduleKey = null) => {
    if (moduleKey) {
      return finalConfig[sectionKey]?.[moduleKey] === true;
    } else {
      return finalConfig[sectionKey]?.enabled === true;
    }
  };

  // 妫€鏌ュ尯鍩熸槸鍚︽湁浠讳綍鍙鐨勫姛鑳?
  const hasSectionVisibleModules = (sectionKey) => {
    const section = finalConfig[sectionKey];
    if (!section?.enabled) return false;

    return Object.keys(section).some(
      (key) => key !== 'enabled' && section[key] === true,
    );
  };

  // 鑾峰彇鍖哄煙鐨勫彲瑙佸姛鑳藉垪琛?
  const getVisibleModules = (sectionKey) => {
    const section = finalConfig[sectionKey];
    if (!section?.enabled) return [];

    return Object.keys(section).filter(
      (key) => key !== 'enabled' && section[key] === true,
    );
  };

  return {
    loading,
    adminConfig,
    userConfig,
    finalConfig,
    isModuleVisible,
    hasSectionVisibleModules,
    getVisibleModules,
    refreshUserConfig,
  };
};
