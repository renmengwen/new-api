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
  new URL('./ModelPricingEditor.jsx', import.meta.url),
  'utf8',
);

test('pricing editor list exposes billing mode and advanced rule type columns', () => {
  assert.match(source, /title:\s*t\('计费模式'\)/);
  assert.match(source, /title:\s*t\('规则类型'\)/);
  assert.match(source, /record\.billingMode === 'advanced'/);
  assert.match(source, /t\('高级规则'\)/);
  assert.match(source, /record\.advancedRuleType[\s\S]*\?[\s\S]*record\.advancedRuleType[\s\S]*:[\s\S]*'—'/);
});

test('pricing editor advanced mode hides fixed forms and shows advanced rule summary entry point', () => {
  assert.match(source, /<Radio value='advanced'>\{t\('高级规则'\)\}<\/Radio>/);
  assert.match(source, /selectedModel\.billingMode === 'advanced' \?/);
  assert.match(source, /t\('固定价格配置保留但不生效。'\)/);
  assert.match(source, /t\('编辑高级规则'\)/);
});
