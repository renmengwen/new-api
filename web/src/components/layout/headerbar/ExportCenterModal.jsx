import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Empty, Modal, Space, Table, Tag, Typography } from '@douyinfe/semi-ui';
import { IconDownload, IconRefresh } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess, timestamp2string } from '../../../helpers';
import { downloadAsyncExportFile } from '../../../helpers/asyncExport.js';

const { Text } = Typography;

const EXPORT_CENTER_PAGE_SIZE = 10;
const EXPORT_CENTER_POLL_INTERVAL_MS = 3000;

const STATUS_META = {
  queued: { color: 'grey', label: '等待中' },
  running: { color: 'blue', label: '生成中' },
  succeeded: { color: 'green', label: '已完成' },
  failed: { color: 'red', label: '失败' },
  expired: { color: 'yellow', label: '已过期' },
};

const JOB_TYPE_LABELS = {
  usage_logs: '调用日志',
  quota_ledger: '额度流水',
  quota_cost_summary: '额度成本汇总',
  admin_audit_logs: '审计日志',
  admin_analytics_models: '运营分析-模型',
  admin_analytics_users: '运营分析-用户',
  admin_analytics_daily: '运营分析-每日',
};

const renderTime = (timestamp) => (timestamp ? timestamp2string(timestamp) : '-');

const ExportCenterModal = ({ visible, onClose, isMobile = false }) => {
  const { t } = useTranslation();
  const [jobs, setJobs] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(EXPORT_CENTER_PAGE_SIZE);
  const [loading, setLoading] = useState(false);
  const [downloadingId, setDownloadingId] = useState(null);

  const loadJobs = useCallback(
    async ({ silent = false } = {}) => {
      if (!visible) {
        return;
      }

      if (!silent) {
        setLoading(true);
      }

      try {
        const response = await API.get('/api/export-jobs', {
          params: {
            p: page,
            page_size: pageSize,
          },
          disableDuplicate: true,
          skipErrorHandler: true,
        });

        if (!response.data.success) {
          showError(response.data.message || t('加载导出任务失败'));
          return;
        }

        const data = response.data.data || {};
        setJobs((data.items || []).map((item) => ({ ...item, key: item.id })));
        setTotal(data.total || 0);
      } catch (error) {
        showError(error);
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [page, pageSize, t, visible],
  );

  useEffect(() => {
    if (!visible) {
      return undefined;
    }

    loadJobs();
    const timer = window.setInterval(() => {
      loadJobs({ silent: true });
    }, EXPORT_CENTER_POLL_INTERVAL_MS);

    return () => {
      window.clearInterval(timer);
    };
  }, [loadJobs, visible]);

  const downloadJob = async (job) => {
    if (!job?.download_url || job.status !== 'succeeded') {
      return;
    }

    setDownloadingId(job.id);
    try {
      await downloadAsyncExportFile({
        job,
        apiClient: API,
        fallbackFileName: job.file_name || 'export.xlsx',
      });
      showSuccess(t('下载已开始'));
    } catch (error) {
      showError(error);
    } finally {
      setDownloadingId(null);
    }
  };

  const columns = useMemo(
    () => [
      {
        title: t('文件名'),
        dataIndex: 'file_name',
        width: 230,
        render: (value) => (
          <Text ellipsis={{ showTooltip: true }}>{value || '-'}</Text>
        ),
      },
      {
        title: t('类型'),
        dataIndex: 'job_type',
        width: 150,
        render: (value) => t(JOB_TYPE_LABELS[value] || value || '-'),
      },
      {
        title: t('导出时间'),
        dataIndex: 'created_at',
        width: 170,
        render: renderTime,
      },
      {
        title: t('进度'),
        dataIndex: 'status',
        width: 150,
        render: (value, record) => {
          const status = String(value || '').trim().toLowerCase();
          const meta = STATUS_META[status] || { color: 'grey', label: value || '-' };
          return (
            <Space spacing={4} vertical align='start'>
              <Tag color={meta.color}>{t(meta.label)}</Tag>
              {record.error_message ? (
                <Text type='danger' size='small' ellipsis={{ showTooltip: true }}>
                  {record.error_message}
                </Text>
              ) : null}
            </Space>
          );
        },
      },
      {
        title: t('行数'),
        dataIndex: 'row_count',
        width: 90,
        render: (value) => value || '-',
      },
      {
        title: t('操作'),
        dataIndex: 'operation',
        width: 100,
        render: (_, record) => (
          <Button
            icon={<IconDownload />}
            size='small'
            theme='borderless'
            disabled={record.status !== 'succeeded'}
            loading={downloadingId === record.id}
            onClick={() => downloadJob(record)}
          >
            {t('下载')}
          </Button>
        ),
      },
    ],
    [downloadingId, t],
  );

  return (
    <Modal
      title={t('导出中心')}
      visible={visible}
      onCancel={onClose}
      width={isMobile ? '96vw' : 960}
      footer={
        <Space>
          <Button
            icon={<IconRefresh />}
            onClick={() => loadJobs()}
            loading={loading}
          >
            {t('刷新')}
          </Button>
          <Button onClick={onClose}>{t('关闭')}</Button>
        </Space>
      }
    >
      <Table
        rowKey='id'
        size='small'
        columns={columns}
        dataSource={jobs}
        loading={loading}
        empty={<Empty description={t('暂无导出任务')} />}
        scroll={{ x: 890 }}
        pagination={{
          currentPage: page,
          pageSize,
          total,
          pageSizeOpts: [10, 20, 50],
          showSizeChanger: true,
          onPageChange: setPage,
          onPageSizeChange: (nextPageSize) => {
            setPage(1);
            setPageSize(nextPageSize);
          },
        }}
      />
    </Modal>
  );
};

export default ExportCenterModal;
