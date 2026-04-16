import { MAX_EXCEL_EXPORT_ROWS } from '../../helpers/exportExcel.js';

const normalizeText = (value) => String(value ?? '');

const normalizeLogType = (value) => {
  const parsedValue = Number.parseInt(value ?? 0, 10);
  return Number.isNaN(parsedValue) ? 0 : parsedValue;
};

const toUnixTimestamp = (value) => {
  const parsedValue = Date.parse(value);
  if (Number.isNaN(parsedValue)) {
    return 0;
  }
  return Math.floor(parsedValue / 1000);
};

const getQuotaDisplayType = () => {
  if (typeof localStorage === 'undefined') {
    return 'USD';
  }
  return localStorage.getItem('quota_display_type') || 'USD';
};

export const createUsageLogCommittedQuery = (
  values = {},
  fallbackDateRange = [],
) => {
  const dateRange =
    Array.isArray(values.dateRange) && values.dateRange.length === 2
      ? values.dateRange
      : [];
  const fallbackRange =
    Array.isArray(fallbackDateRange) && fallbackDateRange.length === 2
      ? fallbackDateRange
      : [];
  const startTimestamp =
    dateRange[0] || values.start_timestamp || fallbackRange[0] || '';
  const endTimestamp =
    dateRange[1] || values.end_timestamp || fallbackRange[1] || '';

  return {
    username: normalizeText(values.username),
    token_name: normalizeText(values.token_name),
    model_name: normalizeText(values.model_name),
    start_timestamp: normalizeText(startTimestamp),
    end_timestamp: normalizeText(endTimestamp),
    channel: normalizeText(values.channel),
    group: normalizeText(values.group),
    request_id: normalizeText(values.request_id),
    logType: normalizeLogType(values.logType),
  };
};

export const buildUsageLogExportRequest = ({
  committedQuery,
  visibleColumnKeys,
}) => ({
  type: committedQuery?.logType ?? 0,
  username: normalizeText(committedQuery?.username),
  token_name: normalizeText(committedQuery?.token_name),
  model_name: normalizeText(committedQuery?.model_name),
  start_timestamp: toUnixTimestamp(committedQuery?.start_timestamp),
  end_timestamp: toUnixTimestamp(committedQuery?.end_timestamp),
  channel: normalizeText(committedQuery?.channel),
  group: normalizeText(committedQuery?.group),
  request_id: normalizeText(committedQuery?.request_id),
  quota_display_type: normalizeText(getQuotaDisplayType()),
  column_keys: Array.isArray(visibleColumnKeys) ? visibleColumnKeys : [],
  limit: MAX_EXCEL_EXPORT_ROWS,
});

export const getVisibleUsageLogColumnKeys = ({
  allColumns,
  visibleColumns,
}) => {
  const columns = Array.isArray(allColumns) ? allColumns : [];
  const visibility = visibleColumns || {};

  return columns
    .filter((column) => column?.key && visibility[column.key])
    .map((column) => column.key);
};
