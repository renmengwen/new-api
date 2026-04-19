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
  buildMediaTaskConditionSummary,
  buildMediaTaskPreview,
  buildAdvancedPricingSaveMaps,
  buildTextSegmentConditionSummary,
  buildTextSegmentPreview,
  canUseAdvancedPricingMode,
  getAdvancedPricingMapValidationErrors,
  getAdvancedPricingValidationErrors,
  getTextSegmentRuleEditorMeta,
  findMatchingTextSegmentRule,
  getEffectiveBillingModeForModel,
  mergeAdvancedPricingDraftMap,
  mergeAdvancedPricingModeDraftMap,
  normalizeAdvancedPricingConfig,
  saveAdvancedPricingOptions,
  serializeAdvancedPricingMap,
  serializeAdvancedPricingConfig,
  validateMediaTaskConfig,
  validateTextSegmentRules,
} from './advancedPricingRuleHelpers.js';
import { resolveAdvancedPricingSelectedModelName } from './advancedPricingSelection.js';

test('resolveAdvancedPricingSelectedModelName keeps controlled selection empty until the controlled model becomes available', () => {
  assert.deepEqual(
    resolveAdvancedPricingSelectedModelName({
      currentSelectedModelName: 'alpha',
      modelNames: ['alpha'],
      isControlledSelection: true,
      externalSelectedModelName: 'beta',
    }),
    {
      nextSelectedModelName: '',
      nextAppliedInitialSelectionVersion: null,
    },
  );

  assert.deepEqual(
    resolveAdvancedPricingSelectedModelName({
      currentSelectedModelName: 'alpha',
      modelNames: ['alpha', 'beta'],
      isControlledSelection: true,
      externalSelectedModelName: '',
      initialSelectedModelName: 'beta',
      initialSelectionVersion: 3,
      lastAppliedInitialSelectionVersion: 2,
    }),
    {
      nextSelectedModelName: '',
      nextAppliedInitialSelectionVersion: 2,
    },
  );

  assert.deepEqual(
    resolveAdvancedPricingSelectedModelName({
      currentSelectedModelName: '',
      modelNames: ['alpha', 'beta'],
      isControlledSelection: true,
      externalSelectedModelName: 'beta',
    }),
    {
      nextSelectedModelName: 'beta',
      nextAppliedInitialSelectionVersion: null,
    },
  );
});

test('resolveAdvancedPricingSelectedModelName applies legacy initial selection only once per version', () => {
  assert.deepEqual(
    resolveAdvancedPricingSelectedModelName({
      currentSelectedModelName: 'alpha',
      modelNames: ['alpha', 'beta'],
      isControlledSelection: false,
      initialSelectedModelName: 'beta',
      initialSelectionVersion: 2,
      lastAppliedInitialSelectionVersion: 1,
    }),
    {
      nextSelectedModelName: 'beta',
      nextAppliedInitialSelectionVersion: 2,
    },
  );

  assert.deepEqual(
    resolveAdvancedPricingSelectedModelName({
      currentSelectedModelName: 'alpha',
      modelNames: ['alpha', 'beta'],
      isControlledSelection: false,
      initialSelectedModelName: 'beta',
      initialSelectionVersion: 2,
      lastAppliedInitialSelectionVersion: 2,
    }),
    {
      nextSelectedModelName: 'alpha',
      nextAppliedInitialSelectionVersion: 2,
    },
  );
});

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

test('validateTextSegmentRules allows overlapping ranges when service tiers differ', () => {
  const errors = validateTextSegmentRules([
    {
      id: 'rule-standard',
      enabled: true,
      priority: 1,
      inputMin: 0,
      inputMax: 32000,
      outputMin: '',
      outputMax: '',
      serviceTier: 'standard',
      inputPrice: '0.2',
      outputPrice: '0.4',
    },
    {
      id: 'rule-priority',
      enabled: true,
      priority: 2,
      inputMin: 0,
      inputMax: 32000,
      outputMin: '',
      outputMax: '',
      serviceTier: 'priority',
      inputPrice: '0.25',
      outputPrice: '0.5',
    },
  ]);

  assert.equal(errors.length, 0);
});

test('validateTextSegmentRules allows overlapping ranges when text modalities differ', () => {
  const errors = validateTextSegmentRules([
    {
      id: 'rule-text',
      enabled: true,
      priority: 1,
      inputMin: 0,
      inputMax: 32000,
      outputMin: 0,
      outputMax: 16000,
      inputModality: 'text',
      outputModality: 'text',
      inputPrice: '0.2',
      outputPrice: '0.4',
    },
    {
      id: 'rule-audio',
      enabled: true,
      priority: 2,
      inputMin: 0,
      inputMax: 32000,
      outputMin: 0,
      outputMax: 16000,
      inputModality: 'audio',
      outputModality: 'audio',
      inputPrice: '0.25',
      outputPrice: '0.5',
    },
  ]);

  assert.equal(errors.length, 0);
});

test('validateTextSegmentRules allows a single default text rule without conditions', () => {
  const errors = validateTextSegmentRules([
    {
      id: 'rule-no-condition',
      enabled: true,
      priority: 1,
      inputMin: '',
      inputMax: '',
      outputMin: '',
      outputMax: '',
      serviceTier: '',
      inputPrice: '0.2',
      outputPrice: '0.4',
    },
  ]);

  assert.equal(errors.length, 0);
});

