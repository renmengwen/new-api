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

const MODEL_STATUS_DISPLAY = {
  healthy: { value: 'healthy', label: '正常', color: 'green' },
  partial: { value: 'partial', label: '部分异常', color: 'yellow' },
  unavailable: { value: 'unavailable', label: '不可用', color: 'red' },
  skipped: { value: 'skipped', label: '已跳过', color: 'grey' },
  no_channel: { value: 'no_channel', label: '无可测渠道', color: 'grey' },
  unknown: { value: 'unknown', label: '未知', color: 'grey' },
};

const MODEL_STATUS_ALIASES = {
  ok: 'healthy',
  success: 'healthy',
  normal: 'healthy',
  healthy: 'healthy',
  partial: 'partial',
  degraded: 'partial',
  warning: 'partial',
  partial_failure: 'partial',
  failed: 'unavailable',
  failure: 'unavailable',
  error: 'unavailable',
  unavailable: 'unavailable',
  skipped: 'skipped',
  disabled: 'skipped',
  no_channel: 'no_channel',
  no_channels: 'no_channel',
  no_available_channel: 'no_channel',
};

const CHANNEL_STATUS_DISPLAY = {
  success: { value: 'success', label: '成功', color: 'green' },
  failed: { value: 'failed', label: '失败', color: 'red' },
  timeout: { value: 'timeout', label: '超时', color: 'yellow' },
  skipped: { value: 'skipped', label: '已跳过', color: 'grey' },
  unknown: { value: 'unknown', label: '未测试', color: 'grey' },
};

const CHANNEL_STATUS_ALIASES = {
  ok: 'success',
  success: 'success',
  healthy: 'success',
  normal: 'success',
  failed: 'failed',
  failure: 'failed',
  error: 'failed',
  unavailable: 'failed',
  timeout: 'timeout',
  timed_out: 'timeout',
  skipped: 'skipped',
  disabled: 'skipped',
  not_tested: 'skipped',
  no_channel: 'skipped',
};

function normalizeStatusValue(status) {
  return String(status || '')
    .trim()
    .toLowerCase()
    .replace(/[\s-]+/g, '_');
}

export function getModelStatusDisplay(status, enabled = true) {
  if (enabled === false) {
    return MODEL_STATUS_DISPLAY.skipped;
  }
  const normalized = normalizeStatusValue(status);
  return MODEL_STATUS_DISPLAY[MODEL_STATUS_ALIASES[normalized] || 'unknown'];
}

export function getChannelStatusDisplay(status) {
  const normalized = normalizeStatusValue(status);
  return CHANNEL_STATUS_DISPLAY[
    CHANNEL_STATUS_ALIASES[normalized] || 'unknown'
  ];
}

export function formatResponseTime(responseTimeMs) {
  if (
    responseTimeMs === null ||
    responseTimeMs === undefined ||
    responseTimeMs === ''
  ) {
    return '-';
  }
  const numeric = Number(responseTimeMs);
  if (!Number.isFinite(numeric)) {
    return '-';
  }
  if (numeric < 1000) {
    return `${Math.round(numeric)} ms`;
  }
  return `${(numeric / 1000).toFixed(2)} s`;
}

export function getChannelTagDisplay(channel, index = 0) {
  const channelId = channel?.channel_id ?? channel?.id ?? '';
  const channelName = channel?.channel_name || channel?.name || '';
  const label =
    channelName || (channelId !== '' ? `#${channelId}` : '未知渠道');
  const statusDisplay = getChannelStatusDisplay(channel?.status);
  const responseTime = formatResponseTime(channel?.response_time_ms);
  const titleParts = [`${label} · ${statusDisplay.label}`];

  if (responseTime !== '-') {
    titleParts.push(responseTime);
  }
  if (channel?.error_message) {
    titleParts.push(channel.error_message);
  }

  return {
    key: `${channelId || index}-${label}`,
    label,
    copyText: getChannelCopyText(channel),
    color: statusDisplay.color,
    statusLabel: statusDisplay.label,
    title: titleParts.join(' · '),
  };
}

