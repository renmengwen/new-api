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
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Banner, Button, Empty, Pagination, Table } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { VChart } from '@visactor/react-vchart';
import { API, renderNumber, renderQuota, showError, timestamp2string } from '../../helpers';
import { buildOperationsAnalyticsSummaryParams } from '../../hooks/operations-analytics/useOperationsAnalyticsData';
import { useOperationsAnalyticsCharts } from '../../hooks/operations-analytics/useOperationsAnalyticsCharts';

const DEFAULT_PAGE_SIZE = 10;
const CHART_LIMIT = 10;

const SORT_FIELD_MAP = {
  user_id: 'user_id',
  username: 'username',
  call_count: 'call_count',
  model_count: 'model_count',
  total_tokens: 'total_tokens',
  total_cost: 'total_cost',
  last_called_at: 'last_called_at',
};

const normalizeSortOrder = (sortOrder) => {
  if (sortOrder === 'ascend') {
    return 'asc';
  }
  if (sortOrder === 'descend') {
    return 'desc';
  }
  return '';
};

const toTableSortOrder = (sortOrder) => {
  if (sortOrder === 'asc') {
    return 'ascend';
  }
  if (sortOrder === 'desc') {
    return 'descend';
  }
  return false;
};

const buildRequestParams = (appliedFilters, extraParams = {}) => {
  const params = {
    ...buildOperationsAnalyticsSummaryParams(appliedFilters),
    ...extraParams,
  };

  Object.keys(params).forEach((key) => {
    if (params[key] === undefined || params[key] === null || params[key] === '') {
      delete params[key];
    }
  });

  return params;
};

const buildUserChartLabel = (item) => item.username || `#${item.user_id}`;

