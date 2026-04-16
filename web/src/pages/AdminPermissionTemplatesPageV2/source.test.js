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

const source = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

test('permission templates page wires guarded delete action into the admin table', () => {
  assert.match(source, /\bPopconfirm\b/);
  assert.match(source, /\bIconDelete\b/);
  assert.match(source, /const handleDelete = async \(templateId\) =>/);
  assert.match(source, /API\.delete\(`\/api\/admin\/permission-templates\/\$\{templateId\}`\)/);
  assert.match(source, /const nextTotal = Math\.max\(0, total - 1\)/);
  assert.match(source, /const nextPage = Math\.min\(page, Math\.max\(1, Math\.ceil\(nextTotal \/ pageSize\)\)\)/);
  assert.match(source, /await loadTemplates\(nextPage, pageSize, profileTypeFilter, keyword\)/);
  assert.match(
    source,
    /<Popconfirm[\s\S]*onConfirm=\{\(\) => handleDelete\(record\.id\)\}[\s\S]*>/,
  );
});
