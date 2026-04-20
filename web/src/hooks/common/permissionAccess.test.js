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

import {
  hasPermissionAction,
  isRootPermissionUser,
  shouldUseStrictSidebarSnapshot,
} from './permissionAccess.js';

test('root users keep full action access even when a new action is missing from snapshot', () => {
  const permissions = {
    profile_type: 'root',
    actions: {},
  };

  assert.equal(
    hasPermissionAction(
      permissions,
      'analytics_management',
      'read',
      { role: 100, user_type: 'root' },
    ),
    true,
  );
});

test('non-root users still require explicit action grants', () => {
  const permissions = {
    profile_type: 'admin',
    actions: {},
  };

  assert.equal(
    hasPermissionAction(
      permissions,
      'analytics_management',
      'read',
      { role: 10, user_type: 'admin' },
    ),
    false,
  );
});

test('root sidebar snapshots do not use strict missing-module denial', () => {
  assert.equal(
    shouldUseStrictSidebarSnapshot({
      role: 100,
      user_type: 'root',
      permissions: {
        profile_type: 'root',
        sidebar_modules: {
          admin: {
            enabled: true,
          },
        },
      },
    }),
    false,
  );
});

test('non-root sidebar snapshots remain strict when backend provides explicit permissions', () => {
  assert.equal(
    shouldUseStrictSidebarSnapshot({
      role: 10,
      user_type: 'admin',
      permissions: {
        profile_type: 'admin',
        sidebar_modules: {
          admin: {
            enabled: true,
          },
        },
      },
    }),
    true,
  );
});

test('role-based root fallback also covers legacy root snapshots without profile_type', () => {
  assert.equal(
    isRootPermissionUser(
      {
        actions: {},
      },
      {
        role: 100,
        user_type: 'end_user',
      },
    ),
    true,
  );
});
