import test from 'node:test';
import assert from 'node:assert/strict';

import {
  QUOTA_ADJUST_MODE,
  calculateAdjustedQuota,
  normalizePositiveAdjustmentValue,
  shouldDisableQuotaAdjustmentConfirm,
  shouldDisableQuotaInput,
} from './editUserModalHelpers.js';

test('normalizePositiveAdjustmentValue keeps only positive numbers for adjustment inputs', () => {
  assert.equal(normalizePositiveAdjustmentValue(20), 20);
  assert.equal(normalizePositiveAdjustmentValue(-20), 20);
  assert.equal(normalizePositiveAdjustmentValue('-15'), 15);
  assert.equal(normalizePositiveAdjustmentValue(''), '');
  assert.equal(normalizePositiveAdjustmentValue(null), '');
});

test('calculateAdjustedQuota applies increase and decrease modes', () => {
  assert.equal(calculateAdjustedQuota(100, 25, QUOTA_ADJUST_MODE.increase), 125);
  assert.equal(calculateAdjustedQuota(100, 25, QUOTA_ADJUST_MODE.decrease), 75);
  assert.equal(calculateAdjustedQuota('100', '-20', QUOTA_ADJUST_MODE.increase), 120);
});

test('shouldDisableQuotaAdjustmentConfirm blocks empty, zero and negative-result adjustments', () => {
  assert.equal(
    shouldDisableQuotaAdjustmentConfirm(100, '', QUOTA_ADJUST_MODE.increase),
    true,
  );
  assert.equal(
    shouldDisableQuotaAdjustmentConfirm(100, 0, QUOTA_ADJUST_MODE.increase),
    true,
  );
  assert.equal(
    shouldDisableQuotaAdjustmentConfirm(100, 20, QUOTA_ADJUST_MODE.increase),
    false,
  );
  assert.equal(
    shouldDisableQuotaAdjustmentConfirm(100, 120, QUOTA_ADJUST_MODE.decrease),
    true,
  );
});

test('shouldDisableQuotaInput disables direct quota edits in edit mode', () => {
  assert.equal(shouldDisableQuotaInput(true), true);
  assert.equal(shouldDisableQuotaInput(false), false);
});
