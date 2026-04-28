const DEFAULT_PAGE = 1;
const DEFAULT_PAGE_SIZE = 10;

export const createDefaultCostSummaryFilters = () => ({
  startTimestamp: '',
  endTimestamp: '',
  modelName: '',
  vendor: '',
  user: '',
  tokenName: '',
  channel: '',
  group: '',
  minCallCount: '',
  minPaidUsd: '',
  sortBy: 'date',
  sortOrder: 'desc',
});

const normalizeFilters = (filters = {}) => {
  const defaultFilters = createDefaultCostSummaryFilters();

  return {
    startTimestamp: filters.startTimestamp ?? defaultFilters.startTimestamp,
    endTimestamp: filters.endTimestamp ?? defaultFilters.endTimestamp,
    modelName: filters.modelName ?? defaultFilters.modelName,
    vendor: filters.vendor ?? defaultFilters.vendor,
    user: filters.user ?? defaultFilters.user,
    tokenName: filters.tokenName ?? defaultFilters.tokenName,
    channel: filters.channel ?? defaultFilters.channel,
    group: filters.group ?? defaultFilters.group,
    minCallCount: filters.minCallCount ?? defaultFilters.minCallCount,
    minPaidUsd: filters.minPaidUsd ?? defaultFilters.minPaidUsd,
    sortBy: filters.sortBy ?? defaultFilters.sortBy,
    sortOrder: filters.sortOrder ?? defaultFilters.sortOrder,
  };
};

const normalizeCommittedRequest = (request = {}) => ({
  page: request.page ?? DEFAULT_PAGE,
  pageSize: request.pageSize ?? DEFAULT_PAGE_SIZE,
  ...normalizeFilters(request),
});

export const createCostSummaryQueryState = (state = {}) => {
  const committedRequest = normalizeCommittedRequest(state.committedRequest);

  return {
    draftFilters: normalizeFilters(state.draftFilters ?? committedRequest),
    committedRequest,
  };
};

export const updateCostSummaryDraftFilters = (
  state,
  nextDraftFilters = {},
) => ({
  ...state,
  draftFilters: normalizeFilters({
    ...state.draftFilters,
    ...nextDraftFilters,
  }),
});

export const commitCostSummaryFilters = (state) => ({
  ...state,
  committedRequest: normalizeCommittedRequest({
    ...state.committedRequest,
    ...state.draftFilters,
    page: DEFAULT_PAGE,
  }),
});

export const resetCostSummaryFilters = (state) => {
  const draftFilters = createDefaultCostSummaryFilters();

  return {
    ...state,
    draftFilters,
    committedRequest: normalizeCommittedRequest({
      ...state.committedRequest,
      ...draftFilters,
      page: DEFAULT_PAGE,
    }),
  };
};

export const getCostSummaryRefreshRequestState = (state) =>
  normalizeCommittedRequest(state.committedRequest);

export const changeCostSummaryCommittedPage = (state, nextPage) => ({
  ...state,
  committedRequest: normalizeCommittedRequest({
    ...state.committedRequest,
    page: nextPage,
  }),
});

export const changeCostSummaryCommittedPageSize = (state, nextPageSize) => ({
  ...state,
  committedRequest: normalizeCommittedRequest({
    ...state.committedRequest,
    page: DEFAULT_PAGE,
    pageSize: nextPageSize,
  }),
});
