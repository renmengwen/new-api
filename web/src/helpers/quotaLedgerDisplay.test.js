import test from 'node:test';
import assert from 'node:assert/strict';

import {
  QUOTA_LEDGER_ENTRY_TYPE_OPTIONS,
  getQuotaAccountName,
  getQuotaEntryTypeLabel,
  getQuotaOperatorName,
} from './quotaLedgerDisplay.js';

test('getQuotaEntryTypeLabel returns stable Chinese labels', () => {
  assert.equal(getQuotaEntryTypeLabel('adjust'), '调额');
  assert.equal(getQuotaEntryTypeLabel('recharge'), '充值');
  assert.equal(getQuotaEntryTypeLabel('reclaim'), '回收');
  assert.equal(getQuotaEntryTypeLabel('consume'), '消耗');
  assert.equal(getQuotaEntryTypeLabel('refund'), '退款');
  assert.equal(getQuotaEntryTypeLabel('reward'), '奖励');
  assert.equal(getQuotaEntryTypeLabel('commission'), '佣金');
  assert.equal(getQuotaEntryTypeLabel('unknown'), 'unknown');
  assert.equal(getQuotaEntryTypeLabel(''), '-');
});

test('QUOTA_LEDGER_ENTRY_TYPE_OPTIONS use Chinese labels', () => {
  assert.deepEqual(QUOTA_LEDGER_ENTRY_TYPE_OPTIONS, [
    { label: '全部类型', value: '' },
    { label: '调额', value: 'adjust' },
    { label: '充值', value: 'recharge' },
    { label: '回收', value: 'reclaim' },
    { label: '消耗', value: 'consume' },
    { label: '退款', value: 'refund' },
    { label: '奖励', value: 'reward' },
    { label: '佣金', value: 'commission' },
  ]);
});

test('account and operator display prefer usernames', () => {
  assert.equal(getQuotaAccountName({ account_username: 'target_user' }), 'target_user');
  assert.equal(getQuotaAccountName({}), '-');

  assert.equal(getQuotaOperatorName({ operator_username: 'admin_user' }), 'admin_user');
  assert.equal(getQuotaOperatorName({ operator_user_id: 99 }), '-');
});
