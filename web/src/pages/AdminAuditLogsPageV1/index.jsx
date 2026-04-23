import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Banner, Button, Empty, Input, Modal, Select, Table, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import CardPro from '../../components/common/ui/CardPro';
import {
  API,
  createCardProPagination,
  MAX_EXCEL_EXPORT_ROWS,
  showError,
  showInfo,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import {
  createSmartExportStatusNotifier,
  runSmartExport,
} from '../../helpers/smartExport';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import {
  changeCommittedPage,
  changeCommittedPageSize,
  commitDraftFilters,
  createAuditLogQueryState,
  createRequestSequenceTracker,
  getRefreshRequestState,
  resetDraftAndCommittedFilters,
  updateDraftFilters,
} from './requestState';
import {
  AUDIT_LOG_FILTER_MODULES,
  formatAuditIdentity,
  formatAuditTarget,
  getAuditLogActionLabel,
  getAuditLogModuleLabel,
} from './display';

const { Text } = Typography;

const renderText = (value) => {
  if (value === null || value === undefined || value === '') {
    return '-';
  }
  return value;
};

const moduleOptions = AUDIT_LOG_FILTER_MODULES.map((module) => ({
  label: getAuditLogModuleLabel(module),
  value: module,
}));

const parseOptionalInteger = (value) => {
  const normalizedValue = String(value ?? '').trim();
  if (!/^[+-]?\d+$/.test(normalizedValue)) {
    return 0;
  }

  return Number.parseInt(normalizedValue, 10);
};

const AdminAuditLogsPageV1 = () => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const { loading: permissionLoading, hasActionPermission } = useUserPermissions();
  const canRead = hasActionPermission('audit_management', 'read');

  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState([]);
  const [listError, setListError] = useState('');
  const [queryState, setQueryState] = useState(() => createAuditLogQueryState());
  const [total, setTotal] = useState(0);
  const [exportLoading, setExportLoading] = useState(false);
  const requestTrackerRef = useRef(null);

  if (!requestTrackerRef.current) {
    requestTrackerRef.current = createRequestSequenceTracker();
  }

  const { draftFilters, committedRequest } = queryState;
  const { actionModule, operatorUserId } = draftFilters;
  const { page, pageSize } = committedRequest;

  const loadAuditLogs = async (nextQueryState = queryState) => {
    if (!canRead) {
      return;
    }

    const requestState = getRefreshRequestState(nextQueryState);
    const requestId = requestTrackerRef.current.issue();

    setLoading(true);
    setListError('');
    try {
      const params = new URLSearchParams({
        p: String(requestState.page),
        page_size: String(requestState.pageSize),
      });
      if (requestState.actionModule.trim()) {
        params.set('action_module', requestState.actionModule.trim());
      }
      if (requestState.operatorUserId.trim()) {
        params.set('operator_user_id', requestState.operatorUserId.trim());
      }

      const res = await API.get(`/api/admin/audit-logs?${params.toString()}`);
      if (!requestTrackerRef.current.shouldAccept(requestId)) {
        return;
      }
      if (!res.data.success) {
        setItems([]);
        setTotal(0);
        setListError(res.data.message || t('加载审计日志失败'));
        return;
      }

      const data = res.data.data || {};
      setItems((data.items || []).map((item) => ({ ...item, key: item.id })));
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
      setListError(t('加载审计日志失败，请稍后重试'));
      showError(error);
    } finally {
      if (requestTrackerRef.current.shouldAccept(requestId)) {
        setLoading(false);
      }
    }
  };

  const resetFilters = async () => {
    const nextQueryState = resetDraftAndCommittedFilters(queryState);
    setQueryState(nextQueryState);
    await loadAuditLogs(nextQueryState);
  };

  const runExport = async () => {
    setExportLoading(true);
    try {
      await runSmartExport({
        url: '/api/admin/audit-logs/export-auto',
        payload: {
          action_module: committedRequest.actionModule.trim(),
          operator_user_id: parseOptionalInteger(committedRequest.operatorUserId),
          limit: total,
        },
        fallbackFileName: 'audit-logs.xlsx',
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

  const exportAuditLogs = async () => {
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
      loadAuditLogs(createAuditLogQueryState());
    }
  }, [permissionLoading, canRead]);

  const columns = useMemo(
    () => [
      {
        title: t('ID'),
        dataIndex: 'id',
        width: 80,
      },
      {
        title: t('操作人'),
        dataIndex: 'operator_user_id',
        width: 220,
        render: (_, record) => formatAuditIdentity({
          userId: record.operator_user_id,
          username: record.operator_username,
          displayName: record.operator_display_name,
        }),
      },
      {
        title: t('动作模块'),
        dataIndex: 'action_module',
        width: 140,
        render: (value) => renderText(getAuditLogModuleLabel(value)),
      },
      {
        title: t('动作类型'),
        dataIndex: 'action_type',
        width: 120,
        render: (value) => renderText(getAuditLogActionLabel(value)),
      },
      {
        title: t('目标'),
        dataIndex: 'target_id',
        width: 220,
        render: (_, record) => formatAuditTarget({
          targetType: record.target_type,
          targetId: record.target_id,
          targetUsername: record.target_username,
          targetDisplayName: record.target_display_name,
        }),
      },
      {
        title: t('IP'),
        dataIndex: 'ip',
        width: 150,
        render: renderText,
      },
      {
        title: t('时间'),
        dataIndex: 'created_at',
        width: 180,
        render: (value) => (value ? timestamp2string(value) : '-'),
      },
    ],
    [t],
  );

  if (permissionLoading) {
    return (
      <div className='mt-[60px] px-2'>
        <Text>{t('加载中')}</Text>
      </div>
    );
  }

  if (!canRead) {
    return (
      <div className='mt-[60px] px-2'>
        <Banner type='warning' closeIcon={null} description={t('你没有审计日志的查看权限')} />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <CardPro
        type='type3'
        descriptionArea={
          <div className='flex flex-col gap-1'>
            <Text strong>{t('审计日志')}</Text>
            <Text type='tertiary'>{t('用于查看后台管理操作产生的审计记录。')}</Text>
          </div>
        }
        actionsArea={
          <div className='flex flex-wrap items-center gap-2'>
            <Button
              size='small'
              type='tertiary'
              onClick={exportAuditLogs}
              disabled={loading || exportLoading}
            >
              {t('导出 Excel')}
            </Button>
            <Button
              size='small'
              type='tertiary'
              onClick={() => loadAuditLogs(queryState)}
            >
              {t('刷新')}
            </Button>
          </div>
        }
        searchArea={
          <div className='flex flex-col md:flex-row items-center gap-2 w-full'>
            <Select
              size='small'
              placeholder={t('按动作模块筛选')}
              value={actionModule}
              onChange={(value) => setQueryState((currentState) => updateDraftFilters(currentState, { actionModule: value }))}
              optionList={moduleOptions}
              style={{ width: isMobile ? '100%' : 220 }}
            />
            <Input
              size='small'
              placeholder={t('按操作人 ID 筛选')}
              value={operatorUserId}
              onChange={(value) => setQueryState((currentState) => updateDraftFilters(currentState, { operatorUserId: value }))}
              style={{ width: isMobile ? '100%' : 220 }}
            />
            <Button
              size='small'
              type='tertiary'
              onClick={() => {
                const nextQueryState = commitDraftFilters(queryState);
                setQueryState(nextQueryState);
                loadAuditLogs(nextQueryState);
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
            const nextQueryState = changeCommittedPage(queryState, nextPage);
            setQueryState(nextQueryState);
            loadAuditLogs(nextQueryState);
          },
          onPageSizeChange: (nextSize) => {
            const nextQueryState = changeCommittedPageSize(queryState, nextSize);
            setQueryState(nextQueryState);
            loadAuditLogs(nextQueryState);
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
              <Button
                size='small'
                type='tertiary'
                onClick={() => loadAuditLogs(queryState)}
              >
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
          empty={
            <Empty
              description={
                committedRequest.actionModule || committedRequest.operatorUserId
                  ? t('没有匹配的审计日志')
                  : t('暂无审计日志')
              }
            />
          }
        />
      </CardPro>
    </div>
  );
};

export default AdminAuditLogsPageV1;
