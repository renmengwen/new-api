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
import path from 'node:path';

const siderBarSource = fs.readFileSync(
  new URL('./SiderBar.jsx', import.meta.url),
  'utf8',
);
const indexCssSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/index.css'),
  'utf8',
);

test('sidebar nav pins its width to the container width', () => {
  assert.match(
    siderBarSource,
    /<Nav[\s\S]*style=\{\{\s*width:\s*'100%'\s*,\s*minWidth:\s*0\s*,\s*maxWidth:\s*'100%'\s*,?[\s\S]*\}\}/,
  );
});

test('sidebar scroll area no longer adds extra bottom padding', () => {
  assert.doesNotMatch(
    indexCssSource,
    /\.sidebar-nav\s+\.semi-navigation-list-wrapper\s*\{[\s\S]*padding-bottom:\s*56px;[\s\S]*\}/,
  );
});

test('sidebar nav uses flex sizing instead of forcing full-height in the scroll column', () => {
  assert.match(
    indexCssSource,
    /\.sidebar-nav\s*\{[^}]*flex:\s*1;[^}]*overflow-y:\s*auto;[^}]*min-height:\s*0;[^}]*\}/,
  );
  assert.doesNotMatch(
    indexCssSource,
    /\.sidebar-nav\s*\{[^}]*height:\s*100%;[^}]*\}/,
  );
});

test('sidebar constrains Semi navigation inner scroll chain so the list can actually scroll', () => {
  assert.match(
    indexCssSource,
    /\.sidebar-nav\s+\.semi-navigation-header-list-outer\s*\{[^}]*flex:\s*1;[^}]*min-height:\s*0;[^}]*display:\s*flex;[^}]*flex-direction:\s*column;[^}]*\}/,
  );
  assert.match(
    indexCssSource,
    /\.sidebar-nav\s+\.semi-navigation-list-wrapper\s*\{[^}]*flex:\s*1;[^}]*min-height:\s*0;[^}]*scrollbar-width:\s*none;[^}]*\}/,
  );
});

test('sidebar hides the actual Semi navigation scrollbar while preserving scroll behavior', () => {
  assert.match(
    indexCssSource,
    /\.sidebar-nav\s+\.semi-navigation-list-wrapper\s*\{[^}]*scrollbar-width:\s*none;[^}]*-ms-overflow-style:\s*none;[^}]*\}/,
  );
  assert.match(
    indexCssSource,
    /\.sidebar-nav\s+\.semi-navigation-list-wrapper::-webkit-scrollbar\s*\{[^}]*display:\s*none;[^}]*\}/,
  );
});
