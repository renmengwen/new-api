import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const dashboardDataSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/hooks/dashboard/useDashboardData.js'),
  'utf8',
);

test('dashboard scoped endpoint is used by admins and agents', () => {
  assert.match(dashboardDataSource, /isAgentUser/);
  assert.match(
    dashboardDataSource,
    /const canViewScopedDashboard = isAdmin\(\) \|\| isAgentUser\(\);/,
  );
  assert.match(dashboardDataSource, /if \(canViewScopedDashboard\)/);
});
