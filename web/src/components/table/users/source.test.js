import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const usersEntrySource = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');
const columnDefsSource = fs.readFileSync(new URL('./UsersColumnDefs.jsx', import.meta.url), 'utf8');

test('managed users table keeps extra user-management capabilities configurable', () => {
  assert.match(usersEntrySource, /canResetPasskey:\s*capabilities\.canResetPasskey === true/);
  assert.match(usersEntrySource, /canResetTwoFA:\s*capabilities\.canResetTwoFA === true/);
  assert.match(usersEntrySource, /canManageSubscriptions:\s*capabilities\.canManageSubscriptions === true/);
  assert.match(usersEntrySource, /canManageBindings:\s*capabilities\.canManageBindings === true/);
});

test('user operation column no longer exposes promote or demote buttons', () => {
  assert.doesNotMatch(columnDefsSource, /\{t\('鎻愬崌'\)\}/);
  assert.doesNotMatch(columnDefsSource, /\{t\('闄嶇骇'\)\}/);
});

test('user quota usage display keeps six decimal places for admin audit views', () => {
  assert.match(columnDefsSource, /const quotaDigits = 6;/);
  assert.match(columnDefsSource, /renderQuota\(used,\s*quotaDigits\)/);
  assert.match(columnDefsSource, /renderQuota\(remain,\s*quotaDigits\)/);
  assert.match(columnDefsSource, /renderQuota\(total,\s*quotaDigits\)/);
});
