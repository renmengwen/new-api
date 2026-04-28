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
  assert.match(source, /resolveModelPricingBridgeSelection/);
  assert.match(source, /resolveModelPricingSelectedModelName/);
  assert.match(source, /for \(const model of models\)/);
  assert.match(source, /AdvancedPricingMode/);
  assert.doesNotMatch(source, /ModelBillingMode/);
});

test('pricing state hook reads advanced pricing mode and can persist copied advanced rules on save', () => {
  assert.match(source, /options\.AdvancedPricingConfig/);
  assert.match(source, /canonicalAdvancedPricingConfig/);
  assert.match(
    source,
    /AdvancedPricingMode:\s*Object\.keys\(canonicalAdvancedPricingConfig\.billing_mode\)\.length > 0\s*\?\s*canonicalAdvancedPricingConfig\.billing_mode\s*:\s*parseOptionJSON\(options\.AdvancedPricingMode\)/,
  );
  assert.match(
    source,
    /AdvancedPricingRules:\s*Object\.keys\(canonicalAdvancedPricingConfig\.rules\)\.length > 0\s*\?\s*canonicalAdvancedPricingConfig\.rules\s*:\s*parseOptionJSON\(options\.AdvancedPricingRules\)/,
  );
  assert.match(source, /const advancedRuleType = resolveAdvancedRuleType\(/);
  assert.match(
    source,
    /const billingModeState = resolveBillingMode\(\{[\s\S]*explicitMode: sourceMaps\.AdvancedPricingMode\[name\],[\s\S]*fixedPrice,[\s\S]*advancedRuleType,[\s\S]*\}\)/,
  );
  assert.match(source, /const latestOptionsRes = await API\.get\('\/api\/option\/'\)/);
  assert.match(
    source,
    /const latestCanonicalAdvancedPricingConfig\s*=\s*parseAdvancedPricingConfigOption\(/,
  );
  assert.match(
    source,
    /const \[advancedPricingRules,\s*setAdvancedPricingRules\] = useState\(\{\}\);/,
  );
  assert.match(
    source,
    /const \[copiedAdvancedRuleNames,\s*setCopiedAdvancedRuleNames\] = useState\(/,
  );
  assert.match(source, /copyAdvancedPricingRulesForModels\(\{/);
  assert.match(source, /setAdvancedPricingRules\(advancedPricingCopyState\.rulesMap\)/);
  assert.match(
    source,
    /output\.AdvancedPricingConfig =\s*buildAdvancedPricingConfigPayloadForPricingEditor\(\{/,
  );
  assert.match(
    source,
    /if \(copiedAdvancedRuleNames\.size > 0\) \{[\s\S]*output\.AdvancedPricingConfig =\s*buildAdvancedPricingConfigPayloadForPricingEditor\(\{/,
  );
  assert.match(
    source,
    /\} else \{[\s\S]*output\.AdvancedPricingMode = buildAdvancedPricingModePayload\(\{/,
  );
  assert.match(
    source,
    /latestModeMap:\s*latestAdvancedPricingModeMap/,
  );
  assert.match(
    source,
    /latestRulesMap:\s*latestAdvancedPricingRulesMap/,
  );
  assert.match(
    source,
    /draftRulesMap:\s*advancedPricingRules,/,
  );
  assert.match(
    source,
    /copiedRuleNames:\s*copiedAdvancedRuleNames,/,
  );
  assert.match(
    source,
    /models,\s*dirtyModeNames: billingModeDirtyNames,/,
  );
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

test('pricing state hook localizes advanced rule type labels in summary text', () => {
  assert.match(helperSource, /export const getAdvancedRuleTypeText = \(advancedRuleType, t\) => \{/);
  assert.match(helperSource, /advancedRuleType === 'media_task'/);
  assert.match(helperSource, /advancedRuleType === 'text_segment'/);
  assert.match(source, /getAdvancedRuleTypeText/);
  assert.match(
    source,
    /`\$\{t\('高级规则'\)\} · \$\{getAdvancedRuleTypeText\(model\.advancedRuleType, t\)\}`/,
  );
});

test('pricing state hook only previews AdvancedPricingMode when the model is explicit or user-dirty', () => {
  assert.match(source, /shouldPersistAdvancedPricingMode\(/);
  assert.match(source, /dirtyModeNames: model\.billingModeDirty \? \[model\.name\] : \[\]/);
});

test('pricing state hook applies initial selected model bridge once per version and jumps to the target page', () => {
  assert.match(source, /initialSelectedModelName = ''/);
  assert.match(source, /initialSelectionVersion = 0/);
  assert.match(source, /const lastAppliedInitialSelectionVersionRef = useRef\(null\);/);
  assert.match(source, /const pendingSelectionPageRef = useRef\(null\);/);
  assert.match(source, /resolveModelPricingBridgeSelection/);
  assert.match(
    source,
    /const \{\s*nextSelectedModelName,\s*nextAppliedInitialSelectionVersion,\s*shouldSyncSelection,\s*\} = resolveModelPricingSelectedModelName\(/,
  );
  assert.match(
    source,
    /const bridgeSelectionState = resolveModelPricingBridgeSelection\(\{[\s\S]*shouldSyncSelection,[\s\S]*selectedModelName: nextSelectedModelName,[\s\S]*pageSize: PAGE_SIZE,[\s\S]*searchText,[\s\S]*conflictOnly,[\s\S]*\}\);/,
  );
  assert.match(
    source,
    /pendingSelectionPageRef\.current = bridgeSelectionState\.pendingSelectionPage;/,
  );
  assert.match(
    source,
    /if \(bridgeSelectionState\.shouldResetSearchText\) \{\s*setSearchText\(''\);\s*\}/s,
  );
  assert.match(
    source,
    /if \(bridgeSelectionState\.shouldResetConflictOnly\) \{\s*setConflictOnly\(false\);\s*\}/s,
  );
  assert.match(
    source,
    /setCurrentPage\(bridgeSelectionState\.nextCurrentPage\);/,
  );
  assert.doesNotMatch(source, /const nextSelectionPage = resolveModelPricingSelectionPage\(/);
});

test('pricing state hook keeps advanced pricing literals on the existing contract names', () => {
  assert.match(helperSource, /BILLING_MODE_PER_TOKEN = 'per_token'/);
  assert.match(helperSource, /BILLING_MODE_PER_REQUEST = 'per_request'/);
  assert.match(helperSource, /BILLING_MODE_ADVANCED = 'advanced'/);
  assert.doesNotMatch(source, /ModelBillingMode/);
  assert.doesNotMatch(source, /pricing_mode/);
  assert.doesNotMatch(source, /text-segment/);
  assert.doesNotMatch(source, /media-task/);
});

test('pricing state hook uses readable Chinese save failure copy when refreshing latest options before submit', () => {
  assert.match(
    source,
    /throw new Error\(latestOptionsMessage \|\| t\('保存失败，请重试'\)\)/,
  );
});
