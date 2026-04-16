/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useState } from 'react';
import dayjs from 'dayjs';
import {
  API,
  MAX_EXCEL_EXPORT_ROWS,
  downloadExcelBlob,
  showError,
} from '../../helpers';

const DEFAULT_ACTIVE_TAB = 'models';
const DEFAULT_DATE_PRESET = 'last7days';
const FIXED_ANALYTICS_TIMEZONE_OFFSET_MINUTES = 8 * 60;
const FIXED_ANALYTICS_TIMEZONE_OFFSET_SECONDS =
  FIXED_ANALYTICS_TIMEZONE_OFFSET_MINUTES * 60;

const createEmptySummary = () => ({
  total_calls: 0,
  total_cost: 0,
  active_users: 0,
  active_models: 0,
  wow: {},
});

const createDefaultDraftFilters = () => ({
  modelKeyword: '',
  usernameKeyword: '',
  startDate: null,
  endDate: null,
});

const filterKeywordStateByActiveTab = (filters, activeTab) => ({
  ...filters,
  modelKeyword: activeTab === 'models' ? (filters.modelKeyword || '').trim() : '',
  usernameKeyword:
    activeTab === 'users' ? (filters.usernameKeyword || '').trim() : '',
});

const createAppliedFilters = (
  activeTab,
  datePreset = DEFAULT_DATE_PRESET,
  draftFilters = createDefaultDraftFilters(),
) =>
  filterKeywordStateByActiveTab(
    {
      datePreset,
      modelKeyword: draftFilters.modelKeyword.trim(),
      usernameKeyword: draftFilters.usernameKeyword.trim(),
      startDate: datePreset === 'custom' ? draftFilters.startDate : null,
      endDate: datePreset === 'custom' ? draftFilters.endDate : null,
    },
    activeTab,
  );

const serializeFixedUtc8DayTimestamp = (value, boundary) => {
  if (!value) {
    return undefined;
  }

  const dayValue = dayjs(value);
  const utcDayStart = Date.UTC(
    dayValue.year(),
    dayValue.month(),
    dayValue.date(),
  ) / 1000;

  return boundary === 'end'
    ? utcDayStart - FIXED_ANALYTICS_TIMEZONE_OFFSET_SECONDS + 24 * 60 * 60 - 1
    : utcDayStart - FIXED_ANALYTICS_TIMEZONE_OFFSET_SECONDS;
};

const validateFilters = (filters, t) => {
  if (filters.datePreset !== 'custom') {
    return '';
  }

  if (!filters.startDate || !filters.endDate) {
    return t('请选择开始日期和结束日期');
  }

  if (dayjs(filters.endDate).isBefore(dayjs(filters.startDate), 'day')) {
    return t('结束日期不能早于开始日期');
  }

  return '';
};

const getQuotaDisplayType = () => {
  if (typeof localStorage === 'undefined') {
    return 'USD';
  }
  return localStorage.getItem('quota_display_type') || 'USD';
};

export const buildOperationsAnalyticsSummaryParams = (appliedFilters) => {
  const params = {
    date_preset: appliedFilters.datePreset,
  };

  if (appliedFilters.datePreset === 'custom') {
    params.start_timestamp = serializeFixedUtc8DayTimestamp(
      appliedFilters.startDate,
      'start',
    );
    params.end_timestamp = serializeFixedUtc8DayTimestamp(
      appliedFilters.endDate,
      'end',
    );
  }

  if (appliedFilters.modelKeyword) {
    params.model_keyword = appliedFilters.modelKeyword;
  }

  if (appliedFilters.usernameKeyword) {
    params.username_keyword = appliedFilters.usernameKeyword;
  }

  return params;
};

export const buildOperationsAnalyticsExportPayload = ({
  activeTab,
  datePreset,
  filters,
  sortState,
}) => {
  const payload = {
    view: activeTab,
    date_preset: datePreset,
    limit: MAX_EXCEL_EXPORT_ROWS,
  };
  payload.quota_display_type = getQuotaDisplayType();

  if (datePreset === 'custom') {
    payload.start_timestamp = serializeFixedUtc8DayTimestamp(
      filters.startDate,
      'start',
    );
    payload.end_timestamp = serializeFixedUtc8DayTimestamp(
      filters.endDate,
      'end',
    );
  }

  if (filters.modelKeyword) {
    payload.model_keyword = filters.modelKeyword;
  }

  if (filters.usernameKeyword) {
    payload.username_keyword = filters.usernameKeyword;
  }

  if (sortState?.sortBy) {
    payload.sort_by = sortState.sortBy;
  }

  if (sortState?.sortOrder) {
    payload.sort_order = sortState.sortOrder;
  }

  return payload;
};

