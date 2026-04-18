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

const hookSource = fs.readFileSync(
  new URL('./useUsageLogsData.jsx', import.meta.url),
  'utf8',
);

test('useUsageLogsData renders advanced billing details with a dedicated branch', () => {
  assert.match(hookSource, /const renderAdvancedBillingDetails = \(t, other\) => \{/);
  assert.match(hookSource, /const renderAdvancedBillingProcess = \(t, log, other\) => \{/);
  assert.match(
    hookSource,
    /other\?\.billing_mode === 'advanced'\s*\?\s*renderAdvancedBillingDetails\(t,\s*other\)\s*:/,
  );
  assert.match(
    hookSource,
    /if \(other\?\.billing_mode === 'advanced'\) \{\s*content = renderAdvancedBillingProcess\(t,\s*logs\[i\],\s*other\);/,
  );
});

test('advanced billing branch prioritizes structured rule fields instead of legacy fixed-price helpers', () => {
  assert.match(hookSource, /advanced_rule_type/);
  assert.match(hookSource, /advanced_rule/);
  assert.match(hookSource, /match_summary/);
  assert.match(hookSource, /condition_tags/);
  assert.match(hookSource, /price_snapshot/);
  assert.match(hookSource, /threshold_snapshot/);
  assert.match(hookSource, /model_price/);
  assert.match(hookSource, /规则类型/);
  assert.match(hookSource, /命中条件/);
  assert.match(hookSource, /单价摘要/);
  assert.match(hookSource, /实际计费依据/);
  const detailsBlock = hookSource.match(
    /const renderAdvancedBillingDetails = \(t, other\) => \{[\s\S]*?\n};/,
  )?.[0];
  const processBlock = hookSource.match(
    /const renderAdvancedBillingProcess = \(t, log, other\) => \{[\s\S]*?\n};/,
  )?.[0];
  assert.ok(detailsBlock, 'advanced billing details helper should exist');
  assert.ok(processBlock, 'advanced billing process helper should exist');
  assert.doesNotMatch(detailsBlock, /render(LogContent|ModelPrice)\(/);
  assert.doesNotMatch(processBlock, /render(LogContent|ModelPrice)\(/);
});
