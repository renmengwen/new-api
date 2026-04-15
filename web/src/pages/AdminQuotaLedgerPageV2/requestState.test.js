import test from 'node:test';
import assert from 'node:assert/strict';
import * as requestStateHelpers from './requestState.js';

test('quota ledger draft filter edits do not change the committed request until search is submitted', () => {
  assert.equal(typeof requestStateHelpers.createQuotaLedgerQueryState, 'function');
  assert.equal(typeof requestStateHelpers.updateDraftFilters, 'function');
  assert.equal(typeof requestStateHelpers.commitDraftFilters, 'function');

  const initialState = requestStateHelpers.createQuotaLedgerQueryState({
    committedRequest: {
      page: 3,
      pageSize: 20,
      userId: '2001',
      entryType: 'adjust',
    },
  });

  const editedState = requestStateHelpers.updateDraftFilters(initialState, {
    userId: '3002',
    entryType: 'topup',
  });

  assert.deepEqual(editedState.draftFilters, {
    userId: '3002',
    entryType: 'topup',
  });
  assert.deepEqual(editedState.committedRequest, {
    page: 3,
    pageSize: 20,
    userId: '2001',
    entryType: 'adjust',
  });

  const searchedState = requestStateHelpers.commitDraftFilters(editedState);
  assert.deepEqual(searchedState.committedRequest, {
    page: 1,
    pageSize: 20,
    userId: '3002',
    entryType: 'topup',
  });
});

test('quota ledger refresh, page change, and page size change derive from the committed request instead of draft filters', () => {
  assert.equal(typeof requestStateHelpers.getRefreshRequestState, 'function');
  assert.equal(typeof requestStateHelpers.changeCommittedPage, 'function');
  assert.equal(typeof requestStateHelpers.changeCommittedPageSize, 'function');

  const initialState = requestStateHelpers.createQuotaLedgerQueryState({
    committedRequest: {
      page: 2,
      pageSize: 10,
      userId: '2001',
      entryType: 'adjust',
    },
  });

  const editedState = requestStateHelpers.updateDraftFilters(initialState, {
    userId: '3002',
    entryType: 'topup',
  });

  assert.deepEqual(requestStateHelpers.getRefreshRequestState(editedState), {
    page: 2,
    pageSize: 10,
    userId: '2001',
    entryType: 'adjust',
  });

  const pagedState = requestStateHelpers.changeCommittedPage(editedState, 5);
  assert.deepEqual(pagedState.committedRequest, {
    page: 5,
    pageSize: 10,
    userId: '2001',
    entryType: 'adjust',
  });

  const resizedState = requestStateHelpers.changeCommittedPageSize(editedState, 30);
  assert.deepEqual(resizedState.committedRequest, {
    page: 1,
    pageSize: 30,
    userId: '2001',
    entryType: 'adjust',
  });
});

test('quota ledger reset clears both draft and committed filters while preserving the committed page size', () => {
  assert.equal(typeof requestStateHelpers.resetDraftAndCommittedFilters, 'function');

  const initialState = requestStateHelpers.createQuotaLedgerQueryState({
    committedRequest: {
      page: 4,
      pageSize: 50,
      userId: '2001',
      entryType: 'adjust',
    },
    draftFilters: {
      userId: '3002',
      entryType: 'topup',
    },
  });

  const resetState = requestStateHelpers.resetDraftAndCommittedFilters(initialState);

  assert.deepEqual(resetState.draftFilters, {
    userId: '',
    entryType: '',
  });
  assert.deepEqual(resetState.committedRequest, {
    page: 1,
    pageSize: 50,
    userId: '',
    entryType: '',
  });
});
