import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const utilsSource = fs.readFileSync(new URL('./utils.jsx', import.meta.url), 'utf8');

test('model plaza price calculation branches on active advanced segment pricing', () => {
  assert.match(utilsSource, /record\.billing_mode\s*===\s*['"]advanced['"]/);
  assert.match(utilsSource, /record\.advanced_rule_set/);
  assert.match(utilsSource, /advanced.*segment/i);
});
