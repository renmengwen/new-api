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

test('pricing editor accepts initial model selection bridge props and forwards them to the state hook', () => {
  assert.match(source, /initialSelectedModelName = ''/);
  assert.match(source, /initialSelectionVersion = 0/);
  assert.match(
    source,
    /useModelPricingEditorState\(\{[\s\S]*candidateModelNames,[\s\S]*filterMode,[\s\S]*initialSelectedModelName,[\s\S]*initialSelectionVersion,[\s\S]*\}\)/,
  );
});

test('pricing editor keeps billing mode literals aligned with the persisted contract', () => {
  assert.match(source, /<Radio value='per_token'>/);
  assert.match(source, /<Radio value='per_request'>/);
  assert.match(source, /<Radio value='advanced'/);
  assert.doesNotMatch(source, /ModelBillingMode/);
  assert.doesNotMatch(source, /pricing_mode/);
  assert.doesNotMatch(source, /per-token/);
  assert.doesNotMatch(source, /per-request/);
});

test('pricing editor exposes advanced rule status summary and edit entry point in the detail panel', () => {
  assert.match(source, /const advancedBillingAvailable = canUseAdvancedBilling\(selectedModel\);/);
  assert.match(source, /<Radio value='advanced' disabled=\{!advancedBillingAvailable\}>/);
  assert.match(source, /const hasAdvancedRulesConfigured = Boolean\(selectedModel\?\.advancedRuleType\);/);
  assert.match(
    source,
    /const hasReservedFixedPricing = hasEditableFixedPricingConfig\(selectedModel\);/,
  );
  assert.doesNotMatch(source, /const hasReservedFixedPricing = Boolean\([\s\S]*rawRatios/);
  assert.match(source, /t\('高级规则状态'\)/);
  assert.match(source, /t\('当前生效模式'\)/);
  assert.match(source, /t\('高级规则配置'\)/);
  assert.match(source, /t\('当前规则类型'\)/);
  assert.match(source, /t\('固定价格配置'\)/);
  assert.match(source, /hasAdvancedRulesConfigured \? t\('已配置'\) : t\('未配置'\)/);
  assert.match(source, /hasReservedFixedPricing \? t\('已保留'\) : t\('未配置'\)/);
  assert.match(source, /t\('固定价格配置保留但不生效。'\)/);
  assert.match(
    source,
    /selectedModel\.billingMode !== 'advanced'[\s\S]*hasAdvancedRulesConfigured/,
  );
  assert.match(source, /t\('已配置高级规则，但当前未生效，可切换。'\)/);
  assert.match(source, /t\('编辑高级规则'\)/);
  assert.doesNotMatch(source, /高级规则内容将在后续任务中接入，此处仅展示当前状态。/);
  assert.match(
    source,
    /t\(\s*'当前模型未配置高级规则，需先配置高级规则后才能切换为高级规则计费模式。',?\s*\)/,
  );
});

test('pricing editor keeps preserved fixed pricing editors visible in advanced mode', () => {
  assert.doesNotMatch(
    source,
    /selectedModel\.billingMode === 'advanced' \? null : selectedModel\.billingMode === 'per_request' \?/,
  );
  assert.match(
    source,
    /selectedModel\.billingMode === 'per_request' \|\|[\s\S]*selectedModel\.billingMode === 'advanced'/,
  );
  assert.match(
    source,
    /selectedModel\.billingMode === 'per_token' \|\|[\s\S]*selectedModel\.billingMode === 'advanced'/,
  );
  assert.match(source, /t\('按次固定价格'\)/);
  assert.match(
    source,
    /t\('按次固定价格配置已保留，可继续维护，当前高级规则生效。'\)/,
  );
  assert.match(
    source,
    /t\('以下按量固定价格配置已保留，可继续维护，当前高级规则生效。'\)/,
  );
});

test('pricing editor confirms billing mode changes in the price settings page before mutating state', () => {
  assert.match(source, /const handleBillingModeSelect = \(nextBillingMode\) => \{/);
  assert.match(source, /const handleBatchApplyConfirm = \(\) => \{/);
  assert.match(source, /const batchBillingModeConfirmation = resolveBatchBillingModeConfirmation\(/);
  assert.match(source, /Modal\.confirm\(\{/);
  assert.match(source, /title:\s*t\(BILLING_MODE_CHANGE_CONFIRM_TITLE\)/);
  assert.match(
    source,
    /content:\s*t\(BILLING_MODE_CHANGE_CONFIRM_CONTENT\)/,
  );
  assert.match(source, /onOk:\s*\(\)\s*=>\s*handleBillingModeChange\(nextBillingMode\)/);
  assert.match(
    source,
    /if \(batchBillingModeConfirmation\.requiresConfirmation\) \{[\s\S]*Modal\.confirm\(\{/,
  );
  assert.match(
    source,
    /title:\s*t\(batchBillingModeConfirmation\.title\)[\s\S]*content:\s*t\(batchBillingModeConfirmation\.content\)/,
  );
  assert.match(source, /onOk=\{handleBatchApplyConfirm\}/);
  assert.match(
    source,
    /onChange=\{\(event\) => handleBillingModeSelect\(event\.target\.value\)\}/,
  );
  assert.doesNotMatch(
    source,
    /onChange=\{\(event\) => handleBillingModeChange\(event\.target\.value\)\}/,
  );
  assert.doesNotMatch(
    source,
    /onOk=\{\(\) => \{\s*if \(applySelectedModelPricing\(\)\) \{\s*setBatchVisible\(false\);\s*\}\s*\}\}/,
  );
});
