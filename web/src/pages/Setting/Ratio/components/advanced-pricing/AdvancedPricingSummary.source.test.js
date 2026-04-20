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
  new URL('./AdvancedPricingSummary.jsx', import.meta.url),
  'utf8',
);

test('advanced pricing summary distinguishes persisted effective mode from unsaved local draft mode', () => {
  assert.match(source, /selectedModel\.effectiveMode/);
  assert.match(source, /selectedModel\.selectedMode/);
  assert.match(source, /当前生效模式/);
  assert.match(source, /本地草稿模式/);
  assert.match(source, /切换本地草稿模式/);
  assert.match(source, /selectedModel\.effectiveMode !== selectedModel\.selectedMode/);
  assert.match(source, /本地未保存/);
});

test('advanced pricing summary surfaces capability scaffolding from the selected advanced config', () => {
  assert.match(source, /selectedModel\?\.advancedConfig/);
  assert.match(source, /inputModality/);
  assert.match(source, /outputModality/);
  assert.match(source, /billingUnit/);
  assert.match(source, /imageSizeTier/);
  assert.match(source, /cacheStoragePrice/);
  assert.match(source, /toolUsageType/);
});
