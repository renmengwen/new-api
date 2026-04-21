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
  assert.match(
    filtersSource,
    /disabled=\{loading \|\| exportLoading \|\| !isExportReady\}/,
  );
  assert.match(filtersSource, /导出 Excel/);
});

test('useUsageLogsData uses committed query state for export, refresh, paging, and visible column ordered export payloads', () => {
  assert.match(hookSource, /from '\.\/exportState'/);
  assert.match(hookSource, /isAgentUser/);
  assert.match(hookSource, /useUserPermissions/);
  assert.match(
    hookSource,
    /hasActionPermission\(\s*'quota_management',\s*'ledger_read',?\s*\)/,
  );
  assert.match(
    hookSource,
    /hasActionPermission\(\s*'quota_management',\s*'read_summary',?\s*\)/,
  );
  assert.match(hookSource, /buildUsageLogExportRequest/);
  assert.match(hookSource, /createUsageLogCommittedQuery/);
  assert.match(hookSource, /getVisibleUsageLogColumnKeys/);
  assert.match(hookSource, /const \[committedQuery, setCommittedQuery\] = useState\(/);
  assert.match(hookSource, /const \[listRequestsInFlight, setListRequestsInFlight\] = useState\(0\)/);
  assert.match(hookSource, /const isExportReady = listRequestsInFlight === 0;/);
  assert.match(
    hookSource,
    /const isAdminUser = isAdmin\(\) \|\| \(isAgentUser\(\) && canReadScopedUsageLogs\);/,
  );
  assert.match(hookSource, /const canShowChannelAffinityUsageCache = isAdmin\(\);/);
  assert.doesNotMatch(hookSource, /const isAdminUser = isAdmin\(\) \|\| isAgentUser\(\);/);
  assert.match(hookSource, /downloadExcelBlob/);
  assert.match(hookSource, /\/api\/log\/export/);
  assert.match(hookSource, /\/api\/log\/self\/export/);
  assert.match(hookSource, /quota_display_type/);
  assert.match(hookSource, /showInfo\(t\('无可导出数据'\)\)/);
  assert.match(hookSource, /Modal\.confirm/);
  assert.match(hookSource, /const nextCommittedQuery = getFormValues\(\)/);
  assert.doesNotMatch(hookSource, /await handleEyeClick\(nextCommittedQuery\)/);
  assert.match(hookSource, /handleEyeClick\(nextCommittedQuery\)/);
  assert.match(hookSource, /await loadLogs\(1,\s*pageSize,\s*nextCommittedQuery\)/);
  assert.match(
    hookSource,
    /await loadLogs\(1,\s*pageSize,\s*nextCommittedQuery\)[\s\S]*setCommittedQuery\(nextCommittedQuery\)/,
  );
  assert.match(hookSource, /loadLogs\(page,\s*pageSize,\s*committedQuery\)/);
  assert.match(hookSource, /loadLogs\(1,\s*size,\s*committedQuery\)/);
  assert.match(hookSource, /setListRequestsInFlight\(\(count\) => count \+ 1\)/);
  assert.match(
    hookSource,
    /setListRequestsInFlight\(\(count\) => Math\.max\(0,\s*count - 1\)\)/,
  );
  assert.match(
    hookSource,
    /createUsageLogCommittedQuery\(formValues,\s*formInitValues\.dateRange\)/,
  );
  assert.match(
    hookSource,
    /if \(loading \|\| exportLoading \|\| !isExportReady\) \{\s*return;\s*\}/,
  );
});

test('useUsageLogsData swallows stat request failures so refresh still reaches the list request', () => {
  assert.match(
    hookSource,
    /const handleEyeClick = async \(query = committedQuery\) => \{[\s\S]*try \{/,
  );
  assert.match(
    hookSource,
    /catch \(error\) \{[\s\S]*showError\(error\);[\s\S]*\}/,
  );
  assert.match(
    hookSource,
    /finally \{[\s\S]*setLoadingStat\(false\);[\s\S]*\}/,
  );
});
