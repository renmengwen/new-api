import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const pageSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/pages/AdminAgentsPageV2/index.jsx'),
  'utf8',
);

test('admin agents page loads assignable token groups for agent forms', () => {
  assert.ok(pageSource.includes("params: { mode: 'assignable_token' }"));
  assert.ok(pageSource.includes('toManagedGroupOptions(res.data.data, userGroup)'));
});

test('admin agents page submits allowed token groups alongside the primary group', () => {
  assert.ok(pageSource.includes('allowed_token_groups_enabled'));
  assert.ok(pageSource.includes('allowed_token_groups: normalizedAllowedTokenGroups'));
  assert.ok(pageSource.includes('multiple'));
});

test('admin agents page shows the same allowed token group helper copy as user management', () => {
  assert.ok(
    pageSource.includes(
      "t('开启后，用户创建令牌时只能选择下列分组，主分组会自动纳入')",
    ),
  );
  assert.ok(
    pageSource.includes(
      "t('仅在开启限制令牌分组后生效，不影响用户主分组和计费语义')",
    ),
  );
});

test('admin agents page keeps create-only credential fields available when not editing', () => {
  assert.ok(pageSource.includes('!editingAgent ? ('));
  assert.ok(!pageSource.includes('{false ? ('));
});

test('admin agents page shows allowed token group controls without removing submit logic', () => {
  assert.ok(!pageSource.includes('hideAllowedTokenGroupFields'));
  assert.ok(pageSource.includes("<Text type='tertiary'>{t('限制令牌分组')}</Text>"));
  assert.ok(pageSource.includes("<Text type='tertiary'>{t('可创建令牌分组')}</Text>"));
});
