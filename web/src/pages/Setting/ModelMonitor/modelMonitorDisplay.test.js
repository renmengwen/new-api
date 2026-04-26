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

import test from 'node:test';
import assert from 'node:assert/strict';

import {
  buildChannelTagDisplays,
  getChannelCopyText,
  getChannelStatusDisplay,
  getEffectiveModelEnabled,
  getModelCopyText,
  getModelOverride,
  getModelStatusDisplay,
  isModelExcludedByPatterns,
  textToPatterns,
} from './modelMonitorDisplay.js';

test('getModelStatusDisplay maps model monitor states to Chinese labels and Semi colors', () => {
  assert.deepEqual(getModelStatusDisplay('healthy'), {
    value: 'healthy',
    label: '正常',
    color: 'green',
  });
  assert.deepEqual(getModelStatusDisplay('partial'), {
    value: 'partial',
    label: '部分异常',
    color: 'yellow',
  });
  assert.deepEqual(getModelStatusDisplay('unavailable'), {
    value: 'unavailable',
    label: '不可用',
    color: 'red',
  });
  assert.deepEqual(getModelStatusDisplay('skipped'), {
    value: 'skipped',
    label: '已跳过',
    color: 'grey',
  });
  assert.deepEqual(getModelStatusDisplay('future_status'), {
    value: 'unknown',
    label: '未知',
    color: 'grey',
  });
});

test('getChannelStatusDisplay normalizes per-channel test states', () => {
  assert.deepEqual(getChannelStatusDisplay('success'), {
    value: 'success',
    label: '成功',
    color: 'green',
  });
  assert.equal(getChannelStatusDisplay('failed').color, 'red');
  assert.equal(getChannelStatusDisplay('timeout').color, 'yellow');
  assert.equal(getChannelStatusDisplay('disabled').label, '已跳过');
});

test('buildChannelTagDisplays keeps multiple channels for the same model visible', () => {
  const result = buildChannelTagDisplays(
    [
      { channel_id: 1, channel_name: 'OpenAI', status: 'success' },
      { channel_id: 2, channel_name: 'Azure', status: 'failed' },
      { channel_id: 3, channel_name: 'Gemini', status: 'timeout' },
    ],
    2,
  );

  assert.deepEqual(
    result.visibleTags.map((tag) => ({
      key: tag.key,
      label: tag.label,
      color: tag.color,
    })),
    [
      { key: '1-OpenAI', label: 'OpenAI', color: 'green' },
      { key: '2-Azure', label: 'Azure', color: 'red' },
    ],
  );
  assert.equal(result.restCount, 1);
});

test('getModelOverride returns model level override without mutating settings', () => {
  const settings = {
    model_overrides: {
      'gpt-4o': {
        enabled: false,
        timeout_seconds: 45,
      },
    },
  };

  const override = getModelOverride(settings, 'gpt-4o');
  assert.deepEqual(override, { enabled: false, timeout_seconds: 45 });
  override.enabled = true;
  assert.equal(settings.model_overrides['gpt-4o'].enabled, false);
  assert.deepEqual(getModelOverride(settings, 'missing'), {});
});

test('textToPatterns accepts comma and newline separated excluded patterns', () => {
  assert.deepEqual(textToPatterns('gpt-image-*,\n  *video* \n\nrealtime*'), [
    'gpt-image-*',
    '*video*',
    'realtime*',
  ]);
});

test('excluded model patterns force scheduled testing off', () => {
  const settings = {
    excluded_model_patterns: ['legacy-*', '*image*'],
    model_overrides: {
      'legacy-chat': {
        enabled: true,
      },
    },
  };

  assert.equal(isModelExcludedByPatterns(settings, 'legacy-chat'), true);
  assert.equal(isModelExcludedByPatterns(settings, 'openai/gpt-image-1'), true);
  assert.equal(
    getEffectiveModelEnabled(settings, {
      model_name: 'legacy-chat',
      enabled: true,
    }),
    false,
  );
});

test('copy text helpers return stable model and channel labels', () => {
  assert.equal(
    getModelCopyText({ model_name: 'anthropic/claude-opus-4-7' }),
    'anthropic/claude-opus-4-7',
  );
  assert.equal(
    getChannelCopyText({ channel_id: 12, channel_name: 'Claude 主渠道' }),
    'Claude 主渠道',
  );
  assert.equal(getChannelCopyText({ id: 7, name: '' }), '#7');
  assert.equal(getChannelCopyText({}), '');
});
