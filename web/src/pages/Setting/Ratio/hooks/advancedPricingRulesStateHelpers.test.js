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

const loadHelpers = async () => {
  try {
    return await import('./advancedPricingRulesStateHelpers.js');
  } catch (error) {
    return {};
  }
};

const createOptions = (overrides = {}) => ({
  AdvancedPricingMode: '{}',
  AdvancedPricingRules: '{}',
  ModelPrice: '{}',
  ModelRatio: '{}',
  ...overrides,
});

test('advanced pricing state uses launch model only as the initial selection', async () => {
  const { buildAdvancedPricingState } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingState, 'function');

  const firstPass = buildAdvancedPricingState({
    options: createOptions(),
    enabledModelNames: ['alpha', 'launch-model'],
    launchModelName: 'launch-model',
    previousSelectedModelName: 'alpha',
  });

  assert.equal(firstPass.selectedModelName, 'launch-model');

  const secondPass = buildAdvancedPricingState({
    options: createOptions(),
    enabledModelNames: ['alpha', 'launch-model'],
    launchModelName: '',
    previousSelectedModelName: 'alpha',
  });

  assert.equal(secondPass.selectedModelName, 'alpha');
});

test('advanced pricing state keeps the current launch-only model until selection changes', async () => {
  const { buildAdvancedPricingState } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingState, 'function');

  const firstPass = buildAdvancedPricingState({
    options: createOptions(),
    launchModelName: 'launch-only',
  });

  assert.deepEqual(firstPass.models.map((model) => model.name), ['launch-only']);
  assert.equal(firstPass.selectedModelName, 'launch-only');

  const secondPass = buildAdvancedPricingState({
    options: createOptions(),
    enabledModelNames: [],
    previousDraftRules: {
      ...firstPass.draftRules,
      'launch-only': {
        ...firstPass.draftRules['launch-only'],
        display_name: 'Local draft',
      },
    },
    previousDraftBillingModes: firstPass.draftBillingModes,
    previousSelectedModelName: firstPass.selectedModelName,
    preserveDraftRuleModelNames: new Set(['launch-only']),
  });

  assert.deepEqual(secondPass.models.map((model) => model.name), ['launch-only']);
  assert.equal(secondPass.selectedModelName, 'launch-only');
  assert.equal(secondPass.draftRules['launch-only'].display_name, 'Local draft');

  const thirdPass = buildAdvancedPricingState({
    options: createOptions(),
    enabledModelNames: ['alpha'],
    previousDraftRules: secondPass.draftRules,
    previousDraftBillingModes: secondPass.draftBillingModes,
    previousSelectedModelName: 'alpha',
    preserveDraftRuleModelNames: new Set(['launch-only']),
  });

  assert.deepEqual(thirdPass.models.map((model) => model.name), ['alpha']);
  assert.equal(thirdPass.selectedModelName, 'alpha');
  assert.equal(thirdPass.draftRules['launch-only'], undefined);
});

test('advanced pricing state keeps existing drafts while appending defaults for newly enabled models', async () => {
  const { buildAdvancedPricingState, buildRuleDraft } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingState, 'function');
  assert.equal(typeof buildRuleDraft, 'function');

  const initialState = buildAdvancedPricingState({
    options: createOptions({
      AdvancedPricingMode: JSON.stringify({
        alpha: 'advanced',
      }),
      AdvancedPricingRules: JSON.stringify({
        alpha: {
          rule_type: 'text_segment',
          display_name: 'Alpha server',
          note: 'from server',
          default_price: 3,
        },
      }),
    }),
  });

  const nextState = buildAdvancedPricingState({
    options: createOptions({
      AdvancedPricingMode: JSON.stringify({
        alpha: 'advanced',
      }),
      AdvancedPricingRules: JSON.stringify({
        alpha: {
          rule_type: 'text_segment',
          display_name: 'Alpha saved remotely',
          note: 'from refresh',
          default_price: 6,
        },
      }),
    }),
    enabledModelNames: ['alpha', 'beta'],
    previousDraftRules: {
      ...initialState.draftRules,
      alpha: {
        ...initialState.draftRules.alpha,
        display_name: 'Alpha local draft',
        default_price: '99',
      },
    },
    previousDraftBillingModes: {
      ...initialState.draftBillingModes,
      alpha: 'per_request',
    },
    preserveDraftRuleModelNames: new Set(['alpha']),
    preserveDraftBillingModeModelNames: new Set(['alpha']),
  });

  assert.equal(nextState.draftRules.alpha.display_name, 'Alpha local draft');
  assert.equal(nextState.draftRules.alpha.default_price, '99');
  assert.equal(nextState.draftBillingModes.alpha, 'per_request');
  assert.deepEqual(nextState.draftRules.beta, buildRuleDraft('text_segment', {}));
  assert.equal(nextState.draftBillingModes.beta, 'per_token');
});

