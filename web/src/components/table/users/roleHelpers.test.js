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

import { resolveDisplayUserRole } from './roleHelpers.js';

test('resolveDisplayUserRole treats root role as root even when user_type is stale end_user', () => {
  assert.equal(resolveDisplayUserRole(100, 'end_user'), 'root');
});

test('resolveDisplayUserRole treats admin role as admin even when user_type is stale end_user', () => {
  assert.equal(resolveDisplayUserRole(10, 'end_user'), 'admin');
});

test('resolveDisplayUserRole keeps agent user type when role is common user', () => {
  assert.equal(resolveDisplayUserRole(1, 'agent'), 'agent');
});

test('resolveDisplayUserRole keeps common users as end_user', () => {
  assert.equal(resolveDisplayUserRole(1, 'end_user'), 'end_user');
  assert.equal(resolveDisplayUserRole(1, ''), 'end_user');
});
