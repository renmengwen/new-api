import test from 'node:test';
import assert from 'node:assert/strict';

import { buildFinalSidebarConfig } from './sidebarPermissionSnapshot.js';

const ADMIN_CONFIG = {
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
    subscription: true,
    setting: true,
  },
};

test('permission sidebar snapshots deny missing admin modules by default', () => {
  const finalConfig = buildFinalSidebarConfig(
    ADMIN_CONFIG,
    {
      admin: {
        enabled: true,
        setting: false,
      },
    },
    { strictSnapshot: true },
  );

  assert.equal(finalConfig.admin.enabled, true);
  assert.equal(finalConfig.admin.channel, false);
  assert.equal(finalConfig.admin.models, false);
  assert.equal(finalConfig.admin.setting, false);
});

test('user sidebar preferences keep missing admin modules enabled by default', () => {
  const finalConfig = buildFinalSidebarConfig(
    ADMIN_CONFIG,
    {
      admin: {
        enabled: true,
        setting: false,
      },
    },
    { strictSnapshot: false },
  );

  assert.equal(finalConfig.admin.enabled, true);
  assert.equal(finalConfig.admin.channel, true);
  assert.equal(finalConfig.admin.models, true);
  assert.equal(finalConfig.admin.setting, false);
});
