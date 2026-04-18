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

test('advanced pricing state hook uses current option keys and literals', () => {
  assert.match(source, /latestModeMap: parseOptionJSON\(latestOptionsByKey\.AdvancedPricingMode\),/);
  assert.match(source, /latestRulesMap: parseOptionJSON\(latestOptionsByKey\.AdvancedPricingRules\),/);
  assert.match(source, /API\.put\('\/api\/option\/', \{\s*key: 'AdvancedPricingConfig'/);
  assert.doesNotMatch(source, /ModelBillingMode/);
  assert.doesNotMatch(source, /pricing_mode/);
  assert.doesNotMatch(source, /text-segment/);
  assert.doesNotMatch(source, /media-task/);
});
