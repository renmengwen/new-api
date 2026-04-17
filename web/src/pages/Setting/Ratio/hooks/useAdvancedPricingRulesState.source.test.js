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
    /buildAdvancedPricingState,/,
  );
  assert.match(
    source,
    /buildRuleDraft,/,
  );
  assert.match(
    source,
    /parseOptionJSON,/,
  );
  assert.match(source, /reduceOptionsByKey,/);
  assert.match(source, /RULE_TYPE_TEXT_SEGMENT,/);
  assert.match(source, /const res = await API\.get\('\/api\/channel\/models_enabled'\);/);
  assert.match(
    source,
    /const nextState = buildAdvancedPricingState\(\{[\s\S]*options,[\s\S]*enabledModelNames,[\s\S]*launchModelName,[\s\S]*previousDraftRules:[\s\S]*previousDraftBillingModes:[\s\S]*previousSelectedModelName:[\s\S]*\}\);/s,
  );
  assert.match(source, /setModels\(nextState\.models\);/);
  assert.match(source, /setDraftRules\(nextState\.draftRules\);/);
  assert.match(source, /setDraftBillingModes\(nextState\.draftBillingModes\);/);
  assert.match(source, /setSelectedModelName\(nextState\.selectedModelName\);/);
  assert.match(source, /const \[searchText, setSearchText\] = useState\(''\);/);
  assert.match(source, /model\.name\.toLowerCase\(\)\.includes\(keyword\)/);
  assert.match(source, /const \[selectedModelName, setSelectedModelName\] = useState\(''\);/);
  assert.doesNotMatch(source, /const parseOptionJSON =/);
  assert.doesNotMatch(source, /const reduceOptionsByKey =/);
  assert.doesNotMatch(source, /const buildRuleDraft =/);
  assert.doesNotMatch(source, /const buildModelState =/);
  assert.doesNotMatch(source, /const formatSegmentLine =/);
  assert.doesNotMatch(source, /const cloneRule =/);
});

test('advanced pricing state treats launch model selection as a one-shot event instead of a sticky override', () => {
  assert.match(source, /initialModelSelectionKey = 0,/);
  assert.match(source, /const \[launchModelName, setLaunchModelName\] = useState\(''\);/);
  assert.match(source, /const draftRulesRef = useRef\(\{\}\);/);
  assert.match(source, /const draftBillingModesRef = useRef\(\{\}\);/);
  assert.match(source, /const dirtyRuleModelNamesRef = useRef\(new Set\(\)\);/);
  assert.match(source, /const dirtyBillingModeModelNamesRef = useRef\(new Set\(\)\);/);
  assert.match(source, /const selectedModelNameRef = useRef\(''\);/);
  assert.match(
    source,
    /useEffect\(\(\) => \{\s*if \(!initialModelSelectionKey\) \{\s*return;\s*\}\s*setLaunchModelName\(initialModelName \|\| ''\);/s,
  );
  assert.match(source, /selectedModelNameRef\.current = selectedModelName;/);
  assert.match(source, /preserveDraftRuleModelNames: dirtyRuleModelNamesRef\.current,/);
  assert.match(
    source,
    /preserveDraftBillingModeModelNames: dirtyBillingModeModelNamesRef\.current,/,
  );
  assert.match(source, /setLaunchModelName\(''\);/);
});

test('advanced pricing state rebuilds editable rules through the shared draft helper', () => {
  assert.match(
    source,
    /const selectedRule = draftRules\[selectedModelName\] \|\| buildRuleDraft\(RULE_TYPE_TEXT_SEGMENT\);/,
  );
  assert.match(
    source,
    /const nextDraftRules = \{\s*\.\.\.draftRulesRef\.current,\s*\[selectedModelName\]: buildRuleDraft\(ruleType, draftRulesRef\.current\[selectedModelName\]\),/s,
  );
  assert.match(source, /dirtyRuleModelNamesRef\.current\.add\(selectedModelName\);/);
  assert.match(
    source,
    /draftRulesRef\.current = nextDraftRules;\s*setDraftRules\(nextDraftRules\);/s,
  );
  assert.match(
    source,
    /const nextDraftRules = \{\s*\.\.\.draftRulesRef\.current,\s*\[selectedModelName\]: \{\s*\.\.\.buildRuleDraft\(currentRuleType, draftRulesRef\.current\[selectedModelName\]\),/s,
  );
  assert.match(source, /dirtyRuleModelNamesRef\.current\.delete\(selectedModel\.name\);/);
  assert.match(
    source,
    /dirtyBillingModeModelNamesRef\.current\.add\(selectedModelName\);/,
  );
  assert.match(
    source,
    /dirtyBillingModeModelNamesRef\.current\.delete\(selectedModel\.name\);/,
  );
});

test('advanced pricing state exposes current rule type and a minimal save api for rule mode json', () => {
  assert.match(source, /const currentRuleType = selectedRule\.rule_type \|\| RULE_TYPE_TEXT_SEGMENT;/);
  assert.match(source, /const currentBillingMode = selectedModel\?\.billingMode \|\| BILLING_MODE_PER_TOKEN;/);
  assert.match(source, /const saveSelectedRule = async \(\) => \{/);
  assert.match(source, /const latestOptionsRes = await API\.get\('\/api\/option\/'\);/);
  assert.match(source, /const latestOptionsByKey = reduceOptionsByKey\(latestOptionsData\);/);
  assert.match(source, /const savePayload = buildAdvancedPricingSavePayload\(\{/);
  assert.match(source, /latestModeMap: parseOptionJSON\(latestOptionsByKey\.AdvancedPricingMode\),/);
  assert.match(source, /latestRulesMap: parseOptionJSON\(latestOptionsByKey\.AdvancedPricingRules\),/);
  assert.match(source, /API\.put\('\/api\/option\/', \{\s*key: 'AdvancedPricingConfig'/);
  assert.doesNotMatch(source, /Promise\.all\(\[/);
  assert.doesNotMatch(source, /key: 'AdvancedPricingMode'/);
  assert.doesNotMatch(source, /key: 'AdvancedPricingRules'/);
});
