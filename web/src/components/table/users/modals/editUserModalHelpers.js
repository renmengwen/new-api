export const QUOTA_ADJUST_MODE = {
  increase: 'increase',
  decrease: 'decrease',
};

const normalizeInteger = (value) => Math.abs(parseInt(value, 10) || 0);

export const normalizePositiveAdjustmentValue = (value) => {
  if (value === '' || value === null || value === undefined) {
    return '';
  }
  const normalizedValue = Math.abs(Number(value));
  return Number.isNaN(normalizedValue) ? '' : normalizedValue;
};

export const calculateAdjustedQuota = (
  currentQuota,
  adjustmentQuota,
  mode = QUOTA_ADJUST_MODE.increase,
) => {
  const normalizedCurrent = normalizeInteger(currentQuota);
  const normalizedAdjustment = normalizeInteger(adjustmentQuota);

  if (mode === QUOTA_ADJUST_MODE.decrease) {
    return normalizedCurrent - normalizedAdjustment;
  }

  return normalizedCurrent + normalizedAdjustment;
};

export const shouldDisableQuotaAdjustmentConfirm = (
  currentQuota,
  adjustmentQuota,
  mode = QUOTA_ADJUST_MODE.increase,
) => {
  const normalizedAdjustment = normalizeInteger(adjustmentQuota);
  if (normalizedAdjustment <= 0) {
    return true;
  }

  return calculateAdjustedQuota(currentQuota, normalizedAdjustment, mode) < 0;
};

export const shouldDisableQuotaInput = (isEdit) => Boolean(isEdit);
