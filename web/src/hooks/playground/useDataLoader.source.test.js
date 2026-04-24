import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const hookSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/hooks/playground/useDataLoader.js'),
  'utf8',
);

test('playground loads group options from token-selectable user groups', () => {
  assert.ok(hookSource.includes('API.get(API_ENDPOINTS.USER_GROUPS, {'));
  assert.ok(hookSource.includes("params: { mode: 'token' }"));
  assert.ok(hookSource.includes('processGroupsData(data, userGroup)'));
});
