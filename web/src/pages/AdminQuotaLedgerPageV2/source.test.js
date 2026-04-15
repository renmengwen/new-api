import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const pageSource = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');
const adminManagersSource = fs.readFileSync(new URL('../AdminManagersPageV2/index.jsx', import.meta.url), 'utf8');
const topupHistoryModalSource = fs.readFileSync(
  new URL('../../components/topup/modals/TopupHistoryModal.jsx', import.meta.url),
  'utf8',
);
const indexCssSource = fs.readFileSync(new URL('../../index.css', import.meta.url), 'utf8');

test('AdminQuotaLedgerPageV2 uses the renamed quota ledger entry type options constant', () => {
  const source = pageSource;

  assert.match(source, /\bQUOTA_LEDGER_ENTRY_TYPE_OPTIONS\b/);
  assert.doesNotMatch(source, /\boptionList=\{QUOTA_ENTRY_TYPE_OPTIONS\}/);
});

test('AdminQuotaLedgerPageV2 formats ledger quota amounts with six decimal places', () => {
  const source = pageSource;

  assert.match(source, /const ADMIN_QUOTA_LEDGER_DIGITS = 6;/);
  assert.match(
    source,
    /dataIndex:\s*'amount'[\s\S]*render:\s*\(value\)\s*=>\s*renderQuota\(value,\s*ADMIN_QUOTA_LEDGER_DIGITS\)/,
  );
  assert.match(
    source,
    /dataIndex:\s*'balance_before'[\s\S]*render:\s*\(value\)\s*=>\s*renderQuota\(value,\s*ADMIN_QUOTA_LEDGER_DIGITS\)/,
  );
  assert.match(
    source,
    /dataIndex:\s*'balance_after'[\s\S]*render:\s*\(value\)\s*=>\s*renderQuota\(value,\s*ADMIN_QUOTA_LEDGER_DIGITS\)/,
  );
});

test('AdminQuotaLedgerPageV2 uses committed request state and wires Excel export from committed filters', () => {
  assert.match(pageSource, /from '\.\/requestState'/);
  assert.match(pageSource, /createQuotaLedgerQueryState/);
  assert.match(pageSource, /commitDraftFilters/);
  assert.match(pageSource, /changeCommittedPage/);
  assert.match(pageSource, /changeCommittedPageSize/);
  assert.match(pageSource, /const \[queryState, setQueryState\] = useState\(\(\) => createQuotaLedgerQueryState\(\)\)/);
  assert.match(pageSource, /const \{ draftFilters, committedRequest \} = queryState/);
  assert.match(pageSource, /postExcelBlob/);
  assert.match(pageSource, /\/api\/admin\/quota\/ledger\/export/);
  assert.match(pageSource, /committedRequest\.userId/);
  assert.match(pageSource, /committedRequest\.entryType/);
  assert.match(pageSource, /limit:\s*MAX_EXCEL_EXPORT_ROWS/);
  assert.match(pageSource, /showInfo\(t\('无可导出数据'\)\)/);
  assert.match(pageSource, /Modal\.confirm/);
  assert.match(pageSource, /导出 Excel/);
});

test('AdminManagersPageV2 direct table explicitly opts into grid-bordered-table', () => {
  assert.match(
    adminManagersSource,
    /<Table[\s\S]*\bclassName=(?:\{[\s\S]*\bgrid-bordered-table\b[\s\S]*\}|['"][^'"]*\bgrid-bordered-table\b[^'"]*['"])/,
  );
});

test('TopupHistoryModal direct table explicitly opts into grid-bordered-table', () => {
  assert.match(
    topupHistoryModalSource,
    /<Table[\s\S]*\bclassName=(?:\{[\s\S]*\bgrid-bordered-table\b[\s\S]*\}|['"][^'"]*\bgrid-bordered-table\b[^'"]*['"])/,
  );
});

test('AdminQuotaLedgerPageV2 relies on the scoped softened table divider contract without debug hooks', () => {
  assert.match(pageSource, /<Table[\s\S]*\bbordered=\{true\}/);
  assert.match(pageSource, /<Table[\s\S]*\bsize=['"]default['"]/);
  assert.doesNotMatch(pageSource, /<Table[\s\S]*\bsize=['"]small['"]/);
  assert.match(pageSource, /<Table[\s\S]*className=['"][^'"]*\bgrid-bordered-table\b[^'"]*['"]/);
  assert.doesNotMatch(pageSource, /<Table[\s\S]*className=['"][^'"]*\bquota-ledger-table\b[^'"]*['"]/);
  assert.doesNotMatch(pageSource, /quota-ledger-debug-table/);
  assert.doesNotMatch(pageSource, /DEBUG BUILD/);
  assert.doesNotMatch(indexCssSource, /\.quota-ledger-debug-table/);
  assert.doesNotMatch(indexCssSource, /(?:^|\r?\n)\.semi-table-wrapper \.semi-table-container\s*\{/);
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper \.semi-table-container\s*\{[\s\S]*border:\s*0;[\s\S]*background:\s*transparent;[\s\S]*border-radius:\s*16px;/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-head[\s\S]*border-right:\s*0 !important;[\s\S]*border-bottom:\s*1px solid #EDEDEE !important;/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-cell[\s\S]*border-right:\s*0 !important;[\s\S]*border-bottom:\s*1px solid #EDEDEE !important;/,
  );
  assert.match(
    indexCssSource,
    /html\.dark \.grid-bordered-table\.semi-table-wrapper \.semi-table-container\s*\{[\s\S]*border:\s*1px solid rgba\(255,\s*255,\s*255,\s*0\.06\);[\s\S]*background:\s*rgba\(24,\s*27,\s*32,\s*0\.88\);/,
  );
  assert.match(
    indexCssSource,
    /html\.dark \.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-head[\s\S]*border-bottom:\s*1px solid #34353A !important;/,
  );
  assert.match(
    indexCssSource,
    /html\.dark \.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-cell[\s\S]*border-bottom:\s*1px solid #34353A !important;/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row:hover[\s\S]*> \.semi-table-row-cell\s*\{[\s\S]*background:\s*rgba\(15,\s*23,\s*42,\s*0\.03\);/,
  );
  assert.match(
    indexCssSource,
    /html\.dark \.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row:hover[\s\S]*> \.semi-table-row-cell\s*\{[\s\S]*background:\s*rgba\(255,\s*255,\s*255,\s*0\.04\);/,
  );
  assert.doesNotMatch(
    indexCssSource,
    /\.quota-ledger-table\.semi-table-wrapper/,
  );
});
