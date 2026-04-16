import test from 'node:test';
import assert from 'node:assert/strict';

const loadDisplayHelpers = () => import('./display.js');

test('getAuditLogModuleLabel renders Chinese and falls back to raw values', async () => {
  const { getAuditLogModuleLabel } = await loadDisplayHelpers();

  assert.equal(getAuditLogModuleLabel('admin_management'), '管理员管理');
  assert.equal(getAuditLogModuleLabel('agent'), '代理管理');
  assert.equal(getAuditLogModuleLabel('setting_system'), '系统设置');
  assert.equal(getAuditLogModuleLabel('setting_model'), '模型相关设置');
  assert.equal(getAuditLogModuleLabel('future_module'), 'future_module');
});

test('getAuditLogActionLabel renders Chinese and falls back to raw values', async () => {
  const { getAuditLogActionLabel } = await loadDisplayHelpers();

  assert.equal(getAuditLogActionLabel('create'), '创建');
  assert.equal(getAuditLogActionLabel('adjust_batch'), '批量额度调整');
  assert.equal(getAuditLogActionLabel('save_general'), '保存通用设置');
  assert.equal(getAuditLogActionLabel('toggle_allow_private_ip'), '允许访问私有 IP');
  assert.equal(getAuditLogActionLabel('refresh_performance_stats'), '刷新统计');
  assert.equal(getAuditLogActionLabel('future_action'), 'future_action');
});

test('formatAuditIdentity renders username plus optional display name plus id', async () => {
  const { formatAuditIdentity } = await loadDisplayHelpers();

  assert.equal(
    formatAuditIdentity({
      userId: 18,
      username: 'alice',
      displayName: 'Alice',
    }),
    'alice（Alice） #18',
  );

  assert.equal(
    formatAuditIdentity({
      userId: 7,
      username: 'bob',
      displayName: 'bob',
    }),
    'bob #7',
  );

  assert.equal(
    formatAuditIdentity({
      userId: 42,
      username: '',
      displayName: '',
    }),
    '#42',
  );
});

test('formatAuditTarget prefers user identity and otherwise falls back to target type plus id', async () => {
  const { formatAuditTarget } = await loadDisplayHelpers();

  assert.equal(
    formatAuditTarget({
      targetType: 'user',
      targetId: 42,
      targetUsername: 'bob',
      targetDisplayName: '鲍勃',
    }),
    'bob（鲍勃） #42',
  );

  assert.equal(
    formatAuditTarget({
      targetType: 'batch',
      targetId: 42,
    }),
    'batch #42',
  );

  assert.equal(
    formatAuditTarget({
      targetType: 'option_key',
      targetId: 0,
    }),
    '配置项',
  );

  assert.equal(
    formatAuditTarget({
      targetType: 'user',
      targetId: 42,
      targetUsername: '',
      targetDisplayName: '',
    }),
    '用户 #42',
  );
});

test('AUDIT_LOG_COVERAGE exactly enumerates the current write points', async () => {
  const { AUDIT_LOG_COVERAGE } = await loadDisplayHelpers();

  assert.deepEqual(AUDIT_LOG_COVERAGE, [
    { module: 'admin_management', actions: ['create', 'update', 'enable', 'disable'] },
    { module: 'agent', actions: ['create', 'update', 'enable', 'disable'] },
    { module: 'user_management', actions: ['create', 'update', 'enable', 'disable', 'delete'] },
    { module: 'permission', actions: ['bind_profile', 'clear_profile'] },
    { module: 'quota', actions: ['adjust', 'adjust_batch'] },
  ]);
});

test('AUDIT_LOG_FILTER_MODULES includes setting modules for the filter select', async () => {
  const { AUDIT_LOG_FILTER_MODULES } = await loadDisplayHelpers();

  assert.equal(AUDIT_LOG_FILTER_MODULES.includes('admin_management'), true);
  assert.equal(AUDIT_LOG_FILTER_MODULES.includes('setting_system'), true);
  assert.equal(AUDIT_LOG_FILTER_MODULES.includes('setting_model'), true);
  assert.equal(AUDIT_LOG_FILTER_MODULES.includes('setting_performance'), true);
});
