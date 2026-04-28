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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Banner, Button, Empty, Input, Modal, Select, Table, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import CardPro from '../../components/common/ui/CardPro';
import {
  API,
  createCardProPagination,
  MAX_EXCEL_EXPORT_ROWS,
  renderNumber,
  showError,
  showInfo,
  showSuccess,
} from '../../helpers';
import {
  createSmartExportStatusNotifier,
  runSmartExport,
} from '../../helpers/smartExport';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import {
  changeCostSummaryCommittedPage,
  changeCostSummaryCommittedPageSize,
  commitCostSummaryFilters,
  createCostSummaryQueryState,
  getCostSummaryRefreshRequestState,
  resetCostSummaryFilters,
  updateCostSummaryDraftFilters,
} from './costSummaryRequestState';
import { createRequestSequenceTracker } from './requestState';

const { Text } = Typography;

const COST_SUMMARY_TABLE_SCROLL_X = 2520;

const parseOptionalInteger = (value) => {
  const normalizedValue = String(value ?? '').trim();
  if (!/^[+-]?\d+$/.test(normalizedValue)) {
    return 0;
  }

  return Number.parseInt(normalizedValue, 10);
};

const parseOptionalNumber = (value) => {
  const normalizedValue = String(value ?? '').trim();
  if (!normalizedValue) {
    return 0;
  }

  const parsedValue = Number.parseFloat(normalizedValue);
  return Number.isFinite(parsedValue) ? parsedValue : 0;
};

const appendTrimmedParam = (params, key, value) => {
  const normalizedValue = String(value ?? '').trim();
  if (normalizedValue) {
    params.set(key, normalizedValue);
  }
};

const buildCostSummaryParams = (requestState) => {
  const params = new URLSearchParams({
    p: String(requestState.page),
    page_size: String(requestState.pageSize),
  });

  appendTrimmedParam(params, 'start_timestamp', requestState.startTimestamp);
  appendTrimmedParam(params, 'end_timestamp', requestState.endTimestamp);
  appendTrimmedParam(params, 'model_name', requestState.modelName);
  appendTrimmedParam(params, 'vendor', requestState.vendor);
  appendTrimmedParam(params, 'user', requestState.user);
  appendTrimmedParam(params, 'token_name', requestState.tokenName);
  appendTrimmedParam(params, 'channel', requestState.channel);
  appendTrimmedParam(params, 'group', requestState.group);
  appendTrimmedParam(params, 'min_call_count', requestState.minCallCount);
  appendTrimmedParam(params, 'min_paid_usd', requestState.minPaidUsd);
  appendTrimmedParam(params, 'sort_by', requestState.sortBy);
  appendTrimmedParam(params, 'sort_order', requestState.sortOrder);

  return params;
};

const hasCommittedFilters = (committedRequest) =>
  Boolean(
    committedRequest.startTimestamp ||
      committedRequest.endTimestamp ||
      committedRequest.modelName ||
      committedRequest.vendor ||
      committedRequest.user ||
      committedRequest.tokenName ||
      committedRequest.channel ||
      committedRequest.group ||
      committedRequest.minCallCount ||
      committedRequest.minPaidUsd,
  );

const renderTextValue = (value) => value || '-';

const renderIntegerValue = (value) => renderNumber(Number(value || 0));

const renderUsdValue = (value) => Number(value || 0).toFixed(6);

