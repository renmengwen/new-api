export const applyQuotaDelta = (currentQuota, delta) => {
  const normalizedCurrent = parseInt(currentQuota, 10) || 0;
  const normalizedDelta = parseInt(delta, 10) || 0;
  return normalizedCurrent + normalizedDelta;
};

export const shouldDisableQuotaInput = (isEdit) => Boolean(isEdit);
