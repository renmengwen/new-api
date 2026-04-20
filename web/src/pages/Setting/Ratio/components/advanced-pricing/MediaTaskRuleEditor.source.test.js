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
  new URL('./MediaTaskRuleEditor.jsx', import.meta.url),
  'utf8',
);

test('media task rule editor keeps the advanced rule workflow in a SideSheet with summary metadata and task preview', () => {
  assert.match(
    source,
    /export default function MediaTaskRuleEditor\(\{\s*config,\s*validationErrors = \[\],\s*onChange,\s*\}\)/,
  );
  assert.match(source, /serializeAdvancedPricingConfig/);
  assert.match(source, /buildMediaTaskPreview/);
  assert.match(source, /const ruleMeta = useMemo/);
  assert.match(source, /ruleMeta\.totalRules/);
  assert.match(source, /ruleMeta\.enabledRules/);
  assert.match(source, /config\?\.displayName/);
  assert.match(source, /config\?\.taskType/);
  assert.match(source, /config\?\.billingUnit/);
  assert.match(source, /config\?\.note/);
  assert.match(source, /SideSheet/);
  assert.match(source, /sideSheetVisible/);
  assert.match(source, /placement='right'/);
  assert.match(source, /visible={sideSheetVisible}/);
  assert.match(source, /sheetPreviewInput/);
  assert.match(source, /sheetPreviewResult/);
  assert.match(source, /previewInput\?\.rawAction/);
  assert.match(source, /previewInput\?\.inferenceMode/);
  assert.match(source, /previewInput\?\.usageTotalTokens/);
  assert.match(source, /previewInput\?\.inputVideo/);
  assert.match(source, /previewInput\?\.audio/);
  assert.match(source, /previewInput\?\.draft/);
  assert.match(source, /previewInput\?\.resolution/);
  assert.match(source, /previewInput\?\.aspectRatio/);
  assert.match(source, /previewInput\?\.outputDuration/);
  assert.match(source, /previewInput\?\.inputVideoDuration/);
  assert.match(source, /previewResult\?\.matchedRule/);
  assert.match(source, /previewResult\?\.priceSummary/);
  assert.match(source, /previewResult\?\.logPreview/);
  assert.match(source, /sheetPreviewResult\?\.priceSummary\?\.estimatedCost/);
  assert.match(source, /serializeMediaTaskRule\(previewResult\.matchedRule\)/);
  assert.match(source, /serializedConfig\.segments \|\| \[\]/);
  assert.doesNotMatch(source, /<Modal/);
});

test('media task rule editor renders modality, image tier, billing unit, and tool scaffolding fields', () => {
  assert.match(source, /field: 'inferenceMode'/);
  assert.match(source, /field: 'inputModality'/);
  assert.match(source, /field: 'outputModality'/);
  assert.match(source, /field: 'billingUnit'/);
  assert.match(source, /field: 'imageSizeTier'/);
  assert.match(source, /field: 'toolUsageType'/);
  assert.match(source, /field: 'toolUsageCount'/);
  assert.match(source, /field: 'toolOveragePrice'/);
  assert.match(source, /field: 'freeQuota'/);
  assert.match(source, /field: 'overageThreshold'/);
});

test('media task rule editor keeps delete closures fresh and uses candidate preview counts inside the side sheet', () => {
  assert.match(source, /const candidateEnabledRuleCount = useMemo/);
  assert.match(
    source,
    /candidatePreviewRules\.filter\(\(rule\) => rule\?\.enabled !== false\)\.length/,
  );
  assert.match(source, /\],\s*\[config,\s*rules,\s*t\]\s*,?\s*\)/);
  assert.doesNotMatch(
    source,
    /const columns = useMemo\([\s\S]*?\],\s*\[t\]\s*,?\s*\)/,
  );
});

test('media task rule editor matches duplicate priority validation errors exactly', () => {
  assert.match(
    source,
    /const hasMatchingPriorityError = \(error, priorityValue\) =>/,
  );
  assert.match(
    source,
    /error\.startsWith\(`priority \$\{priority\} duplicated:`\)/,
  );
  assert.match(source, /hasMatchingPriorityError\(error, candidateRule\.priority\)/);
  assert.doesNotMatch(
    source,
    /error\.includes\(`priority \$\{candidateRule\.priority\}`\)/,
  );
});

test('media task rule editor uses readable chinese summary labels instead of technical english tags', () => {
  assert.match(source, /t\('任务类型'\)/);
  assert.match(source, /t\('计费单位'\)/);
  assert.match(source, /t\('规则总数'\)/);
  assert.match(source, /t\('启用规则'\)/);
  assert.match(source, /t\('单价'\)/);
  assert.match(source, /t\('最低结算 Token'\)/);
  assert.match(source, /t\('草稿系数'\)/);
  assert.match(source, /t\('本次上报 Token'\)/);
  assert.match(source, /t\('结算 Token'\)/);
  assert.match(source, /t\('预估费用'\)/);
  assert.match(source, /t\('草稿模式'\)/);
  assert.match(source, /t\('推理模式'\)/);
  assert.match(source, /t\('分辨率'\)/);
  assert.match(source, /t\('宽高比'\)/);
  assert.match(source, /t\('输出时长'\)/);
  assert.match(source, /t\('输入视频时长'\)/);
  assert.match(source, /t\('命中规则 JSON'\)/);
  assert.doesNotMatch(source, /label:\s*'unit_price'/);
  assert.doesNotMatch(source, /label:\s*'min_tokens'/);
  assert.doesNotMatch(source, /label:\s*'draft_coefficient'/);
  assert.doesNotMatch(source, /`segments=/);
  assert.doesNotMatch(source, /`enabled=/);
  assert.doesNotMatch(source, /t\('Draft'\)/);
  assert.doesNotMatch(source, /rule_type/);
  assert.doesNotMatch(source, /label:\s*'resolution'/);
  assert.doesNotMatch(source, /label:\s*'aspect_ratio'/);
  assert.doesNotMatch(source, /label:\s*'output_duration'/);
  assert.doesNotMatch(source, /label:\s*'input_video_duration'/);
  assert.doesNotMatch(source, /保存后 segments JSON/);
  assert.doesNotMatch(source, /命中 segment JSON/);
});
test('media task rule editor uses Semi top-level TextArea export instead of Input.TextArea', () => {
  assert.match(
    source,
    /import \{[\s\S]*TextArea,[\s\S]*\} from '@douyinfe\/semi-ui';/,
  );
  assert.doesNotMatch(source, /Input\.TextArea/);
});
