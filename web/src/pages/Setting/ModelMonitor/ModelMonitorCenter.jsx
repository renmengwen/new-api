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

import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  Banner,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  InputNumber,
  Modal,
  Row,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { BellRing, Play, RefreshCw, Save } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../helpers';
import { useUserPermissions } from '../../../hooks/common/useUserPermissions';
import {
  buildChannelTagDisplays,
  buildModelMonitorResultMessage,
  buildModelOverrideSettings,
  formatResponseTime,
  getChannelCopyText,
  getChannelStatusDisplay,
  getEffectiveModelEnabled,
  getModelCopyText,
  getModelOverride,
  getModelStatusDisplay,
  isModelExcludedByPatterns,
  patternsToText,
  textToPatterns,
} from './modelMonitorDisplay';

const { Text, Title } = Typography;

const DEFAULT_SETTINGS = {
  enabled: false,
  interval_minutes: 10,
  batch_size: 5,
  default_timeout_seconds: 30,
  failure_threshold: 3,
  excluded_model_patterns: [],
  model_overrides: {},
  notification_disabled_user_ids: [],
};

const DEFAULT_SUMMARY = {
  total_models: 0,
  healthy_models: 0,
  partial_models: 0,
  unavailable_models: 0,
  skipped_models: 0,
  total_channels: 0,
  failed_channels: 0,
};

function normalizeBoolean(value) {
  if (value === 'true') return true;
  if (value === 'false') return false;
  return Boolean(value);
}

function normalizeSettings(settings) {
  const next = {
    ...DEFAULT_SETTINGS,
    ...(settings || {}),
  };

  return {
    ...next,
    enabled: normalizeBoolean(next.enabled),
    interval_minutes:
      Number(next.interval_minutes) || DEFAULT_SETTINGS.interval_minutes,
    batch_size: Number(next.batch_size) || DEFAULT_SETTINGS.batch_size,
    default_timeout_seconds:
      Number(next.default_timeout_seconds) ||
      DEFAULT_SETTINGS.default_timeout_seconds,
    failure_threshold:
      Number(next.failure_threshold) || DEFAULT_SETTINGS.failure_threshold,
    excluded_model_patterns: Array.isArray(next.excluded_model_patterns)
      ? next.excluded_model_patterns
      : textToPatterns(next.excluded_model_patterns),
    model_overrides:
      next.model_overrides && typeof next.model_overrides === 'object'
        ? next.model_overrides
        : {},
    notification_disabled_user_ids: Array.isArray(
      next.notification_disabled_user_ids,
    )
      ? next.notification_disabled_user_ids.map(Number).filter(Number.isFinite)
      : [],
  };
}

function formatTestedAt(value) {
  if (!value) return '-';
  const numeric = Number(value);
  const date = Number.isFinite(numeric)
    ? new Date(numeric > 100000000000 ? numeric : numeric * 1000)
    : new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function getModelEnabled(settings, record) {
  return getEffectiveModelEnabled(settings, record);
}

function getModelTimeout(settings, record) {
  const override = getModelOverride(settings, record.model_name);
  return (
    Number(override.timeout_seconds) ||
    Number(record.timeout_seconds) ||
    settings.default_timeout_seconds
  );
}

function getModelChannels(record) {
  return Array.isArray(record?.channels) ? record.channels : [];
}

function getChannelCount(record) {
  return Number(record?.channel_count) || getModelChannels(record).length;
}

function getChannelStatusCounts(record) {
  const channels = getModelChannels(record);
  if (
    record?.success_count !== undefined ||
    record?.failed_count !== undefined ||
    record?.skipped_count !== undefined
  ) {
    return {
      success: Number(record.success_count) || 0,
      failed: Number(record.failed_count) || 0,
      skipped: Number(record.skipped_count) || 0,
    };
  }
  return channels.reduce(
    (counts, channel) => {
      const status = getChannelStatusDisplay(channel.status).value;
      if (status === 'success') {
        counts.success += 1;
      } else if (status === 'skipped') {
        counts.skipped += 1;
      } else {
        counts.failed += 1;
      }
      return counts;
    },
    { success: 0, failed: 0, skipped: 0 },
  );
}

async function writeClipboardText(text) {
  const value = String(text || '').trim();
  if (!value) {
    return false;
  }
  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return true;
    } catch (error) {
      // Fall through to the textarea fallback for browsers without clipboard permission.
    }
  }
  if (typeof document === 'undefined') {
    return false;
  }
  const textarea = document.createElement('textarea');
  textarea.value = value;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  document.body.appendChild(textarea);
  textarea.select();
  try {
    return document.execCommand('copy');
  } finally {
    document.body.removeChild(textarea);
  }
}

