export const QUOTA_LEDGER_ENTRY_TYPE_LABELS = {
  opening: '初始额度',
  adjust: '调额',
  recharge: '充值',
  reclaim: '回收',
  consume: '消耗',
  refund: '退款',
  reward: '奖励',
  commission: '佣金',
};

export const QUOTA_LEDGER_REASON_LABELS = {
  manual_adjust: '手动调额',
  legacy_user_update: '手动调额',
  managed_user_update: '管理用户调额',
  batch_adjust: '批量调额',
  batch_partial_adjust: '批量部分调额',
  agent_adjust: '代理商调额',
  agent_reclaim: '代理商回收',
  agent_batch_adjust: '代理商批量调额',
  sync_with_user_quota: '同步用户额度',
  checkin_reward: '签到奖励',
  aff_quota_transfer: '推广返佣',
  user_register: '新用户注册赠送',
  invitee_register: '邀请注册赠送',
  wallet_preconsume: '钱包预扣',
  wallet_settle_consume: '钱包结算扣费',
  wallet_settle_refund: '钱包结算退款',
  wallet_refund: '钱包退款',
  post_consume_quota: '后置扣费',
  post_consume_refund: '后置退款',
  task_adjust_consume: '任务重算扣费',
  task_adjust_refund: '任务重算退款',
  midjourney_refund: '绘图任务退款',
  stripe: 'Stripe 支付',
  creem: 'Creem 支付',
  waffo: 'Waffo 支付',
  epay: '易支付',
  alipay: '支付宝',
  wxpay: '微信支付',
  qqpay: 'QQ支付',
  wechat: '微信支付',
};

export const QUOTA_LEDGER_ENTRY_TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: '初始额度', value: 'opening' },
  { label: '调额', value: 'adjust' },
  { label: '充值', value: 'recharge' },
  { label: '回收', value: 'reclaim' },
  { label: '消耗', value: 'consume' },
  { label: '退款', value: 'refund' },
  { label: '奖励', value: 'reward' },
  { label: '佣金', value: 'commission' },
];

export const getQuotaEntryTypeLabel = (entryType) => {
  if (!entryType) {
    return '-';
  }
  return QUOTA_LEDGER_ENTRY_TYPE_LABELS[entryType] || entryType;
};

export const getQuotaReasonLabel = (reason) => {
  if (!reason) {
    return '-';
  }
  return QUOTA_LEDGER_REASON_LABELS[reason] || reason;
};

export const getQuotaAccountName = (item) => item?.account_username || '-';

export const getQuotaOperatorName = (item) => item?.operator_username || '-';