export function buildChannelTagDisplays(channels, limit = 4) {
  const allTags = Array.isArray(channels)
    ? channels.map((channel, index) => getChannelTagDisplay(channel, index))
    : [];
  const visibleLimit = Math.max(0, Number(limit) || 0);

  return {
    visibleTags: allTags.slice(0, visibleLimit),
    restCount: Math.max(allTags.length - visibleLimit, 0),
    total: allTags.length,
  };
}

export function patternsToText(patterns) {
  if (Array.isArray(patterns)) {
    return patterns.join('\n');
  }
  return patterns || '';
}

export function textToPatterns(text) {
  const source = Array.isArray(text) ? text.join('\n') : String(text || '');
  return source
    .split(/[,\n]/g)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function getModelOverride(settings, modelName) {
  const override = settings?.model_overrides?.[modelName];
  return override && typeof override === 'object' ? { ...override } : {};
}

export function modelMonitorPatternMatches(pattern, modelName) {
  const normalizedPattern = String(pattern || '').trim();
  const value = String(modelName || '');
  if (!normalizedPattern) {
    return false;
  }
  if (normalizedPattern === value) {
    return true;
  }
  if (!normalizedPattern.includes('*')) {
    return false;
  }

  const parts = normalizedPattern.split('*');
  let position = 0;
  if (parts[0] !== '') {
    if (!value.startsWith(parts[0])) {
      return false;
    }
    position = parts[0].length;
  }

  const lastIndex = parts.length - 1;
  for (let index = 1; index < lastIndex; index += 1) {
    const part = parts[index];
    if (part === '') {
      continue;
    }
    const nextPosition = value.indexOf(part, position);
    if (nextPosition < 0) {
      return false;
    }
    position = nextPosition + part.length;
  }

  const lastPart = parts[lastIndex];
  return lastPart === '' || value.slice(position).endsWith(lastPart);
}

export function isModelExcludedByPatterns(settings, modelName) {
  const patterns = Array.isArray(settings?.excluded_model_patterns)
    ? settings.excluded_model_patterns
    : textToPatterns(settings?.excluded_model_patterns);
  return patterns.some((pattern) => modelMonitorPatternMatches(pattern, modelName));
}

export function getEffectiveModelEnabled(settings, record) {
  const modelName = record?.model_name || '';
  if (isModelExcludedByPatterns(settings, modelName)) {
    return false;
  }
  const override = getModelOverride(settings, modelName);
  if (Object.prototype.hasOwnProperty.call(override, 'enabled')) {
    return override.enabled !== false;
  }
  return record?.enabled !== false;
}

export function getModelMonitorResultCounts(summary) {
  const total = Number(summary?.total_models) || 0;
  const success = Number(summary?.healthy_models) || 0;
  const failed =
    (Number(summary?.partial_models) || 0) +
    (Number(summary?.unavailable_models) || 0);

  return { total, success, failed };
}

export function buildModelMonitorResultMessage(summary, t) {
  return t(
    '测试完成：总模型 {{total}}，成功 {{success}}，失败 {{failed}}',
    getModelMonitorResultCounts(summary),
  );
}

export function buildModelOverrideSettings(settings, modelName, patch) {
  return {
    ...(settings || {}),
    model_overrides: {
      ...(settings?.model_overrides || {}),
      [modelName]: {
        ...getModelOverride(settings, modelName),
        ...patch,
      },
    },
  };
}

export function getModelCopyText(record) {
  return String(record?.model_name || '').trim();
}

export function getChannelCopyText(channel) {
  const channelName = channel?.channel_name || channel?.name || '';
  if (channelName) {
    return channelName;
  }
  const channelId = channel?.channel_id ?? channel?.id ?? '';
  return channelId !== '' ? `#${channelId}` : '';
}