const UserAnalyticsTab = ({
  activeTab,
  appliedFilters,
  sortState,
  onSortStateChange,
}) => {
  const { t } = useTranslation();
  const { chartOption, specBar } = useOperationsAnalyticsCharts({ t });

  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);
  const [costTopUsers, setCostTopUsers] = useState([]);
  const [tokenTopUsers, setTokenTopUsers] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const sortBy = sortState?.sortBy || '';
  const sortOrder = sortState?.sortOrder || '';
  const [tableError, setTableError] = useState('');
  const [chartError, setChartError] = useState('');
  const tableRequestRef = useRef(0);
  const chartRequestRef = useRef(0);

  const loadTableData = useCallback(async () => {
    if (activeTab !== 'users') {
      return;
    }

    const requestId = tableRequestRef.current + 1;
    tableRequestRef.current = requestId;

    setLoading(true);
    setTableError('');

    try {
      const tableRes = await API.get('/api/admin/analytics/users', {
        params: buildRequestParams(appliedFilters, {
          p: page,
          page_size: pageSize,
          sort_by: sortBy,
          sort_order: sortOrder,
        }),
      });

      if (tableRequestRef.current !== requestId) {
        return;
      }

      if (!tableRes?.data?.success) {
        setItems([]);
        setTotal(0);
        setTableError(tableRes?.data?.message || t('加载用户分析数据失败'));
        return;
      }

      const tableData = tableRes.data.data || {};
      setItems(
        (tableData.items || []).map((item) => ({
          ...item,
          key: item.user_id,
        })),
      );
      setTotal(tableData.total || 0);
    } catch (requestError) {
      if (tableRequestRef.current !== requestId) {
        return;
      }

      setItems([]);
      setTotal(0);
      setTableError(t('加载用户分析数据失败，请稍后重试'));
      showError(requestError);
    } finally {
      if (tableRequestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [activeTab, appliedFilters, page, pageSize, sortBy, sortOrder, t]);

  useEffect(() => {
    loadTableData();
  }, [loadTableData]);

  useEffect(() => {
    if (activeTab !== 'users') {
      tableRequestRef.current += 1;
      chartRequestRef.current += 1;
      setLoading(false);
    }
  }, [activeTab]);

  const loadChartData = useCallback(async () => {
    if (activeTab !== 'users') {
      return;
    }

    const requestId = chartRequestRef.current + 1;
    chartRequestRef.current = requestId;

    setChartError('');

    try {
      const [costTopRes, tokenTopRes] = await Promise.all([
        API.get('/api/admin/analytics/users', {
          params: buildRequestParams(appliedFilters, {
            p: 1,
            page_size: CHART_LIMIT,
            sort_by: 'total_cost',
            sort_order: 'desc',
          }),
        }),
        API.get('/api/admin/analytics/users', {
          params: buildRequestParams(appliedFilters, {
            p: 1,
            page_size: CHART_LIMIT,
            sort_by: 'total_tokens',
            sort_order: 'desc',
          }),
        }),
      ]);

      if (chartRequestRef.current !== requestId) {
        return;
      }

      const responses = [costTopRes, tokenTopRes];
      const failedResponse = responses.find((response) => !response?.data?.success);
      if (failedResponse) {
        setCostTopUsers([]);
        setTokenTopUsers([]);
        setChartError(failedResponse?.data?.message || t('加载用户分析数据失败'));
        return;
      }

      setCostTopUsers(costTopRes.data.data?.items || []);
      setTokenTopUsers(tokenTopRes.data.data?.items || []);
    } catch (requestError) {
      if (chartRequestRef.current !== requestId) {
        return;
      }

      setCostTopUsers([]);
      setTokenTopUsers([]);
      setChartError(t('加载用户分析数据失败，请稍后重试'));
      showError(requestError);
    }
  }, [activeTab, appliedFilters, t]);

  useEffect(() => {
    loadChartData();
  }, [loadChartData]);

  useEffect(() => () => {
    tableRequestRef.current += 1;
    chartRequestRef.current += 1;
  }, []);

  const tableSortOrder = toTableSortOrder(sortOrder);

  const columns = useMemo(
    () => [
      {
        title: t('用户ID'),
        dataIndex: 'user_id',
        sorter: true,
        sortOrder: sortBy === 'user_id' ? tableSortOrder : false,
        width: 110,
      },
      {
        title: t('昵称'),
        dataIndex: 'username',
        sorter: true,
        sortOrder: sortBy === 'username' ? tableSortOrder : false,
        width: 180,
        render: (value, record) => value || `#${record.user_id}`,
      },
      {
        title: t('调用次数'),
        dataIndex: 'call_count',
        sorter: true,
        sortOrder: sortBy === 'call_count' ? tableSortOrder : false,
        width: 120,
        render: (value) => renderNumber(value || 0),
      },
      {
        title: t('使用模型数'),
        dataIndex: 'model_count',
        sorter: true,
        sortOrder: sortBy === 'model_count' ? tableSortOrder : false,
        width: 130,
        render: (value) => renderNumber(value || 0),
      },
      {
        title: t('Token 总量'),
        dataIndex: 'total_tokens',
        sorter: true,
        sortOrder: sortBy === 'total_tokens' ? tableSortOrder : false,
        width: 140,
        render: (value) => renderNumber(value || 0),
      },
      {
        title: t('总费用'),
        dataIndex: 'total_cost',
        sorter: true,
        sortOrder: sortBy === 'total_cost' ? tableSortOrder : false,
        width: 130,
        render: (value) => renderQuota(value),
      },
      {
        title: t('最后调用时间'),
        dataIndex: 'last_called_at',
        sorter: true,
        sortOrder: sortBy === 'last_called_at' ? tableSortOrder : false,
        width: 190,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
      {
        title: t('操作'),
        dataIndex: 'action',
        width: 110,
        render: () => (
          <Button size='small' theme='borderless' disabled>
            {t('查看详情')}
          </Button>
        ),
      },
    ],
    [sortBy, tableSortOrder, t],
  );

  const costSpec = useMemo(
    () => {
      const baseSpec = specBar({
        data: costTopUsers.map((item) => ({
          username: buildUserChartLabel(item),
          total_cost: item.total_cost || 0,
        })),
        title: t('Top 用户使用金额'),
        subtext: `${t('Top')} ${costTopUsers.length}`,
        xField: 'username',
        yField: 'total_cost',
        seriesField: 'username',
        valueField: 'total_cost',
        valueFormatter: (value) => renderQuota(value || 0),
      });

      return {
        ...baseSpec,
        axes: [
          {
            orient: 'bottom',
          },
          {
            orient: 'left',
            label: {
              formatMethod: (value) => renderQuota(Number(value || 0)),
            },
          },
        ],
      };
    },
    [costTopUsers, specBar, t],
  );

  const tokenSpec = useMemo(
    () =>
      specBar({
        data: tokenTopUsers.map((item) => ({
          username: buildUserChartLabel(item),
          total_tokens: item.total_tokens || 0,
        })),
        title: t('Top 用户 Token 总量'),
        subtext: `${t('Top')} ${tokenTopUsers.length}`,
        xField: 'username',
        yField: 'total_tokens',
        seriesField: 'username',
        valueField: 'total_tokens',
        valueFormatter: (value) => renderNumber(value || 0),
      }),
    [tokenTopUsers, specBar, t],
  );

  return (
    <div className='flex flex-col gap-4 w-full'>
      {tableError ? <Banner type='warning' description={tableError} /> : null}
      {chartError ? <Banner type='warning' description={chartError} /> : null}

      <div className='grid gap-4 lg:grid-cols-2'>
        <div
          className='rounded-2xl border p-3'
          style={{ borderColor: 'var(--semi-color-border)' }}
        >
          <div className='h-72'>
            <VChart spec={costSpec} option={chartOption} />
          </div>
        </div>
        <div
          className='rounded-2xl border p-3'
          style={{ borderColor: 'var(--semi-color-border)' }}
        >
          <div className='h-72'>
            <VChart spec={tokenSpec} option={chartOption} />
          </div>
        </div>
      </div>

      <div
        className='rounded-2xl border p-4 flex flex-col gap-4'
        style={{ borderColor: 'var(--semi-color-border)' }}
      >
        <Table
          className='grid-bordered-table'
          size='default'
          bordered={true}
          columns={columns}
          dataSource={items}
          loading={loading}
          pagination={false}
          scroll={{ x: 1120 }}
          onChange={({ sorter }) => {
            const nextSortBy = sorter?.sortOrder
              ? SORT_FIELD_MAP[sorter?.dataIndex] || sorter?.dataIndex || ''
              : '';
            const nextSortOrder = normalizeSortOrder(sorter?.sortOrder);
            setPage(1);
            if (onSortStateChange) {
              onSortStateChange({
                sortBy: nextSortBy,
                sortOrder: nextSortOrder,
              });
            }
          }}
          empty={
            <Empty
              description={
                appliedFilters.usernameKeyword
                  ? t('没有匹配的用户统计')
                  : t('暂无用户统计数据')
              }
            />
          }
        />

        {total > 0 ? (
          <div className='flex justify-end'>
            <Pagination
              currentPage={page}
              pageSize={pageSize}
              total={total}
              showSizeChanger
              pageSizeOpts={[10, 20, 50, 100]}
              onPageChange={setPage}
              onPageSizeChange={(nextPageSize) => {
                setPage(1);
                setPageSize(nextPageSize);
              }}
            />
          </div>
        ) : null}
      </div>
    </div>
  );
};

export default UserAnalyticsTab;
