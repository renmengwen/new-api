import test from 'node:test';
import assert from 'node:assert/strict';

import {
  buildActionOverrideMap,
  buildActionOverridePayload,
  buildDataScopeOverrideMap,
  buildDataScopeOverridePayload,
  buildMenuOverrideMap,
  buildMenuOverridePayload,
  filterTemplateOptions,
  getEffectiveScopeText,
  getUserTypeText,
} from './helpers.js';

test('buildActionOverrideMap indexes action overrides by resource and action', () => {
  const result = buildActionOverrideMap([
    { resource_key: 'permission_management', action_key: 'read', effect: 'allow' },
    { resource_key: 'user_management', action_key: 'update_status', effect: 'deny' },
  ]);

  assert.deepEqual(result, {
    'permission_management.read': 'allow',
    'user_management.update_status': 'deny',
  });
});

test('buildMenuOverrideMap indexes menu overrides by section and module', () => {
  const result = buildMenuOverrideMap([
    { section_key: 'admin', module_key: 'user-permissions', effect: 'show' },
    { section_key: 'admin', module_key: 'agents', effect: 'hide' },
  ]);

  assert.deepEqual(result, {
    'admin.user-permissions': 'show',
    'admin.agents': 'hide',
  });
});

test('buildDataScopeOverrideMap normalizes assigned scope values into comma-separated strings', () => {
  const result = buildDataScopeOverrideMap([
    { resource_key: 'user_management', scope_type: 'assigned', scope_value: [3, 9, 12] },
    { resource_key: 'quota_management', scope_type: 'all', scope_value: [] },
  ]);

  assert.deepEqual(result, {
    user_management: { scopeType: 'assigned', scopeValue: '3,9,12' },
    quota_management: { scopeType: 'all', scopeValue: '' },
  });
});

test('buildActionOverridePayload keeps only explicit allow and deny values', () => {
  const result = buildActionOverridePayload({
    'permission_management.read': 'allow',
    'permission_management.bind_profile': 'inherit',
    'user_management.update_status': 'deny',
  });

  assert.deepEqual(result, [
    { resource_key: 'permission_management', action_key: 'read', effect: 'allow' },
    { resource_key: 'user_management', action_key: 'update_status', effect: 'deny' },
  ]);
});

test('buildMenuOverridePayload keeps only explicit show and hide values', () => {
  const result = buildMenuOverridePayload({
    'admin.user-permissions': 'show',
    'admin.agents': 'hide',
    'admin.setting': 'inherit',
  });

  assert.deepEqual(result, [
    { section_key: 'admin', module_key: 'user-permissions', effect: 'show' },
    { section_key: 'admin', module_key: 'agents', effect: 'hide' },
  ]);
});

test('buildDataScopeOverridePayload strips inherit values and invalid assigned ids', () => {
  const result = buildDataScopeOverridePayload({
    user_management: { scopeType: 'assigned', scopeValue: '1, 2, 0, x, -5, 9' },
    quota_management: { scopeType: 'all', scopeValue: '' },
    audit_management: { scopeType: 'inherit', scopeValue: '7' },
  });

  assert.deepEqual(result, [
    { resource_key: 'user_management', scope_type: 'assigned', scope_value: [1, 2, 9] },
    { resource_key: 'quota_management', scope_type: 'all', scope_value: [] },
  ]);
});

test('filterTemplateOptions keeps matching templates and allows root users to use admin templates', () => {
  const templates = [
    { id: 1, profile_name: '管理员默认', profile_type: 'admin' },
    { id: 2, profile_name: '代理商默认', profile_type: 'agent' },
    { id: 3, profile_name: '用户默认', profile_type: 'end_user' },
  ];

  assert.deepEqual(filterTemplateOptions(templates, { user_type: 'agent' }), [
    { label: '代理商默认 (agent)', value: 2 },
  ]);

  assert.deepEqual(filterTemplateOptions(templates, { user_type: 'root' }), [
    { label: '管理员默认 (admin)', value: 1 },
  ]);
});

test('getEffectiveScopeText returns stable Chinese labels for known scopes', () => {
  assert.equal(getEffectiveScopeText('all'), '全部用户');
  assert.equal(getEffectiveScopeText('self'), '仅自己');
  assert.equal(getEffectiveScopeText('agent_only'), '仅绑定用户');
  assert.equal(getEffectiveScopeText('assigned'), '指定用户');
  assert.equal(getEffectiveScopeText('unknown'), '默认范围');
});

test('getUserTypeText returns stable Chinese labels for known types', () => {
  assert.equal(getUserTypeText('root'), '超级管理员');
  assert.equal(getUserTypeText('admin'), '管理员');
  assert.equal(getUserTypeText('agent'), '代理商');
  assert.equal(getUserTypeText('end_user'), '普通用户');
  assert.equal(getUserTypeText('mystery'), 'mystery');
  assert.equal(getUserTypeText(''), '-');
});
