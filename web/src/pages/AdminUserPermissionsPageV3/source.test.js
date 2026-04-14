import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(new URL('./CleanPage.jsx', import.meta.url), 'utf8');

test('permission modal keeps template binding above three override tabs', () => {
  assert.match(source, /\bTabs\b/);
  assert.match(source, /\bTabPane\b/);
  assert.match(source, /activePermissionTab/);
  assert.match(source, /setActivePermissionTab\('action'\)/);
  assert.match(
    source,
    /<Tabs[\s\S]*activeKey=\{activePermissionTab\}[\s\S]*onChange=\{setActivePermissionTab\}/,
  );
  assert.match(source, /itemKey='action'/);
  assert.match(source, /itemKey='menu'/);
  assert.match(source, /itemKey='data-scope'/);

  const selectedProfileIndex = source.indexOf('value={selectedProfileId}');
  const tabsIndex = source.indexOf('<Tabs');

  assert.notEqual(selectedProfileIndex, -1);
  assert.notEqual(tabsIndex, -1);
  assert.ok(selectedProfileIndex < tabsIndex);
});
