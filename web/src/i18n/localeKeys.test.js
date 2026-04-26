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
import { readFileSync, readdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const localeDir = join(dirname(fileURLToPath(import.meta.url)), 'locales');

const requiredModelMonitorCopyKeys = ['点击复制渠道名称', '已复制渠道名称'];

test('all locales include model monitor channel copy keys', () => {
  const localeFiles = readdirSync(localeDir).filter((file) =>
    file.endsWith('.json'),
  );

  assert.ok(localeFiles.length > 0);
  for (const localeFile of localeFiles) {
    const localeFileContent = JSON.parse(
      readFileSync(join(localeDir, localeFile), 'utf8'),
    );
    const locale = localeFileContent.translation || localeFileContent;
    for (const key of requiredModelMonitorCopyKeys) {
      assert.equal(
        Object.prototype.hasOwnProperty.call(locale, key),
        true,
        `${localeFile} missing ${key}`,
      );
    }
  }
});
