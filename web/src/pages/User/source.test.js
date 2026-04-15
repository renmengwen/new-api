import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

test('managed user page maps extra user management actions into capabilities', () => {
  assert.match(
    source,
    /hasActionPermission\(\s*'user_management',\s*'reset_passkey',?\s*\)/,
  );
  assert.match(source, /hasActionPermission\('user_management', 'reset_2fa'\)/);
  assert.match(
    source,
    /hasActionPermission\(\s*'user_management',\s*'manage_subscriptions',?\s*\)/,
  );
  assert.match(
    source,
    /hasActionPermission\(\s*'user_management',\s*'manage_bindings',?\s*\)/,
  );
});
