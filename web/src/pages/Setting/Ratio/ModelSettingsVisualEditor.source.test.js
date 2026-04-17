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
  new URL('./ModelSettingsVisualEditor.jsx', import.meta.url),
  'utf8',
);

test('price settings wrapper loads channel-enabled models and disables manual model creation', () => {
  assert.match(source, /API\.get\('\/api\/channel\/models_enabled'\)/);
  assert.match(source, /candidateModelNames=\{enabledModels\}/);
  assert.match(source, /filterMode='enabled'/);
  assert.match(source, /allowAddModel=\{false\}/);
  assert.match(source, /setEnabledModels\(\[\]\)/);
});

test('price settings wrapper routes edit advanced rules into the advanced pricing tab flow', () => {
  assert.match(source, /onEditAdvancedRules=\{\(model\) => props\.onOpenAdvancedPricingRules\?\.\(model\)\}/);
  assert.doesNotMatch(source, /onEditAdvancedRules=\{\(model\)\s*=>\s*showError\(/);
});
