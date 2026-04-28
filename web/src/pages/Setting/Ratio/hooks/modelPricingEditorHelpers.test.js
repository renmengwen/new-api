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
  BILLING_MODE_ADVANCED,
  BILLING_MODE_CHANGE_CONFIRM_CONTENT,
  BILLING_MODE_CHANGE_CONFIRM_TITLE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_PER_TOKEN,
  buildAdvancedPricingConfigPayloadForPricingEditor,
  buildAdvancedPricingModePayload,
  copyAdvancedPricingRulesForModels,
  canUseAdvancedBilling,
  hasEditableFixedPricingConfig,
  isBasePricingUnset,
  resolveBatchBillingModeConfirmation,
  resolveBillingMode,
} from './modelPricingEditorHelpers.js';

test('resolveBillingMode keeps explicit mode separate from legacy inferred mode', () => {
  assert.deepEqual(
    resolveBillingMode({
      explicitMode: BILLING_MODE_PER_REQUEST,
      fixedPrice: '',
      advancedRuleType: '',
    }),
    {
      billingMode: BILLING_MODE_PER_REQUEST,
      explicitBillingMode: BILLING_MODE_PER_REQUEST,
      hasExplicitBillingMode: true,
      hasInvalidExplicitAdvancedMode: false,
    },
  );

  assert.deepEqual(
    resolveBillingMode({
      explicitMode: '',
      fixedPrice: '3',
      advancedRuleType: '',
    }),
    {
      billingMode: BILLING_MODE_PER_REQUEST,
      explicitBillingMode: '',
      hasExplicitBillingMode: false,
      hasInvalidExplicitAdvancedMode: false,
    },
  );

  assert.deepEqual(
    resolveBillingMode({
      explicitMode: '',
      fixedPrice: '',
      advancedRuleType: '',
    }),
    {
      billingMode: BILLING_MODE_PER_TOKEN,
      explicitBillingMode: '',
      hasExplicitBillingMode: false,
      hasInvalidExplicitAdvancedMode: false,
    },
  );

  assert.deepEqual(
    resolveBillingMode({
      explicitMode: BILLING_MODE_ADVANCED,
      fixedPrice: '3',
      advancedRuleType: '',
    }),
    {
      billingMode: BILLING_MODE_PER_REQUEST,
      explicitBillingMode: '',
      hasExplicitBillingMode: false,
      hasInvalidExplicitAdvancedMode: true,
    },
  );

  assert.deepEqual(
    resolveBillingMode({
      explicitMode: BILLING_MODE_ADVANCED,
      fixedPrice: '',
      advancedRuleType: 'tiered',
    }),
    {
      billingMode: BILLING_MODE_ADVANCED,
      explicitBillingMode: BILLING_MODE_ADVANCED,
      hasExplicitBillingMode: true,
      hasInvalidExplicitAdvancedMode: false,
    },
  );
});

test('buildAdvancedPricingModePayload merges latest server state without persisting unchanged inferred modes', () => {
  const merged = buildAdvancedPricingModePayload({
    latestModeMap: {
      preserved_remote: BILLING_MODE_ADVANCED,
      dirty_model: BILLING_MODE_PER_TOKEN,
      explicit_existing: BILLING_MODE_PER_REQUEST,
    },
    latestRulesMap: {
      dirty_model: {
        rule_type: 'tiered',
      },
    },
    models: [
      {
        name: 'dirty_model',
        billingMode: BILLING_MODE_ADVANCED,
        hasExplicitBillingMode: false,
      },
      {
        name: 'explicit_existing',
        billingMode: BILLING_MODE_PER_REQUEST,
        hasExplicitBillingMode: true,
      },
      {
        name: 'explicit_missing_remote',
        billingMode: BILLING_MODE_PER_TOKEN,
        hasExplicitBillingMode: true,
      },
      {
        name: 'inferred_unchanged',
        billingMode: BILLING_MODE_PER_REQUEST,
        hasExplicitBillingMode: false,
      },
    ],
    dirtyModeNames: new Set(['dirty_model']),
  });

  assert.deepEqual(merged, {
    preserved_remote: BILLING_MODE_ADVANCED,
    dirty_model: BILLING_MODE_ADVANCED,
    explicit_existing: BILLING_MODE_PER_REQUEST,
    explicit_missing_remote: BILLING_MODE_PER_TOKEN,
  });
  assert.equal('inferred_unchanged' in merged, false);
});

