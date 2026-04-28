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

test('model monitor table shows model last tested time', () => {
  assert.match(source, /title: t\('最后测试时间'\)/);
  assert.match(source, /dataIndex: 'tested_at'/);
  assert.match(source, /render: \(value\) => formatTestedAt\(value\)/);
});

test('manual model monitor test reports final model result summary', () => {
  assert.match(source, /buildModelMonitorResultMessage/);
  assert.match(source, /finalData = await waitForManualTestResult\(\);/);
  assert.match(source, /showSuccess\(buildModelMonitorResultMessage\(finalData\.summary, t\)\)/);
});

test('model monitor page refreshes state while scheduled monitoring is enabled', () => {
  assert.match(source, /setInterval\(\(\) => \{/);
  assert.match(source, /fetchMonitorData\(\{ showLoading: false, disableDuplicate: true \}\)/);
  assert.match(source, /settings\.enabled/);
});
