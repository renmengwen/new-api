import test from 'node:test';
import assert from 'node:assert/strict';

import {
  QUOTA_LEDGER_ENTRY_TYPE_OPTIONS,
  getQuotaAccountName,
  getQuotaEntryTypeLabel,
  getQuotaOperatorName,
  getQuotaReasonLabel,
} from './quotaLedgerDisplay.js';

test('getQuotaEntryTypeLabel returns stable Chinese labels', () => {
  assert.equal(getQuotaEntryTypeLabel('opening'), '初始额度');
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
    { label: '初始额度', value: 'opening' },
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

test('getQuotaReasonLabel maps stable reason codes to Chinese labels', () => {
  assert.equal(getQuotaReasonLabel('manual_adjust'), '手动调额');
  assert.equal(getQuotaReasonLabel('legacy_user_update'), '手动调额');
  assert.equal(getQuotaReasonLabel('managed_user_update'), '管理用户调额');
  assert.equal(getQuotaReasonLabel('batch_adjust'), '批量调额');
  assert.equal(getQuotaReasonLabel('batch_partial_adjust'), '批量部分调额');
  assert.equal(getQuotaReasonLabel('agent_adjust'), '代理商调额');
  assert.equal(getQuotaReasonLabel('agent_reclaim'), '代理商回收');
  assert.equal(getQuotaReasonLabel('agent_batch_adjust'), '代理商批量调额');
  assert.equal(getQuotaReasonLabel('sync_with_user_quota'), '同步用户额度');
  assert.equal(getQuotaReasonLabel('checkin_reward'), '签到奖励');
  assert.equal(getQuotaReasonLabel('aff_quota_transfer'), '推广返佣');
  assert.equal(getQuotaReasonLabel('user_register'), '新用户注册赠送');
  assert.equal(getQuotaReasonLabel('invitee_register'), '邀请注册赠送');
  assert.equal(getQuotaReasonLabel('wallet_preconsume'), '钱包预扣');
  assert.equal(getQuotaReasonLabel('wallet_settle_consume'), '钱包结算扣费');
  assert.equal(getQuotaReasonLabel('wallet_settle_refund'), '钱包结算退款');
  assert.equal(getQuotaReasonLabel('wallet_refund'), '钱包退款');
  assert.equal(getQuotaReasonLabel('midjourney_refund'), '绘图任务退款');
  assert.equal(getQuotaReasonLabel('stripe'), 'Stripe 支付');
  assert.equal(getQuotaReasonLabel('creem'), 'Creem 支付');
  assert.equal(getQuotaReasonLabel('waffo'), 'Waffo 支付');
  assert.equal(getQuotaReasonLabel('epay'), '易支付');
  assert.equal(getQuotaReasonLabel('alipay'), '支付宝');
  assert.equal(getQuotaReasonLabel('wxpay'), '微信支付');
  assert.equal(getQuotaReasonLabel('qqpay'), 'QQ支付');
  assert.equal(getQuotaReasonLabel('unknown_reason'), 'unknown_reason');
  assert.equal(getQuotaReasonLabel(''), '-');
});