test('buildAdvancedPricingModePayload removes stale invalid advanced entries from latest server state', () => {
  const merged = buildAdvancedPricingModePayload({
    latestModeMap: {
      stale_advanced: BILLING_MODE_ADVANCED,
      unrelated_explicit: BILLING_MODE_PER_REQUEST,
    },
    models: [
      {
        name: 'stale_advanced',
        billingMode: BILLING_MODE_PER_REQUEST,
        hasExplicitBillingMode: false,
        hasInvalidExplicitAdvancedMode: true,
      },
      {
        name: 'unrelated_explicit',
        billingMode: BILLING_MODE_PER_REQUEST,
        hasExplicitBillingMode: true,
        hasInvalidExplicitAdvancedMode: false,
      },
    ],
  });

  assert.deepEqual(merged, {
    unrelated_explicit: BILLING_MODE_PER_REQUEST,
  });
});

test('buildAdvancedPricingModePayload preserves refreshed advanced entries when latest rules are now valid', () => {
  const merged = buildAdvancedPricingModePayload({
    latestModeMap: {
      refreshed_advanced: BILLING_MODE_ADVANCED,
    },
    latestRulesMap: {
      refreshed_advanced: {
        rule_type: 'tiered',
      },
    },
    models: [
      {
        name: 'refreshed_advanced',
        billingMode: BILLING_MODE_PER_REQUEST,
        hasExplicitBillingMode: false,
        hasInvalidExplicitAdvancedMode: true,
      },
    ],
  });

  assert.deepEqual(merged, {
    refreshed_advanced: BILLING_MODE_ADVANCED,
  });
});

test('buildAdvancedPricingModePayload drops stale explicit advanced modes when latest rules no longer exist', () => {
  const merged = buildAdvancedPricingModePayload({
    latestModeMap: {
      stale_explicit_advanced: BILLING_MODE_ADVANCED,
    },
    latestRulesMap: {},
    models: [
      {
        name: 'stale_explicit_advanced',
        billingMode: BILLING_MODE_ADVANCED,
        hasExplicitBillingMode: true,
        hasInvalidExplicitAdvancedMode: false,
      },
    ],
  });

  assert.deepEqual(merged, {});
});

test('buildAdvancedPricingModePayload does not write advanced mode back for dirty models without latest rules', () => {
  const merged = buildAdvancedPricingModePayload({
    latestModeMap: {
      dirty_without_rule: BILLING_MODE_PER_REQUEST,
    },
    latestRulesMap: {},
    models: [
      {
        name: 'dirty_without_rule',
        billingMode: BILLING_MODE_ADVANCED,
        hasExplicitBillingMode: false,
        hasInvalidExplicitAdvancedMode: false,
      },
    ],
    dirtyModeNames: new Set(['dirty_without_rule']),
  });

  assert.deepEqual(merged, {
    dirty_without_rule: BILLING_MODE_PER_REQUEST,
  });
});

test('copyAdvancedPricingRulesForModels deep copies source rules to selected targets', () => {
  const sourceRuleSet = {
    rule_type: 'text_segment',
    segments: [
      {
        priority: 10,
        input_min: 0,
        input_max: 100,
        input_price: 1.2,
      },
    ],
  };
  const result = copyAdvancedPricingRulesForModels({
    sourceModelName: 'source-model',
    targetModelNames: ['target-model', 'other-target'],
    rulesMap: {
      'source-model': sourceRuleSet,
      'target-model': {
        rule_type: 'media_task',
        segments: [{ priority: 1, unit_price: 0.5 }],
      },
    },
  });

  assert.deepEqual(result.rulesMap['target-model'], sourceRuleSet);
  assert.deepEqual(result.rulesMap['other-target'], sourceRuleSet);
  assert.equal(result.advancedRuleType, 'text_segment');
  assert.notEqual(result.rulesMap['target-model'], sourceRuleSet);
  assert.notEqual(result.rulesMap['target-model'].segments, sourceRuleSet.segments);
});

