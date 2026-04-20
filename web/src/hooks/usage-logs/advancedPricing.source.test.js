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
  assert.match(
    hookSource,
    /const renderAdvancedBillingDetails = \(t, other\) => \{/,
  );
  assert.match(
    hookSource,
    /const renderAdvancedBillingProcess = \(t, log, other\) => \{/,
  );
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

test('advanced billing process includes final formulas for text segments and media tasks', () => {
  assert.match(
    hookSource,
    /const buildAdvancedTextSegmentFormula = \(t, log, other, snapshot\) => \{/,
  );
  assert.match(
    hookSource,
    /const buildAdvancedMediaTaskFormula = \(t, log, other, snapshot\) => \{/,
  );
  assert.match(hookSource, /最终计费公式/);
  assert.match(hookSource, /实际计费用量/);
  assert.match(hookSource, /生效计费用量/);
  assert.match(hookSource, /effectiveTokens/);
  assert.match(hookSource, /ruleType === 'media_task'/);
  assert.match(
    hookSource,
    /\?\s*buildAdvancedMediaTaskFormula\(t,\s*log,\s*other,\s*snapshot\)/,
  );
  assert.match(
    hookSource,
    /:\s*buildAdvancedTextSegmentFormula\(t,\s*log,\s*other,\s*snapshot\)/,
  );
  assert.match(hookSource, /groupRatio/);
  assert.match(hookSource, /priceSnapshot\.input_price/);
  assert.match(hookSource, /priceSnapshot\.output_price/);
  assert.match(hookSource, /thresholdSnapshot\.min_tokens/);
  assert.match(hookSource, /convertUSDToCurrency/);
  assert.doesNotMatch(
    hookSource,
    /const buildAdvancedMediaTaskFormula[\s\S]*renderQuota\(unitPrice\)/,
  );
  assert.doesNotMatch(
    hookSource,
    /const buildAdvancedTextSegmentFormula[\s\S]*renderQuota\(inputPrice\)/,
  );
  assert.doesNotMatch(
    hookSource,
    /const buildAdvancedTextSegmentFormula[\s\S]*renderQuota\(outputPrice\)/,
  );
});

test('advanced billing helpers keep coexistence extras instead of swallowing legacy surcharge details', () => {
  assert.match(
    hookSource,
    /const buildAdvancedExtraChargeLines = \(t, log, other, snapshot\) => \{/,
  );
  assert.match(hookSource, /web_search_price/);
  assert.match(hookSource, /file_search_price/);
  assert.match(hookSource, /image_generation_call_price/);
  assert.match(hookSource, /cache_creation_tokens/);
  assert.match(hookSource, /audio_output/);
  assert.match(hookSource, /audio_completion_ratio/);
  assert.match(
    hookSource,
    /const derivedAudioOutputPrice =[\s\S]*audioCompletionRatio[\s\S]*\?/,
  );
  assert.match(
    hookSource,
    /pushTokenItem\(t\('音频输出'\), other\?\.audio_output, derivedAudioOutputPrice\);/,
  );
  assert.match(
    hookSource,
    /buildAdvancedExtraChargeLines\(t, log, other, snapshot\)/,
  );
  assert.match(
    hookSource,
    /buildAdvancedExtraChargeLines\(t, null, other, snapshot\)/,
  );
  assert.match(
    hookSource,
    /const buildAdvancedTextSegmentFormula = \(t, log, other, snapshot\) => \{[\s\S]*buildAdvancedExtraChargeItems\(t, other, snapshot\)/,
  );
  assert.match(
    hookSource,
    /const buildAdvancedMediaTaskFormula = \(t, log, other, snapshot\) => \{[\s\S]*getAdvancedActualUsageTokens\(log, other\)/,
  );
});

test('advanced billing helpers read structured advanced pricing context and switch formulas by billing unit', () => {
  assert.match(
    hookSource,
    /const getAdvancedPricingContext = \(other\) => \{/,
  );
  assert.match(hookSource, /advanced_pricing_context/);
  assert.match(hookSource, /live_duration_secs/);
  assert.match(hookSource, /image_count/);
  assert.match(hookSource, /tool_usage_type/);
  assert.match(hookSource, /tool_usage_count/);
  assert.match(hookSource, /per_second/);
  assert.match(hookSource, /per_image/);
  assert.match(hookSource, /per_1000_calls/);
  assert.match(hookSource, /switch \(billingUnit\)/);
  assert.match(hookSource, /free_quota/);
});

test('advanced non-token billing strings stay readable in UTF-8 source', () => {
  assert.doesNotMatch(
    hookSource,
    /鍗曚环鎽樿|瀹為檯璁|鏈€缁堣|鐢熸晥璁|鍒嗙粍鍊嶇巼/,
  );
  assert.match(hookSource, /单价摘要/);
  assert.match(hookSource, /实际计费用量/);
  assert.match(hookSource, /生效计费用量/);
  assert.match(hookSource, /最终计费公式/);
  assert.match(hookSource, /分组倍率/);
});

test('advanced non-token unit price helper matches settlement inputs and keeps draft coefficient in formula', () => {
  assert.match(
    hookSource,
    /const resolveAdvancedNonTokenUnitPrice = \(snapshot, other\) => \{/,
  );
  assert.match(hookSource, /priceSnapshot\.cache_storage_price/);
  assert.match(
    hookSource,
    /buildAdvancedPriceSummary[\s\S]*resolveAdvancedNonTokenUnitPrice\(snapshot,\s*other\)/,
  );
  assert.match(
    hookSource,
    /const buildAdvancedTextSegmentFormula = \(t, log, other, snapshot\) => \{[\s\S]*resolveAdvancedNonTokenUnitPrice\(snapshot,\s*other\)/,
  );
  assert.match(
    hookSource,
    /const nonTokenFormula = buildAdvancedNonTokenFormula\(\s*[\s\S]*?nonTokenUnitPrice,\s*[\s\S]*?groupRatio,\s*[\s\S]*?multiplier,\s*[\s\S]*?\);/,
  );
});