const CostSummaryTab = ({ canRead, permissionLoading = false }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState([]);
  const [listError, setListError] = useState('');
  const [total, setTotal] = useState(0);
  const [exportLoading, setExportLoading] = useState(false);
  const [queryState, setQueryState] = useState(() => createCostSummaryQueryState());
  const requestTrackerRef = useRef(null);

  if (!requestTrackerRef.current) {
    requestTrackerRef.current = createRequestSequenceTracker();
  }

  const { draftFilters, committedRequest } = queryState;
  const { page, pageSize } = committedRequest;

  const loadCostSummary = async (nextQueryState = queryState) => {
    if (!canRead) {
      return;
    }

    const requestState = getCostSummaryRefreshRequestState(nextQueryState);
    const requestId = requestTrackerRef.current.issue();

    setLoading(true);
    setListError('');
    try {
      const params = buildCostSummaryParams(requestState);
      const res = await API.get(`/api/admin/quota/cost-summary?${params.toString()}`);

      if (!requestTrackerRef.current.shouldAccept(requestId)) {
        return;
      }

      if (!res.data.success) {
        setItems([]);
        setTotal(0);
        setListError(res.data.message || t('加载成本汇总失败'));
        return;
      }

      const data = res.data.data || {};
      setItems((data.items || []).map((item, index) => ({
        ...item,
        key: `${item.date || 'date'}-${item.model_name || 'model'}-${item.vendor_name || 'vendor'}-${index}`,
      })));
      setQueryState((currentState) => ({
        ...currentState,
        committedRequest: {
          ...requestState,
          page: data.page || requestState.page,
          pageSize: data.page_size || requestState.pageSize,
        },
      }));
      setTotal(data.total || 0);
    } catch (error) {
      if (!requestTrackerRef.current.shouldAccept(requestId)) {
        return;
      }
      setItems([]);
      setTotal(0);
      setListError(t('加载成本汇总失败，请稍后重试'));
      showError(error);
    } finally {
      if (requestTrackerRef.current.shouldAccept(requestId)) {
        setLoading(false);
      }
    }
  };

  const resetFilters = async () => {
    const nextQueryState = resetCostSummaryFilters(queryState);
    setQueryState(nextQueryState);
    await loadCostSummary(nextQueryState);
  };

  const runExport = async () => {
    setExportLoading(true);
    try {
      await runSmartExport({
        url: '/api/admin/quota/cost-summary/export-auto',
        payload: {
          start_timestamp: parseOptionalInteger(committedRequest.startTimestamp),
          end_timestamp: parseOptionalInteger(committedRequest.endTimestamp),
          model_name: committedRequest.modelName,
          vendor: committedRequest.vendor,
          user: committedRequest.user,
          token_name: committedRequest.tokenName,
          channel: parseOptionalInteger(committedRequest.channel),
          group: committedRequest.group,
          min_call_count: parseOptionalInteger(committedRequest.minCallCount),
          min_paid_usd: parseOptionalNumber(committedRequest.minPaidUsd),
          sort_by: committedRequest.sortBy,
          sort_order: committedRequest.sortOrder,
          limit: total,
        },
        fallbackFileName: 'quota-cost-summary.xlsx',
        onAsyncProgress: createSmartExportStatusNotifier({
          t,
          showInfo,
          showSuccess,
        }),
      });
    } catch (error) {
      showError(error);
    } finally {
      setExportLoading(false);
    }
  };

  const exportCostSummary = async () => {
    if (loading || exportLoading) {
      return;
    }

    if (!total) {
      showInfo(t('无可导出数据'));
      return;
    }

    if (total > MAX_EXCEL_EXPORT_ROWS) {
      Modal.confirm({
        title: t('导出 Excel'),
        content: t('当前筛选结果较大，导出可能切换为后台生成，是否继续？'),
        okText: t('继续导出'),
        cancelText: t('取消'),
        onOk: runExport,
      });
      return;
    }

    await runExport();
  };

  useEffect(() => {
    if (!permissionLoading && canRead) {
      loadCostSummary(createCostSummaryQueryState());
    }
  }, [permissionLoading, canRead]);

  const sortFieldOptions = useMemo(
    () => [
      { label: t('日期'), value: 'date' },
      { label: t('模型名称'), value: 'model_name' },
      { label: t('供应商名称'), value: 'vendor_name' },
      { label: t('调用次数'), value: 'call_count' },
      { label: t('输入tokens'), value: 'input_tokens' },
      { label: t('输出tokens'), value: 'output_tokens' },
      { label: t('实付金额（USD）'), value: 'paid_usd' },
    ],
    [t],
  );

  const sortOrderOptions = useMemo(
    () => [
      { label: t('降序'), value: 'desc' },
      { label: t('升序'), value: 'asc' },
    ],
    [t],
  );

  const columns = useMemo(
    () => [
      {
        title: t('日期'),
        dataIndex: 'date',
        width: 120,
        fixed: 'left',
        render: renderTextValue,
      },
      {
        title: t('模型名称'),
        dataIndex: 'model_name',
        width: 180,
        fixed: 'left',
        render: renderTextValue,
      },
      {
        title: t('供应商名称'),
        dataIndex: 'vendor_name',
        width: 160,
        render: renderTextValue,
      },
      {
        title: t('结算含税价input'),
        dataIndex: 'input_unit_price_usd',
        width: 150,
        render: renderUsdValue,
      },
      {
        title: t('结算含税价output'),
        dataIndex: 'output_unit_price_usd',
        width: 150,
        render: renderUsdValue,
      },
      {
        title: t('输入tokens'),
        dataIndex: 'input_tokens',
        width: 130,
        render: renderIntegerValue,
      },
      {
        title: t('输出tokens'),
        dataIndex: 'output_tokens',
        width: 130,
        render: renderIntegerValue,
      },
      {
        title: t('调用次数'),
        dataIndex: 'call_count',
        width: 110,
        render: renderIntegerValue,
      },
      {
        title: t('input费用'),
        dataIndex: 'input_cost_usd',
        width: 120,
        render: renderUsdValue,
      },
      {
        title: t('output费用'),
        dataIndex: 'output_cost_usd',
        width: 120,
        render: renderUsdValue,
      },
      {
        title: t('缓存创建'),
        dataIndex: 'cache_create_tokens',
        width: 120,
        render: renderIntegerValue,
      },
      {
        title: t('缓存读取'),
        dataIndex: 'cache_read_tokens',
        width: 120,
        render: renderIntegerValue,
      },
      {
        title: t('缓存创建单价'),
        dataIndex: 'cache_create_unit_price_usd',
        width: 150,
        render: renderUsdValue,
      },
      {
        title: t('缓存读取单价'),
        dataIndex: 'cache_read_unit_price_usd',
        width: 150,
        render: renderUsdValue,
      },
      {
        title: t('cache的token'),
        dataIndex: 'cache_tokens',
        width: 130,
        render: renderIntegerValue,
      },
      {
        title: t('cache的金额'),
        dataIndex: 'cache_cost_usd',
        width: 130,
        render: renderUsdValue,
      },
      {
        title: t('总费用（USD）'),
        dataIndex: 'total_cost_usd',
        width: 140,
        render: renderUsdValue,
      },
      {
        title: t('折扣'),
        dataIndex: 'discount_usd',
        width: 120,
        render: renderUsdValue,
      },
      {
        title: t('实付金额（USD）'),
        dataIndex: 'paid_usd',
        width: 140,
        render: renderUsdValue,
      },
    ],
    [t],
  );

  const updateDraftFilters = (nextDraftFilters) => {
    setQueryState((currentState) => updateCostSummaryDraftFilters(currentState, nextDraftFilters));
  };

  const controlWidth = isMobile ? '100%' : 160;
  const wideControlWidth = isMobile ? '100%' : 200;

  if (permissionLoading) {
    return <Text>{t('加载中')}</Text>;
  }

  if (!canRead) {
    return <Banner type='warning' closeIcon={null} description={t('你没有成本汇总的查看权限')} />;
  }

  return (
    <CardPro
      type='type3'
      descriptionArea={
        <div className='flex flex-col gap-1'>
          <Text strong>{t('成本汇总')}</Text>
          <Text type='tertiary'>{t('按日期、模型和供应商汇总调用成本与实付金额。')}</Text>
        </div>
      }
      actionsArea={
        <div className='flex flex-wrap items-center gap-2'>
          <Button
            size='small'
            type='tertiary'
            onClick={exportCostSummary}
            disabled={loading || exportLoading}
          >
            {t('导出 Excel')}
          </Button>
          <Button size='small' type='tertiary' onClick={() => loadCostSummary(queryState)}>
            {t('刷新')}
          </Button>
        </div>
      }
      searchArea={
        <div className='flex flex-wrap items-center gap-2 w-full'>
          <Input
            size='small'
            placeholder={t('开始时间戳')}
            value={draftFilters.startTimestamp}
            onChange={(value) => updateDraftFilters({ startTimestamp: value })}
            style={{ width: controlWidth }}
          />
          <Input
            size='small'
            placeholder={t('结束时间戳')}
            value={draftFilters.endTimestamp}
            onChange={(value) => updateDraftFilters({ endTimestamp: value })}
            style={{ width: controlWidth }}
          />
          <Input
            size='small'
            placeholder={t('模型名称')}
            value={draftFilters.modelName}
            onChange={(value) => updateDraftFilters({ modelName: value })}
            style={{ width: wideControlWidth }}
          />
          <Input
            size='small'
            placeholder={t('供应商')}
            value={draftFilters.vendor}
            onChange={(value) => updateDraftFilters({ vendor: value })}
            style={{ width: controlWidth }}
          />
          <Input
            size='small'
            placeholder={t('用户 ID 或用户名')}
            value={draftFilters.user}
            onChange={(value) => updateDraftFilters({ user: value })}
            style={{ width: wideControlWidth }}
          />
          <Input
            size='small'
            placeholder={t('令牌名称')}
            value={draftFilters.tokenName}
            onChange={(value) => updateDraftFilters({ tokenName: value })}
            style={{ width: controlWidth }}
          />
          <Input
            size='small'
            placeholder={t('渠道 ID')}
            value={draftFilters.channel}
            onChange={(value) => updateDraftFilters({ channel: value })}
            style={{ width: controlWidth }}
          />
          <Input
            size='small'
            placeholder={t('分组')}
            value={draftFilters.group}
            onChange={(value) => updateDraftFilters({ group: value })}
            style={{ width: controlWidth }}
          />
          <Input
            size='small'
            placeholder={t('最小调用次数')}
            value={draftFilters.minCallCount}
            onChange={(value) => updateDraftFilters({ minCallCount: value })}
            style={{ width: controlWidth }}
          />
          <Input
            size='small'
            placeholder={t('最小实付 USD')}
            value={draftFilters.minPaidUsd}
            onChange={(value) => updateDraftFilters({ minPaidUsd: value })}
            style={{ width: controlWidth }}
          />
          <Select
            size='small'
            value={draftFilters.sortBy}
            optionList={sortFieldOptions}
            onChange={(value) => updateDraftFilters({ sortBy: value })}
            style={{ width: controlWidth }}
          />
          <Select
            size='small'
            value={draftFilters.sortOrder}
            optionList={sortOrderOptions}
            onChange={(value) => updateDraftFilters({ sortOrder: value })}
            style={{ width: controlWidth }}
          />
          <Button
            size='small'
            type='tertiary'
            onClick={() => {
              const nextQueryState = commitCostSummaryFilters(queryState);
              setQueryState(nextQueryState);
              loadCostSummary(nextQueryState);
            }}
          >
            {t('查询')}
          </Button>
          <Button size='small' type='tertiary' onClick={resetFilters}>
            {t('重置')}
          </Button>
        </div>
      }
      paginationArea={createCardProPagination({
        currentPage: page,
        pageSize,
        total,
        onPageChange: (nextPage) => {
          const nextQueryState = changeCostSummaryCommittedPage(queryState, nextPage);
          setQueryState(nextQueryState);
          loadCostSummary(nextQueryState);
        },
        onPageSizeChange: (nextSize) => {
          const nextQueryState = changeCostSummaryCommittedPageSize(queryState, nextSize);
          setQueryState(nextQueryState);
          loadCostSummary(nextQueryState);
        },
        isMobile,
        t,
      })}
      t={t}
    >
      {listError ? (
        <div className='mb-3 flex flex-col gap-2'>
          <Banner type='warning' closeIcon={null} description={listError} />
          <div>
            <Button size='small' type='tertiary' onClick={() => loadCostSummary(queryState)}>
              {t('重新加载')}
            </Button>
          </div>
        </div>
      ) : null}
      <Table
        className='grid-bordered-table'
        size='small'
        bordered={true}
        columns={columns}
        dataSource={items}
        loading={loading}
        pagination={false}
        scroll={{ x: COST_SUMMARY_TABLE_SCROLL_X }}
        empty={
          <Empty
            description={
              hasCommittedFilters(committedRequest)
                ? t('没有匹配的成本汇总')
                : t('暂无成本汇总')
            }
          />
        }
      />
    </CardPro>
  );
};

export default CostSummaryTab;