test('advanced pricing state rebuilds bootstrap defaults from server options when the model was not edited', async () => {
  const { buildAdvancedPricingState } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingState, 'function');

  const bootstrapState = buildAdvancedPricingState({
    options: createOptions(),
    enabledModelNames: ['alpha'],
  });

  assert.deepEqual(bootstrapState.draftRules.alpha, {
    display_name: '',
    note: '',
    rule_type: 'text_segment',
    segment_basis: 'token',
    billing_unit: '1K tokens',
    default_price: '',
    segments_text: '',
  });
  assert.equal(bootstrapState.draftBillingModes.alpha, 'per_token');

  const hydratedState = buildAdvancedPricingState({
    options: createOptions({
      AdvancedPricingMode: JSON.stringify({
        alpha: 'advanced',
      }),
      AdvancedPricingRules: JSON.stringify({
        alpha: {
          rule_type: 'text_segment',
          display_name: 'Alpha server',
          note: 'hydrated from server',
          default_price: 6,
        },
      }),
    }),
    enabledModelNames: ['alpha'],
    previousDraftRules: bootstrapState.draftRules,
    previousDraftBillingModes: bootstrapState.draftBillingModes,
  });

  assert.equal(hydratedState.draftRules.alpha.display_name, 'Alpha server');
  assert.equal(hydratedState.draftRules.alpha.note, 'hydrated from server');
  assert.equal(hydratedState.draftRules.alpha.default_price, '6');
  assert.equal(hydratedState.draftBillingModes.alpha, 'advanced');
});

test('buildRuleDraft clears fields from the other rule type when switching modes', async () => {
  const { buildRuleDraft } = await loadHelpers();

  assert.equal(typeof buildRuleDraft, 'function');

  const mediaDraft = buildRuleDraft('media_task', {
    rule_type: 'text_segment',
    display_name: 'Shared name',
    note: 'Shared note',
    segment_basis: 'character',
    billing_unit: '1M chars',
    default_price: 7,
    segments_text: '0-100: 7',
  });

  assert.deepEqual(mediaDraft, {
    display_name: 'Shared name',
    note: 'Shared note',
    rule_type: 'media_task',
    task_type: 'image_generation',
    billing_unit: 'task',
    unit_price: '',
  });

  const textDraft = buildRuleDraft('text_segment', {
    rule_type: 'media_task',
    display_name: 'Shared name',
    note: 'Shared note',
    task_type: 'video_generation',
    billing_unit: 'minute',
    unit_price: 8,
  });

  assert.deepEqual(textDraft, {
    display_name: 'Shared name',
    note: 'Shared note',
    rule_type: 'text_segment',
    segment_basis: 'token',
    billing_unit: '1K tokens',
    default_price: '',
    segments_text: '',
  });
});

test('buildRuleDraft hydrates canonical backend rules into shell fields', async () => {
  const { buildRuleDraft } = await loadHelpers();

  assert.equal(typeof buildRuleDraft, 'function');

  assert.deepEqual(
    buildRuleDraft('text_segment', {
      rule_type: 'text_segment',
      display_name: 'Tiered text',
      segment_basis: 'character',
      billing_unit: '1M chars',
      default_price: 6.6,
      note: 'saved text note',
      segments: [
        {
          priority: 10,
          input_min: 0,
          input_max: 100,
          input_price: 1.2,
        },
        {
          priority: 20,
          input_min: 101,
          input_max: 200,
          input_price: 2.4,
        },
      ],
    }),
    {
      display_name: 'Tiered text',
      note: 'saved text note',
      rule_type: 'text_segment',
      segment_basis: 'character',
      billing_unit: '1M chars',
      default_price: '6.6',
      segments_text: '0-100: 1.2\n101-200: 2.4',
    },
  );

  assert.deepEqual(
    buildRuleDraft('media_task', {
      rule_type: 'media_task',
      display_name: 'Tiered media',
      task_type: 'video_generation',
      billing_unit: 'minute',
      note: 'saved media note',
      segments: [
        {
          priority: 10,
          unit_price: 8.8,
          remark: 'legacy remark',
        },
      ],
    }),
    {
      display_name: 'Tiered media',
      note: 'saved media note',
      rule_type: 'media_task',
      task_type: 'video_generation',
      billing_unit: 'minute',
      unit_price: '8.8',
    },
  );
});

