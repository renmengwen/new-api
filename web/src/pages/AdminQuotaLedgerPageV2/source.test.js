import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const pageSource = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');
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

test('AdminQuotaLedgerPageV2 relies on the shared table border style without debug hooks', () => {
  assert.match(pageSource, /<Table[\s\S]*\bbordered=\{true\}/);
  assert.doesNotMatch(pageSource, /quota-ledger-debug-table/);
  assert.doesNotMatch(pageSource, /DEBUG BUILD/);
  assert.doesNotMatch(indexCssSource, /\.quota-ledger-debug-table/);
  assert.match(indexCssSource, /\.semi-table-wrapper \.semi-table-container\s*\{[\s\S]*border:\s*1px solid #34353A;/);
  assert.match(indexCssSource, /\.semi-table-wrapper \.semi-table-container > \.semi-table-header\s*\{[\s\S]*border-bottom:\s*1px solid #34353A !important;/);
  assert.match(indexCssSource, /\.semi-table-wrapper \.semi-table-container > \.semi-table-body[\s\S]*> \.semi-table-thead[\s\S]*border-bottom:\s*1px solid #34353A !important;/);
});
