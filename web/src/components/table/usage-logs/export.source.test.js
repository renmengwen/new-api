import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const filtersSource = fs.readFileSync(
  new URL('./UsageLogsFilters.jsx', import.meta.url),
  'utf8',
);
const hookSource = fs.readFileSync(
  new URL('../../../hooks/usage-logs/useUsageLogsData.jsx', import.meta.url),
  'utf8',
);

test('UsageLogsFilters adds an export excel button wired to the export handler', () => {
  assert.match(filtersSource, /handleExport/);
  assert.match(filtersSource, /onClick=\{handleExport\}/);
  assert.match(filtersSource, /导出 Excel/);
});

test('useUsageLogsData uses committed query state for export, refresh, paging, and visible column ordered export payloads', () => {
  assert.match(hookSource, /from '\.\/exportState'/);
  assert.match(hookSource, /buildUsageLogExportRequest/);
  assert.match(hookSource, /createUsageLogCommittedQuery/);
  assert.match(hookSource, /getVisibleUsageLogColumnKeys/);
  assert.match(hookSource, /const \[committedQuery, setCommittedQuery\] = useState\(/);
  assert.match(hookSource, /downloadExcelBlob/);
  assert.match(hookSource, /\/api\/log\/export/);
  assert.match(hookSource, /\/api\/log\/self\/export/);
  assert.match(hookSource, /showInfo\(t\('无可导出数据'\)\)/);
  assert.match(hookSource, /Modal\.confirm/);
  assert.match(hookSource, /const nextCommittedQuery = getFormValues\(\)/);
  assert.match(hookSource, /setCommittedQuery\(nextCommittedQuery\)/);
  assert.match(hookSource, /await loadLogs\(1,\s*pageSize,\s*nextCommittedQuery\)/);
  assert.match(hookSource, /loadLogs\(page,\s*pageSize,\s*committedQuery\)/);
  assert.match(hookSource, /loadLogs\(1,\s*size,\s*committedQuery\)/);
});
