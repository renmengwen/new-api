import test from 'node:test';
import assert from 'node:assert/strict';

import { applyQuotaDelta, shouldDisableQuotaInput } from './editUserModalHelpers.js';

test('applyQuotaDelta adds numeric deltas onto current quota', () => {
  assert.equal(applyQuotaDelta(100, 25), 125);
  assert.equal(applyQuotaDelta('100', '-20'), 80);
  assert.equal(applyQuotaDelta(undefined, undefined), 0);
});

test('shouldDisableQuotaInput disables direct quota edits in edit mode', () => {
  assert.equal(shouldDisableQuotaInput(true), true);
  assert.equal(shouldDisableQuotaInput(false), false);
});