test('advanced pricing state merges enabled models, stored json models, and the launch model', async () => {
  const { buildAdvancedPricingState } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingState, 'function');

  const state = buildAdvancedPricingState({
    options: createOptions({
      AdvancedPricingMode: JSON.stringify({
        'mode-only': 'per_request',
      }),
      AdvancedPricingRules: JSON.stringify({
        'rule-only': {
          rule_type: 'media_task',
          task_type: 'image_generation',
          unit_price: 5,
        },
      }),
      ModelPrice: JSON.stringify({
        'price-only': 2,
      }),
      ModelRatio: JSON.stringify({
        'ratio-only': 3,
      }),
    }),
    enabledModelNames: ['enabled-only'],
    launchModelName: 'launch-only',
  });

  assert.deepEqual(
    state.models.map((model) => model.name),
    ['enabled-only', 'launch-only', 'mode-only', 'price-only', 'ratio-only', 'rule-only'],
  );
  assert.equal(state.selectedModelName, 'launch-only');
});

test('advanced pricing helpers normalize text shell drafts into backend rule payloads', async () => {
  const { normalizeAdvancedPricingDraftRule } = await loadHelpers();

  assert.equal(typeof normalizeAdvancedPricingDraftRule, 'function');

  assert.deepEqual(
    normalizeAdvancedPricingDraftRule({
      rule_type: 'text_segment',
      display_name: 'Tiered text',
      segment_basis: 'character',
      billing_unit: '1M chars',
      default_price: '9.9',
      segments_text: '0-100: 1.2\n101-200: 2.4',
      note: 'preserved note',
    }),
    {
      rule_type: 'text_segment',
      display_name: 'Tiered text',
      segment_basis: 'character',
      billing_unit: '1M chars',
      default_price: 9.9,
      note: 'preserved note',
      segments: [
        {
          priority: 10,
          input_min: 0,
          input_max: 100,
          input_price: 1.2,
        },
        {
          priority: 20,
          input_min: 101,
          input_max: 200,
          input_price: 2.4,
        },
      ],
    },
  );
});

test('advanced pricing helpers normalize media shell drafts into backend rule payloads', async () => {
  const { normalizeAdvancedPricingDraftRule } = await loadHelpers();

  assert.equal(typeof normalizeAdvancedPricingDraftRule, 'function');

  assert.deepEqual(
    normalizeAdvancedPricingDraftRule({
      rule_type: 'media_task',
      display_name: 'Tiered media',
      task_type: 'video_generation',
      billing_unit: 'minute',
      unit_price: '8.8',
      note: 'preserved note',
    }),
    {
      rule_type: 'media_task',
      display_name: 'Tiered media',
      task_type: 'video_generation',
      billing_unit: 'minute',
      note: 'preserved note',
      segments: [
        {
          priority: 10,
          unit_price: 8.8,
          remark: 'preserved note',
        },
      ],
    },
  );
});

test('advanced pricing helpers preserve incompatible text canonical fields during metadata-only saves', async () => {
  const { buildAdvancedPricingSavePayload, buildRuleDraft } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingSavePayload, 'function');
  assert.equal(typeof buildRuleDraft, 'function');

  const canonicalRule = {
    rule_type: 'text_segment',
    display_name: 'Tiered text',
    segment_basis: 'token',
    billing_unit: '1M tokens',
    default_price: 6.6,
    note: 'server note',
    segments: [
      {
        priority: 10,
        input_min: 0,
        input_max: 100,
        input_price: 1.2,
        output_min: 0,
        output_max: 100,
        output_price: 2.4,
        cache_read_price: 0.3,
        service_tier: 'premium',
      },
    ],
  };

  const draftRule = {
    ...buildRuleDraft('text_segment', canonicalRule),
    display_name: 'Edited text',
    segment_basis: 'character',
    billing_unit: '1K chars',
    default_price: '9.9',
    note: 'edited note',
  };

  const payload = buildAdvancedPricingSavePayload({
    modelName: 'alpha',
    billingMode: 'advanced',
    draftRule,
  });

  assert.deepEqual(payload.normalizedRule, {
    ...canonicalRule,
    display_name: 'Edited text',
    segment_basis: 'character',
    billing_unit: '1K chars',
    default_price: 9.9,
    note: 'edited note',
  });
});

