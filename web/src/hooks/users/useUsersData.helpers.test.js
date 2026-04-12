import test from 'node:test';
import assert from 'node:assert/strict';

import {
  normalizeUserPageData,
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
