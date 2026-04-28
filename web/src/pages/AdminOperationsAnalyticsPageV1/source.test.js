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

const hookSource = fs.readFileSync(
  new URL('../../hooks/operations-analytics/useOperationsAnalyticsData.js', import.meta.url),
  'utf8',
);

test('useOperationsAnalyticsData wires smart export through the analytics export-auto endpoint', () => {
  assert.match(hookSource, /from '\.\.\/\.\.\/helpers\/smartExport'/);
  assert.match(hookSource, /\brunSmartExport\b/);
  assert.match(hookSource, /createExportCenterStartNotifier/);
  assert.match(hookSource, /autoDownloadAsync:\s*false/);
  assert.match(hookSource, /url:\s*'\/api\/admin\/analytics\/export-auto'/);
});

test('useOperationsAnalyticsData builds the smart export payload from active analytics filters and sort state', () => {
  assert.match(
    hookSource,
    /payload:\s*buildOperationsAnalyticsExportPayload\(\{\s*activeTab,\s*datePreset:\s*appliedFilters\.datePreset,\s*filters:\s*appliedFilters,\s*sortState:\s*sortStateByTab\[activeTab\],\s*limit:\s*getOperationsAnalyticsExportLimit\(/,
  );
  assert.doesNotMatch(hookSource, /limit:\s*MAX_EXCEL_EXPORT_ROWS/);
  assert.match(hookSource, /fallbackFileName:\s*`operations-analytics-\$\{activeTab\}\.xlsx`/);
});
