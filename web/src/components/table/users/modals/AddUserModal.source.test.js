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

const modalSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/components/table/users/modals/AddUserModal.jsx'),
  'utf8',
);

test('add user modal exposes a required group selector backed by loaded group options', () => {
  assert.ok(modalSource.includes("field='group'"));
  assert.ok(modalSource.includes('optionList={props.groupOptions}'));
  assert.ok(
    modalSource.includes(
      "rules={[{ required: true, message: t('请选择分组') }]}",
    ),
  );
});

test('add user modal defaults group to the first available option instead of hardcoded default', () => {
  assert.ok(
    modalSource.includes(
      "const getDefaultGroupValue = () => props.groupOptions?.[0]?.value || '';",
    ),
  );
  assert.ok(modalSource.includes('group: getDefaultGroupValue(),'));
  assert.ok(
    modalSource.includes("formApiRef.current?.setValue('group', nextDefaultGroup);"),
  );
});

test('add user modal keeps allowed token groups logic while hiding the controls', () => {
  assert.ok(modalSource.includes('props.supportsAllowedTokenGroups'));
  assert.ok(modalSource.includes("field='allowed_token_groups_enabled'"));
  assert.ok(modalSource.includes("field='allowed_token_groups'"));
  assert.ok(modalSource.includes('multiple'));
  assert.ok(
    modalSource.includes(
      "style={props.hideAllowedTokenGroupFields ? { display: 'none' } : undefined}",
    ),
  );
});
