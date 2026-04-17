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
  new URL('./useAdvancedPricingRulesState.js', import.meta.url),
  'utf8',
);

test('advanced pricing state parses advanced mode and rules and derives searchable models', () => {
  assert.match(
    source,
    /const \[enabledModelNames, setEnabledModelNames\] = useState\(\[\]\);/,
  );
  assert.match(
    source,
    /buildAdvancedPricingModels as buildDerivedAdvancedPricingModels/,
  );
  assert.match(
    source,
    /buildAdvancedPricingDraftRules as mergeAdvancedPricingDraftRules/,
  );
  assert.match(
    source,
    /buildAdvancedPricingDraftBillingModes as mergeAdvancedPricingDraftBillingModes/,
  );
  assert.match(source, /const res = await API\.get\('\/api\/channel\/models_enabled'\);/);
  assert.match(
    source,
    /const nextModels = buildDerivedAdvancedPricingModels\(\{[\s\S]*options,[\s\S]*enabledModelNames,[\s\S]*launchModelName,[\s\S]*\}\);/s,
  );
  assert.match(
    source,
    /setDraftRules\(\(previous\) =>\s*mergeAdvancedPricingDraftRules\(\{[\s\S]*models: nextModels,[\s\S]*previousDraftRules: previous,[\s\S]*\}\),/s,
  );
  assert.match(
    source,
    /setDraftBillingModes\(\(previous\) =>\s*mergeAdvancedPricingDraftBillingModes\(\{[\s\S]*models: nextModels,[\s\S]*previousDraftBillingModes: previous,[\s\S]*\}\),/s,
  );
  assert.match(source, /const \[searchText, setSearchText\] = useState\(''\);/);
  assert.match(source, /model\.name\.toLowerCase\(\)\.includes\(keyword\)/);
  assert.match(source, /const \[selectedModelName, setSelectedModelName\] = useState\(''\);/);
});

test('advanced pricing state treats launch model selection as a one-shot event instead of a sticky override', () => {
  assert.match(source, /initialModelSelectionKey = 0,/);
  assert.match(source, /const \[launchModelName, setLaunchModelName\] = useState\(''\);/);
  assert.match(
    source,
    /useEffect\(\(\) => \{\s*if \(!initialModelSelectionKey\) \{\s*return;\s*\}\s*setLaunchModelName\(initialModelName \|\| ''\);/s,
  );
  assert.match(
    source,
    /resolveAdvancedPricingSelectedModelName as resolveDerivedSelectedModelName/,
  );
  assert.match(
    source,
    /return resolveDerivedSelectedModelName\(\{[\s\S]*models: nextModels,[\s\S]*launchModelName,[\s\S]*previousSelectedModelName: previous,[\s\S]*\}\);/s,
  );
  assert.match(source, /setLaunchModelName\(''\);/);
});

test('advanced pricing state isolates rule-type specific fields when rebuilding drafts', () => {
  assert.match(
    source,
    /const shouldPreserveTypeSpecificFields = normalized\.rule_type === ruleType;/,
  );
  assert.match(
    source,
    /display_name:\s*typeof normalized\.display_name === 'string' \? normalized\.display_name : ''/,
  );
  assert.match(
    source,
    /billing_unit:\s*shouldPreserveTypeSpecificFields[\s\S]*'task'/,
  );
  assert.match(
    source,
    /billing_unit:\s*shouldPreserveTypeSpecificFields[\s\S]*'1K tokens'/,
  );
  assert.doesNotMatch(source, /return \{\s*\.\.\.normalized,\s*task_type:/);
  assert.doesNotMatch(source, /return \{\s*\.\.\.normalized,\s*segment_basis:/);
});

test('advanced pricing state exposes current rule type and a minimal save api for rule mode json', () => {
  assert.match(source, /const currentRuleType = selectedRule\.rule_type \|\| RULE_TYPE_TEXT_SEGMENT;/);
  assert.match(source, /const currentBillingMode = selectedModel\?\.billingMode \|\| BILLING_MODE_PER_TOKEN;/);
  assert.match(source, /const saveSelectedRule = async \(\) => \{/);
  assert.match(source, /const latestOptionsRes = await API\.get\('\/api\/option\/'\);/);
  assert.match(source, /API\.put\('\/api\/option\/', \{\s*key: 'AdvancedPricingMode'/);
  assert.match(source, /API\.put\('\/api\/option\/', \{\s*key: 'AdvancedPricingRules'/);
});
