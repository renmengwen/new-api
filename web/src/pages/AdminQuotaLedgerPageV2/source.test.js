import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

test('AdminQuotaLedgerPageV2 uses the renamed quota ledger entry type options constant', () => {
  const source = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

  assert.match(source, /\bQUOTA_LEDGER_ENTRY_TYPE_OPTIONS\b/);
  assert.doesNotMatch(source, /\boptionList=\{QUOTA_ENTRY_TYPE_OPTIONS\}/);
});

test('AdminQuotaLedgerPageV2 formats ledger quota amounts with six decimal places', () => {
  const source = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

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