test('validateTextSegmentRules rejects multiple default text rules without conditions', () => {
  const errors = validateTextSegmentRules([
    {
      id: 'rule-no-condition-a',
      enabled: true,
      priority: 1,
      inputMin: '',
      inputMax: '',
      outputMin: '',
      outputMax: '',
      serviceTier: '',
      inputPrice: '0.2',
      outputPrice: '0.4',
    },
    {
      id: 'rule-no-condition-b',
      enabled: true,
      priority: 2,
      inputMin: '',
      inputMax: '',
      outputMin: '',
      outputMax: '',
      serviceTier: '',
      inputPrice: '0.3',
      outputPrice: '0.5',
    },
  ]);

  assert.ok(errors.some((error) => error.includes('default')));
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

test('buildTextSegmentPreview uses default text rule when no conditional rule matches', () => {
  const preview = buildTextSegmentPreview(
    [
      {
        id: 'unconditional',
        enabled: true,
        priority: 1,
        inputMin: '',
        inputMax: '',
        outputMin: '',
        outputMax: '',
        serviceTier: '',
        inputPrice: '0.2',
        outputPrice: '0.4',
      },
    ],
    {
      inputTokens: 4096,
      outputTokens: 2048,
    },
  );

  assert.equal(preview.matchedRule?.id, 'unconditional');
  assert.match(preview.formulaSummary, /4096/);
  assert.notEqual(preview.priceSummary.totalCost, '');
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

test('buildTextSegmentPreview matches service tier specific segments and exposes formula/log preview', () => {
  const rules = [
    {
      id: 'segment-standard',
      enabled: true,
      priority: 20,
      inputMin: 0,
      inputMax: 16000,
      outputMin: 0,
      outputMax: 8000,
      serviceTier: 'standard',
      inputPrice: '0.2',
      outputPrice: '0.4',
    },
    {
      id: 'segment-priority',
      enabled: true,
      priority: 10,
      inputMin: 0,
      inputMax: 16000,
      outputMin: 0,
      outputMax: 8000,
      serviceTier: 'priority',
      inputPrice: '0.3',
      outputPrice: '0.6',
    },
  ];

  const preview = buildTextSegmentPreview(rules, {
    inputTokens: 1024,
    outputTokens: 512,
    serviceTier: 'priority',
  });

  assert.equal(preview.matchedRule?.id, 'segment-priority');
  assert.match(preview.conditionSummary, /服务层=priority/);
  assert.match(preview.formulaSummary, /1024/);
  assert.match(preview.logPreview.detailSummary, /priority/i);
});

test('buildTextSegmentPreview matches service tier case-insensitively', () => {
  const preview = buildTextSegmentPreview(
    [
      {
        id: 'segment-default',
        enabled: true,
        priority: 10,
        inputMin: 0,
        inputMax: 16000,
        outputMin: 0,
        outputMax: 8000,
        serviceTier: 'Default',
        inputPrice: '0.3',
        outputPrice: '0.6',
      },
    ],
    {
      inputTokens: 1024,
      outputTokens: 512,
      serviceTier: 'default',
    },
  );

  assert.equal(preview.matchedRule?.id, 'segment-default');
  assert.match(preview.conditionSummary, /default/i);
});

test('buildTextSegmentConditionSummary includes modality and schema-supported extension fields', () => {
  const summary = buildTextSegmentConditionSummary({
    inputModality: 'audio',
    outputModality: 'text',
    imageSizeTier: 'hd',
    toolUsageType: 'web_search',
    toolUsageCount: '1000',
  });

  assert.match(summary, /input_modality=audio/);
  assert.match(summary, /output_modality=text/);
  assert.match(summary, /image_size_tier=hd/);
  assert.match(summary, /tool_usage_type=web_search/);
  assert.match(summary, /tool_usage_count>=1000/);
});

test('buildTextSegmentPreview matches modality-aware rules and exposes cache/tool scaffolding fields', () => {
  const preview = buildTextSegmentPreview(
    [
      {
        id: 'segment-audio',
        enabled: true,
        priority: 1,
        inputModality: 'audio',
        outputModality: 'text',
        billingUnit: 'per_million_tokens',
        cacheStoragePrice: '1',
        toolUsageType: 'google_search',
        freeQuota: '500',
        overageThreshold: '1000',
        inputPrice: '1',
        outputPrice: '2.5',
      },
    ],
    {
      inputModality: 'audio',
      outputModality: 'text',
      toolUsageType: 'google_search',
      toolUsageCount: '750',
      inputTokens: '128',
      outputTokens: '64',
    },
  );

  assert.equal(preview.matchedRule?.id, 'segment-audio');
  assert.equal(preview.priceSummary.cacheStoragePrice, '1');
  assert.equal(preview.priceSummary.toolUsageCount, '750');
  assert.equal(preview.priceSummary.freeQuota, '500');
  assert.equal(preview.priceSummary.overageThreshold, '1000');
  assert.equal(preview.priceSummary.billingUnit, 'per_million_tokens');
  assert.equal(preview.matchedSegmentPreview.input_modality, 'audio');
  assert.equal(preview.matchedSegmentPreview.output_modality, 'text');
  assert.equal(preview.matchedSegmentPreview.tool_usage_type, 'google_search');
});

test('buildTextSegmentPreview ignores schema-only text selectors that backend matching does not implement', () => {
  const preview = buildTextSegmentPreview(
    [
      {
        id: 'segment-audio',
        enabled: true,
        priority: 1,
        inputModality: 'audio',
        outputModality: 'text',
        imageSizeTier: 'hd',
        toolUsageType: 'google_search',
        toolUsageCount: '1000',
        inputPrice: '1',
        outputPrice: '2',
      },
    ],
    {
      inputModality: 'audio',
      outputModality: 'text',
      imageSizeTier: 'sd',
      toolUsageType: 'code_interpreter',
      toolUsageCount: '1',
      inputTokens: '64',
      outputTokens: '32',
    },
  );

  assert.equal(preview.matchedRule?.id, 'segment-audio');
});

test('getTextSegmentRuleEditorMeta counts enabled rules and treats explicit zero default price as configured', () => {
  assert.deepEqual(
    getTextSegmentRuleEditorMeta(
      {
        defaultPrice: 0,
      },
      [
        { id: 'rule-enabled', enabled: true },
        { id: 'rule-disabled', enabled: false },
        { id: 'rule-implicit-enabled' },
      ],
    ),
    {
      ruleType: TEXT_SEGMENT_RULE_TYPE,
      totalRules: 3,
      enabledRules: 2,
      hasDefaultPrice: true,
      defaultPrice: '0',
    },
  );

  assert.deepEqual(getTextSegmentRuleEditorMeta({}, []), {
    ruleType: TEXT_SEGMENT_RULE_TYPE,
    totalRules: 0,
    enabledRules: 0,
    hasDefaultPrice: false,
    defaultPrice: '',
  });
});

test('buildMediaTaskPreview returns the highest-priority matching segment and explains min token billing floor', () => {
  const rules = [
    {
      id: 'media-fallback',
      priority: 20,
      rawAction: 'generate',
      unitPrice: '0.12',
      minTokens: '400',
    },
    {
      id: 'media-draft-priority',
      priority: 5,
      rawAction: 'generate',
      inferenceMode: 'quality',
      inputVideo: 'true',
      audio: 'false',
      draft: 'true',
      resolution: '1080p',
      aspectRatio: '16:9',
      outputDurationMin: '0',
      outputDurationMax: '8',
      inputVideoDurationMin: '3',
      inputVideoDurationMax: '12',
      unitPrice: '0.2',
      minTokens: '1000',
      draftCoefficient: '0.5',
    },
  ];

  const preview = buildMediaTaskPreview(rules, {
    rawAction: 'generate',
    inferenceMode: 'quality',
    inputVideo: 'true',
    audio: 'false',
    resolution: '1080p',
    aspectRatio: '16:9',
    outputDuration: '6',
    inputVideoDuration: '9',
    draft: 'true',
    usageTotalTokens: '600',
  });

  assert.equal(preview.matchedRule?.id, 'media-draft-priority');
  assert.equal(
    preview.conditionSummary,
    buildMediaTaskConditionSummary(rules[1]),
  );
  assert.equal(preview.priceSummary.usageTotalTokens, '600');
  assert.equal(preview.priceSummary.billableTokens, '1000');
  assert.equal(preview.priceSummary.minTokens, '1000');
  assert.equal(preview.priceSummary.unitPrice, '0.2');
  assert.equal(preview.priceSummary.draftCoefficient, '0.5');
  assert.equal(preview.priceSummary.estimatedCost, '0.0001');
  assert.match(preview.formulaSummary, /max\(600, 1000\)/);
  assert.match(preview.formulaSummary, /0\.5/);
  assert.match(preview.formulaSummary, /1,000,000/);
  assert.match(preview.logPreview.detailSummary, /media-draft-priority/);
  assert.match(preview.logPreview.processSummary, /1000/);
  assert.equal(preview.matchedSegmentPreview.priority, 5);
  assert.equal(preview.matchedSegmentPreview.min_tokens, 1000);
});

test('buildMediaTaskPreview returns an unmatched preview when task conditions miss every segment', () => {
  const preview = buildMediaTaskPreview(
    [
      {
        id: 'media-quality-only',
        priority: 1,
        rawAction: 'generate',
        inferenceMode: 'quality',
        unitPrice: '0.3',
      },
    ],
    {
      rawAction: 'generate',
      inferenceMode: 'fast',
      usageTotalTokens: '256',
    },
  );

  assert.equal(preview.matchedRule, null);
  assert.equal(preview.matchedSegmentPreview, null);
  assert.equal(preview.formulaSummary, '');
  assert.equal(preview.priceSummary.estimatedCost, '');
});

test('buildMediaTaskPreview matches output modality, image tier, and tool usage scaffolding fields', () => {
  const preview = buildMediaTaskPreview(
    [
      {
        id: 'media-image',
        priority: 1,
        outputModality: 'image',
        billingUnit: 'per_image',
        imageSizeTier: '2k',
        toolUsageType: 'google_search',
        freeQuota: '100',
        overageThreshold: '250',
        unitPrice: '0.4',
      },
    ],
    {
      outputModality: 'image',
      imageSizeTier: '2k',
      toolUsageType: 'google_search',
      toolUsageCount: '120',
      usageTotalTokens: '1',
    },
  );

  assert.equal(preview.matchedRule?.id, 'media-image');
  assert.equal(preview.priceSummary.toolUsageCount, '120');
  assert.equal(preview.priceSummary.freeQuota, '100');
  assert.equal(preview.priceSummary.overageThreshold, '250');
  assert.equal(preview.priceSummary.billingUnit, 'per_image');
  assert.equal(preview.matchedSegmentPreview.output_modality, 'image');
  assert.equal(preview.matchedSegmentPreview.image_size_tier, '2k');
  assert.equal(preview.matchedSegmentPreview.tool_usage_type, 'google_search');
});

test('buildMediaTaskPreview ignores media selectors that backend P1 matching does not implement', () => {
  const preview = buildMediaTaskPreview(
    [
      {
        id: 'media-backend-aligned',
        priority: 1,
        rawAction: 'generate',
        inputModality: 'image',
        outputModality: 'video',
        imageSizeTier: '4k',
        toolUsageType: 'web_search',
        toolUsageCount: '3',
        inferenceMode: 'quality',
        resolution: '1080p',
        aspectRatio: '16:9',
        outputDurationMin: '0',
        outputDurationMax: '8',
        draft: 'true',
        unitPrice: '0.4',
      },
    ],
    {
      rawAction: 'different-action',
      inputModality: 'audio',
      outputModality: 'image',
      imageSizeTier: '1k',
      toolUsageType: 'code_interpreter',
      toolUsageCount: '99',
      inferenceMode: 'quality',
      resolution: '1080p',
      aspectRatio: '16:9',
      outputDuration: '6',
      draft: 'true',
      usageTotalTokens: '100',
    },
  );

  assert.equal(preview.matchedRule?.id, 'media-backend-aligned');
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

test('normalizeAdvancedPricingConfig maps persisted media_task canonical fields to editor draft shape', () => {
  const normalizedConfig = normalizeAdvancedPricingConfig({
    rule_type: 'media_task',
    display_name: '即梦视频任务',
    task_type: 'video_generation',
    billing_unit: 'total_tokens',
    note: '按媒体任务命中分段计费',
    segments: [
      {
        priority: 10,
        raw_action: 'generate',
        inference_mode: 'fast',
        audio: false,
        input_video: true,
        resolution: '1080p',
        aspect_ratio: '16:9',
        output_duration_min: 0,
        output_duration_max: 5,
        draft: true,
        draft_coefficient: 0.6,
        remark: '首屏短视频',
        unit_price: 0.24,
        min_tokens: 1200,
      },
    ],
  });

  assert.equal(normalizedConfig.ruleType, 'media_task');
  assert.equal(normalizedConfig.displayName, '即梦视频任务');
  assert.equal(normalizedConfig.taskType, 'video_generation');
  assert.equal(normalizedConfig.billingUnit, 'total_tokens');
  assert.equal(normalizedConfig.note, '按媒体任务命中分段计费');
  assert.equal(normalizedConfig.rules.length, 1);
  assert.match(normalizedConfig.rules[0].id, /^media_task-/);
  assert.equal(normalizedConfig.rules[0].priority, '10');
  assert.equal(normalizedConfig.rules[0].rawAction, 'generate');
  assert.equal(normalizedConfig.rules[0].audio, 'false');
  assert.equal(normalizedConfig.rules[0].inputVideo, 'true');
  assert.equal(normalizedConfig.rules[0].draft, 'true');
  assert.equal(normalizedConfig.rules[0].draftCoefficient, '0.6');
  assert.equal(normalizedConfig.rules[0].unitPrice, '0.24');
  assert.equal(normalizedConfig.rules[0].minTokens, '1200');
});

test('normalizeAdvancedPricingConfig round-trips canonical text_segment configs without dropping canonical segments', () => {
  const canonicalConfig = {
    rule_type: 'text_segment',
    display_name: 'Tiered text',
    segment_basis: 'character',
    billing_unit: '1M chars',
    default_price: 6.6,
    note: 'preserve note',
    segments: [
      {
        priority: 10,
        input_min: 0,
        input_max: 1024,
        input_price: 1.2,
        output_min: 0,
        output_max: 512,
        output_price: 0,
        cache_read_price: 0,
        cache_write_price: 0.4,
        service_tier: 'premium',
      },
    ],
  };

  const normalizedConfig = normalizeAdvancedPricingConfig(canonicalConfig);

  assert.equal(normalizedConfig.ruleType, 'text_segment');
  assert.equal(normalizedConfig.displayName, 'Tiered text');
  assert.equal(normalizedConfig.segmentBasis, 'character');
  assert.equal(normalizedConfig.billingUnit, '1M chars');
  assert.equal(normalizedConfig.defaultPrice, '6.6');
  assert.equal(normalizedConfig.note, 'preserve note');
  assert.equal(normalizedConfig.rules.length, 1);
  assert.equal(normalizedConfig.rules[0].priority, '10');
  assert.equal(normalizedConfig.rules[0].inputMin, '0');
  assert.equal(normalizedConfig.rules[0].inputMax, '1024');
  assert.equal(normalizedConfig.rules[0].outputMin, '0');
  assert.equal(normalizedConfig.rules[0].outputMax, '512');
  assert.equal(normalizedConfig.rules[0].inputPrice, '1.2');
  assert.equal(normalizedConfig.rules[0].outputPrice, '0');
  assert.equal(normalizedConfig.rules[0].cacheReadPrice, '0');
  assert.equal(normalizedConfig.rules[0].cacheWritePrice, '0.4');
  assert.equal(normalizedConfig.rules[0].service_tier, 'premium');

  assert.deepEqual(serializeAdvancedPricingConfig(normalizedConfig), canonicalConfig);
});

test('normalizeAdvancedPricingConfig round-trips canonical text_segment configs with modality-aware extension fields', () => {
  const canonicalConfig = {
    rule_type: 'text_segment',
    display_name: 'Gemini 2.5 Flash',
    billing_unit: 'per_million_tokens',
    segments: [
      {
        priority: 1,
        input_modality: 'audio',
        output_modality: 'text',
        billing_unit: 'per_million_tokens',
        cache_storage_price: 1,
        tool_usage_type: 'google_search',
        free_quota: 500,
        overage_threshold: 1000,
        input_price: 1,
        output_price: 2.5,
      },
    ],
  };

  const normalizedConfig = normalizeAdvancedPricingConfig(canonicalConfig);

  assert.equal(normalizedConfig.rules[0].inputModality, 'audio');
  assert.equal(normalizedConfig.rules[0].outputModality, 'text');
  assert.equal(normalizedConfig.rules[0].billingUnit, 'per_million_tokens');
  assert.equal(normalizedConfig.rules[0].cacheStoragePrice, '1');
  assert.equal(normalizedConfig.rules[0].toolUsageType, 'google_search');
  assert.equal(normalizedConfig.rules[0].freeQuota, '500');
  assert.equal(normalizedConfig.rules[0].overageThreshold, '1000');
  assert.deepEqual(serializeAdvancedPricingConfig(normalizedConfig), canonicalConfig);
});

test('normalizeAdvancedPricingConfig round-trips text image tier and tool usage count scaffolding fields', () => {
  const canonicalConfig = {
    rule_type: 'text_segment',
    display_name: 'Gemini Audio',
    billing_unit: 'per_second',
    segments: [
      {
        priority: 1,
        input_modality: 'audio',
        output_modality: 'text',
        image_size_tier: 'hd',
        tool_usage_count: 1000,
        input_price: 1.2,
      },
    ],
  };

  const normalizedConfig = normalizeAdvancedPricingConfig(canonicalConfig);

  assert.equal(normalizedConfig.rules[0].imageSizeTier, 'hd');
  assert.equal(normalizedConfig.rules[0].toolUsageCount, '1000');
  assert.deepEqual(serializeAdvancedPricingConfig(normalizedConfig), canonicalConfig);
});

test('serializeAdvancedPricingConfig emits canonical media_task json and preserves explicit zero and false values', () => {
  const serializedConfig = serializeAdvancedPricingConfig({
    ruleType: 'media_task',
    displayName: 'Veo 任务',
    taskType: 'video_generation',
    billingUnit: 'total_tokens',
    note: '',
    rules: [
      {
        id: 'media_task-1',
        priority: '5',
        rawAction: 'firstTailGenerate',
        inferenceMode: 'quality',
        audio: 'false',
        inputVideo: 'true',
        resolution: '720p',
        aspectRatio: '16:9',
        outputDurationMin: '0',
        outputDurationMax: '8',
        inputVideoDurationMin: '',
        inputVideoDurationMax: '',
        draft: 'true',
        draftCoefficient: '0.5',
        remark: '草稿模式',
        unitPrice: '0.18',
        minTokens: '0',
      },
    ],
  });

  assert.deepEqual(serializedConfig, {
    rule_type: 'media_task',
    display_name: 'Veo 任务',
    task_type: 'video_generation',
    billing_unit: 'total_tokens',
    segments: [
      {
        priority: 5,
        raw_action: 'firstTailGenerate',
        inference_mode: 'quality',
        audio: false,
        input_video: true,
        resolution: '720p',
        aspect_ratio: '16:9',
        output_duration_min: 0,
        output_duration_max: 8,
        draft: true,
        draft_coefficient: 0.5,
        remark: '草稿模式',
        unit_price: 0.18,
        min_tokens: 0,
      },
    ],
  });
});

test('serializeAdvancedPricingConfig emits media task extension fields for modality, tier, and tool scaffolding', () => {
  const serializedConfig = serializeAdvancedPricingConfig({
    ruleType: 'media_task',
    displayName: 'Gemini Image',
    taskType: 'image_generation',
    billingUnit: 'per_image',
    rules: [
      {
        id: 'media_task-1',
        priority: '1',
        inputModality: 'image',
        outputModality: 'image',
        billingUnit: 'per_image',
        imageSizeTier: '2k',
        toolUsageType: 'google_search',
        toolUsageCount: '3',
        freeQuota: '100',
        overageThreshold: '250',
        unitPrice: '0.4',
      },
    ],
  });

  assert.deepEqual(serializedConfig, {
    rule_type: 'media_task',
    display_name: 'Gemini Image',
    task_type: 'image_generation',
    billing_unit: 'per_image',
    segments: [
      {
        priority: 1,
        input_modality: 'image',
        output_modality: 'image',
        billing_unit: 'per_image',
        image_size_tier: '2k',
        tool_usage_type: 'google_search',
        tool_usage_count: 3,
        free_quota: 100,
        overage_threshold: 250,
        unit_price: 0.4,
      },
    ],
  });
});

test('serializeAdvancedPricingMap keeps canonical text rules while dropping models whose media rules were deleted', () => {
  const advancedPricingMap = {
    'text-model': {
      rule_type: 'text_segment',
      display_name: 'Tiered text',
      segment_basis: 'token',
      billing_unit: '1M tokens',
      default_price: 3.3,
      note: 'preserve note',
      segments: [
        {
          priority: 10,
          input_min: 0,
          input_max: 100,
          input_price: 1.2,
          output_price: 2.4,
          cache_read_price: 0,
        },
      ],
    },
    'media-model': {
      ruleType: 'media_task',
      taskType: 'video_generation',
      billingUnit: 'total_tokens',
      note: 'delete me',
      rules: [],
    },
  };

  assert.deepEqual(serializeAdvancedPricingMap(advancedPricingMap), {
    'text-model': {
      rule_type: 'text_segment',
      display_name: 'Tiered text',
      segment_basis: 'token',
      billing_unit: '1M tokens',
      default_price: 3.3,
      note: 'preserve note',
      segments: [
        {
          priority: 10,
          input_min: 0,
          input_max: 100,
          input_price: 1.2,
          output_price: 2.4,
          cache_read_price: 0,
        },
      ],
    },
  });
});

test('getAdvancedPricingMapValidationErrors reports invalid drafts from non-selected models', () => {
  const errors = getAdvancedPricingMapValidationErrors({
    'valid-text-model': {
      ruleType: 'text_segment',
      rules: [
        {
          id: 'text-1',
          enabled: true,
          priority: '1',
          inputMin: '0',
          inputMax: '32000',
          outputMin: '',
          outputMax: '',
          inputPrice: '0.2',
          outputPrice: '0.4',
        },
      ],
    },
    'invalid-media-model': {
      ruleType: 'media_task',
      taskType: 'video_generation',
      billingUnit: 'total_tokens',
      rules: [
        {
          id: 'media-1',
          priority: '',
          unitPrice: '',
        },
      ],
    },
  });

  assert.ok(
    errors.some(
      (error) =>
        error.includes('invalid-media-model') &&
        error.includes('priority is required'),
    ),
  );
  assert.ok(
    errors.some(
      (error) =>
        error.includes('invalid-media-model') &&
        error.includes('unit_price is required'),
    ),
  );
});

test('mergeAdvancedPricingDraftMap preserves dirty model drafts while refreshing clean models from server', () => {
  const previousDraftMap = {
    dirtyModel: {
      ruleType: 'media_task',
      taskType: 'video_generation',
      billingUnit: 'total_tokens',
      rules: [{ id: 'dirty-rule', priority: '9', unitPrice: '0.9' }],
    },
    cleanModel: {
      ruleType: 'text_segment',
      rules: [
        {
          id: 'clean-old',
          enabled: true,
          priority: '1',
          inputMin: '0',
          inputMax: '1024',
          inputPrice: '0.1',
          outputPrice: '0.2',
        },
      ],
    },
  };

  const mergedDraftMap = mergeAdvancedPricingDraftMap(
    previousDraftMap,
    {
      dirtyModel: {
        ruleType: 'media_task',
        taskType: 'video_generation',
        billingUnit: 'total_tokens',
        rules: [{ id: 'server-rule', priority: '1', unitPrice: '0.1' }],
      },
      cleanModel: {
        ruleType: 'text_segment',
        rules: [
          {
            id: 'clean-new',
            enabled: true,
            priority: '2',
            inputMin: '0',
            inputMax: '2048',
            inputPrice: '0.2',
            outputPrice: '0.4',
          },
        ],
      },
    },
    new Set(['dirtyModel']),
  );

  assert.equal(mergedDraftMap.dirtyModel.rules[0].id, 'dirty-rule');
  assert.equal(mergedDraftMap.cleanModel.rules[0].id, 'clean-new');
});

test('mergeAdvancedPricingModeDraftMap preserves dirty model modes while refreshing clean models from server', () => {
  const mergedModeMap = mergeAdvancedPricingModeDraftMap(
    {
      dirtyModel: 'advanced',
      cleanModel: 'per_token',
    },
    {
      dirtyModel: 'per_request',
      cleanModel: 'advanced',
    },
    new Set(['dirtyModel']),
  );

  assert.equal(mergedModeMap.dirtyModel, 'advanced');
  assert.equal(mergedModeMap.cleanModel, 'advanced');
});

test('buildAdvancedPricingSaveMaps patches only dirty models onto the latest server snapshot', () => {
  const payload = buildAdvancedPricingSaveMaps({
    latestModeMap: {
      cleanModel: 'per_request',
      untouchedModel: 'advanced',
    },
    latestRulesMap: {
      cleanModel: {
        rule_type: 'text_segment',
        display_name: 'server-clean',
        segments: [
          {
            priority: 10,
            input_min: 0,
            input_max: 100,
            input_price: 1.2,
          },
        ],
      },
      untouchedModel: {
        rule_type: 'text_segment',
        segments: [
          {
            priority: 10,
            input_min: 0,
            input_max: 100,
            input_price: 1.5,
          },
        ],
      },
    },
    draftModeMap: {
      dirtyModel: 'advanced',
      cleanModel: 'per_token',
    },
    draftConfigMap: {
      dirtyModel: {
        ruleType: 'text_segment',
        displayName: 'dirty-local',
        rules: [
          {
            id: 'dirty-1',
            enabled: true,
            priority: '20',
            inputMin: '0',
            inputMax: '200',
            inputPrice: '2.4',
            outputPrice: '4.8',
          },
        ],
      },
      cleanModel: {
        ruleType: 'text_segment',
        displayName: 'stale-local-clean',
        rules: [
          {
            id: 'clean-stale',
            enabled: true,
            priority: '99',
            inputMin: '0',
            inputMax: '999',
            inputPrice: '9.9',
          },
        ],
      },
    },
    dirtyModelNames: new Set(['dirtyModel']),
    fixedBillingModes: {
      dirtyModel: 'per_token',
      cleanModel: 'per_token',
      untouchedModel: 'per_token',
    },
  });

  assert.equal(payload.modeMap.cleanModel, 'per_request');
  assert.equal(payload.modeMap.dirtyModel, 'advanced');
  assert.equal(payload.modeMap.untouchedModel, 'advanced');
  assert.equal(payload.rulesMap.cleanModel.display_name, 'server-clean');
  assert.equal(payload.rulesMap.dirtyModel.display_name, 'dirty-local');
  assert.equal(payload.rulesMap.untouchedModel.rule_type, 'text_segment');
});

test('buildAdvancedPricingSaveMaps removes deleted dirty rules and drops stale advanced mode entries without rules', () => {
  const payload = buildAdvancedPricingSaveMaps({
    latestModeMap: {
      deletedDirty: 'advanced',
      staleAdvanced: 'advanced',
    },
    latestRulesMap: {
      deletedDirty: {
        rule_type: 'text_segment',
        segments: [
          {
            priority: 10,
            input_min: 0,
            input_max: 100,
            input_price: 1.2,
          },
        ],
      },
    },
    draftModeMap: {
      deletedDirty: 'advanced',
    },
    draftConfigMap: {
      deletedDirty: {
        ruleType: 'text_segment',
        rules: [],
      },
    },
    dirtyModelNames: new Set(['deletedDirty']),
    fixedBillingModes: {
      deletedDirty: 'per_request',
      staleAdvanced: 'per_token',
    },
  });

  assert.equal(payload.modeMap.deletedDirty, 'per_request');
  assert.equal('staleAdvanced' in payload.modeMap, false);
  assert.equal('deletedDirty' in payload.rulesMap, false);
});

test('validateMediaTaskConfig rejects media_task configs without segments', () => {
  const errors = validateMediaTaskConfig({
    ruleType: 'media_task',
    displayName: 'empty',
    taskType: 'video_generation',
    billingUnit: 'total_tokens',
    note: '',
    rules: [],
  });

  assert.ok(
    errors.some((error) => error.includes('segments') && error.includes('required')),
  );
});

test('getAdvancedPricingValidationErrors allows saving an empty media rule set so the model config can be removed', () => {
  assert.deepEqual(
    getAdvancedPricingValidationErrors({
      ruleType: 'media_task',
      taskType: 'video_generation',
      billingUnit: 'total_tokens',
      rules: [],
    }),
    [],
  );

  assert.ok(
    getAdvancedPricingValidationErrors({
      ruleType: 'media_task',
      taskType: 'video_generation',
      billingUnit: 'total_tokens',
      rules: [
        {
          id: 'media_task-1',
          priority: '',
          unitPrice: '',
        },
      ],
    }).length > 0,
  );
});

test('validateMediaTaskConfig rejects half-open and decimal media task duration ranges', () => {
  const errors = validateMediaTaskConfig({
    ruleType: 'media_task',
    displayName: 'invalid-durations',
    taskType: 'video_generation',
    billingUnit: 'total_tokens',
    note: '',
    rules: [
      {
        id: 'media_task-1',
        priority: '1',
        outputDurationMin: '3',
        outputDurationMax: '',
        unitPrice: '0.2',
      },
      {
        id: 'media_task-2',
        priority: '2',
        inputVideoDurationMin: '1.5',
        inputVideoDurationMax: '4',
        unitPrice: '0.3',
      },
    ],
  });

  assert.ok(
    errors.some(
      (error) =>
        error.includes('media_task-1') &&
        error.includes('output_duration') &&
        error.includes('min') &&
        error.includes('max'),
    ),
  );
  assert.ok(
    errors.some(
      (error) =>
        error.includes('media_task-2') &&
        error.includes('input_video_duration') &&
        error.includes('integer'),
    ),
  );
});

test('serializeAdvancedPricingConfig omits invalid media task duration ranges from canonical json', () => {
  const serializedConfig = serializeAdvancedPricingConfig({
    ruleType: 'media_task',
    displayName: 'sanitize-invalid-durations',
    taskType: 'video_generation',
    billingUnit: 'total_tokens',
    note: '',
    rules: [
      {
        id: 'media_task-1',
        priority: '5',
        outputDurationMin: '0.5',
        outputDurationMax: '8',
        inputVideoDurationMin: '3',
        inputVideoDurationMax: '',
        unitPrice: '0.18',
      },
      {
        id: 'media_task-2',
        priority: '6',
        outputDurationMin: '0',
        outputDurationMax: '12',
        inputVideoDurationMin: '5',
        inputVideoDurationMax: '15',
        unitPrice: '0.28',
      },
    ],
  });

  assert.deepEqual(serializedConfig.segments[0], {
    priority: 5,
    unit_price: 0.18,
  });
  assert.deepEqual(serializedConfig.segments[1], {
    priority: 6,
    output_duration_min: 0,
    output_duration_max: 12,
    input_video_duration_min: 5,
    input_video_duration_max: 15,
    unit_price: 0.28,
  });
});

test('validateMediaTaskConfig rejects missing required fields, duplicate priorities, and inverted ranges', () => {
  const errors = validateMediaTaskConfig({
    ruleType: 'media_task',
    displayName: '',
    taskType: '',
    billingUnit: '',
    note: '',
    rules: [
      {
        id: 'media_task-1',
        priority: '1',
        outputDurationMin: '10',
        outputDurationMax: '5',
        unitPrice: '',
      },
      {
        id: 'media_task-2',
        priority: '1',
        inputVideoDurationMin: '12',
        inputVideoDurationMax: '6',
        unitPrice: '0.3',
      },
    ],
  });

  assert.ok(errors.some((error) => error.includes('task_type')));
  assert.ok(errors.some((error) => error.includes('billing_unit')));
  assert.ok(errors.some((error) => error.includes('unit_price')));
  assert.ok(errors.some((error) => error.includes('priority 1')));
  assert.ok(
    errors.some((error) => error.includes('media_task-1') && error.includes('output_duration')),
  );
  assert.ok(
    errors.some((error) => error.includes('media_task-2') && error.includes('input_video_duration')),
  );
});

test('buildMediaTaskConditionSummary summarizes key media task filters for operator preview', () => {
  const summary = buildMediaTaskConditionSummary({
    rawAction: 'generate',
    inferenceMode: 'fast',
    audio: 'false',
    inputVideo: 'true',
    resolution: '1080p',
    aspectRatio: '16:9',
    outputDurationMin: '0',
    outputDurationMax: '5',
    draft: 'true',
    toolUsageCount: '100',
    minTokens: '1200',
  });

  assert.match(summary, /raw_action=generate/);
  assert.match(summary, /fast/);
  assert.match(summary, /1080p/);
  assert.match(summary, /16:9/);
  assert.match(summary, /0/);
  assert.match(summary, /5/);
  assert.match(summary, /100/);
  assert.match(summary, /1200/);
});

test('saveAdvancedPricingOptions persists canonical AdvancedPricingConfig in a single request', async () => {
  const calls = [];
  const api = {
    put: async (url, payload) => {
      calls.push({ url, payload });
      return {
        data: {
          success: true,
        },
      };
    },
  };

  await saveAdvancedPricingOptions({
    api,
    t: (key) => key,
    savePayload: {
      rulesMap: {
        'dirty-model': {
          rule_type: 'text_segment',
          segments: [
            {
              priority: 1,
              input_min: 0,
              input_max: 128,
              input_price: 0.2,
            },
          ],
        },
      },
      modeMap: {
        'dirty-model': 'advanced',
      },
    },
  });

  assert.deepEqual(calls, [
    {
      url: '/api/option/',
      payload: {
        key: 'AdvancedPricingConfig',
        value: JSON.stringify(
          {
            billing_mode: {
              'dirty-model': 'advanced',
            },
            rules: {
              'dirty-model': {
                rule_type: 'text_segment',
                segments: [
                  {
                    priority: 1,
                    input_min: 0,
                    input_max: 128,
                    input_price: 0.2,
                  },
                ],
              },
            },
          },
          null,
          2,
        ),
      },
    },
  ]);
});

test('saveAdvancedPricingOptions surfaces backend failure from canonical config save', async () => {
  const callKeys = [];

  await assert.rejects(
    saveAdvancedPricingOptions({
      api: {
        put: async (url, payload) => {
          callKeys.push(payload.key);
          return { data: { success: false, message: 'mode save failed' } };
        },
      },
      t: (key) => key,
      savePayload: {
        rulesMap: { 'dirty-model': { rule_type: 'text_segment', segments: [] } },
        modeMap: { 'dirty-model': 'advanced' },
      },
    }),
    /mode save failed/,
  );

  assert.deepEqual(callKeys, ['AdvancedPricingConfig']);
});
