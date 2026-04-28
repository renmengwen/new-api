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
const localeDir = new URL('../../../i18n/locales/', import.meta.url);
const exportCenterLocaleKeys = [
  '导出任务已创建，请到导出中心查看进度',
  '导出中心',
  '加载导出任务失败',
  '下载已开始',
  '导出时间',
  '行数',
  '暂无导出任务',
  '生成中',
  '调用日志',
  '额度流水',
  '额度成本汇总',
  '审计日志',
  '运营分析-模型',
  '运营分析-用户',
  '运营分析-每日',
];

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

test('ExportCenterModal labels persisted operations analytics job types', () => {
  assert.match(exportCenterSource, /operations_analytics_models:\s*'运营分析-模型'/);
  assert.match(exportCenterSource, /operations_analytics_users:\s*'运营分析-用户'/);
  assert.match(exportCenterSource, /operations_analytics_daily:\s*'运营分析-每日'/);
  assert.doesNotMatch(exportCenterSource, /admin_analytics_models/);
});

test('Export center copy is present in all supported locale files', () => {
  for (const localeFile of [
    'zh-CN.json',
    'zh-TW.json',
    'en.json',
    'fr.json',
    'ja.json',
    'ru.json',
    'vi.json',
  ]) {
    const messages = JSON.parse(
      fs.readFileSync(new URL(localeFile, localeDir), 'utf8'),
    ).translation;

    for (const key of exportCenterLocaleKeys) {
      assert.ok(
        Object.prototype.hasOwnProperty.call(messages, key),
        `${localeFile} missing ${key}`,
      );
    }
  }
});
