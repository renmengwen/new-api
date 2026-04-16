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

const projectRoot = process.cwd();
const adminManagersSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/pages/AdminManagersPageV2/index.jsx'),
  'utf8',
);
const adminAgentsSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/pages/AdminAgentsPageV2/index.jsx'),
  'utf8',
);
const permissionTemplatesSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/pages/AdminPermissionTemplatesPageV2/index.jsx'),
  'utf8',
);
const userPermissionsSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/pages/AdminUserPermissionsPageV3/CleanPage.jsx'),
  'utf8',
);

test('admin list action columns no longer use borderless text-link buttons', () => {
  for (const source of [
    adminManagersSource,
    adminAgentsSource,
    permissionTemplatesSource,
    userPermissionsSource,
  ]) {
    assert.doesNotMatch(source, /theme='borderless'/);
    assert.doesNotMatch(source, /style=\{actionLinkStyle\}/);
  }
});

test('manager and agent action columns use solid danger buttons for disable actions', () => {
  assert.match(
    adminManagersSource,
    /theme=\{record\.status === 1 \? 'solid' : 'light'\}[\s\S]*type=\{record\.status === 1 \? 'danger' : 'primary'\}/,
  );
  assert.match(
    adminAgentsSource,
    /theme=\{record\.status === 1 \? 'solid' : 'light'\}[\s\S]*type=\{record\.status === 1 \? 'danger' : 'primary'\}/,
  );
});

test('permission template delete action uses a solid danger button with popconfirm', () => {
  assert.match(
    permissionTemplatesSource,
    /<Popconfirm[\s\S]*onConfirm=\{\(\) => handleDelete\(record\.id\)\}[\s\S]*<Button[\s\S]*size='small'[\s\S]*theme='solid'[\s\S]*type='danger'/,
  );
});

test('user permission action uses a regular small tertiary button', () => {
  assert.match(
    userPermissionsSource,
    /<Button[\s\S]*size='small'[\s\S]*type='tertiary'[\s\S]*onClick=\{\(\) => openModal\(record\)\}/,
  );
});

test.skip('permission template delete action uses a solid danger button', () => {
  assert.match(permissionTemplatesSource, /<Button[\s\S]*theme='solid'[\s\S]*type='danger'[\s\S]*>\s*\{t\('鍒犻櫎'\)\}/);
});

test.skip('user permission action uses a regular small button instead of text-link styling', () => {
  assert.match(userPermissionsSource, /<Button[\s\S]*size='small'[\s\S]*>\s*\{t\('閰嶇疆鏉冮檺'\)\}/);
});
