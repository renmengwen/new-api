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

test('advanced pricing rules state prefers canonical AdvancedPricingConfig while keeping legacy mode fallback', () => {
  assert.match(source, /options\.AdvancedPricingConfig/);
  assert.match(source, /parseAdvancedPricingConfigOption/);
  assert.match(source, /canonicalAdvancedPricingConfig/);
  assert.match(source, /options\.AdvancedPricingMode/);
  assert.match(source, /options\.AdvancedPricingRules/);
  assert.match(
    source,
    /const serverAdvancedPricingModeMap = useMemo\([\s\S]*buildBillingModeMap\([\s\S]*Object\.keys\(canonicalAdvancedPricingConfig\.billing_mode\)\.length > 0[\s\S]*canonicalAdvancedPricingConfig\.billing_mode[\s\S]*parseOptionJSON\(options\.AdvancedPricingMode\)/,
  );
  assert.match(
    source,
    /const serverAdvancedPricingMap = useMemo\([\s\S]*buildAdvancedPricingMap\([\s\S]*Object\.keys\(canonicalAdvancedPricingConfig\.rules\)\.length > 0[\s\S]*canonicalAdvancedPricingConfig\.rules[\s\S]*parseOptionJSON\(options\.AdvancedPricingRules\)/,
  );
  assert.doesNotMatch(source, /ModelBillingMode/);
  assert.match(source, /!selectedModel\.hasAdvancedPricing/);
  assert.match(source, /getAdvancedPricingValidationErrors/);
  assert.match(source, /saveAdvancedPricingOptions/);
  assert.match(source, /handleTextSegmentConfigChange/);
  assert.match(source, /handleMediaTaskConfigChange/);
  assert.match(source, /serviceTier/);
  assert.match(source, /inputModality: ''/);
  assert.match(source, /outputModality: ''/);
  assert.match(source, /imageSizeTier: ''/);
  assert.match(source, /toolUsageType: ''/);
  assert.match(source, /toolUsageCount: ''/);
  assert.match(source, /buildMediaTaskPreview/);
  assert.match(source, /rawAction: ''/);
  assert.match(source, /usageTotalTokens: ''/);
  assert.match(
    source,
    /selectedAdvancedConfig\.ruleType === MEDIA_TASK_RULE_TYPE/,
  );
  assert.match(
    source,
    /return buildMediaTaskPreview\(\s*selectedAdvancedConfig\.rules,\s*previewInput,\s*selectedAdvancedConfig\.taskType,\s*\)/,
  );
  assert.match(
    source,
    /const PREVIEW_BOOLEAN_FIELDS = new Set\(\['inputVideo', 'audio', 'draft'\]\);/,
  );
  assert.match(
    source,
    /const PREVIEW_NUMERIC_FIELDS = new Set\(\[[\s\S]*'toolUsageCount'[\s\S]*\]\);/,
  );
  assert.match(source, /typeof refresh === 'function'/);
});

test('advanced pricing rules state distinguishes controlled selection from legacy initial selection', () => {
  assert.match(source, /initialSelectedModelName = ''/);
  assert.match(source, /initialSelectionVersion = 0/);
  assert.match(
    source,
    /const isControlledSelection = externalSelectedModelName !== undefined;/,
  );
  assert.match(
    source,
    /const \[selectedModelName, setSelectedModelNameState\] = useState\(\s*isControlledSelection\s*\?\s*\(externalSelectedModelName \?\? ''\)\s*:\s*\(initialSelectedModelName \|\| ''\),/s,
  );
  assert.doesNotMatch(
    source,
    /externalSelectedModelName\s*\|\|\s*initialSelectedModelName/,
  );
  assert.match(source, /resolveAdvancedPricingSelectedModelName/);
  assert.match(
    source,
    /const handleSelectedModelNameChange = \(nextSelectedModelName\) => \{/,
  );
  assert.match(
    source,
    /if \(!isControlledSelection\) \{\s*setSelectedModelNameState\(nextSelectedModelName\);\s*\}/s,
  );
  assert.doesNotMatch(
    source,
    /useEffect\(\(\) => \{\s*if \(selectedModelName && typeof onSelectedModelChange === 'function'\)/s,
  );
  assert.doesNotMatch(
    source,
    /selectedModelName: externalSelectedModelName = ''/,
  );
});

test('advanced pricing rules state does not blindly overwrite dirty drafts when the parent rerenders options', () => {
  assert.match(source, /mergeAdvancedPricingModeDraftMap/);
  assert.match(source, /mergeAdvancedPricingDraftMap/);
  assert.match(source, /dirtyModelNamesRef/);
  assert.match(source, /canonicalAdvancedPricingConfig/);
  assert.match(source, /options\?\.AdvancedPricingConfig/);
  assert.match(source, /options\?\.AdvancedPricingMode/);
  assert.match(source, /options\?\.AdvancedPricingRules/);
  assert.doesNotMatch(
    source,
    /setAdvancedPricingModeMap\(buildBillingModeMap\(options\)\);\s*setAdvancedPricingMap\(buildAdvancedPricingMap\(options\)\);/s,
  );
});

test('advanced pricing rules state builds the model list strictly from candidate model names', () => {
  assert.match(
    source,
    /const models = useMemo\(\(\) => \{[\s\S]*Array\.from\(new Set\(\(candidateModelNames \|\| \[\]\)\.filter\(Boolean\)\)\)/,
  );
  assert.doesNotMatch(
    source,
    /MODEL_OPTION_KEYS\.forEach\(\(key\) => \{[\s\S]*modelNames\.add\(modelName\)/,
  );
  assert.doesNotMatch(
    source,
    /Object\.keys\(advancedPricingModeMap\)\.forEach\(\(modelName\) => \{[\s\S]*modelNames\.add\(modelName\)/,
  );
  assert.doesNotMatch(
    source,
    /Object\.keys\(advancedPricingMap\)\.forEach\(\(modelName\) => \{[\s\S]*modelNames\.add\(modelName\)/,
  );
});

test('advanced pricing rules state saves against the latest server snapshot and handles partial success without Promise.all fanout', () => {
  assert.match(source, /buildAdvancedPricingSaveMaps/);
  assert.match(source, /saveAdvancedPricingOptions/);
  assert.match(source, /buildAdvancedPricingConfigPayload/);
  assert.match(source, /const savePreview = useMemo\(/);
  assert.match(source, /configOptionKey: 'AdvancedPricingConfig'/);
  assert.match(source, /configEntry:\s*buildAdvancedPricingConfigPayload\(/);
  assert.match(source, /effectiveMode:/);
  assert.match(source, /savePreview,/);
  assert.match(source, /const savePayload = buildAdvancedPricingSaveMaps\(/);
  assert.match(
    source,
    /const handleSave = async \(\) => \{[\s\S]*?const savePayload = buildAdvancedPricingSaveMaps\(/,
  );
  assert.match(
    source,
    /const handleSave = async \(\) => \{[\s\S]*?latestModeMap:\s*Object\.keys\(canonicalAdvancedPricingConfig\.billing_mode\)\.length > 0\s*\?\s*canonicalAdvancedPricingConfig\.billing_mode\s*:\s*parseOptionJSON\(options\.AdvancedPricingMode\)/,
  );
  assert.match(
    source,
    /const handleSave = async \(\) => \{[\s\S]*?latestRulesMap:\s*Object\.keys\(canonicalAdvancedPricingConfig\.rules\)\.length > 0\s*\?\s*canonicalAdvancedPricingConfig\.rules\s*:\s*parseOptionJSON\(options\.AdvancedPricingRules\)/,
  );
  assert.match(source, /await saveAdvancedPricingOptions\(\{/);
  assert.match(source, /configOptionKey/);
  assert.doesNotMatch(source, /onPartialFailure:\s*async\s*\(/);
  assert.doesNotMatch(source, /await API\.put\('/);
  assert.doesNotMatch(source, /Promise\.all\(\[/);
});

test('advanced pricing rules state uses readable Chinese success and failure copy in the save flow', () => {
  assert.match(source, /showSuccess\(t\('保存成功'\)\)/);
  assert.match(source, /showError\(error\.message \|\| t\('保存失败，请重试'\)\)/);
  assert.match(
    source,
    /saveFailureMessage:\s*t\('保存失败，请重试'\)/,
  );
  assert.match(
    source,
    /showError\(t\('请至少先保存一条高级规则，再切换为高级规则生效'\)\)/,
  );
  assert.match(source, /showError\(t\('请先选择模型'\)\)/);
  assert.match(source, /console\.error\('刷新高级定价规则失败:', refreshError\)/);
  assert.match(
    source,
    /showError\(refreshError\.message \|\| t\('刷新失败，请手动重试'\)\)/,
  );
  assert.match(source, /console\.error\('保存高级定价规则失败:', error\)/);
});
