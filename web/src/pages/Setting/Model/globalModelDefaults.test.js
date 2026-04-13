import test from 'node:test';
import assert from 'node:assert/strict';

import {
  CHAT_COMPLETIONS_TO_RESPONSES_POLICY_ALL_CHANNELS_EXAMPLE,
  CHAT_COMPLETIONS_TO_RESPONSES_POLICY_TEMPLATE,
  DEFAULT_GLOBAL_SETTING_INPUTS,
} from './globalModelDefaults.js';

test('chat completions to responses policy template matches recommended volcengine whitelist', () => {
  const parsed = JSON.parse(CHAT_COMPLETIONS_TO_RESPONSES_POLICY_TEMPLATE);

  assert.deepEqual(parsed, {
    enabled: true,
    channel_types: [45],
    model_patterns: [
      '^doubao-seed-translation-.*$',
      '^doubao-seed-1-6-thinking-.*$',
    ],
  });
});

test('default global setting inputs use the recommended responses policy template', () => {
  assert.equal(
    DEFAULT_GLOBAL_SETTING_INPUTS['global.chat_completions_to_responses_policy'],
    CHAT_COMPLETIONS_TO_RESPONSES_POLICY_TEMPLATE,
  );
});

test('all channels example keeps the same volcengine whitelist patterns', () => {
  const parsed = JSON.parse(CHAT_COMPLETIONS_TO_RESPONSES_POLICY_ALL_CHANNELS_EXAMPLE);

  assert.deepEqual(parsed, {
    enabled: true,
    all_channels: true,
    model_patterns: [
      '^doubao-seed-translation-.*$',
      '^doubao-seed-1-6-thinking-.*$',
    ],
  });
});
