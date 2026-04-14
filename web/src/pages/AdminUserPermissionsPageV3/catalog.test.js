import test from 'node:test';
import assert from 'node:assert/strict';

import { ADMIN_PERMISSION_RESOURCES as v3Resources } from './catalog.js';
import { ADMIN_PERMISSION_RESOURCES as cleanResources } from '../AdminConsole/permissionCatalogUiClean.js';
import { ADMIN_PERMISSION_RESOURCES as uiResources } from '../AdminConsole/permissionCatalogUi.js';
import { ADMIN_PERMISSION_RESOURCES as legacyResources } from '../AdminConsole/permissionCatalog.js';

const requiredUserManagementActions = [
  'read',
  'create',
  'update',
  'update_status',
  'delete',
  'reset_passkey',
  'reset_2fa',
  'manage_subscriptions',
  'manage_bindings',
];

const getUserManagementActionKeys = (resources) =>
  resources
    .find((item) => item.resourceKey === 'user_management')
    ?.actions?.map((action) => action.actionKey) || [];

test('all user management permission catalogs include all user action overrides', () => {
  [v3Resources, cleanResources, uiResources, legacyResources].forEach((resources) => {
    const actionKeys = getUserManagementActionKeys(resources);
    assert.deepEqual(actionKeys, requiredUserManagementActions);
  });
});
