import test from 'node:test';
import assert from 'node:assert/strict';

import {
  ADMIN_MENU_OPTIONS as v3Menu,
  ADMIN_PERMISSION_RESOURCES as v3Resources,
} from './catalog.js';
import {
  ADMIN_MENU_OPTIONS as cleanMenu,
  ADMIN_PERMISSION_RESOURCES as cleanResources,
} from '../AdminConsole/permissionCatalogUiClean.js';
import {
  ADMIN_MENU_OPTIONS as uiMenu,
  ADMIN_PERMISSION_RESOURCES as uiResources,
} from '../AdminConsole/permissionCatalogUi.js';
import {
  ADMIN_MENU_OPTIONS as legacyMenu,
  ADMIN_PERMISSION_RESOURCES as legacyResources,
} from '../AdminConsole/permissionCatalog.js';

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

const catalogs = [v3Resources, cleanResources, uiResources, legacyResources];
const menuCatalogs = [v3Menu, cleanMenu, uiMenu, legacyMenu];

test('all user management permission catalogs include all user action overrides', () => {
  catalogs.forEach((resources) => {
    const actionKeys = getUserManagementActionKeys(resources);
    assert.deepEqual(actionKeys, requiredUserManagementActions);
  });
});

test('all admin menu catalogs include the model monitor module', () => {
  menuCatalogs.forEach((menuOptions) => {
    const moduleKeys = menuOptions.map((item) => item.moduleKey);
    assert.ok(moduleKeys.includes('model-monitor'));
  });
});

test('all action permission catalogs include model monitor permissions', () => {
  catalogs.forEach((resources) => {
    const modelMonitor = resources.find(
      (item) => item.resourceKey === 'model_monitor_management',
    );
    const actionKeys = modelMonitor?.actions?.map((action) => action.actionKey) || [];
    assert.deepEqual(actionKeys, ['read', 'update', 'test']);
  });
});
