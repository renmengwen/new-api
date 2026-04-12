export const ADMIN_PERMISSION_RESOURCES = [
  {
    resourceKey: 'permission_management',
    label: '权限管理',
    actions: [
      { actionKey: 'read', label: '查看' },
      { actionKey: 'bind_profile', label: '配置权限' },
    ],
  },
  {
    resourceKey: 'agent_management',
    label: '代理商管理',
    actions: [
      { actionKey: 'read', label: '查看' },
      { actionKey: 'create', label: '新增' },
      { actionKey: 'update', label: '编辑' },
      { actionKey: 'update_status', label: '启停' },
    ],
  },
  {
    resourceKey: 'user_management',
    label: '用户管理',
    actions: [
      { actionKey: 'read', label: '查看' },
      { actionKey: 'create', label: '新增' },
      { actionKey: 'update', label: '编辑' },
      { actionKey: 'update_status', label: '启停' },
      { actionKey: 'delete', label: '删除' },
    ],
  },
  {
    resourceKey: 'quota_management',
    label: '额度管理',
    actions: [
      { actionKey: 'read_summary', label: '额度摘要' },
      { actionKey: 'adjust', label: '单用户调额' },
      { actionKey: 'adjust_batch', label: '批量调额' },
      { actionKey: 'ledger_read', label: '额度流水' },
    ],
  },
  {
    resourceKey: 'audit_management',
    label: '审计日志',
    actions: [{ actionKey: 'read', label: '查看' }],
  },
];

export const ADMIN_MENU_OPTIONS = [
  { sectionKey: 'admin', moduleKey: 'user', label: '用户管理' },
  { sectionKey: 'admin', moduleKey: 'agents', label: '代理商管理' },
  { sectionKey: 'admin', moduleKey: 'permission-templates', label: '权限模板管理' },
  { sectionKey: 'admin', moduleKey: 'user-permissions', label: '用户权限管理' },
  { sectionKey: 'admin', moduleKey: 'quota-ledger', label: '额度流水' },
  { sectionKey: 'admin', moduleKey: 'channel', label: '渠道管理' },
  { sectionKey: 'admin', moduleKey: 'subscription', label: '订阅管理' },
  { sectionKey: 'admin', moduleKey: 'models', label: '模型管理' },
  { sectionKey: 'admin', moduleKey: 'deployment', label: '模型部署' },
  { sectionKey: 'admin', moduleKey: 'redemption', label: '兑换码管理' },
  { sectionKey: 'admin', moduleKey: 'setting', label: '系统设置' },
];

export const ADMIN_DATA_SCOPE_RESOURCES = [
  { resourceKey: 'user_management', label: '用户管理' },
  { resourceKey: 'quota_management', label: '额度管理' },
];

export const PERMISSION_PROFILE_TYPE_OPTIONS = [
  { label: '管理员', value: 'admin' },
  { label: '代理商', value: 'agent' },
  { label: '普通用户', value: 'end_user' },
];

export const USER_PERMISSION_TYPE_OPTIONS = [
  { label: '全部对象', value: '' },
  { label: '管理员', value: 'admin' },
  { label: '代理商', value: 'agent' },
  { label: '普通用户', value: 'end_user' },
];

export const ACTION_OVERRIDE_OPTIONS = [
  { label: '继承模板', value: 'inherit' },
  { label: '显式允许', value: 'allow' },
  { label: '显式禁止', value: 'deny' },
];

export const MENU_OVERRIDE_OPTIONS = [
  { label: '继承模板', value: 'inherit' },
  { label: '显示菜单', value: 'show' },
  { label: '隐藏菜单', value: 'hide' },
];

export const DATA_SCOPE_OPTIONS = [
  { label: '继承默认范围', value: 'inherit' },
  { label: '全部用户', value: 'all' },
  { label: '仅自己', value: 'self' },
  { label: '仅绑定用户', value: 'agent_only' },
  { label: '指定用户', value: 'assigned' },
];

export const QUOTA_ENTRY_TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: 'adjust', value: 'adjust' },
  { label: 'recharge', value: 'recharge' },
  { label: 'reclaim', value: 'reclaim' },
  { label: 'consume', value: 'consume' },
  { label: 'refund', value: 'refund' },
];
