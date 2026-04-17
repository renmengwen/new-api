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
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_PER_TOKEN,
  buildAdvancedPricingModePayload,
  canUseAdvancedBilling,
  isBasePricingUnset,
  resolveBillingMode,
} from './modelPricingEditorHelpers.js';

test('resolveBillingMode keeps explicit mode separate from legacy inferred mode', () => {
  assert.deepEqual(
    resolveBillingMode({
      explicitMode: BILLING_MODE_PER_REQUEST,
      fixedPrice: '',
    }),
    {
      billingMode: BILLING_MODE_PER_REQUEST,
      explicitBillingMode: BILLING_MODE_PER_REQUEST,
      hasExplicitBillingMode: true,
    },
  );

  assert.deepEqual(
    resolveBillingMode({
      explicitMode: '',
      fixedPrice: '3',
    }),
    {
      billingMode: BILLING_MODE_PER_REQUEST,
      explicitBillingMode: '',
      hasExplicitBillingMode: false,
    },
  );

  assert.deepEqual(
    resolveBillingMode({
      explicitMode: '',
      fixedPrice: '',
    }),
    {
      billingMode: BILLING_MODE_PER_TOKEN,
      explicitBillingMode: '',
      hasExplicitBillingMode: false,
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
