import test from 'node:test';
import assert from 'node:assert/strict';

const loadDisplayHelpers = () => import('./display.js');

test('getAuditLogModuleLabel renders Chinese and falls back to raw values', async () => {
  const { getAuditLogModuleLabel } = await loadDisplayHelpers();

  assert.equal(getAuditLogModuleLabel('admin_management'), '管理员管理');
  assert.equal(getAuditLogModuleLabel('agent'), '代理管理');
  assert.equal(getAuditLogModuleLabel('future_module'), 'future_module');
});

test('getAuditLogActionLabel renders Chinese and falls back to raw values', async () => {
  const { getAuditLogActionLabel } = await loadDisplayHelpers();

  assert.equal(getAuditLogActionLabel('create'), '创建');
  assert.equal(getAuditLogActionLabel('adjust_batch'), '批量额度调整');
  assert.equal(getAuditLogActionLabel('future_action'), 'future_action');
});

test('formatAuditIdentity renders username plus optional display name plus id', async () => {
  const { formatAuditIdentity } = await loadDisplayHelpers();

  assert.equal(
    formatAuditIdentity({
      userId: 23,
      username: 'alice',
      displayName: '爱丽丝',
    }),
    'alice（爱丽丝）#23',
  );

  assert.equal(
    formatAuditIdentity({
      userId: 23,
      username: 'alice',
      displayName: 'alice',
    }),
    'alice#23',
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
    'bob（鲍勃）#42',
  );

  assert.equal(
    formatAuditTarget({
      targetType: 'quota_adjustment',
      targetId: 99,
    }),
    'quota_adjustment#99',
  );
});

test('AUDIT_LOG_COVERAGE exactly enumerates the current write points', async () => {
  const { AUDIT_LOG_COVERAGE } = await loadDisplayHelpers();

  assert.deepEqual(AUDIT_LOG_COVERAGE, {
    admin_management: ['create', 'update', 'enable', 'disable'],
    agent: ['create', 'update', 'enable', 'disable'],
    user_management: ['create', 'update', 'enable', 'disable', 'delete'],
    permission: ['bind_profile', 'clear_profile'],
    quota: ['adjust', 'adjust_batch'],
  });
});
