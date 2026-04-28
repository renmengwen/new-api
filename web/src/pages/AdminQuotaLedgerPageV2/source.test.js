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
const costSummaryPath = new URL('./CostSummaryTab.jsx', import.meta.url);
const costSummarySource = fs.existsSync(costSummaryPath)
  ? fs.readFileSync(costSummaryPath, 'utf8')
  : '';

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

test('AdminQuotaLedgerPageV2 exposes ledger and cost summary tabs', () => {
  assert.match(pageSource, /import \{[^}]*\bTabs\b[^}]*\} from '@douyinfe\/semi-ui'/s);
  assert.match(pageSource, /import CostSummaryTab from '\.\/CostSummaryTab'/);
  assert.match(pageSource, /<Tabs[^>]*type='line'[^>]*defaultActiveKey='ledger'[^>]*>/s);
  assert.match(pageSource, /<Tabs\.TabPane[^>]*itemKey='ledger'[^>]*>/s);
  assert.match(pageSource, /<Tabs\.TabPane[^>]*itemKey='cost-summary'[^>]*>/s);
  assert.match(pageSource, /<CostSummaryTab\s+canRead=\{canRead\}\s+permissionLoading=\{permissionLoading\}\s*\/>/);
});

test('CostSummaryTab wires the quota cost summary list and export endpoints', () => {
  assert.match(costSummarySource, /\/api\/admin\/quota\/cost-summary\?\$\{params\.toString\(\)\}/);
  assert.match(costSummarySource, /url:\s*'\/api\/admin\/quota\/cost-summary\/export-auto'/);
  assert.match(costSummarySource, /fallbackFileName:\s*'quota-cost-summary\.xlsx'/);
});

test('CostSummaryTab builds the export payload from committed cost summary filters', () => {
  const payloadStart = costSummarySource.indexOf('payload: {');
  const payloadEnd = costSummarySource.indexOf('fallbackFileName:', payloadStart);
  const payloadSource = costSummarySource.slice(payloadStart, payloadEnd);

  assert.match(payloadSource, /model_name:\s*committedRequest\.modelName/);
  assert.match(payloadSource, /vendor:\s*committedRequest\.vendor/);
  assert.match(payloadSource, /user:\s*committedRequest\.user/);
  assert.match(payloadSource, /token_name:\s*committedRequest\.tokenName/);
  assert.match(payloadSource, /channel:\s*parseOptionalInteger\(committedRequest\.channel\)/);
  assert.match(payloadSource, /group:\s*committedRequest\.group/);
  assert.match(payloadSource, /min_call_count:\s*parseOptionalInteger\(committedRequest\.minCallCount\)/);
  assert.match(payloadSource, /min_paid_usd:\s*parseOptionalNumber\(committedRequest\.minPaidUsd\)/);
  assert.doesNotMatch(payloadSource, /draftFilters\.[a-zA-Z]+/);
});
