export const normalizeUserPageData = (data, fallbackPage = 1) => {
  const items = Array.isArray(data?.items) ? data.items : [];
  const page =
    Number.isInteger(data?.page) && data.page > 0 ? data.page : fallbackPage;
  const total =
    typeof data?.total === 'number' && Number.isFinite(data.total)
      ? data.total
      : items.length;

  return {
    items,
    page,
    total,
  };
};

export const toGroupOptions = (payload) => {
  if (!payload?.success || !Array.isArray(payload?.data)) {
    return [];
  }

  return payload.data.map((group) => ({
    label: group,
    value: group,
  }));
};
