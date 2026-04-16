import test from 'node:test';
import assert from 'node:assert/strict';
import { MAX_EXCEL_EXPORT_ROWS } from '../../helpers/exportExcel.js';
import {
  buildUsageLogExportRequest,
  createUsageLogCommittedQuery,
  getVisibleUsageLogColumnKeys,
} from './exportState.js';

test('createUsageLogCommittedQuery normalizes submitted usage log filters into a committed snapshot', () => {
  const committedQuery = createUsageLogCommittedQuery({
    username: 'alice',
    token_name: 'demo-token',
    model_name: 'gpt-4.1',
    channel: '18',
    group: 'default',
    request_id: 'req_123',
    dateRange: ['2026-04-15 00:00:00', '2026-04-16 00:00:00'],
    logType: '0',
  });

  assert.deepEqual(committedQuery, {
    username: 'alice',
    token_name: 'demo-token',
    model_name: 'gpt-4.1',
    start_timestamp: '2026-04-15 00:00:00',
    end_timestamp: '2026-04-16 00:00:00',
    channel: '18',
    group: 'default',
    request_id: 'req_123',
    logType: 0,
  });
});

test('createUsageLogCommittedQuery falls back to the default date range when the current date range is cleared', () => {
  const committedQuery = createUsageLogCommittedQuery(
    {
      username: 'alice',
      token_name: 'demo-token',
      dateRange: [],
      logType: '2',
    },
    ['2026-04-15 00:00:00', '2026-04-16 01:00:00'],
  );

  assert.deepEqual(committedQuery, {
    username: 'alice',
    token_name: 'demo-token',
    model_name: '',
    start_timestamp: '2026-04-15 00:00:00',
    end_timestamp: '2026-04-16 01:00:00',
    channel: '',
    group: '',
    request_id: '',
    logType: 2,
  });
});

test('buildUsageLogExportRequest uses the committed query snapshot, visible column order, and the shared export cap', () => {
  const exportRequest = buildUsageLogExportRequest({
    committedQuery: {
      username: 'alice',
      token_name: 'demo-token',
      model_name: 'gpt-4.1',
      start_timestamp: '2026-04-15T00:00:00.000Z',
      end_timestamp: '2026-04-16T12:34:56.000Z',
      channel: '18',
      group: 'default',
      request_id: 'req_123',
      logType: 2,
    },
    visibleColumnKeys: ['time', 'model', 'details'],
  });

  assert.deepEqual(exportRequest, {
    type: 2,
    username: 'alice',
    token_name: 'demo-token',
    model_name: 'gpt-4.1',
    start_timestamp: 1776211200,
    end_timestamp: 1776342896,
    channel: '18',
    group: 'default',
    request_id: 'req_123',
    quota_display_type: 'USD',
    column_keys: ['time', 'model', 'details'],
    limit: MAX_EXCEL_EXPORT_ROWS,
  });
});

test('getVisibleUsageLogColumnKeys preserves the current visible column order', () => {
  const visibleColumnKeys = getVisibleUsageLogColumnKeys({
    allColumns: [
      { key: 'time' },
      { key: 'channel' },
      { key: 'username' },
      { key: 'model' },
      { key: 'details' },
    ],
    visibleColumns: {
      time: true,
      channel: false,
      username: true,
      model: true,
      details: false,
    },
  });

  assert.deepEqual(visibleColumnKeys, ['time', 'username', 'model']);
});
