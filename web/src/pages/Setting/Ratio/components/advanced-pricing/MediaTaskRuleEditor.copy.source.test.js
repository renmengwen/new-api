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

const source = fs.readFileSync(
  new URL('./MediaTaskRuleEditor.jsx', import.meta.url),
  'utf8',
);

test('media task rule editor adds a copy action that opens a new draft with max priority plus one', () => {
  assert.match(
    source,
    /import \{ IconCopy, IconDelete, IconEdit, IconPlus \} from '@douyinfe\/semi-icons';/,
  );
  assert.match(source, /const getNextPriority = \(\) =>/);
  assert.match(source, /const openCopySideSheet = \(rule\) =>/);
  assert.match(source, /setEditingRuleId\(''\);/);
  assert.match(source, /createEmptyMediaTaskRule\(Date\.now\(\)\)\.id/);
  assert.match(source, /priority: getNextPriority\(\)/);
  assert.match(source, /icon={<IconCopy \/>}/);
  assert.match(source, /onClick=\{\(\) => openCopySideSheet\(record\)\}/);
});
