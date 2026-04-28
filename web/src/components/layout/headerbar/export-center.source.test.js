import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const userAreaSource = fs.readFileSync(
  new URL('./UserArea.jsx', import.meta.url),
  'utf8',
);
const exportCenterSource = fs.readFileSync(
  new URL('./ExportCenterModal.jsx', import.meta.url),
  'utf8',
);

test('UserArea exposes export center from the account dropdown', () => {
  assert.match(userAreaSource, /import ExportCenterModal from '\.\/ExportCenterModal'/);
  assert.match(userAreaSource, /useState\(false\)/);
  assert.match(userAreaSource, /setExportCenterVisible\(true\)/);
  assert.match(userAreaSource, /t\('导出中心'\)/);
  assert.match(userAreaSource, /<ExportCenterModal/);
});

test('ExportCenterModal lists jobs and only polls while visible', () => {
  assert.match(exportCenterSource, /API\.get\('\/api\/export-jobs'/);
  assert.match(exportCenterSource, /params:\s*\{\s*p:\s*page,\s*page_size:\s*pageSize/s);
  assert.match(exportCenterSource, /if \(!visible\) \{/);
  assert.match(exportCenterSource, /window\.setInterval/);
  assert.match(exportCenterSource, /window\.clearInterval/);
});

test('ExportCenterModal renders expected table fields and download action', () => {
  assert.match(exportCenterSource, /title:\s*t\('文件名'\)/);
  assert.match(exportCenterSource, /title:\s*t\('导出时间'\)/);
  assert.match(exportCenterSource, /title:\s*t\('进度'\)/);
  assert.match(exportCenterSource, /title:\s*t\('操作'\)/);
  assert.match(exportCenterSource, /downloadAsyncExportFile/);
  assert.match(exportCenterSource, /disabled=\{record\.status !== 'succeeded'\}/);
});
