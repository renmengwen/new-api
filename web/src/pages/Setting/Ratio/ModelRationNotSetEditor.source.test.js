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
  new URL('./ModelRationNotSetEditor.jsx', import.meta.url),
  'utf8',
);

test('unset pricing page uses the same enabled-model fallback strategy as the price settings page', () => {
  assert.match(source, /API\.get\('\/api\/channel\/models_enabled'\)/);
  assert.match(source, /import \{ buildFallbackEnabledModelNames \} from '\.\/enabledModelCandidates';/);
  assert.match(
    source,
    /buildFallbackEnabledModelNames\(\{\s*options: props\.options,\s*initialModelName: props\.initialModelName,\s*\}\)/s,
  );
  assert.match(source, /const resolvedEnabledModels = shouldUseFallbackEnabledModels/);
  assert.match(source, /setEnabledModels\(fallbackEnabledModels\);\s*showError\(message\);/s);
  assert.match(source, /setEnabledModels\(fallbackEnabledModels\);\s*console\.error\(/s);
  assert.match(source, /candidateModelNames=\{resolvedEnabledModels\}/);
  assert.match(source, /filterMode='unset'/);
  assert.doesNotMatch(source, /setEnabledModels\(\[\]\)/);
});
