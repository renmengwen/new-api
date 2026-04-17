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
  assert.match(source, /AdvancedPricingMode:\s*parseOptionJSON\(options\.AdvancedPricingMode\)/);
  assert.match(source, /AdvancedPricingRules:\s*parseOptionJSON\(options\.AdvancedPricingRules\)/);
  assert.match(source, /ModelPrice:\s*parseOptionJSON\(options\.ModelPrice\)/);
  assert.match(source, /ModelRatio:\s*parseOptionJSON\(options\.ModelRatio\)/);
  assert.match(source, /const \[searchText, setSearchText\] = useState\(''\);/);
  assert.match(source, /model\.name\.toLowerCase\(\)\.includes\(keyword\)/);
  assert.match(source, /const \[selectedModelName, setSelectedModelName\] = useState\(''\);/);
});

test('advanced pricing state exposes current rule type and a minimal save api for rule mode json', () => {
  assert.match(source, /const currentRuleType = selectedRule\.rule_type \|\| RULE_TYPE_TEXT_SEGMENT;/);
  assert.match(source, /const currentBillingMode = selectedModel\?\.billingMode \|\| BILLING_MODE_PER_TOKEN;/);
  assert.match(source, /const saveSelectedRule = async \(\) => \{/);
  assert.match(source, /const latestOptionsRes = await API\.get\('\/api\/option\/'\);/);
  assert.match(source, /API\.put\('\/api\/option\/', \{\s*key: 'AdvancedPricingMode'/);
  assert.match(source, /API\.put\('\/api\/option\/', \{\s*key: 'AdvancedPricingRules'/);
});
