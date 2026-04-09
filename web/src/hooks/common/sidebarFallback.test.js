import test from 'node:test';
import assert from 'node:assert/strict';
import { buildFallbackUserSidebarConfig } from './sidebarFallback.js';

const DEFAULT_ADMIN_CONFIG = {
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
    agents: true,
    'permission-templates': true,
    'user-permissions': true,
    'quota-ledger': true,
    setting: true,
  },
};

test('ordinary users do not receive admin sidebar access by fallback', () => {
  const config = buildFallbackUserSidebarConfig(DEFAULT_ADMIN_CONFIG, {
    role: 1,
    user_type: 'end_user',
  });

  assert.equal(config.admin.enabled, false);
  assert.equal(config.console.enabled, true);
});

test('agents do not receive admin sidebar access by fallback', () => {
  const config = buildFallbackUserSidebarConfig(DEFAULT_ADMIN_CONFIG, {
    role: 10,
    user_type: 'agent',
  });

  assert.equal(config.admin.enabled, false);
});

test('admins receive admin sidebar access but not settings by fallback', () => {
  const config = buildFallbackUserSidebarConfig(DEFAULT_ADMIN_CONFIG, {
    role: 10,
    user_type: 'admin',
  });

  assert.equal(config.admin.enabled, true);
  assert.equal(config.admin.agents, true);
  assert.equal(config.admin.setting, false);
});

test('root users keep settings access by fallback', () => {
  const config = buildFallbackUserSidebarConfig(DEFAULT_ADMIN_CONFIG, {
    role: 100,
    user_type: 'root',
  });

  assert.equal(config.admin.enabled, true);
  assert.equal(config.admin.setting, true);
});

test('root role overrides stale end_user user_type during fallback inference', () => {
  const config = buildFallbackUserSidebarConfig(DEFAULT_ADMIN_CONFIG, {
    role: 100,
    user_type: 'end_user',
  });

  assert.equal(config.admin.enabled, true);
  assert.equal(config.admin.setting, true);
});
