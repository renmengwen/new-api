import test from 'node:test';
import assert from 'node:assert/strict';

import { isUserDeleted } from './statusHelpers.js';

test('isUserDeleted treats missing deleted fields as not deleted', () => {
  assert.equal(isUserDeleted({}), false);
  assert.equal(isUserDeleted({ DeletedAt: undefined }), false);
  assert.equal(isUserDeleted({ deleted_at: undefined }), false);
  assert.equal(isUserDeleted({ DeletedAt: null }), false);
  assert.equal(isUserDeleted({ deleted_at: null }), false);
});

test('isUserDeleted treats existing deleted markers as deleted', () => {
  assert.equal(isUserDeleted({ DeletedAt: '2026-04-10T10:00:00Z' }), true);
  assert.equal(isUserDeleted({ deleted_at: 1712736000 }), true);
});