test('advanced pricing helpers preserve incompatible media canonical fields during metadata-only saves', async () => {
  const { buildAdvancedPricingSavePayload, buildRuleDraft } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingSavePayload, 'function');
  assert.equal(typeof buildRuleDraft, 'function');

  const canonicalRule = {
    rule_type: 'media_task',
    display_name: 'Tiered media',
    task_type: 'video_generation',
    billing_unit: 'minute',
    note: 'server note',
    segments: [
      {
        priority: 10,
        unit_price: 8.8,
        resolution: '1080p',
        aspect_ratio: '16:9',
        min_tokens: 1024,
        draft: false,
        remark: 'legacy remark',
      },
    ],
  };

  const draftRule = {
    ...buildRuleDraft('media_task', canonicalRule),
    display_name: 'Edited media',
    task_type: 'image_generation',
    billing_unit: 'task',
    note: 'edited note',
  };

  const payload = buildAdvancedPricingSavePayload({
    modelName: 'beta',
    billingMode: 'advanced',
    draftRule,
  });

  assert.deepEqual(payload.normalizedRule, {
    ...canonicalRule,
    display_name: 'Edited media',
    task_type: 'image_generation',
    billing_unit: 'task',
    note: 'edited note',
  });
});

test('advanced pricing helpers reject shell edits for incompatible text canonical rules', async () => {
  const { buildAdvancedPricingSavePayload, buildRuleDraft } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingSavePayload, 'function');
  assert.equal(typeof buildRuleDraft, 'function');

  const canonicalRule = {
    rule_type: 'text_segment',
    display_name: 'Tiered text',
    segment_basis: 'token',
    billing_unit: '1M tokens',
    default_price: 6.6,
    segments: [
      {
        priority: 10,
        input_min: 0,
        input_max: 100,
        input_price: 1.2,
        output_price: 2.4,
        service_tier: 'premium',
      },
    ],
  };

  assert.throws(
    () =>
      buildAdvancedPricingSavePayload({
        modelName: 'alpha',
        billingMode: 'advanced',
        draftRule: {
          ...buildRuleDraft('text_segment', canonicalRule),
          segments_text: '0-100: 9.9',
        },
      }),
    /cannot safely round-trip/i,
  );
});

test('advanced pricing helpers reject shell edits for incompatible media canonical rules', async () => {
  const { buildAdvancedPricingSavePayload, buildRuleDraft } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingSavePayload, 'function');
  assert.equal(typeof buildRuleDraft, 'function');

  const canonicalRule = {
    rule_type: 'media_task',
    display_name: 'Tiered media',
    task_type: 'video_generation',
    billing_unit: 'minute',
    segments: [
      {
        priority: 10,
        unit_price: 8.8,
        resolution: '1080p',
        draft: false,
      },
    ],
  };

  assert.throws(
    () =>
      buildAdvancedPricingSavePayload({
        modelName: 'beta',
        billingMode: 'advanced',
        draftRule: {
          ...buildRuleDraft('media_task', canonicalRule),
          unit_price: '12.3',
        },
      }),
    /cannot safely round-trip/i,
  );
});

test('advanced pricing helpers merge normalized rules into a single AdvancedPricingConfig payload', async () => {
  const { buildAdvancedPricingSavePayload } = await loadHelpers();

  assert.equal(typeof buildAdvancedPricingSavePayload, 'function');

  const payload = buildAdvancedPricingSavePayload({
    modelName: 'beta',
    billingMode: 'advanced',
    draftRule: {
      rule_type: 'text_segment',
      display_name: 'Beta tiered text',
      segment_basis: 'character',
      billing_unit: '1M chars',
      default_price: '6.6',
      note: 'preserved text note',
      segments_text: '0-100: 1.2',
    },
    latestModeMap: {
      alpha: 'per_token',
    },
    latestRulesMap: {
      alpha: {
        rule_type: 'media_task',
        segments: [
          {
            priority: 10,
            unit_price: 9.9,
          },
        ],
      },
    },
  });

  assert.deepEqual(payload.previewPayload, {
    AdvancedPricingMode: {
      beta: 'advanced',
    },
      AdvancedPricingRules: {
        beta: {
          rule_type: 'text_segment',
          display_name: 'Beta tiered text',
          segment_basis: 'character',
          billing_unit: '1M chars',
          default_price: 6.6,
          note: 'preserved text note',
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
  });

  assert.deepEqual(JSON.parse(payload.optionValue), {
    billing_mode: {
      alpha: 'per_token',
      beta: 'advanced',
    },
    rules: {
      alpha: {
        rule_type: 'media_task',
        segments: [
          {
            priority: 10,
            unit_price: 9.9,
          },
        ],
      },
      beta: {
        rule_type: 'text_segment',
        display_name: 'Beta tiered text',
        segment_basis: 'character',
        billing_unit: '1M chars',
        default_price: 6.6,
        note: 'preserved text note',
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
  });
});
