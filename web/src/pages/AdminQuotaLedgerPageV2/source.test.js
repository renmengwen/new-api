/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const pageSource = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

test('AdminQuotaLedgerPageV2 wires smart export through the quota ledger export-auto endpoint', () => {
  assert.match(pageSource, /from '\.\.\/\.\.\/helpers\/smartExport'/);
  assert.match(pageSource, /runSmartExport/);
  assert.match(pageSource, /createSmartExportStatusNotifier/);
  assert.match(pageSource, /url:\s*'\/api\/admin\/quota\/ledger\/export-auto'/);
});

test('AdminQuotaLedgerPageV2 builds the smart export payload from committed ledger filters', () => {
  assert.match(pageSource, /payload:\s*\{/);
  assert.match(pageSource, /user_id:\s*parseOptionalInteger\(committedRequest\.userId\)/);
  assert.match(pageSource, /entry_type:\s*committedRequest\.entryType/);
  assert.match(pageSource, /limit:\s*total/);
  assert.doesNotMatch(pageSource, /limit:\s*MAX_EXCEL_EXPORT_ROWS/);
  assert.match(pageSource, /fallbackFileName:\s*'quota-ledger\.xlsx'/);
});

test('AdminQuotaLedgerPageV2 keeps the quota export button disabled while the smart export request is active', () => {
  assert.match(pageSource, /const \[exportLoading, setExportLoading\] = useState\(false\)/);
  assert.match(pageSource, /if \(loading \|\| exportLoading\) \{/);
  assert.match(pageSource, /disabled=\{loading \|\| exportLoading\}/);
});