test('buildAdvancedPricingConfigPayloadForPricingEditor includes copied advanced rules before validating advanced modes', () => {
  const copiedRuleSet = {
    rule_type: 'text_segment',
    segments: [{ priority: 10, input_min: 0, input_price: 1.2 }],
  };
  const payload = buildAdvancedPricingConfigPayloadForPricingEditor({
    latestModeMap: {
      existing: BILLING_MODE_PER_REQUEST,
    },
    latestRulesMap: {
      existing: {
        rule_type: 'media_task',
        segments: [{ priority: 1, unit_price: 0.5 }],
      },
    },
    draftRulesMap: {
      target: copiedRuleSet,
    },
    copiedRuleNames: new Set(['target']),
    models: [
      {
        name: 'target',
        billingMode: BILLING_MODE_ADVANCED,
        hasExplicitBillingMode: false,
      },
    ],
    dirtyModeNames: new Set(['target']),
  });

  assert.deepEqual(payload, {
    billing_mode: {
      existing: BILLING_MODE_PER_REQUEST,
      target: BILLING_MODE_ADVANCED,
    },
    rules: {
      existing: {
        rule_type: 'media_task',
        segments: [{ priority: 1, unit_price: 0.5 }],
      },
      target: copiedRuleSet,
    },
  });
  assert.notEqual(payload.rules.target, copiedRuleSet);
});

test('advanced availability and unset state require a real advanced rule type', () => {
  assert.equal(
    canUseAdvancedBilling({
      billingMode: BILLING_MODE_ADVANCED,
      advancedRuleType: '',
    }),
    false,
  );
  assert.equal(
    isBasePricingUnset({
      billingMode: BILLING_MODE_ADVANCED,
      fixedPrice: '',
      inputPrice: '',
      advancedRuleType: '',
    }),
    true,
  );

  assert.equal(
    canUseAdvancedBilling({
      advancedRuleType: 'tiered',
    }),
    true,
  );
  assert.equal(
    isBasePricingUnset({
      billingMode: BILLING_MODE_ADVANCED,
      fixedPrice: '',
      inputPrice: '',
      advancedRuleType: 'tiered',
    }),
    false,
  );
});

test('hasEditableFixedPricingConfig only reflects current editable pricing fields instead of raw ratio snapshots', () => {
  assert.equal(
    hasEditableFixedPricingConfig({
      fixedPrice: '',
      inputPrice: '',
      completionPrice: '',
      cachePrice: '',
      createCachePrice: '',
      imagePrice: '',
      audioInputPrice: '',
      audioOutputPrice: '',
      rawRatios: {
        modelRatio: '0.5',
        completionRatio: '2',
      },
    }),
    false,
  );

  assert.equal(
    hasEditableFixedPricingConfig({
      fixedPrice: '',
      inputPrice: '',
      completionPrice: '',
      cachePrice: '0.3',
      createCachePrice: '',
      imagePrice: '',
      audioInputPrice: '',
      audioOutputPrice: '',
      rawRatios: {
        cacheRatio: '0.1',
      },
    }),
    true,
  );
});

test('resolveBatchBillingModeConfirmation only requires confirmation when batch apply changes target billing modes', () => {
  assert.deepEqual(
    resolveBatchBillingModeConfirmation({
      selectedModel: {
        name: 'template',
        billingMode: BILLING_MODE_ADVANCED,
      },
      selectedModelNames: ['same-mode', 'different-mode', 'missing'],
      models: [
        {
          name: 'same-mode',
          billingMode: BILLING_MODE_ADVANCED,
        },
        {
          name: 'different-mode',
          billingMode: BILLING_MODE_PER_TOKEN,
        },
      ],
    }),
    {
      requiresConfirmation: true,
      changedModelNames: ['different-mode'],
      title: BILLING_MODE_CHANGE_CONFIRM_TITLE,
      content: BILLING_MODE_CHANGE_CONFIRM_CONTENT,
    },
  );

  assert.deepEqual(
    resolveBatchBillingModeConfirmation({
      selectedModel: {
        name: 'template',
        billingMode: BILLING_MODE_PER_REQUEST,
      },
      selectedModelNames: ['same-mode'],
      models: [
        {
          name: 'same-mode',
          billingMode: BILLING_MODE_PER_REQUEST,
        },
      ],
    }),
    {
      requiresConfirmation: false,
      changedModelNames: [],
      title: BILLING_MODE_CHANGE_CONFIRM_TITLE,
      content: BILLING_MODE_CHANGE_CONFIRM_CONTENT,
    },
  );
});
