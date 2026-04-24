import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const usersPageSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/components/table/users/index.jsx'),
  'utf8',
);

test('users page shows allowed token group controls outside managed mode', () => {
  assert.ok(
    usersPageSource.includes(
      'const supportsAllowedTokenGroups = capabilities.supportsAllowedTokenGroups !== false;',
    ),
  );
  assert.ok(
    usersPageSource.includes('const hideAllowedTokenGroupFields = false;'),
  );
  assert.ok(
    usersPageSource.includes(
      'supportsAllowedTokenGroups={supportsAllowedTokenGroups}',
    ),
  );
  assert.ok(
    usersPageSource.includes(
      'hideAllowedTokenGroupFields={hideAllowedTokenGroupFields}',
    ),
  );
  assert.ok(
    !usersPageSource.includes('supportsAllowedTokenGroups={isManagedMode}'),
  );
});

test('UsersPage only enables role and status filters outside managed mode', () => {
  assert.match(usersPageSource, /showRoleFilter=\{!isManagedMode\}/);
  assert.match(usersPageSource, /showStatusFilter=\{!isManagedMode\}/);
});
