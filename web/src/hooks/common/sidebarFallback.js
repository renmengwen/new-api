const ROLE_ADMIN_THRESHOLD = 10;
const ROLE_ROOT_THRESHOLD = 100;

export const inferSidebarUserType = (user = {}) => {
  if (typeof user?.user_type === 'string' && user.user_type) {
    return user.user_type;
  }

  const role = Number(user?.role ?? 0);
  if (role >= ROLE_ROOT_THRESHOLD) {
    return 'root';
  }
  if (role >= ROLE_ADMIN_THRESHOLD) {
    return 'admin';
  }
  return 'end_user';
};

export const buildFallbackUserSidebarConfig = (adminConfig, user = {}) => {
  if (!adminConfig || typeof adminConfig !== 'object') {
    return {};
  }

  const userType = inferSidebarUserType(user);
  const canAccessAdminSection = userType === 'root' || userType === 'admin';
  const isRootUser = userType === 'root';
  const fallbackConfig = {};

  Object.entries(adminConfig).forEach(([sectionKey, sectionConfig]) => {
    if (!sectionConfig || typeof sectionConfig !== 'object') {
      return;
    }

    if (sectionKey === 'admin' && !canAccessAdminSection) {
      fallbackConfig[sectionKey] = { enabled: false };
      return;
    }

    const nextSection = {
      enabled: sectionConfig.enabled !== false,
    };

    Object.entries(sectionConfig).forEach(([moduleKey, enabled]) => {
      if (moduleKey === 'enabled' || enabled === false) {
        return;
      }

      if (sectionKey === 'admin' && moduleKey === 'setting' && !isRootUser) {
        nextSection[moduleKey] = false;
        return;
      }

      nextSection[moduleKey] = true;
    });

    fallbackConfig[sectionKey] = nextSection;
  });

  if (!fallbackConfig.admin && !canAccessAdminSection) {
    fallbackConfig.admin = { enabled: false };
  }

  return fallbackConfig;
};
