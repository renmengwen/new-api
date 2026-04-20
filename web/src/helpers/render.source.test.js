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

const renderSource = fs.readFileSync(new URL('./render.jsx', import.meta.url), 'utf8');

test('usage log render helpers treat zero model_price with positive model_ratio as token billing', () => {
  assert.match(
    renderSource,
    /function normalizeModelPriceForLog\(modelPrice = -1, modelRatio = 0\) \{/,
  );
  assert.match(
    renderSource,
    /if \(price === 0 && Number\.isFinite\(ratio\) && ratio > 0\) \{\s*return -1;\s*\}/,
  );
  assert.match(
    renderSource,
    /export function renderModelPrice\([\s\S]*?modelPrice = normalizeModelPriceForLog\(modelPrice, modelRatio\);/,
  );
  assert.match(
    renderSource,
    /export function renderClaudeModelPrice\([\s\S]*?modelPrice = normalizeModelPriceForLog\(modelPrice, modelRatio\);/,
  );
  assert.match(
    renderSource,
    /export function renderAudioModelPrice\([\s\S]*?modelPrice = normalizeModelPriceForLog\(modelPrice, modelRatio\);/,
  );
});
