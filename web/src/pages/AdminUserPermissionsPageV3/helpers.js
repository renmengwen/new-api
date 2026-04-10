export const buildActionOverrideMap = (items = []) => {
  const nextMap = {};
  items.forEach((item) => {
    nextMap[`${item.resource_key}.${item.action_key}`] = item.effect;
  });
  return nextMap;
};

export const buildMenuOverrideMap = (items = []) => {
  const nextMap = {};
  items.forEach((item) => {
    nextMap[`${item.section_key}.${item.module_key}`] = item.effect;
  });
  return nextMap;
};

export const buildDataScopeOverrideMap = (items = []) => {
  const nextMap = {};
  items.forEach((item) => {
    nextMap[item.resource_key] = {
      scopeType: item.scope_type,
      scopeValue: Array.isArray(item.scope_value) ? item.scope_value.join(',') : '',
    };
  });
  return nextMap;
};

export const buildActionOverridePayload = (overrideMap = {}) =>
  Object.entries(overrideMap)
    .filter(([, effect]) => effect === 'allow' || effect === 'deny')
    .map(([key, effect]) => {
      const [resource_key, action_key] = key.split('.');
      return { resource_key, action_key, effect };
    });

export const buildMenuOverridePayload = (overrideMap = {}) =>
  Object.entries(overrideMap)
    .filter(([, effect]) => effect === 'show' || effect === 'hide')
    .map(([key, effect]) => {
      const [section_key, module_key] = key.split('.');
      return { section_key, module_key, effect };
    });

export const buildDataScopeOverridePayload = (overrideMap = {}) =>
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

export const filterTemplateOptions = (templates = [], editingUser) =>
  templates
    .filter((template) => {
      if (!editingUser) {
        return true;
      }

      return (
        template.profile_type === editingUser.user_type ||
        (editingUser.user_type === 'root' && template.profile_type === 'admin')
      );
    })
    .map((template) => ({
      label: `${template.profile_name} (${template.profile_type})`,
      value: template.id,
    }));

export const getEffectiveScopeText = (value) => {
  switch (value) {
    case 'all':
      return '全部用户';
    case 'self':
      return '仅自己';
    case 'agent_only':
      return '仅绑定用户';
    case 'assigned':
      return '指定用户';
    default:
      return '默认范围';
  }
};

export const getUserTypeText = (userType) => {
  switch (userType) {
    case 'root':
      return '超级管理员';
    case 'admin':
      return '管理员';
    case 'agent':
      return '代理商';
    case 'end_user':
      return '普通用户';
    case '':
    case null:
    case undefined:
      return '-';
    default:
      return userType;
  }
};
