import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync(new URL('./ModelMonitorCenter.jsx', import.meta.url), 'utf8');

test('model monitor page offsets content below the fixed header', () => {
  assert.match(source, /className=['"]mt-\[60px\] px-2['"]/);
});
