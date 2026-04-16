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
  normalizeUserPageData,
  toManagedGroupOptions,
  toGroupOptions,
} from './useUsersData.helpers.js';

test('normalizeUserPageData falls back to an empty page when items is null', () => {
  assert.deepEqual(normalizeUserPageData({ items: null, page: null, total: null }, 2), {
    items: [],
    page: 2,
    total: 0,
  });
});

test('normalizeUserPageData keeps valid paging data', () => {
  assert.deepEqual(
    normalizeUserPageData({ items: [{ id: 1 }], page: 3, total: 9 }, 1),
    {
      items: [{ id: 1 }],
      page: 3,
      total: 9,
    },
  );
});

test('toGroupOptions returns empty options when group api payload is denied', () => {
  assert.deepEqual(
    toGroupOptions({ success: false, message: '无权进行此操作，权限不足', data: null }),
    [],
  );
});

test('toGroupOptions maps string groups into select options', () => {
  assert.deepEqual(toGroupOptions({ success: true, data: ['default', 'vip'] }), [
    { label: 'default', value: 'default' },
    { label: 'vip', value: 'vip' },
  ]);
});

test('toManagedGroupOptions keeps managed-mode labels aligned with raw group names', () => {
  assert.deepEqual(
    toManagedGroupOptions(
      {
        agent_group: { desc: '用户分组', ratio: 1 },
        default: { desc: '默认分组', ratio: 1 },
        vip: { desc: 'vip分组', ratio: 1 },
      },
      'agent_group',
    ),
    [
      {
        label: 'agent_group',
        value: 'agent_group',
        ratio: 1,
        fullLabel: '用户分组',
      },
      {
        label: 'default',
        value: 'default',
        ratio: 1,
        fullLabel: '默认分组',
      },
      {
        label: 'vip',
        value: 'vip',
        ratio: 1,
        fullLabel: 'vip分组',
      },
    ],
  );
});
