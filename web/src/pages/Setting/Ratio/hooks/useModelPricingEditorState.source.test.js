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
import fs from 'node:fs';

const source = fs.readFileSync(
  new URL('./useModelPricingEditorState.js', import.meta.url),
  'utf8',
);
const helperSource = fs.readFileSync(
  new URL('./modelPricingEditorHelpers.js', import.meta.url),
  'utf8',
);

test('pricing state hook resolves visible models separately from full-model save serialization', () => {
  assert.match(source, /resolveInitialVisibleModelNames/);
  assert.match(source, /resolveVisibleModels/);
  assert.match(source, /for \(const model of models\)/);
});

test('pricing state hook reads advanced pricing mode and rules while keeping advanced rules read-only on save', () => {
  assert.match(source, /AdvancedPricingMode:\s*parseOptionJSON\(options\.AdvancedPricingMode\)/);
  assert.match(source, /AdvancedPricingRules:\s*parseOptionJSON\(options\.AdvancedPricingRules\)/);
  assert.match(source, /const advancedRuleType = resolveAdvancedRuleType\(/);
  assert.match(
    source,
    /const billingModeState = resolveBillingMode\(\{[\s\S]*explicitMode: sourceMaps\.AdvancedPricingMode\[name\],[\s\S]*fixedPrice,[\s\S]*advancedRuleType,[\s\S]*\}\)/,
  );
  assert.match(source, /const latestOptionsRes = await API\.get\('\/api\/option\/'\)/);
  assert.match(source, /output\.AdvancedPricingMode = buildAdvancedPricingModePayload\(/);
  assert.doesNotMatch(source, /AdvancedPricingRules:\s*\{\}/);
  assert.doesNotMatch(source, /output\.AdvancedPricingMode\[model\.name\]\s*=\s*model\.billingMode/);
});

test('pricing state hook keeps legacy billing fallback when AdvancedPricingMode is absent', () => {
  assert.match(
    helperSource,
    /export const resolveBillingMode = \(\{[\s\S]*explicitMode,[\s\S]*fixedPrice,[\s\S]*advancedRuleType,[\s\S]*\}\) => \{/,
  );
  assert.match(helperSource, /hasInvalidExplicitAdvancedMode =[\s\S]*explicitMode === BILLING_MODE_ADVANCED/);
  assert.match(helperSource, /hasValue\(fixedPrice\)/);
  assert.match(helperSource, /explicitBillingMode: hasExplicitBillingMode \? explicitMode : ''/);
});

test('pricing state hook only previews AdvancedPricingMode when the model is explicit or user-dirty', () => {
  assert.match(source, /shouldPersistAdvancedPricingMode\(/);
  assert.match(source, /dirtyModeNames: model\.billingModeDirty \? \[model\.name\] : \[\]/);
});
