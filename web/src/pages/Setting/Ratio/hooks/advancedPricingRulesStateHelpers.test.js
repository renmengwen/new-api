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
