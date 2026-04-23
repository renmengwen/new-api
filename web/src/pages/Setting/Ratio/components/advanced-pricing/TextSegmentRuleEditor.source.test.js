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

test('text segment rule editor keeps the preview input state, reset, and binding chain for image size tier and tool usage count', () => {
  assert.match(
    source,
    /const TEXT_SEGMENT_PREVIEW_NUMERIC_FIELDS = new Set\(\[[\s\S]*'toolUsageCount'[\s\S]*\]\);/,
  );
  assert.match(source, /sheetPreviewInput, setSheetPreviewInput/);
  assert.match(source, /imageSizeTier: ''/);
  assert.match(source, /toolUsageCount: ''/);
  assert.match(source, /previewInput\?\.imageSizeTier/);
  assert.match(source, /previewInput\?\.toolUsageCount/);
  assert.match(source, /handlePreviewInputChange\('imageSizeTier', value\)/);
  assert.match(source, /handlePreviewInputChange\('toolUsageCount', value\)/);
  assert.match(source, /field: 'toolOveragePrice'/);
});

test('text segment rule editor uses readable UTF-8 Chinese labels for advanced pricing fields', () => {
  assert.match(
    source,
    /field: 'imageSizeTier'[\s\S]*label: '图像档位'[\s\S]*placeholder: '例如 hd \/ 2k \/ 4k'/,
  );
  assert.match(
    source,
    /field: 'toolUsageType'[\s\S]*label: '工具调用类型'[\s\S]*placeholder: '例如 google_search'/,
  );
  assert.match(
    source,
    /field: 'toolUsageCount'[\s\S]*label: '工具调用次数'[\s\S]*placeholder: '选填，填写整数值'/,
  );
  assert.match(
    source,
    /field: 'toolOveragePrice'[\s\S]*label: '超额单价'[\s\S]*placeholder: '可选，超额部分单价'/,
  );
  assert.match(source, /field: 'freeQuota'[\s\S]*label: '免费额度'/);
  assert.match(source, /field: 'overageThreshold'[\s\S]*label: '超额阈值'/);
  assert.match(source, /t\('图像档位'\)/);
  assert.match(source, /t\('工具调用次数'\)/);
  assert.match(source, /t\('例如 hd \/ 2k \/ 4k'\)/);
  assert.match(source, /t\('例如 3（整数）'\)/);
  assert.doesNotMatch(source, /Tool Usage Count/);
  assert.doesNotMatch(source, /Tool Usage/);
  assert.doesNotMatch(source, /Tool Overage Price/);
  assert.doesNotMatch(source, /Free Quota/);
  assert.doesNotMatch(source, /Overage Threshold/);
});

test('text segment rule editor uses a constrained top-level billing unit selector with chinese labels', () => {
  assert.match(
    source,
    /import \{[\s\S]*Select,[\s\S]*\} from '@douyinfe\/semi-ui';/,
  );
  assert.match(source, /const billingUnitOptions = useMemo/);
  assert.match(source, /value: 'per_million_tokens'/);
  assert.match(source, /label: t\('每百万 Tokens'\)/);
  assert.match(source, /value: 'per_second'/);
  assert.match(source, /label: t\('每秒'\)/);
  assert.match(source, /value: 'per_minute'/);
  assert.match(source, /label: t\('每分钟'\)/);
  assert.match(source, /value: 'per_image'/);
  assert.match(source, /label: t\('每张图片'\)/);
  assert.match(source, /value: 'per_1000_calls'/);
  assert.match(source, /label: t\('每千次调用'\)/);
  assert.match(source, /<Select/);
  assert.match(source, /optionList=\{billingUnitOptions\}/);
  assert.match(source, /value=\{config\?\.billingUnit \|\| ''\}/);
  assert.match(
    source,
    /<Select[\s\S]*value=\{config\?\.billingUnit \|\| ''\}[\s\S]*optionList=\{billingUnitOptions\}[\s\S]*style=\{\{ width: '100%' \}\}/,
  );
  assert.match(source, /field: 'billingUnit'[\s\S]*label: '计费单位'/);
  assert.doesNotMatch(source, /placeholder=\{t\('例如：1M tokens'\)\}/);
});

test('text segment rule editor still uses Semi top-level TextArea export instead of Input.TextArea', () => {
  assert.match(
    source,
    /import \{[\s\S]*TextArea,[\s\S]*\} from '@douyinfe\/semi-ui';/,
  );
  assert.doesNotMatch(source, /const \{ TextArea \} = Input;/);
  assert.doesNotMatch(source, /Input\.TextArea/);
});

test('text segment rule editor exposes a current rule-set JSON edit dialog', () => {
  assert.match(
    source,
    /import \{[\s\S]*Modal,[\s\S]*TextArea,[\s\S]*\} from '@douyinfe\/semi-ui';/,
  );
  assert.match(source, /onRuleSetJsonApply/);
  assert.match(source, /parseAdvancedRuleSetJsonImport/);
  assert.match(source, /jsonEditorVisible/);
  assert.match(source, /jsonEditorText/);
  assert.match(source, /setJsonEditorText\(JSON\.stringify\(serializedConfig, null, 2\)\)/);
  assert.match(source, /编辑规则 JSON/);
  assert.match(source, /保存 JSON/);
  assert.match(source, /expectedRuleType={TEXT_SEGMENT_RULE_TYPE}/);
});

test('text segment rule editor renders JSON blocks through the shared collapsible JSON component', () => {
  assert.match(source, /import CollapsibleJsonBlock from '\.\/CollapsibleJsonBlock';/);
  assert.match(source, /<CollapsibleJsonBlock title=\{t\('当前规则 JSON'\)\}>/);
  assert.match(source, /<CollapsibleJsonBlock title=\{t\('生成的规则 JSON'\)\}>/);
  assert.match(source, /<CollapsibleJsonBlock title=\{t\('命中规则 JSON'\)\}>/);
  assert.match(source, /<CollapsibleJsonBlock title=\{t\('保存后配置 JSON'\)\}>/);
});
