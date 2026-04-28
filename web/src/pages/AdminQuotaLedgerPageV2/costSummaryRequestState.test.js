import test from 'node:test';
import assert from 'node:assert/strict';
import * as requestStateHelpers from './costSummaryRequestState.js';

const defaultFilters = {
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
};

test('cost summary query state starts with default filters and pagination', () => {
  assert.equal(typeof requestStateHelpers.createDefaultCostSummaryFilters, 'function');
  assert.equal(typeof requestStateHelpers.createCostSummaryQueryState, 'function');

  assert.deepEqual(
    requestStateHelpers.createDefaultCostSummaryFilters(),
    defaultFilters,
  );
  assert.deepEqual(requestStateHelpers.createCostSummaryQueryState(), {
    draftFilters: defaultFilters,
    committedRequest: {
      page: 1,
      pageSize: 10,
      ...defaultFilters,
    },
  });
});

test('cost summary draft filter edits do not change the committed request until committed', () => {
  assert.equal(typeof requestStateHelpers.updateCostSummaryDraftFilters, 'function');
  assert.equal(typeof requestStateHelpers.commitCostSummaryFilters, 'function');

  const initialState = requestStateHelpers.createCostSummaryQueryState({
    committedRequest: {
      page: 3,
      pageSize: 20,
      startTimestamp: '1704067200',
      endTimestamp: '1706745599',
      modelName: 'gpt-4o-mini',
      vendor: 'openai',
      user: 'alice',
      tokenName: 'prod-token',
      channel: 'primary',
      group: 'default',
      minCallCount: '5',
      minPaidUsd: '1.25',
      sortBy: 'paidUsd',
      sortOrder: 'asc',
    },
  });

  const editedState = requestStateHelpers.updateCostSummaryDraftFilters(
    initialState,
    {
      modelName: 'claude-3-5-sonnet',
      vendor: 'anthropic',
      minCallCount: '0',
      minPaidUsd: '0',
      sortBy: 'callCount',
      sortOrder: 'desc',
    },
  );

  assert.deepEqual(editedState.draftFilters, {
    startTimestamp: '1704067200',
    endTimestamp: '1706745599',
    modelName: 'claude-3-5-sonnet',
    vendor: 'anthropic',
    user: 'alice',
    tokenName: 'prod-token',
    channel: 'primary',
    group: 'default',
    minCallCount: '0',
    minPaidUsd: '0',
    sortBy: 'callCount',
    sortOrder: 'desc',
  });
  assert.deepEqual(editedState.committedRequest, {
    page: 3,
    pageSize: 20,
    startTimestamp: '1704067200',
    endTimestamp: '1706745599',
    modelName: 'gpt-4o-mini',
    vendor: 'openai',
    user: 'alice',
    tokenName: 'prod-token',
    channel: 'primary',
    group: 'default',
    minCallCount: '5',
    minPaidUsd: '1.25',
    sortBy: 'paidUsd',
    sortOrder: 'asc',
  });

  const committedState =
    requestStateHelpers.commitCostSummaryFilters(editedState);
  assert.deepEqual(committedState.committedRequest, {
    page: 1,
    pageSize: 20,
    startTimestamp: '1704067200',
    endTimestamp: '1706745599',
    modelName: 'claude-3-5-sonnet',
    vendor: 'anthropic',
    user: 'alice',
    tokenName: 'prod-token',
    channel: 'primary',
    group: 'default',
    minCallCount: '0',
    minPaidUsd: '0',
    sortBy: 'callCount',
    sortOrder: 'desc',
  });
});

test('cost summary refresh, page changes, and page size changes use committed filters', () => {
  assert.equal(
    typeof requestStateHelpers.getCostSummaryRefreshRequestState,
    'function',
  );
  assert.equal(
    typeof requestStateHelpers.changeCostSummaryCommittedPage,
    'function',
  );
  assert.equal(
    typeof requestStateHelpers.changeCostSummaryCommittedPageSize,
    'function',
  );

  const initialState = requestStateHelpers.createCostSummaryQueryState({
    committedRequest: {
      page: 2,
      pageSize: 10,
      modelName: 'gpt-4o-mini',
      vendor: 'openai',
      sortBy: 'paidUsd',
      sortOrder: 'asc',
    },
  });
  const editedState = requestStateHelpers.updateCostSummaryDraftFilters(
    initialState,
    {
      modelName: 'claude-3-5-sonnet',
      vendor: 'anthropic',
      sortBy: 'callCount',
    },
  );

  assert.deepEqual(
    requestStateHelpers.getCostSummaryRefreshRequestState(editedState),
    {
      page: 2,
      pageSize: 10,
      ...defaultFilters,
      modelName: 'gpt-4o-mini',
      vendor: 'openai',
      sortBy: 'paidUsd',
      sortOrder: 'asc',
    },
  );

  const pagedState = requestStateHelpers.changeCostSummaryCommittedPage(
    editedState,
    5,
  );
  assert.deepEqual(pagedState.committedRequest, {
    page: 5,
    pageSize: 10,
    ...defaultFilters,
    modelName: 'gpt-4o-mini',
    vendor: 'openai',
    sortBy: 'paidUsd',
    sortOrder: 'asc',
  });

  const resizedState =
    requestStateHelpers.changeCostSummaryCommittedPageSize(editedState, 30);
  assert.deepEqual(resizedState.committedRequest, {
    page: 1,
    pageSize: 30,
    ...defaultFilters,
    modelName: 'gpt-4o-mini',
    vendor: 'openai',
    sortBy: 'paidUsd',
    sortOrder: 'asc',
  });
});

test('cost summary reset clears filters and preserves the committed page size', () => {
  assert.equal(typeof requestStateHelpers.resetCostSummaryFilters, 'function');

  const initialState = requestStateHelpers.createCostSummaryQueryState({
    committedRequest: {
      page: 4,
      pageSize: 50,
      modelName: 'gpt-4o-mini',
      vendor: 'openai',
      user: 'alice',
      sortBy: 'paidUsd',
      sortOrder: 'asc',
    },
    draftFilters: {
      modelName: 'claude-3-5-sonnet',
      vendor: 'anthropic',
      user: 'bob',
      tokenName: 'test-token',
      minPaidUsd: '0',
      sortBy: 'callCount',
      sortOrder: 'desc',
    },
  });

  const resetState = requestStateHelpers.resetCostSummaryFilters(initialState);

  assert.deepEqual(resetState.draftFilters, defaultFilters);
  assert.deepEqual(resetState.committedRequest, {
    page: 1,
    pageSize: 50,
    ...defaultFilters,
  });
});
