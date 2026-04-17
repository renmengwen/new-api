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
import { Banner, Empty, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { VChart } from '@visactor/react-vchart';
import { API, renderNumber, renderQuota, showError } from '../../helpers';
import { buildOperationsAnalyticsSummaryParams } from '../../hooks/operations-analytics/useOperationsAnalyticsData';
import { useOperationsAnalyticsCharts } from '../../hooks/operations-analytics/useOperationsAnalyticsCharts';

const DailyAnalyticsTab = ({ activeTab, appliedFilters }) => {
  const { t } = useTranslation();
  const { chartOption, specLine } = useOperationsAnalyticsCharts({ t });

  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const requestRef = useRef(0);

  const loadData = useCallback(async () => {
    if (activeTab !== 'daily') {
      return;
    }

    const requestId = requestRef.current + 1;
    requestRef.current = requestId;

    setLoading(true);
    setError('');

    try {
      const res = await API.get('/api/admin/analytics/daily', {
        params: buildOperationsAnalyticsSummaryParams(appliedFilters),
      });

      if (requestRef.current !== requestId) {
        return;
      }

      if (!res?.data?.success) {
        setItems([]);
        setError(res?.data?.message || t('加载按日分析数据失败'));
        return;
      }

      setItems(res.data.data?.items || []);
    } catch (requestError) {
      if (requestRef.current !== requestId) {
        return;
      }

      setItems([]);
      setError(t('加载按日分析数据失败，请稍后重试'));
      showError(requestError);
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [activeTab, appliedFilters, t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  useEffect(() => {
    if (activeTab !== 'daily') {
      requestRef.current += 1;
      setLoading(false);
    }
  }, [activeTab]);

  useEffect(() => () => {
    requestRef.current += 1;
  }, []);

  const createSeriesData = useCallback(
    (field, metricLabel) =>
      items.map((item) => ({
        bucket_day: item.bucket_day,
        metric: metricLabel,
        [field]: Number(item[field] || 0),
      })),
    [items],
  );

  const callSpec = useMemo(
    () =>
      specLine({
        data: createSeriesData('call_count', t('调用次数')),
        title: t('按日调用次数'),
        subtext: `${t('总计')}：${renderNumber(
          items.reduce((sum, item) => sum + Number(item.call_count || 0), 0),
        )}`,
        xField: 'bucket_day',
        yField: 'call_count',
        seriesField: 'metric',
        valueFormatter: (value) => renderNumber(value || 0),
      }),
    [createSeriesData, items, specLine, t],
  );

  const costSpec = useMemo(
    () => {
      const baseSpec = specLine({
        data: createSeriesData('total_cost', t('总费用')),
        title: t('按日费用'),
        subtext: `${t('总计')}：${renderQuota(
          items.reduce((sum, item) => sum + Number(item.total_cost || 0), 0),
        )}`,
        xField: 'bucket_day',
        yField: 'total_cost',
        seriesField: 'metric',
        valueFormatter: (value) => renderQuota(value || 0),
      });

      return {
        ...baseSpec,
        tooltip: {
          ...baseSpec.tooltip,
          dimension: {
            visible: false,
          },
        },
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
    [createSeriesData, items, specLine, t],
  );

  const activeUsersSpec = useMemo(
    () =>
      specLine({
        data: createSeriesData('active_users', t('活跃用户')),
        title: t('按日活跃用户'),
        subtext: `${t('峰值')}：${renderNumber(
          items.reduce(
            (maxValue, item) => Math.max(maxValue, Number(item.active_users || 0)),
            0,
          ),
        )}`,
        xField: 'bucket_day',
        yField: 'active_users',
        seriesField: 'metric',
        valueFormatter: (value) => renderNumber(value || 0),
      }),
    [createSeriesData, items, specLine, t],
  );

  const activeModelsSpec = useMemo(
    () =>
      specLine({
        data: createSeriesData('active_models', t('活跃模型')),
        title: t('按日活跃模型'),
        subtext: `${t('峰值')}：${renderNumber(
          items.reduce(
            (maxValue, item) => Math.max(maxValue, Number(item.active_models || 0)),
            0,
          ),
        )}`,
        xField: 'bucket_day',
        yField: 'active_models',
        seriesField: 'metric',
        valueFormatter: (value) => renderNumber(value || 0),
      }),
    [createSeriesData, items, specLine, t],
  );

  if (loading) {
    return (
      <div className='flex flex-col gap-4 w-full'>
        {error ? <Banner type='warning' description={error} /> : null}
        <div
          className='rounded-2xl border p-6 min-h-[320px] flex items-center justify-center'
          style={{ borderColor: 'var(--semi-color-border)' }}
        >
          <Spin spinning={loading} tip={t('加载中')}>
            <div className='h-24 w-24' />
          </Spin>
        </div>
      </div>
    );
  }

  if (!error && items.length === 0) {
    return <Empty description={t('暂无按日分析数据')} />;
  }

  return (
    <div className='flex flex-col gap-4 w-full'>
      {error ? <Banner type='warning' description={error} /> : null}

      <div className='grid gap-4 lg:grid-cols-2'>
        {[callSpec, costSpec, activeUsersSpec, activeModelsSpec].map((spec, index) => (
          <div
            key={`${spec.title?.text || 'daily'}-${index}`}
            className='rounded-2xl border p-3'
            style={{ borderColor: 'var(--semi-color-border)' }}
          >
            <div className='h-72'>
              <VChart spec={spec} option={chartOption} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default DailyAnalyticsTab;
