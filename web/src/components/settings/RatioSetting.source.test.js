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
  new URL('./RatioSetting.jsx', import.meta.url),
  'utf8',
);

test('ratio setting adds an advanced pricing rules tab with stable item key', () => {
  assert.match(source, /import AdvancedPricingRulesPage from ['"]..\/..\/pages\/Setting\/Ratio\/AdvancedPricingRulesPage['"]/);
  assert.match(source, /tab=\{t\('高级定价规则'\)\}\s+itemKey='advanced_pricing'/);
  assert.match(source, /<AdvancedPricingRulesPage[\s\S]*options=\{inputs\}[\s\S]*refresh=\{onRefresh\}/);
});

test('ratio setting keeps tab state so price settings can open the advanced pricing page directly', () => {
  assert.match(source, /const \[activeTab, setActiveTab\] = useState\('visual'\);/);
  assert.match(source, /const \[pendingAdvancedPricingModelName, setPendingAdvancedPricingModelName\] = useState\(''\);/);
  assert.match(
    source,
    /const \[advancedPricingInitialModelSelectionKey, setAdvancedPricingInitialModelSelectionKey\] = useState\(0\);/,
  );
  assert.match(source, /<Tabs type='card' activeKey=\{activeTab\} onChange=\{handleTabChange\}>/);
  assert.match(source, /onOpenAdvancedPricingRules=\{handleOpenAdvancedPricingRules\}/);
  assert.match(
    source,
    /setAdvancedPricingInitialModelSelectionKey\(\(previous\) => previous \+ 1\);/,
  );
  assert.match(source, /initialModelName=\{pendingAdvancedPricingModelName\}/);
  assert.match(
    source,
    /initialModelSelectionKey=\{advancedPricingInitialModelSelectionKey\}/,
  );
});
