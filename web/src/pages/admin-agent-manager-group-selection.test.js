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

const readPageSource = (relativePath) =>
  fs.readFileSync(path.join(process.cwd(), 'web/src/pages', relativePath), 'utf8');

test('agent and manager create modals expose group selectors with loaded options', () => {
  const agentsSource = readPageSource('AdminAgentsPageV2/index.jsx');
  const managersSource = readPageSource('AdminManagersPageV2/index.jsx');

  for (const source of [agentsSource, managersSource]) {
    assert.ok(source.includes('const [groupOptions, setGroupOptions] = useState([]);'));
    assert.ok(source.includes("const [defaultGroup, setDefaultGroup] = useState('');"));
    assert.ok(source.includes("API.get('/api/group/')"));
    assert.ok(source.includes("const nextDefaultGroup = options[0]?.value || '';"));
    assert.ok(source.includes('value={formState.group}'));
  }

  assert.ok(agentsSource.includes("{t('分组')}"));
  assert.ok(managersSource.includes("{t('分组')}"));
});
