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

const entrySource = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

test('global layer order stylesheet loads before PageLayout triggers Semi on-demand styles', () => {
  const indexCssImport = entrySource.indexOf("import './index.css';");
  const pageLayoutImport = entrySource.indexOf("import PageLayout from './components/layout/PageLayout';");

  assert.notEqual(indexCssImport, -1, 'missing web/src/index.css import');
  assert.notEqual(pageLayoutImport, -1, 'missing PageLayout import');
  assert.ok(
    indexCssImport < pageLayoutImport,
    'index.css must load before PageLayout so Tailwind/Semi layer order is established first',
  );
});
