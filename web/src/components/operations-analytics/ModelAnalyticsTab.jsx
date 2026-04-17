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
import { API, renderNumber, renderQuota, showError } from '../../helpers';
import { buildOperationsAnalyticsSummaryParams } from '../../hooks/operations-analytics/useOperationsAnalyticsData';
import { useOperationsAnalyticsCharts } from '../../hooks/operations-analytics/useOperationsAnalyticsCharts';

const DEFAULT_PAGE_SIZE = 10;
const CHART_LIMIT = 10;

const SORT_FIELD_MAP = {
  model_name: 'model_name',
  call_count: 'call_count',
  prompt_tokens: 'prompt_tokens',
  completion_tokens: 'completion_tokens',
  total_cost: 'total_cost',
  avg_use_time: 'avg_use_time',
  success_rate: 'success_rate',
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

const buildModelChartLabel = (item) => item.model_name || '(empty)';

const hasNoTokenDetails = (record) =>
  Number(record?.prompt_tokens || 0) === 0 &&
  Number(record?.completion_tokens || 0) === 0 &&
  Number(record?.total_cost || 0) > 0;

const renderTokenValue = (value, record, t) =>
  hasNoTokenDetails(record) ? t('暂无 token 数据') : renderNumber(value || 0);

const ModelAnalyticsTab = ({
  activeTab,
  appliedFilters,
  sortState,
  onSortStateChange,
}) => {
  const { t } = useTranslation();
  const { chartOption, specLine, specBar, specPie } = useOperationsAnalyticsCharts({ t });

  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);
  const [dailyItems, setDailyItems] = useState([]);
  const [callRankItems, setCallRankItems] = useState([]);
  const [costRankItems, setCostRankItems] = useState([]);
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
    if (activeTab !== 'models') {
      return;
    }

    const requestId = tableRequestRef.current + 1;
    tableRequestRef.current = requestId;

    setLoading(true);
    setTableError('');

    try {
      const tableRes = await API.get('/api/admin/analytics/models', {
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
        setTableError(tableRes?.data?.message || t('加载模型分析数据失败'));
        return;
      }

      const tableData = tableRes.data.data || {};
      const nextItems = (tableData.items || []).map((item, index) => ({
        ...item,
        key: `${item.model_name || 'model'}-${page}-${index}`,
      }));

      setItems(nextItems);
      setTotal(tableData.total || 0);
    } catch (requestError) {
      if (tableRequestRef.current !== requestId) {
        return;
      }

      setItems([]);
      setTotal(0);
      setTableError(t('加载模型分析数据失败，请稍后重试'));
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
    if (activeTab !== 'models') {
      tableRequestRef.current += 1;
      chartRequestRef.current += 1;
      setLoading(false);
    }
  }, [activeTab]);

  const loadChartData = useCallback(async () => {
    if (activeTab !== 'models') {
      return;
    }

    const requestId = chartRequestRef.current + 1;
    chartRequestRef.current = requestId;

    setChartError('');

    try {
      const [dailyRes, callRankRes, costRankRes] = await Promise.all([
        API.get('/api/admin/analytics/daily', {
          params: buildRequestParams(appliedFilters),
        }),
        API.get('/api/admin/analytics/models', {
          params: buildRequestParams(appliedFilters, {
            p: 1,
            page_size: CHART_LIMIT,
            sort_by: 'call_count',
            sort_order: 'desc',
          }),
        }),
        API.get('/api/admin/analytics/models', {
          params: buildRequestParams(appliedFilters, {
            p: 1,
            page_size: CHART_LIMIT,
            sort_by: 'total_cost',
            sort_order: 'desc',
          }),
        }),
      ]);

      if (chartRequestRef.current !== requestId) {
        return;
      }

      const responses = [dailyRes, callRankRes, costRankRes];
      const failedResponse = responses.find((response) => !response?.data?.success);
      if (failedResponse) {
        setDailyItems([]);
        setCallRankItems([]);
        setCostRankItems([]);
        setChartError(failedResponse?.data?.message || t('加载模型分析数据失败'));
        return;
      }

      setDailyItems(dailyRes.data.data?.items || []);
      setCallRankItems(callRankRes.data.data?.items || []);
      setCostRankItems(costRankRes.data.data?.items || []);
    } catch (requestError) {
      if (chartRequestRef.current !== requestId) {
        return;
      }

      setDailyItems([]);
      setCallRankItems([]);
      setCostRankItems([]);
      setChartError(t('加载模型分析数据失败，请稍后重试'));
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
        title: t('模型名称'),
        dataIndex: 'model_name',
        sorter: true,
        sortOrder: sortBy === 'model_name' ? tableSortOrder : false,
        width: 180,
        render: (value) => value || '(empty)',
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
        title: t('输入Token'),
        dataIndex: 'prompt_tokens',
        sorter: true,
        sortOrder: sortBy === 'prompt_tokens' ? tableSortOrder : false,
        width: 130,
        render: (value, record) => renderTokenValue(value, record, t),
      },
      {
        title: t('输出Token'),
        dataIndex: 'completion_tokens',
        sorter: true,
        sortOrder: sortBy === 'completion_tokens' ? tableSortOrder : false,
        width: 130,
        render: (value, record) => renderTokenValue(value, record, t),
      },
      {
        title: t('总费用'),
        dataIndex: 'total_cost',
        sorter: true,
        sortOrder: sortBy === 'total_cost' ? tableSortOrder : false,
        width: 140,
        render: (value) => renderQuota(value),
      },
      {
        title: t('平均响应时间(ms)'),
        dataIndex: 'avg_use_time',
        sorter: true,
        sortOrder: sortBy === 'avg_use_time' ? tableSortOrder : false,
        width: 160,
        render: (value) => Number(value || 0).toFixed(1),
      },
      {
        title: t('成功率'),
        dataIndex: 'success_rate',
        sorter: true,
        sortOrder: sortBy === 'success_rate' ? tableSortOrder : false,
        width: 120,
        render: (value) => `${(Number(value || 0) * 100).toFixed(1)}%`,
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

  const trendSpec = useMemo(
    () =>
      specLine({
        data: dailyItems.map((item) => ({
          bucket_day: item.bucket_day,
          metric: t('调用次数'),
          call_count: item.call_count || 0,
        })),
        title: t('日调用量趋势'),
        subtext: `${t('总计')}：${renderNumber(
          dailyItems.reduce((sum, item) => sum + Number(item.call_count || 0), 0),
        )}`,
        xField: 'bucket_day',
        yField: 'call_count',
        seriesField: 'metric',
        valueFormatter: (value) => renderNumber(value || 0),
      }),
    [dailyItems, specLine, t],
  );

  const barSpec = useMemo(
    () =>
      specBar({
        data: callRankItems.map((item) => ({
          model_name: buildModelChartLabel(item),
          call_count: item.call_count || 0,
        })),
        title: t('模型调用量柱状图'),
        subtext: `${t('Top')} ${callRankItems.length}`,
        xField: 'model_name',
        yField: 'call_count',
        seriesField: 'model_name',
        valueField: 'call_count',
        valueFormatter: (value) => renderNumber(value || 0),
      }),
    [callRankItems, specBar, t],
  );

  const pieSpec = useMemo(
    () =>
      specPie({
        data: costRankItems.map((item) => ({
          type: buildModelChartLabel(item),
          value: item.total_cost || 0,
        })),
        title: t('模型费用占比'),
        subtext: `${t('总计')}：${renderQuota(
          costRankItems.reduce((sum, item) => sum + Number(item.total_cost || 0), 0),
        )}`,
        valueFormatter: (value) => renderQuota(value || 0),
      }),
    [costRankItems, specPie, t],
  );

  return (
    <div className='flex flex-col gap-4 w-full'>
      {tableError ? <Banner type='warning' description={tableError} /> : null}
      {chartError ? <Banner type='warning' description={chartError} /> : null}

      <div className='grid gap-4 w-full xl:grid-cols-[minmax(0,3fr)_minmax(360px,2fr)]'>
        <div className='grid gap-4'>
          <div
            className='rounded-2xl border p-3'
            style={{ borderColor: 'var(--semi-color-border)' }}
          >
            <div className='h-80'>
              <VChart spec={trendSpec} option={chartOption} />
            </div>
          </div>

          <div className='grid gap-4 xl:grid-cols-2'>
            <div
              className='rounded-2xl border p-3'
              style={{ borderColor: 'var(--semi-color-border)' }}
            >
              <div className='h-72'>
                <VChart spec={barSpec} option={chartOption} />
              </div>
            </div>
            <div
              className='rounded-2xl border p-3'
              style={{ borderColor: 'var(--semi-color-border)' }}
            >
              <div className='h-72'>
                <VChart spec={pieSpec} option={chartOption} />
              </div>
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
            scroll={{ x: 1080 }}
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
                  appliedFilters.modelKeyword
                    ? t('没有匹配的模型统计')
                    : t('暂无模型统计数据')
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
    </div>
  );
};

export default ModelAnalyticsTab;
