const hasOwn = (value, key) =>
  value &&
  typeof value === 'object' &&
  Object.prototype.hasOwnProperty.call(value, key);

export const normalizeUserSidebarConfig = (config) => {
  if (!config || typeof config !== 'object') return {};

  const normalized = {};
  Object.entries(config).forEach(([sectionKey, sectionValue]) => {
    if (sectionValue === false) {
      normalized[sectionKey] = { enabled: false };
      return;
    }
    if (sectionValue === true) {
      normalized[sectionKey] = { enabled: true };
      return;
    }
    if (!sectionValue || typeof sectionValue !== 'object') {
      return;
    }

    normalized[sectionKey] = {
      enabled: sectionValue.enabled !== false,
    };
    Object.keys(sectionValue).forEach((moduleKey) => {
      if (moduleKey === 'enabled') return;
      normalized[sectionKey][moduleKey] = sectionValue[moduleKey] !== false;
    });
  });

  return normalized;
};

export const buildFinalSidebarConfig = (
  adminConfig,
  userConfig,
  { strictSnapshot = false } = {},
) => {
  const result = {};

  if (!adminConfig || Object.keys(adminConfig).length === 0 || !userConfig) {
    return result;
  }

  Object.keys(adminConfig).forEach((sectionKey) => {
    const adminSection = adminConfig[sectionKey];
    const hasUserSection = hasOwn(userConfig, sectionKey);
    const userSection = hasUserSection ? userConfig[sectionKey] : undefined;

    if (!adminSection?.enabled) {
      result[sectionKey] = { enabled: false };
      return;
    }

    const sectionEnabled = hasUserSection
      ? userSection?.enabled !== false
      : !strictSnapshot;
    result[sectionKey] = { enabled: sectionEnabled };

    Object.keys(adminSection).forEach((moduleKey) => {
      if (moduleKey === 'enabled') return;

      const adminAllowed = adminSection[moduleKey];
      let userAllowed;
      if (!hasUserSection) {
        userAllowed = !strictSnapshot;
      } else if (hasOwn(userSection, moduleKey)) {
        userAllowed = userSection[moduleKey] !== false;
      } else {
        userAllowed = !strictSnapshot;
      }

      result[sectionKey][moduleKey] =
        adminAllowed && userAllowed && sectionEnabled;
    });
  });

  return result;
};

export const isPermissionSidebarSectionAllowed = (sidebarModules, sectionKey) => {
  if (!sidebarModules) return true;
  if (!hasOwn(sidebarModules, sectionKey)) return false;

  const sectionPerms = sidebarModules[sectionKey];
  if (sectionPerms === false) return false;
  if (!sectionPerms || typeof sectionPerms !== 'object') return false;

  return sectionPerms.enabled !== false;
};

export const isPermissionSidebarModuleAllowed = (
  sidebarModules,
  sectionKey,
  moduleKey,
) => {
  if (!sidebarModules) return true;
  if (!isPermissionSidebarSectionAllowed(sidebarModules, sectionKey)) {
    return false;
  }

  const sectionPerms = sidebarModules[sectionKey];
  return hasOwn(sectionPerms, moduleKey) && sectionPerms[moduleKey] === true;
};
