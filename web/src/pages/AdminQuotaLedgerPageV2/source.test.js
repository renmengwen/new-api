import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

test('AdminQuotaLedgerPageV2 uses the renamed quota ledger entry type options constant', () => {
  const source = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

  assert.match(source, /\bQUOTA_LEDGER_ENTRY_TYPE_OPTIONS\b/);
  assert.doesNotMatch(source, /\boptionList=\{QUOTA_ENTRY_TYPE_OPTIONS\}/);
});
