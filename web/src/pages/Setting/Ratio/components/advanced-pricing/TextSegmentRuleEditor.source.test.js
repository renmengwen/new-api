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
  new URL('./TextSegmentRuleEditor.jsx', import.meta.url),
  'utf8',
);

test('text segment rule editor keeps the modern config contract, summary metadata, and SideSheet workflow', () => {
  assert.match(
    source,
    /export default function TextSegmentRuleEditor\(\{\s*config,\s*rules,\s*validationErrors = \[\],\s*onChange,\s*onConfigChange,\s*\}\)/,
  );
  assert.match(source, /serializeAdvancedPricingConfig/);
  assert.match(source, /getTextSegmentRuleEditorMeta/);
  assert.match(source, /const ruleMeta = useMemo/);
  assert.match(source, /ruleMeta\.totalRules/);
  assert.match(source, /ruleMeta\.enabledRules/);
  assert.match(source, /ruleMeta\.hasDefaultPrice/);
  assert.match(source, /priorityHint/);
  assert.match(source, /config\?\.displayName/);
  assert.match(source, /config\?\.segmentBasis/);
  assert.match(source, /config\?\.billingUnit/);
  assert.match(source, /config\?\.defaultPrice/);
  assert.match(source, /config\?\.note/);
  assert.match(source, /field: 'serviceTier'/);
  assert.match(source, /SideSheet/);
  assert.match(source, /sideSheetVisible/);
  assert.match(source, /placement='right'/);
  assert.match(source, /sheetPreviewInput/);
  assert.match(source, /sheetPreviewResult/);
  assert.match(source, /buildTextSegmentPreview/);
  assert.match(source, /candidatePreviewRules/);
  assert.match(source, /previewInput\?\.inputTokens/);
  assert.match(source, /previewInput\?\.outputTokens/);
  assert.match(source, /previewInput\?\.serviceTier/);
  assert.match(source, /previewInput\?\.inputModality/);
  assert.match(source, /previewInput\?\.outputModality/);
  assert.match(source, /previewInput\?\.imageSizeTier/);
  assert.match(source, /previewInput\?\.toolUsageCount/);
  assert.match(source, /error\.includes\(candidateRule\.id\)/);
  assert.doesNotMatch(source, /error\.includes\(String\(candidateRule\.priority\)\)/);
  assert.doesNotMatch(source, /LegacyTextSegmentRuleEditor/);
  assert.doesNotMatch(source, /<Modal/);
  assert.doesNotMatch(source, /onRuleTypeChange/);
  assert.doesNotMatch(source, /onRuleFieldChange/);
});

test('text segment rule editor renders modality, unit, cache storage, and tool scaffolding fields', () => {
  assert.match(source, /field: 'inputModality'/);
  assert.match(source, /field: 'outputModality'/);
  assert.match(source, /field: 'imageSizeTier'/);
  assert.match(source, /field: 'billingUnit'/);
  assert.match(source, /field: 'cacheStoragePrice'/);
  assert.match(source, /field: 'toolUsageType'/);
  assert.match(source, /field: 'toolUsageCount'/);
  assert.match(source, /field: 'freeQuota'/);
  assert.match(source, /field: 'overageThreshold'/);
});

test('text segment rule editor uses readable chinese billing summary labels in tables and preview tags', () => {
  assert.match(source, /t\('输入单价'\)/);
  assert.match(source, /t\('输出单价'\)/);
  assert.match(source, /t\('缓存读单价'\)/);
  assert.match(source, /t\('缓存写单价'\)/);
  assert.match(source, /t\('规则总数'\)/);
  assert.match(source, /t\('优先级'\)/);
  assert.match(source, /t\('计费单位'\)/);
  assert.match(source, /t\('当前规则 JSON'\)/);
  assert.match(source, /t\('命中规则 JSON'\)/);
  assert.doesNotMatch(source, /`input_price:/);
  assert.doesNotMatch(source, /`output_price:/);
  assert.doesNotMatch(source, /`cache_read_price:/);
  assert.doesNotMatch(source, /`cache_write_price:/);
  assert.doesNotMatch(source, /`segments=/);
  assert.doesNotMatch(source, /`priority=/);
  assert.doesNotMatch(source, /当前 segments JSON/);
  assert.doesNotMatch(source, /命中 segment JSON/);
});
test('text segment rule editor uses Semi top-level TextArea export instead of Input.TextArea', () => {
  assert.match(
    source,
    /import \{[\s\S]*TextArea,[\s\S]*\} from '@douyinfe\/semi-ui';/,
  );
  assert.doesNotMatch(source, /const \{ TextArea \} = Input;/);
  assert.doesNotMatch(source, /Input\.TextArea/);
});
