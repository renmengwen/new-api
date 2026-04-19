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

const previewSource = fs.readFileSync(
  new URL('./AdvancedPricingPreview.jsx', import.meta.url),
  'utf8',
);
const pageSource = fs.readFileSync(
  new URL('../../AdvancedPricingRulesPage.jsx', import.meta.url),
  'utf8',
);

test('advanced pricing preview uses the text segment preview contract', () => {
  assert.match(
    previewSource,
    /function AdvancedPricingPreview\(\{\s*selectedModel,\s*selectedAdvancedConfig,\s*previewInput,\s*previewResult,\s*savePreview,\s*onPreviewInputChange,\s*\}\)/s,
  );
  assert.doesNotMatch(previewSource, /\bpreviewPayload\b/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('inputTokens', value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('outputTokens', value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('serviceTier', value\)/);
  assert.match(previewSource, /savePreview\?\.effectiveMode/);
  assert.match(previewSource, /getEffectiveModeLabel/);
  assert.match(previewSource, /effectiveModeLabel/);
  assert.match(previewSource, /savePreview\?\.configOptionKey/);
  assert.match(previewSource, /savePreview\?\.configEntry/);
  assert.match(previewSource, /selectedModel\?\.name/);
  assert.match(previewSource, /previewResult\?\.matchedRule/);
  assert.match(previewSource, /previewResult\?\.formulaSummary/);
  assert.match(previewSource, /previewResult\?\.logPreview/);

  assert.match(
    pageSource,
    /<AdvancedPricingPreview[\s\S]*selectedModel={selectedModel}[\s\S]*selectedAdvancedConfig={selectedAdvancedConfig}[\s\S]*previewInput={previewInput}[\s\S]*previewResult={previewResult}[\s\S]*savePreview={savePreview}[\s\S]*onPreviewInputChange={handlePreviewInputChange}[\s\S]*\/>/,
  );
});

test('advanced pricing preview supports media task inputs, summaries, and matched segment json', () => {
  assert.match(previewSource, /MEDIA_TASK_RULE_TYPE/);
  assert.match(previewSource, /serializeMediaTaskRule/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('rawAction', value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('usageTotalTokens', value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('audio', event\.target\.value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('inputVideo', event\.target\.value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('draft', event\.target\.value\)/);
  assert.match(previewSource, /previewResult\?\.conditionSummary/);
  assert.match(previewSource, /previewResult\?\.priceSummary/);
  assert.match(previewSource, /matchedSegmentPreview/);
});

test('advanced pricing preview wires modality, image tier, and tool usage scaffolding inputs', () => {
  assert.match(previewSource, /onPreviewInputChange\?\.\('inputModality', value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('outputModality', value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('imageSizeTier', value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('toolUsageType', value\)/);
  assert.match(previewSource, /onPreviewInputChange\?\.\('toolUsageCount', value\)/);
  assert.match(previewSource, /freeQuota/);
  assert.match(previewSource, /overageThreshold/);
  assert.match(previewSource, /cacheStoragePrice/);
});

test('advanced pricing preview localizes save preview and media task summary copy for the chinese admin ui', () => {
  assert.match(previewSource, /title=\{t\('保存预览'\)\}/);
  assert.match(previewSource, /t\('当前模型'\)/);
  assert.match(previewSource, /t\('保存后生效模式'\)/);
  assert.match(previewSource, /t\('保存配置键'\)/);
  assert.match(previewSource, /t\('配置预览'\)/);
  assert.match(previewSource, /t\('任务类型'\)/);
  assert.match(previewSource, /t\('计费单位'\)/);
  assert.match(previewSource, /t\('规则数'\)/);
  assert.match(previewSource, /t\('本次上报 Token'\)/);
  assert.match(previewSource, /t\('结算 Token'\)/);
  assert.match(previewSource, /t\('最低结算 Token'\)/);
  assert.match(previewSource, /t\('单价'\)/);
  assert.match(previewSource, /t\('草稿系数'\)/);
  assert.match(previewSource, /t\('预估费用'\)/);
  assert.match(previewSource, /t\('命中规则 JSON'\)/);
  assert.doesNotMatch(previewSource, /title='Save Preview'/);
  assert.doesNotMatch(previewSource, /Preview which option entries will be written/);
  assert.doesNotMatch(previewSource, /task_type=/);
  assert.doesNotMatch(previewSource, /billing_unit=/);
  assert.doesNotMatch(previewSource, /segments=/);
  assert.doesNotMatch(
    previewSource,
    /\{ key: 'usageTotalTokens', label: 'usage_total_tokens' \}/,
  );
  assert.doesNotMatch(
    previewSource,
    /\{ key: 'billableTokens', label: 'billable_tokens' \}/,
  );
  assert.doesNotMatch(previewSource, /t\('Draft'\)/);
});

test('advanced pricing preview localizes effective mode labels instead of exposing internal mode keys', () => {
  assert.match(previewSource, /ADVANCED_PRICING_MODE_ADVANCED/);
  assert.match(previewSource, /FIXED_BILLING_MODE_PER_REQUEST/);
  assert.match(previewSource, /FIXED_BILLING_MODE_PER_TOKEN/);
  assert.match(previewSource, /getEffectiveModeLabel/);
  assert.match(previewSource, /effectiveModeLabel/);
  assert.match(previewSource, /t\('高级规则'\)/);
  assert.match(previewSource, /t\('固定按次'\)/);
  assert.match(previewSource, /t\('固定按量'\)/);
});
