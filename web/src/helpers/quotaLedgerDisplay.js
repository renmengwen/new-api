export const QUOTA_LEDGER_ENTRY_TYPE_LABELS = {
  adjust: '调额',
  recharge: '充值',
  reclaim: '回收',
  consume: '消耗',
  refund: '退款',
  reward: '奖励',
  commission: '佣金',
};

export const QUOTA_LEDGER_ENTRY_TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
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

export const getQuotaAccountName = (item) => item?.account_username || '-';

export const getQuotaOperatorName = (item) => item?.operator_username || '-';
