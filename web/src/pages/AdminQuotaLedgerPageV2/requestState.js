const DEFAULT_PAGE = 1;
const DEFAULT_PAGE_SIZE = 10;

const normalizeFilters = (filters = {}) => ({
  userId: filters.userId ?? '',
  entryType: filters.entryType ?? '',
});

const normalizeCommittedRequest = (request = {}) => ({
  page: request.page ?? DEFAULT_PAGE,
  pageSize: request.pageSize ?? DEFAULT_PAGE_SIZE,
  ...normalizeFilters(request),
});

export const createQuotaLedgerQueryState = (state = {}) => {
  const committedRequest = normalizeCommittedRequest(state.committedRequest);

  return {
    draftFilters: normalizeFilters(state.draftFilters ?? committedRequest),
    committedRequest,
  };
};

export const updateDraftFilters = (state, nextDraftFilters = {}) => ({
  ...state,
  draftFilters: normalizeFilters({
    ...state.draftFilters,
    ...nextDraftFilters,
  }),
});

export const commitQuotaLedgerFilters = (state) => ({
  ...state,
  committedRequest: normalizeCommittedRequest({
    ...state.committedRequest,
    ...state.draftFilters,
    page: DEFAULT_PAGE,
  }),
});

export const commitDraftFilters = commitQuotaLedgerFilters;

export const resetDraftAndCommittedFilters = (state) => {
  const draftFilters = normalizeFilters();

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

export const getRefreshRequestState = (state) =>
  normalizeCommittedRequest(state.committedRequest);

export const changeCommittedPage = (state, nextPage) => ({
  ...state,
  committedRequest: normalizeCommittedRequest({
    ...state.committedRequest,
    page: nextPage,
  }),
});

export const changeCommittedPageSize = (state, nextPageSize) => ({
  ...state,
  committedRequest: normalizeCommittedRequest({
    ...state.committedRequest,
    page: DEFAULT_PAGE,
    pageSize: nextPageSize,
  }),
});

export const createRequestSequenceTracker = (initialRequestId = 0) => {
  let latestRequestId = initialRequestId;

  return {
    issue() {
      latestRequestId += 1;
      return latestRequestId;
    },
    shouldAccept(requestId) {
      return requestId === latestRequestId;
    },
  };
};
