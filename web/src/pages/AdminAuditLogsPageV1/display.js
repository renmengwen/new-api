export const AUDIT_LOG_MODULE_LABELS = {
  admin_management: '管理员管理',
  agent: '代理管理',
  user_management: '用户管理',
  permission: '权限管理',
  quota: '额度管理',
};

export const AUDIT_LOG_ACTION_LABELS = {
  create: '创建',
  update: '更新',
  enable: '启用',
  disable: '禁用',
  delete: '删除',
  bind_profile: '绑定权限模板',
  clear_profile: '清空权限模板',
  adjust: '额度调整',
  adjust_batch: '批量额度调整',
};

export const AUDIT_LOG_COVERAGE = [
  { module: 'admin_management', actions: ['create', 'update', 'enable', 'disable'] },
  { module: 'agent', actions: ['create', 'update', 'enable', 'disable'] },
  { module: 'user_management', actions: ['create', 'update', 'enable', 'disable', 'delete'] },
  { module: 'permission', actions: ['bind_profile', 'clear_profile'] },
  { module: 'quota', actions: ['adjust', 'adjust_batch'] },
];

const hasValue = (value) => value !== null && value !== undefined && value !== '';

const getPreferredValue = (source, keys) => {
  for (const key of keys) {
    const value = source?.[key];
    if (hasValue(value)) {
      return value;
    }
  }

  return '';
};

export const getAuditLogModuleLabel = (value) => AUDIT_LOG_MODULE_LABELS[value] ?? value;

export const getAuditLogActionLabel = (value) => AUDIT_LOG_ACTION_LABELS[value] ?? value;

export const formatAuditIdentity = (identity = {}) => {
  const userId = getPreferredValue(identity, ['userId', 'user_id', 'id']);
  const username = getPreferredValue(identity, ['username']);
  const displayName = getPreferredValue(identity, ['displayName', 'display_name']);

  if (!username) {
    return '-';
  }

  const displayNamePart = displayName && displayName !== username ? `（${displayName}）` : '';
  const idPart = hasValue(userId) ? ` #${userId}` : '';

  return `${username}${displayNamePart}${idPart}`;
};

export const formatAuditTarget = (target = {}) => {
  const targetType = getPreferredValue(target, ['targetType', 'target_type']);
  const targetId = getPreferredValue(target, ['targetId', 'target_id']);
  const targetUsername = getPreferredValue(target, ['targetUsername', 'target_username']);
  const targetDisplayName = getPreferredValue(target, ['targetDisplayName', 'target_display_name']);

  const identity = formatAuditIdentity({
    userId: targetId,
    username: targetUsername,
    displayName: targetDisplayName,
  });

  if (identity !== '-') {
    return identity;
  }

  if (!hasValue(targetType)) {
    return hasValue(targetId) ? `#${targetId}` : '-';
  }

  return hasValue(targetId) ? `${targetType} #${targetId}` : targetType;
};
