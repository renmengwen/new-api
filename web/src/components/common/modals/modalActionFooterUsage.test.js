import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const projectFiles = [
  '../../../pages/AdminPermissionTemplatesPageV2/index.jsx',
  '../../../pages/AdminManagersPageV2/index.jsx',
  '../../../pages/AdminAgentsPageV2/index.jsx',
  '../../../pages/AdminUserPermissionsPageV3/CleanPage.jsx',
  '../../table/users/modals/EnableDisableUserModal.jsx',
  '../../table/users/modals/AddUserModal.jsx',
  '../../table/users/modals/EditUserModal.jsx',
  '../../../pages/User/PermissionManagementTabEnhanced.jsx',
  '../../../pages/User/AgentManagementTabEnhanced.jsx',
  '../../../pages/User/ManagedUsersTabEnhanced.jsx',
];

test('first-batch management dialogs use shared ModalActionFooter', () => {
  projectFiles.forEach((relativePath) => {
    const source = fs.readFileSync(new URL(relativePath, import.meta.url), 'utf8');
    assert.match(source, /\bModalActionFooter\b/, `${relativePath} should reference ModalActionFooter`);
  });
});
