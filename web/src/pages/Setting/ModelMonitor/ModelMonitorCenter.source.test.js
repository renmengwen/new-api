import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync(new URL('./ModelMonitorCenter.jsx', import.meta.url), 'utf8');

test('model monitor page offsets content below the fixed header', () => {
  assert.match(source, /className=['"]mt-\[60px\] px-2['"]/);
});

test('model monitor page exposes notification recipient management', () => {
  assert.match(source, /\/api\/model_monitor\/notification-users/);
  assert.match(source, /notification_disabled_user_ids/);
  assert.match(source, /can_receive/);
  assert.match(source, /disabled_reason/);
  assert.match(source, /通知人管理/);
  assert.match(source, /<Modal/);
  assert.match(source, /<Checkbox/);
  assert.doesNotMatch(source, /onClick=\{saveSettings\}/);
});

test('model monitor settings save applies the mutation response before refetching', () => {
  assert.match(source, /const \{ success, message, data \} = res\.data;/);
  assert.match(source, /if \(data\) \{\s+applyMonitorData\(data\);\s+\} else \{\s+await fetchMonitorData\(\);\s+\}/);
});
