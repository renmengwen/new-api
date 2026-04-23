import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const pageSource = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

test('AdminAuditLogsPageV1 wires smart export through the audit export-auto endpoint', () => {
  assert.match(pageSource, /from '\.\.\/\.\.\/helpers\/smartExport'/);
  assert.match(pageSource, /runSmartExport/);
  assert.match(pageSource, /createSmartExportStatusNotifier/);
  assert.match(pageSource, /url:\s*'\/api\/admin\/audit-logs\/export-auto'/);
});

test('AdminAuditLogsPageV1 builds the smart export payload from committed audit filters', () => {
  assert.match(pageSource, /payload:\s*\{/);
  assert.match(pageSource, /action_module:\s*committedRequest\.actionModule\.trim\(\)/);
  assert.match(pageSource, /operator_user_id:\s*parseOptionalInteger\(committedRequest\.operatorUserId\)/);
  assert.match(pageSource, /limit:\s*total/);
  assert.doesNotMatch(pageSource, /limit:\s*MAX_EXCEL_EXPORT_ROWS/);
  assert.match(pageSource, /fallbackFileName:\s*'audit-logs\.xlsx'/);
});

test('AdminAuditLogsPageV1 keeps the audit export button disabled while the smart export request is active', () => {
  assert.match(pageSource, /const \[exportLoading, setExportLoading\] = useState\(false\)/);
  assert.match(pageSource, /if \(loading \|\| exportLoading\) \{/);
  assert.match(pageSource, /disabled=\{loading \|\| exportLoading\}/);
});
