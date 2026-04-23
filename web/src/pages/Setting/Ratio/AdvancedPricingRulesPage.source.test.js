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
const mediaTaskRuleEditorSource = fs.readFileSync(
  new URL('./components/advanced-pricing/MediaTaskRuleEditor.jsx', import.meta.url),
  'utf8',
);
const textSegmentRuleEditorSource = fs.readFileSync(
  new URL('./components/advanced-pricing/TextSegmentRuleEditor.jsx', import.meta.url),
  'utf8',
);

test('advanced pricing page wires model list, summary, editor, preview, and save flow', () => {
  assert.match(source, /useAdvancedPricingRulesState/);
  assert.match(source, /<AdvancedPricingModelList/);
  assert.match(source, /<AdvancedPricingSummary/);
  assert.match(source, /<TextSegmentRuleEditor/);
  assert.match(source, /<MediaTaskRuleEditor/);
  assert.match(source, /selectedAdvancedConfig\.ruleType === MEDIA_TASK_RULE_TYPE[\s\S]*<MediaTaskRuleEditor[\s\S]*<AdvancedPricingPreview/s);
  assert.match(source, /ruleType === TEXT_SEGMENT_RULE_TYPE/);
  assert.match(source, /ruleType === MEDIA_TASK_RULE_TYPE/);
  assert.match(source, /API\.get\('\/api\/channel\/models_enabled'\)/);
  assert.match(source, /import \{ buildFallbackEnabledModelNames \} from '\.\/enabledModelCandidates';/);
  assert.match(
    source,
    /const fallbackEnabledModels = buildFallbackEnabledModelNames\(\{\s*options: props\.options,\s*initialModelName: props\.initialModelName,\s*\}\);/s,
  );
  assert.match(source, /const resolvedEnabledModels = shouldUseFallbackEnabledModels/);
  assert.match(source, /candidateModelNames:\s*resolvedEnabledModels,/);
  assert.match(source, /setEnabledModels\(fallbackEnabledModels\);\s*showError\(message\);/s);
  assert.match(source, /setEnabledModels\(fallbackEnabledModels\);\s*console\.error\(/s);
  assert.doesNotMatch(
    source,
    /setEnabledModels\(\[\]\);\s*showError\(message\);/s,
  );
  assert.doesNotMatch(
    source,
    /setEnabledModels\(\[\]\);\s*showError\(t\('获取启用模型失败'\)\);/s,
  );
});

test('advanced pricing page keeps controlled selection and legacy initial selection on separate props', () => {
  assert.match(
    source,
    /selectedModelName:\s*props\.selectedModelName,/,
  );
  assert.match(
    source,
    /onSelectedModelChange:\s*props\.onSelectedModelChange,/,
  );
  assert.match(
    source,
    /initialSelectedModelName:\s*props\.initialModelName,/,
  );
  assert.match(
    source,
    /initialSelectionVersion:\s*props\.initialModelSelectionKey,/,
  );
  assert.doesNotMatch(
    source,
    /selectedModelName:\s*props\.selectedModelName \?\? legacySelectedModelName,/,
  );
  assert.doesNotMatch(
    source,
    /onSelectedModelChange:\s*props\.onSelectedModelChange \?\? setLegacySelectedModelName,/,
  );
});

test('media task rule editor constrains duration fields to integer input only', () => {
  assert.match(
    mediaTaskRuleEditorSource,
    /field: 'outputDurationMin',[\s\S]*?regex: INTEGER_INPUT_REGEX,[\s\S]*?\},/,
  );
  assert.match(
    mediaTaskRuleEditorSource,
    /field: 'outputDurationMax',[\s\S]*?regex: INTEGER_INPUT_REGEX,[\s\S]*?\},/,
  );
  assert.match(
    mediaTaskRuleEditorSource,
    /field: 'inputVideoDurationMin',[\s\S]*?regex: INTEGER_INPUT_REGEX,[\s\S]*?\},/,
  );
  assert.match(
    mediaTaskRuleEditorSource,
    /field: 'inputVideoDurationMax',[\s\S]*?regex: INTEGER_INPUT_REGEX,[\s\S]*?\},/,
  );
});

test('media task rule editor exposes rawAction input and keeps the side sheet billing preview wired to media segments', () => {
  assert.match(
    mediaTaskRuleEditorSource,
    /field: 'rawAction',[\s\S]*?label: t\('任务动作'\),[\s\S]*?placeholder: t\('如 generate \/ firstTailGenerate'\),/,
  );
  assert.match(mediaTaskRuleEditorSource, /serializeMediaTaskRule\(draftRule\)/);
  assert.match(
    mediaTaskRuleEditorSource,
    /visible={sideSheetVisible}/,
  );
  assert.match(mediaTaskRuleEditorSource, /sheetPreviewResult\?\.matchedRule/);
  assert.match(
    mediaTaskRuleEditorSource,
    /serializeMediaTaskRule\(previewResult\.matchedRule\)/,
  );
  assert.doesNotMatch(mediaTaskRuleEditorSource, /modalVisible/);
});

test('text segment rule editor is wired through the modern config contract only', () => {
  assert.match(textSegmentRuleEditorSource, /function TextSegmentRulesEditor/);
  assert.match(textSegmentRuleEditorSource, /SideSheet/);
  assert.match(textSegmentRuleEditorSource, /getTextSegmentRuleEditorMeta/);
  assert.match(textSegmentRuleEditorSource, /validationErrors/);
  assert.match(textSegmentRuleEditorSource, /onChange/);
  assert.match(textSegmentRuleEditorSource, /onConfigChange/);
  assert.doesNotMatch(textSegmentRuleEditorSource, /LegacyTextSegmentRuleEditor/);
  assert.doesNotMatch(textSegmentRuleEditorSource, /onRuleTypeChange/);
  assert.doesNotMatch(textSegmentRuleEditorSource, /onRuleFieldChange/);
  assert.match(
    source,
    /<TextSegmentRuleEditor[\s\S]*config={selectedAdvancedConfig}[\s\S]*rules={selectedAdvancedConfig\.rules}[\s\S]*validationErrors={validationErrors}[\s\S]*onChange={handleTextSegmentRulesChange}[\s\S]*onConfigChange={handleTextSegmentConfigChange}[\s\S]*\/>/,
  );
});

test('advanced pricing page wires current-model rule-set JSON editing into both advanced editors', () => {
  assert.match(source, /handleAdvancedRuleSetJsonApply/);
  assert.match(
    source,
    /<TextSegmentRuleEditor[\s\S]*onRuleSetJsonApply={handleAdvancedRuleSetJsonApply}[\s\S]*\/>/,
  );
  assert.match(
    source,
    /<MediaTaskRuleEditor[\s\S]*onRuleSetJsonApply={handleAdvancedRuleSetJsonApply}[\s\S]*\/>/,
  );
  assert.match(textSegmentRuleEditorSource, /onRuleSetJsonApply/);
  assert.match(mediaTaskRuleEditorSource, /onRuleSetJsonApply/);
  assert.match(textSegmentRuleEditorSource, /编辑规则 JSON/);
  assert.match(mediaTaskRuleEditorSource, /编辑规则 JSON/);
  assert.match(textSegmentRuleEditorSource, /parseAdvancedRuleSetJsonImport/);
  assert.match(mediaTaskRuleEditorSource, /parseAdvancedRuleSetJsonImport/);
});