export const useOperationsAnalyticsData = ({
  canRead,
  canExport,
  sortStateByTab = {},
  t,
}) => {
  const [activeTab, setActiveTab] = useState('models');
  const [datePreset, setDatePreset] = useState('last7days');
  const [draftFilters, setDraftFilters] = useState(() => createDefaultDraftFilters());
  const [appliedFilters, setAppliedFilters] = useState(() =>
    createAppliedFilters(DEFAULT_ACTIVE_TAB),
  );
  const [summary, setSummary] = useState(() => createEmptySummary());
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [summaryError, setSummaryError] = useState('');
  const [exportLoading, setExportLoading] = useState(false);

  useEffect(() => {
    setAppliedFilters((currentFilters) => {
      const nextAppliedFilters = filterKeywordStateByActiveTab(
        currentFilters,
        activeTab,
      );

      if (
        currentFilters.modelKeyword === nextAppliedFilters.modelKeyword &&
        currentFilters.usernameKeyword === nextAppliedFilters.usernameKeyword
      ) {
        return currentFilters;
      }

      return nextAppliedFilters;
    });

    setDraftFilters((currentFilters) => {
      const nextDraftFilters = filterKeywordStateByActiveTab(
        currentFilters,
        activeTab,
      );

      if (
        currentFilters.modelKeyword === nextDraftFilters.modelKeyword &&
        currentFilters.usernameKeyword === nextDraftFilters.usernameKeyword
      ) {
        return currentFilters;
      }

      return nextDraftFilters;
    });
  }, [activeTab]);

  useEffect(() => {
    if (!canRead) {
      setSummary(createEmptySummary());
      setSummaryError('');
      setSummaryLoading(false);
      return undefined;
    }

    let disposed = false;

    const loadSummary = async () => {
      setSummaryLoading(true);
      setSummaryError('');

      try {
        const res = await API.get('/api/admin/analytics/summary', {
          params: buildOperationsAnalyticsSummaryParams(appliedFilters),
        });

        if (disposed) {
          return;
        }

        if (!res.data.success) {
          setSummary(createEmptySummary());
          setSummaryError(res.data.message || t('加载汇总数据失败'));
          return;
        }

        setSummary({
          ...createEmptySummary(),
          ...(res.data.data || {}),
        });
      } catch (error) {
        if (disposed) {
          return;
        }

        setSummary(createEmptySummary());
        setSummaryError(t('加载汇总数据失败，请稍后重试'));
        showError(error);
      } finally {
        if (!disposed) {
          setSummaryLoading(false);
        }
      }
    };

    loadSummary();

    return () => {
      disposed = true;
    };
  }, [appliedFilters, canRead, t]);

  const updateDraftFilter = (field, value) => {
    setDraftFilters((current) => ({
      ...current,
      [field]: value,
    }));
  };

  const applyFilters = () => {
    const nextAppliedFilters = createAppliedFilters(
      activeTab,
      datePreset,
      draftFilters,
    );
    const validationMessage = validateFilters(nextAppliedFilters, t);

    if (validationMessage) {
      showError(validationMessage);
      return false;
    }

    setAppliedFilters(nextAppliedFilters);
    return true;
  };

  const resetFilters = () => {
    const nextDraftFilters = createDefaultDraftFilters();
    setDatePreset(DEFAULT_DATE_PRESET);
    setDraftFilters(nextDraftFilters);
    setAppliedFilters(
      createAppliedFilters(DEFAULT_ACTIVE_TAB, DEFAULT_DATE_PRESET, nextDraftFilters),
    );
  };

  const exportAnalytics = async () => {
    if (!canExport || exportLoading) {
      return;
    }

    const validationMessage = validateFilters(appliedFilters, t);

    if (validationMessage) {
      showError(validationMessage);
      return;
    }

    setExportLoading(true);
    try {
      await downloadExcelBlob({
        url: '/api/admin/analytics/export',
        payload: buildOperationsAnalyticsExportPayload({
          activeTab,
          datePreset: appliedFilters.datePreset,
          filters: appliedFilters,
          sortState: sortStateByTab[activeTab],
        }),
        fallbackFileName: `operations-analytics-${activeTab}.xlsx`,
      });
    } catch (error) {
      showError(error);
    } finally {
      setExportLoading(false);
    }
  };

  return {
    activeTab,
    setActiveTab,
    datePreset,
    setDatePreset,
    draftFilters,
    updateDraftFilter,
    appliedFilters,
    applyFilters,
    resetFilters,
    summary,
    summaryLoading,
    summaryError,
    exportLoading,
    exportAnalytics,
  };
};

export const operationsAnalyticsDefaults = {
  DEFAULT_ACTIVE_TAB,
  DEFAULT_DATE_PRESET,
};

export default useOperationsAnalyticsData;
