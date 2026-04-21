import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const pageSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/components/table/users/index.jsx'),
  'utf8',
);

test('users page enables allowed token group controls outside managed mode', () => {
  assert.ok(
    pageSource.includes(
      'const supportsAllowedTokenGroups = capabilities.supportsAllowedTokenGroups !== false;',
    ),
  );
  assert.ok(
    pageSource.includes(
      'supportsAllowedTokenGroups={supportsAllowedTokenGroups}',
    ),
  );
  assert.ok(!pageSource.includes('supportsAllowedTokenGroups={isManagedMode}'));
});
