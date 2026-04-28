import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const hookSource = fs.readFileSync(
  new URL('../../../hooks/usage-logs/useUsageLogsData.jsx', import.meta.url),
  'utf8',
);

test('useUsageLogsData wires smart export through the usage log export-auto endpoints', () => {
  assert.match(hookSource, /from '\.\.\/\.\.\/helpers\/smartExport'/);
  assert.match(hookSource, /runSmartExport/);
  assert.match(hookSource, /createExportCenterStartNotifier/);
  assert.match(hookSource, /autoDownloadAsync:\s*false/);
  assert.match(hookSource, /\/api\/log\/export-auto/);
  assert.match(hookSource, /\/api\/log\/self\/export-auto/);
});

test('useUsageLogsData builds the smart export payload from the shared usage export request builder', () => {
  assert.match(hookSource, /payload:\s*\{\s*\.\.\.buildUsageLogExportRequest\(\{/);
  assert.match(hookSource, /committedQuery/);
  assert.match(hookSource, /visibleColumnKeys:\s*getExportColumnKeys\(\)/);
  assert.match(hookSource, /limit:\s*logCount/);
  assert.match(hookSource, /fallbackFileName:\s*'usage-logs\.xlsx'/);
});
