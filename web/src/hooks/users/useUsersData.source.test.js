/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const hookSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/hooks/users/useUsersData.jsx'),
  'utf8',
);

test('useUsersData loads managed-mode group options from user self groups', () => {
  assert.ok(hookSource.includes('toManagedGroupOptions'));
  assert.ok(hookSource.includes("API.get('/api/user/self/groups', {"));
  assert.ok(hookSource.includes("params: { mode: 'assignable_token' }"));
  assert.ok(hookSource.includes("const userGroup = JSON.parse(localStorage.getItem('user'))?.group;"));
  assert.ok(
    hookSource.includes(
      'setGroupOptions(toManagedGroupOptions(res.data.data, userGroup));',
    ),
  );
});

test('useUsersData keeps legacy group loading on the admin group endpoint', () => {
  assert.ok(hookSource.includes("API.get('/api/group/')"));
  assert.ok(hookSource.includes('setGroupOptions(toGroupOptions(res.data));'));
});

test('useUsersData tracks role and status filters for legacy user search', () => {
  assert.ok(hookSource.includes("searchRole: ''"));
  assert.ok(hookSource.includes("searchStatus: ''"));
  assert.ok(hookSource.includes('searchRole: formValues.searchRole || \'\','));
  assert.ok(hookSource.includes('searchStatus: formValues.searchStatus || \'\','));
  assert.ok(hookSource.includes("params.set('role', searchRole);"));
  assert.ok(hookSource.includes("params.set('status', searchStatus);"));
});

test('useUsersData delegates search-state decisions to the shared mode-aware helper', () => {
  assert.ok(
    hookSource.includes(
      'shouldUseUserSearch',
    ),
  );
  assert.ok(
    hookSource.includes(
      'const shouldSearch = shouldUseUserSearch({',
    ),
  );
});
