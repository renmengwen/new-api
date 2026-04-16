import test from 'node:test';
import assert from 'node:assert/strict';
import * as requestStateHelpers from './requestState.js';

test('draft filter edits do not change the committed request until search is submitted', () => {
  assert.equal(typeof requestStateHelpers.createAuditLogQueryState, 'function');
  assert.equal(typeof requestStateHelpers.updateDraftFilters, 'function');
  assert.equal(typeof requestStateHelpers.commitDraftFilters, 'function');

  const initialState = requestStateHelpers.createAuditLogQueryState({
    committedRequest: {
      page: 3,
      pageSize: 20,
      actionModule: 'quota',
      operatorUserId: '18',
    },
  });

  const editedState = requestStateHelpers.updateDraftFilters(initialState, {
    actionModule: 'agent',
    operatorUserId: '99',
  });

  assert.deepEqual(editedState.draftFilters, {
    actionModule: 'agent',
    operatorUserId: '99',
  });
  assert.deepEqual(editedState.committedRequest, {
    page: 3,
    pageSize: 20,
    actionModule: 'quota',
    operatorUserId: '18',
  });

  const searchedState = requestStateHelpers.commitDraftFilters(editedState);
  assert.deepEqual(searchedState.committedRequest, {
    page: 1,
    pageSize: 20,
    actionModule: 'agent',
    operatorUserId: '99',
  });
});

test('refresh, page change, and page size change derive from the committed request instead of draft filters', () => {
  assert.equal(typeof requestStateHelpers.getRefreshRequestState, 'function');
  assert.equal(typeof requestStateHelpers.changeCommittedPage, 'function');
  assert.equal(typeof requestStateHelpers.changeCommittedPageSize, 'function');

  const initialState = requestStateHelpers.createAuditLogQueryState({
    committedRequest: {
      page: 2,
      pageSize: 10,
      actionModule: 'quota',
      operatorUserId: '18',
    },
  });

  const editedState = requestStateHelpers.updateDraftFilters(initialState, {
    actionModule: 'agent',
    operatorUserId: '77',
  });

  assert.deepEqual(requestStateHelpers.getRefreshRequestState(editedState), {
    page: 2,
    pageSize: 10,
    actionModule: 'quota',
    operatorUserId: '18',
  });

  const pagedState = requestStateHelpers.changeCommittedPage(editedState, 5);
  assert.deepEqual(pagedState.committedRequest, {
    page: 5,
    pageSize: 10,
    actionModule: 'quota',
    operatorUserId: '18',
  });

  const resizedState = requestStateHelpers.changeCommittedPageSize(editedState, 30);
  assert.deepEqual(resizedState.committedRequest, {
    page: 1,
    pageSize: 30,
    actionModule: 'quota',
    operatorUserId: '18',
  });
});

test('reset clears both draft and committed filters while preserving the committed page size', () => {
  assert.equal(typeof requestStateHelpers.resetDraftAndCommittedFilters, 'function');

  const initialState = requestStateHelpers.createAuditLogQueryState({
    committedRequest: {
      page: 4,
      pageSize: 50,
      actionModule: 'quota',
      operatorUserId: '18',
    },
    draftFilters: {
      actionModule: 'agent',
      operatorUserId: '99',
    },
  });

  const resetState = requestStateHelpers.resetDraftAndCommittedFilters(initialState);

  assert.deepEqual(resetState.draftFilters, {
    actionModule: '',
    operatorUserId: '',
  });
  assert.deepEqual(resetState.committedRequest, {
    page: 1,
    pageSize: 50,
    actionModule: '',
    operatorUserId: '',
  });
});

test('request sequence tracker rejects stale responses and accepts only the latest request', () => {
  assert.equal(typeof requestStateHelpers.createRequestSequenceTracker, 'function');

  const tracker = requestStateHelpers.createRequestSequenceTracker();
  const slowRequestId = tracker.issue();
  const fastRequestId = tracker.issue();

  assert.equal(tracker.shouldAccept(slowRequestId), false);
  assert.equal(tracker.shouldAccept(fastRequestId), true);

  const latestRequestId = tracker.issue();
  assert.equal(tracker.shouldAccept(fastRequestId), false);
  assert.equal(tracker.shouldAccept(latestRequestId), true);
});
