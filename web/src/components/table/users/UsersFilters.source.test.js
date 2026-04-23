import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const filtersSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/components/table/users/UsersFilters.jsx'),
  'utf8',
);

test('UsersFilters defines optional role and status filters', () => {
  assert.match(filtersSource, /showRoleFilter = false/);
  assert.match(filtersSource, /showStatusFilter = false/);
  assert.match(filtersSource, /field='searchRole'/);
  assert.match(filtersSource, /field='searchStatus'/);
});
