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

test('advanced pricing page uses the A layout and wires all shell components', () => {
  assert.match(source, /import AdvancedPricingModelList from '.\/components\/advanced-pricing\/AdvancedPricingModelList'/);
  assert.match(source, /import AdvancedPricingSummary from '.\/components\/advanced-pricing\/AdvancedPricingSummary'/);
  assert.match(source, /import AdvancedPricingPreview from '.\/components\/advanced-pricing\/AdvancedPricingPreview'/);
  assert.match(source, /import TextSegmentRuleEditor from '.\/components\/advanced-pricing\/TextSegmentRuleEditor'/);
  assert.match(source, /import MediaTaskRuleEditor from '.\/components\/advanced-pricing\/MediaTaskRuleEditor'/);
  assert.match(source, /gridTemplateColumns:\s*isMobile\s*\?\s*'minmax\(0,\s*1fr\)'\s*:\s*'minmax\(280px,\s*320px\)\s+minmax\(0,\s*1fr\)'/);
  assert.match(source, /<AdvancedPricingModelList[\s\S]*<AdvancedPricingSummary[\s\S]*<AdvancedPricingPreview/);
});

test('advanced pricing page supports both text segment and media task rule shells', () => {
  assert.match(source, /const RULE_TYPE_TEXT_SEGMENT = 'text_segment';/);
  assert.match(source, /const RULE_TYPE_MEDIA_TASK = 'media_task';/);
  assert.match(source, /currentRuleType === RULE_TYPE_MEDIA_TASK \?/);
  assert.match(source, /<MediaTaskRuleEditor/);
  assert.match(source, /<TextSegmentRuleEditor/);
});
