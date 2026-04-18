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
  ADVANCED_PRICING_MODE_ADVANCED,
  FIXED_BILLING_MODE_PER_REQUEST,
  FIXED_BILLING_MODE_PER_TOKEN,
  MEDIA_TASK_RULE_TYPE,
  TEXT_SEGMENT_RULE_TYPE,
  buildTextSegmentConditionSummary,
  buildTextSegmentPreview,
  canUseAdvancedPricingMode,
  findMatchingTextSegmentRule,
  getEffectiveBillingModeForModel,
  normalizeAdvancedPricingConfig,
  validateTextSegmentRules,
} from './advancedPricingRuleHelpers.js';

test('validateTextSegmentRules rejects inverted ranges, duplicate priorities, and overlapping enabled segments', () => {
  const errors = validateTextSegmentRules([
    {
      id: 'rule-a',
      enabled: true,
      priority: 1,
      inputMin: 2000,
      inputMax: 1000,
      outputMin: '',
      outputMax: '',
      inputPrice: '0.2',
      outputPrice: '0.4',
      cacheReadPrice: '',
      cacheWritePrice: '',
    },
    {
      id: 'rule-b',
      enabled: true,
      priority: 1,
      inputMin: 0,
      inputMax: 4000,
      outputMin: '',
      outputMax: '',
      inputPrice: '0.3',
      outputPrice: '0.5',
      cacheReadPrice: '',
      cacheWritePrice: '',
    },
    {
      id: 'rule-c',
      enabled: true,
      priority: 2,
      inputMin: 3000,
      inputMax: 6000,
      outputMin: '',
      outputMax: '',
      inputPrice: '0.35',
      outputPrice: '0.55',
      cacheReadPrice: '',
      cacheWritePrice: '',
    },
  ]);

  assert.ok(
    errors.some((error) => error.includes('输入最小值不能大于输入最大值')),
  );
  assert.ok(errors.some((error) => error.includes('优先级 1 重复')));
  assert.ok(
    errors.some((error) => error.includes('rule-b') && error.includes('rule-c')),
  );
});

test('findMatchingTextSegmentRule returns the enabled highest-priority matching segment', () => {
  const rules = [
    {
      id: 'fallback',
      enabled: true,
      priority: 20,
      inputMin: 0,
      inputMax: '',
      outputMin: '',
      outputMax: '',
      inputPrice: '0.2',
      outputPrice: '0.4',
      cacheReadPrice: '',
      cacheWritePrice: '',
    },
    {
      id: 'small-window',
      enabled: true,
      priority: 5,
      inputMin: 0,
      inputMax: 8000,
      outputMin: 0,
      outputMax: 4000,
      inputPrice: '0.16',
      outputPrice: '0.32',
      cacheReadPrice: '0.04',
      cacheWritePrice: '0.08',
    },
  ];

  const matchedRule = findMatchingTextSegmentRule(rules, {
    inputTokens: 4096,
    outputTokens: 2048,
  });

  assert.equal(matchedRule?.id, 'small-window');
});

test('buildTextSegmentPreview returns matched segment summary and estimated prices', () => {
  const rule = {
    id: 'segment-1',
    enabled: true,
    priority: 3,
    inputMin: 0,
    inputMax: 16000,
    outputMin: 0,
    outputMax: 8000,
    inputPrice: '0.25',
    outputPrice: '0.5',
    cacheReadPrice: '0.05',
    cacheWritePrice: '0.1',
  };

  const preview = buildTextSegmentPreview([rule], {
    inputTokens: 8000,
    outputTokens: 2000,
  });

  assert.equal(preview.matchedRule?.id, 'segment-1');
  assert.equal(
    preview.conditionSummary,
    buildTextSegmentConditionSummary(rule),
  );
  assert.equal(preview.priceSummary.inputCost, '0.002');
  assert.equal(preview.priceSummary.outputCost, '0.001');
  assert.equal(preview.priceSummary.totalCost, '0.003');
  assert.equal(preview.priceSummary.cacheReadPrice, '0.05');
  assert.equal(preview.priceSummary.cacheWritePrice, '0.1');
});

test('advanced pricing helper constants stay aligned with persisted runtime enums', () => {
  assert.equal(ADVANCED_PRICING_MODE_ADVANCED, 'advanced');
  assert.equal(FIXED_BILLING_MODE_PER_TOKEN, 'per_token');
  assert.equal(FIXED_BILLING_MODE_PER_REQUEST, 'per_request');
  assert.equal(TEXT_SEGMENT_RULE_TYPE, 'text_segment');
  assert.equal(MEDIA_TASK_RULE_TYPE, 'media_task');
});

test('normalizeAdvancedPricingConfig preserves persisted rule type enums and advanced mode only applies with rules', () => {
  const emptyConfig = normalizeAdvancedPricingConfig({
    ruleType: 'media_task',
    rules: [],
  });
  const configuredConfig = normalizeAdvancedPricingConfig({
    ruleType: 'media_task',
    rules: [{ id: 'media-rule-1' }],
  });

  assert.equal(emptyConfig.ruleType, 'media_task');
  assert.equal(configuredConfig.ruleType, 'media_task');
  assert.equal(canUseAdvancedPricingMode(emptyConfig), false);
  assert.equal(canUseAdvancedPricingMode(configuredConfig), true);
  assert.equal(
    getEffectiveBillingModeForModel({
      selectedMode: 'advanced',
      fixedBillingMode: 'per_request',
      advancedConfig: emptyConfig,
    }),
    'per_request',
  );
  assert.equal(
    getEffectiveBillingModeForModel({
      selectedMode: 'advanced',
      fixedBillingMode: 'per_request',
      advancedConfig: configuredConfig,
    }),
    'advanced',
  );
});