export default function ModelMonitorCenter() {
  const { t } = useTranslation();
  const {
    loading: permissionLoading,
    hasActionPermission,
  } = useUserPermissions();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [settings, setSettings] = useState(DEFAULT_SETTINGS);
  const [summary, setSummary] = useState(DEFAULT_SUMMARY);
  const [items, setItems] = useState([]);
  const [excludedPatternsText, setExcludedPatternsText] = useState('');
  const [loadError, setLoadError] = useState('');
  const [notificationModalVisible, setNotificationModalVisible] =
    useState(false);
  const [notificationUsersLoading, setNotificationUsersLoading] =
    useState(false);
  const [notificationUsers, setNotificationUsers] = useState([]);
  const [
    notificationDraftDisabledUserIds,
    setNotificationDraftDisabledUserIds,
  ] = useState([]);
  const formRef = useRef();

  const canRead = hasActionPermission('model_monitor_management', 'read');
  const canUpdate = hasActionPermission('model_monitor_management', 'update');
  const canTest = hasActionPermission('model_monitor_management', 'test');

  const applyMonitorData = useCallback((data) => {
    const nextSettings = normalizeSettings(data?.settings);
    setSettings(nextSettings);
    setExcludedPatternsText(
      patternsToText(nextSettings.excluded_model_patterns),
    );
    setSummary({
      ...DEFAULT_SUMMARY,
      ...(data?.summary || {}),
    });
    setItems(Array.isArray(data?.items) ? data.items : []);
  }, []);

  const fetchMonitorData = useCallback(async ({
    showLoading = true,
    disableDuplicate = false,
  } = {}) => {
    if (!canRead) {
      return null;
    }
    if (showLoading) {
      setLoading(true);
    }
    setLoadError('');
    try {
      const res = await API.get('/api/model_monitor', { disableDuplicate });
      const { success, message, data } = res.data;
      if (!success) {
        setLoadError(message || t('加载失败'));
        showError(message || t('加载失败'));
        return null;
      }

      applyMonitorData(data);
      return data;
    } catch (error) {
      setLoadError(t('加载失败'));
      showError(t('加载失败'));
      return null;
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, [applyMonitorData, canRead, t]);

  useEffect(() => {
    if (permissionLoading || !canRead) {
      return;
    }
    fetchMonitorData();
  }, [canRead, fetchMonitorData, permissionLoading]);

  useEffect(() => {
    if (permissionLoading || !canRead || !settings.enabled || testing) {
      return undefined;
    }
    const intervalMinutes =
      Number(settings.interval_minutes) || DEFAULT_SETTINGS.interval_minutes;
    const intervalMs = Math.min(
      Math.max(intervalMinutes * 60 * 1000, 30000),
      60000,
    );
    const timer = setInterval(() => {
      fetchMonitorData({ showLoading: false, disableDuplicate: true });
    }, intervalMs);
    return () => clearInterval(timer);
  }, [
    canRead,
    fetchMonitorData,
    permissionLoading,
    settings.enabled,
    settings.interval_minutes,
    testing,
  ]);

  const modelRows = useMemo(
    () =>
      items.map((item) => ({
        ...item,
        key: item.model_name,
      })),
    [items],
  );
  const formValues = useMemo(
    () => ({
      ...settings,
      excluded_model_patterns_text: excludedPatternsText,
    }),
    [settings, excludedPatternsText],
  );
  const displaySettings = useMemo(
    () => ({
      ...settings,
      excluded_model_patterns: textToPatterns(excludedPatternsText),
    }),
    [settings, excludedPatternsText],
  );
  const notificationDraftDisabledUserIdSet = useMemo(
    () => new Set(notificationDraftDisabledUserIds.map(Number)),
    [notificationDraftDisabledUserIds],
  );

  useEffect(() => {
    if (formRef.current) {
      formRef.current.setValues(formValues);
    }
  }, [formValues]);

  const updateSetting = (key, value) => {
    setSettings((current) => ({
      ...current,
      [key]: value,
    }));
  };

  const updateModelOverride = (modelName, patch) => {
    setSettings((current) =>
      buildModelOverrideSettings(current, modelName, patch),
    );
  };

  const buildSettingsPayload = (sourceSettings = settings) => ({
    ...sourceSettings,
    excluded_model_patterns: textToPatterns(excludedPatternsText),
  });

  const saveSettings = async (sourceSettings = settings) => {
    if (!canUpdate) {
      showError(t('您无权访问此页面，请联系管理员'));
      return false;
    }
    setSaving(true);
    try {
      const payload = buildSettingsPayload(sourceSettings);
      const res = await API.put('/api/model_monitor/settings', payload);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('保存失败，请重试'));
        return false;
      }
      showSuccess(t('保存成功'));
      if (data) {
        applyMonitorData(data);
      } else {
        await fetchMonitorData();
      }
      return true;
    } catch (error) {
      showError(t('保存失败，请重试'));
      return false;
    } finally {
      setSaving(false);
    }
  };

  const openNotificationUsersModal = async () => {
    if (!canUpdate) {
      return;
    }
    setNotificationModalVisible(true);
    setNotificationUsersLoading(true);
    setNotificationDraftDisabledUserIds(
      Array.isArray(settings.notification_disabled_user_ids)
        ? settings.notification_disabled_user_ids
        : [],
    );
    try {
      const res = await API.get('/api/model_monitor/notification-users');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('加载失败'));
        return;
      }
      setNotificationUsers(Array.isArray(data) ? data : []);
    } catch (error) {
      showError(t('加载失败'));
    } finally {
      setNotificationUsersLoading(false);
    }
  };

  const setNotificationRecipientEnabled = (userId, enabled) => {
    const normalizedUserId = Number(userId);
    if (!Number.isFinite(normalizedUserId)) {
      return;
    }
    setNotificationDraftDisabledUserIds((current) => {
      const next = new Set(current.map(Number));
      if (enabled) {
        next.delete(normalizedUserId);
      } else {
        next.add(normalizedUserId);
      }
      return Array.from(next).sort((a, b) => a - b);
    });
  };

  const saveNotificationUsers = async () => {
    const nextSettings = {
      ...settings,
      notification_disabled_user_ids: notificationDraftDisabledUserIds,
    };
    setSettings(nextSettings);
    const saved = await saveSettings(nextSettings);
    if (saved) {
      setNotificationModalVisible(false);
    }
  };

  const waitForManualTestResult = useCallback(async () => {
    const maxAttempts = 120;
    for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      const data = await fetchMonitorData({
        showLoading: false,
        disableDuplicate: true,
      });
      if (!data?.running) {
        return data;
      }
    }
    return null;
  }, [fetchMonitorData]);

  const runManualTest = async () => {
    if (!canTest) {
      showError(t('您无权访问此页面，请联系管理员'));
      return;
    }
    setTesting(true);
    try {
      const res = await API.post('/api/model_monitor/test');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message || t('测试失败'));
        return;
      }
      if (data) {
        applyMonitorData(data);
      }
      showSuccess(message || t('已触发手动测试，测试结果会自动刷新'));
      let finalData = data;
      if (data?.running !== false) {
        finalData = await waitForManualTestResult();
      } else {
        finalData = await fetchMonitorData({
          showLoading: false,
          disableDuplicate: true,
        });
      }
      if (finalData?.summary) {
        showSuccess(buildModelMonitorResultMessage(finalData.summary, t));
      }
    } catch (error) {
      showError(t('测试失败'));
    } finally {
      setTesting(false);
    }
  };

  const copyMonitorText = useCallback(
    async (text, successMessage) => {
      try {
        const copied = await writeClipboardText(text);
        if (!copied) {
          showError(t('复制失败'));
          return;
        }
        showSuccess(successMessage);
      } catch (error) {
        showError(t('复制失败'));
      }
    },
    [t],
  );

  const renderStatusTag = (status, enabled) => {
    const display = getModelStatusDisplay(status, enabled);
    return (
      <Tag color={display.color} shape='circle'>
        {t(display.label)}
      </Tag>
    );
  };

  const renderChannelTags = (channels) => {
    const { visibleTags, restCount } = buildChannelTagDisplays(channels, 4);
    if (!visibleTags.length) {
      return <Text type='tertiary'>-</Text>;
    }

    return (
      <Space wrap spacing={4}>
        {visibleTags.map((tag) => (
          <Tooltip
            key={tag.key}
            content={`${tag.title} · ${t('点击复制渠道名称')}`}
          >
            <Tag
              color={tag.color}
              shape='circle'
              onClick={(event) => {
                event.stopPropagation();
                copyMonitorText(tag.copyText, t('已复制渠道名称'));
              }}
              style={{ cursor: 'pointer' }}
            >
              {tag.label}
            </Tag>
          </Tooltip>
        ))}
        {restCount > 0 && <Tag shape='circle'>{`+${restCount}`}</Tag>}
      </Space>
    );
  };

  const renderChannelDetails = (record) => {
    const channels = Array.isArray(record.channels) ? record.channels : [];
    const channelColumns = [
      {
        title: t('渠道'),
        dataIndex: 'channel_name',
        render: (value, channel) => (
          <div className='flex flex-col'>
            <Tooltip content={t('点击复制渠道名称')}>
              <Text
                strong
                role='button'
                tabIndex={0}
                onClick={(event) => {
                  event.stopPropagation();
                  copyMonitorText(
                    getChannelCopyText(channel),
                    t('已复制渠道名称'),
                  );
                }}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    event.stopPropagation();
                    copyMonitorText(
                      getChannelCopyText(channel),
                      t('已复制渠道名称'),
                    );
                  }
                }}
                style={{ cursor: 'pointer' }}
              >
                {value || `#${channel.channel_id}`}
              </Text>
            </Tooltip>
            <Text type='tertiary' size='small'>
              {channel.channel_type || '-'}
            </Text>
          </div>
        ),
      },
      {
        title: t('测试结果'),
        dataIndex: 'status',
        render: (value) => {
          const display = getChannelStatusDisplay(value);
          return (
            <Tag color={display.color} shape='circle'>
              {t(display.label)}
            </Tag>
          );
        },
      },
      {
        title: t('响应时间'),
        dataIndex: 'response_time_ms',
        render: (value) => formatResponseTime(value),
      },
      {
        title: t('连续失败'),
        dataIndex: 'consecutive_failures',
        render: (value) => value ?? 0,
      },
      {
        title: t('测试时间'),
        dataIndex: 'tested_at',
        render: (value) => formatTestedAt(value),
      },
      {
        title: t('错误信息'),
        dataIndex: 'error_message',
        render: (value) =>
          value ? (
            <Text type='danger' ellipsis={{ showTooltip: true }}>
              {value}
            </Text>
          ) : (
            <Text type='tertiary'>-</Text>
          ),
      },
    ];

    return (
      <div style={{ padding: '8px 0 8px 28px' }}>
        <Table
          columns={channelColumns}
          dataSource={channels.map((channel) => ({
            ...channel,
            key: `${record.model_name}-${channel.channel_id}`,
          }))}
          pagination={false}
          size='small'
          className='grid-bordered-table'
        />
      </div>
    );
  };

  const summaryItems = [
    { label: t('模型总数'), value: summary.total_models },
    { label: t('正常模型'), value: summary.healthy_models, color: 'green' },
    { label: t('部分异常'), value: summary.partial_models, color: 'yellow' },
    { label: t('不可用'), value: summary.unavailable_models, color: 'red' },
    { label: t('已跳过'), value: summary.skipped_models, color: 'grey' },
    { label: t('渠道总数'), value: summary.total_channels },
    { label: t('失败渠道'), value: summary.failed_channels, color: 'red' },
  ];

  const notificationColumns = [
    {
      title: t('接收通知'),
      dataIndex: 'notification_enabled',
      width: 120,
      render: (_, record) => {
        const canReceive = record.can_receive !== false;
        const disabledReason =
          record.disabled_reason === 'no_email'
            ? t('未配置邮箱')
            : record.disabled_reason === 'no_model_monitor_read_permission'
              ? t('无模型监控查看权限')
              : t('不可接收');
        const checkbox = (
          <Checkbox
            checked={
              canReceive &&
              !notificationDraftDisabledUserIdSet.has(Number(record.id))
            }
            disabled={!canUpdate || !canReceive}
            onChange={(event) =>
              setNotificationRecipientEnabled(record.id, event.target.checked)
            }
          >
            {canReceive ? t('接收') : t('不可接收')}
          </Checkbox>
        );
        if (canReceive) {
          return checkbox;
        }
        return (
          <Tooltip content={disabledReason}>
            <span>{checkbox}</span>
          </Tooltip>
        );
      },
    },
    {
      title: t('管理员'),
      dataIndex: 'username',
      render: (value, record) => (
        <div className='flex flex-col'>
          <Text strong>{record.display_name || value || `#${record.id}`}</Text>
          <Text type='tertiary' size='small'>
            {value || '-'}
          </Text>
        </div>
      ),
    },
    {
      title: t('邮箱'),
      dataIndex: 'email',
      render: (value) => value || '-',
    },
    {
      title: t('类型'),
      dataIndex: 'user_type',
      width: 100,
      render: (value, record) => {
        if (value === 'root' || record.role >= 100) {
          return <Tag color='red'>{t('超级管理员')}</Tag>;
        }
        if (value === 'agent') {
          return <Tag color='violet'>{t('代理商')}</Tag>;
        }
        return <Tag color='blue'>{t('管理员')}</Tag>;
      },
    },
  ];

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'model_name',
      width: 240,
      fixed: 'left',
      render: (value, record) => (
        <div className='flex flex-col'>
          <Tooltip content={t('点击复制模型名称')}>
            <Text
              strong
              ellipsis={{ showTooltip: true }}
              role='button'
              tabIndex={0}
              onClick={(event) => {
                event.stopPropagation();
                copyMonitorText(getModelCopyText(record), t('已复制模型名称'));
              }}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault();
                  event.stopPropagation();
                  copyMonitorText(
                    getModelCopyText(record),
                    t('已复制模型名称'),
                  );
                }
              }}
              style={{ cursor: 'pointer' }}
            >
              {value}
            </Text>
          </Tooltip>
          <Text type='tertiary' size='small'>
            {t('所属渠道')} {getChannelCount(record)}
          </Text>
        </div>
      ),
    },
    {
      title: t('所属渠道'),
      dataIndex: 'channels',
      width: 260,
      render: renderChannelTags,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 110,
      render: (value, record) =>
        renderStatusTag(value, getModelEnabled(displaySettings, record)),
    },
    {
      title: t('最后测试时间'),
      dataIndex: 'tested_at',
      width: 180,
      render: (value) => formatTestedAt(value),
    },
    {
      title: t('成功/失败'),
      dataIndex: 'success_count',
      width: 130,
      render: (_, record) => {
        const counts = getChannelStatusCounts(record);
        return (
          <Space spacing={4}>
            <Tag color='green' shape='circle'>
              {counts.success}
            </Tag>
            <Tag color='red' shape='circle'>
              {counts.failed}
            </Tag>
            {counts.skipped > 0 && (
              <Tag color='grey' shape='circle'>
                {counts.skipped}
              </Tag>
            )}
          </Space>
        );
      },
    },
    {
      title: t('最长响应'),
      dataIndex: 'timeout_seconds',
      width: 170,
      render: (_, record) => (
        <Space spacing={6}>
          <InputNumber
            size='small'
            min={1}
            step={1}
            value={getModelTimeout(settings, record)}
            disabled={!canUpdate}
            onChange={(value) =>
              updateModelOverride(record.model_name, {
                timeout_seconds:
                  Number(value) || settings.default_timeout_seconds,
              })
            }
            style={{ width: 96 }}
          />
          <Text type='tertiary'>{t('秒')}</Text>
        </Space>
      ),
    },
    {
      title: t('定时测试'),
      dataIndex: 'enabled',
      width: 110,
      render: (_, record) => {
        const excluded = isModelExcludedByPatterns(
          displaySettings,
          record.model_name,
        );
        const switchNode = (
          <Switch
            size='small'
            checked={getModelEnabled(displaySettings, record)}
            checkedText='｜'
            uncheckedText='〇'
            disabled={!canUpdate || excluded}
            onChange={(value) =>
              updateModelOverride(record.model_name, {
                enabled: value,
              })
            }
          />
        );
        if (!excluded) {
          return switchNode;
        }
        return (
          <Tooltip content={t('该模型已命中排除规则，无法单独开启定时测试')}>
            <span>{switchNode}</span>
          </Tooltip>
        );
      },
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      width: 110,
      fixed: 'right',
      render: () => (
        <Button
          size='small'
          type='tertiary'
          onClick={() => saveSettings()}
          loading={saving}
          disabled={!canUpdate}
        >
          {t('保存')}
        </Button>
      ),
    },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <Spin spinning={permissionLoading || loading} size='large'>
      {!permissionLoading && !canRead && (
        <Banner
          style={{ marginTop: '10px' }}
          type='warning'
          description={t('您无权访问此页面，请联系管理员')}
        />
      )}

      {canRead && (
        <>
          <Card style={{ marginTop: '10px' }}>
            <div className='flex flex-col gap-3'>
              <div className='flex flex-col md:flex-row md:items-center md:justify-between gap-2'>
                <div>
                  <Title heading={5} style={{ margin: 0 }}>
                    {t('模型监控')}
                  </Title>
                  <Text type='tertiary'>
                    {t('按模型聚合展示各所属渠道的最近测试结果')}
                  </Text>
                </div>
                <Space wrap>
                  <Button
                    icon={<RefreshCw size={16} />}
                    onClick={fetchMonitorData}
                    loading={loading}
                    disabled={!canRead}
                  >
                    {t('刷新')}
                  </Button>
                  <Button
                    icon={<BellRing size={16} />}
                    onClick={openNotificationUsersModal}
                    disabled={!canUpdate}
                  >
                    {t('通知人管理')}
                  </Button>
                  <Button
                    type='primary'
                    icon={<Play size={16} />}
                    onClick={runManualTest}
                    loading={testing}
                    disabled={!canTest}
                  >
                    {t('手动测试')}
                  </Button>
                  <Button
                    icon={<Save size={16} />}
                    onClick={() => saveSettings()}
                    loading={saving}
                    disabled={!canUpdate}
                  >
                    {t('保存设置')}
                  </Button>
                </Space>
              </div>

              {loadError && <Banner type='warning' description={loadError} />}

              <Form
                values={formValues}
                getFormApi={(formAPI) => (formRef.current = formAPI)}
              >
                <Row gutter={16}>
                  <Col xs={24} sm={12} md={8} lg={6}>
                    <Form.Switch
                      field='enabled'
                      label={t('启用模型监控')}
                      checkedText='｜'
                      uncheckedText='〇'
                      disabled={!canUpdate}
                      onChange={(value) => updateSetting('enabled', value)}
                    />
                  </Col>
                  <Col xs={24} sm={12} md={8} lg={6}>
                    <Form.InputNumber
                      field='interval_minutes'
                      label={t('测试间隔')}
                      min={1}
                      step={1}
                      suffix={t('分钟')}
                      disabled={!canUpdate}
                      onChange={(value) =>
                        updateSetting('interval_minutes', Number(value) || 1)
                      }
                    />
                  </Col>
                  <Col xs={24} sm={12} md={8} lg={6}>
                    <Form.InputNumber
                      field='batch_size'
                      label={t('批量大小')}
                      min={1}
                      step={1}
                      disabled={!canUpdate}
                      onChange={(value) =>
                        updateSetting('batch_size', Number(value) || 1)
                      }
                    />
                  </Col>
                  <Col xs={24} sm={12} md={8} lg={6}>
                    <Form.InputNumber
                      field='default_timeout_seconds'
                      label={t('默认最长响应')}
                      min={1}
                      step={1}
                      suffix={t('秒')}
                      disabled={!canUpdate}
                      onChange={(value) =>
                        updateSetting('default_timeout_seconds', Number(value) || 1)
                      }
                    />
                  </Col>
                  <Col xs={24} sm={12} md={8} lg={6}>
                    <Form.InputNumber
                      field='failure_threshold'
                      label={t('失败阈值')}
                      min={1}
                      step={1}
                      disabled={!canUpdate}
                      onChange={(value) =>
                        updateSetting('failure_threshold', Number(value) || 1)
                      }
                    />
                  </Col>
                  <Col xs={24} md={16} lg={18}>
                    <Form.TextArea
                      field='excluded_model_patterns_text'
                      label={t('排除模型规则')}
                      placeholder={t('一行一个或用逗号分隔，例如：*image*')}
                      autosize={{ minRows: 2, maxRows: 5 }}
                      disabled={!canUpdate}
                      onChange={setExcludedPatternsText}
                    />
                  </Col>
                </Row>
              </Form>
            </div>
          </Card>

          <Card style={{ marginTop: '10px' }}>
            <div className='flex flex-col gap-3'>
              <Space wrap>
                {summaryItems.map((item) => (
                  <Tag
                    key={item.label}
                    color={item.color}
                    shape='circle'
                    style={{ padding: '6px 10px' }}
                  >
                    {item.label}: {item.value ?? 0}
                  </Tag>
                ))}
              </Space>

              <Table
                columns={columns}
                dataSource={modelRows}
                rowKey='model_name'
                expandedRowRender={renderChannelDetails}
                rowExpandable={(record) =>
                  Array.isArray(record.channels) && record.channels.length > 0
                }
                expandRowByClick
                pagination={{
                  pageSize: 20,
                  showSizeChanger: true,
                  pageSizeOptions: [10, 20, 50, 100],
                }}
                size='small'
                scroll={{ x: 'max-content' }}
                className='grid-bordered-table'
              />
            </div>
          </Card>

          <Modal
            title={t('通知人管理')}
            visible={notificationModalVisible}
            onCancel={() => setNotificationModalVisible(false)}
            onOk={saveNotificationUsers}
            okText={t('保存')}
            cancelText={t('取消')}
            confirmLoading={saving}
            okButtonProps={{ disabled: !canUpdate }}
            size='large'
          >
            <Text type='tertiary'>
              {t('取消勾选后，模型失败邮件不会发送给对应管理员。')}
            </Text>
            <Table
              columns={notificationColumns}
              dataSource={notificationUsers.map((user) => ({
                ...user,
                key: user.id,
              }))}
              loading={notificationUsersLoading}
              pagination={false}
              size='small'
              className='grid-bordered-table mt-3'
            />
          </Modal>
        </>
      )}
      </Spin>
    </div>
  );
}
