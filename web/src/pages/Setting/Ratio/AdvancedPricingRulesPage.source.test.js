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
  new URL('./AdvancedPricingRulesPage.jsx', import.meta.url),
  'utf8',
);

test('advanced pricing page wires model list, summary, editor, preview, and save flow', () => {
  assert.match(source, /useAdvancedPricingRulesState/);
  assert.match(source, /<AdvancedPricingModelList/);
  assert.match(source, /<AdvancedPricingSummary/);
  assert.match(source, /<TextSegmentRuleEditor/);
  assert.match(source, /<AdvancedPricingPreview/);
  assert.match(source, /ruleType === 'text-segment'/);
  assert.match(source, /API\.get\('\/api\/channel\/models_enabled'\)/);
});
