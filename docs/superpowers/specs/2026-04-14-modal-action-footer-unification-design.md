# Modal Action Footer Unification Design

Date: 2026-04-14

## Goal

Unify the footer layout of confirm and edit dialogs in the management flows so cancel and confirm actions consistently match the existing icon-based style already used in the project.

## Current State

The codebase currently mixes two footer patterns:

- custom icon-based action footers, typically used in side sheets and some edit modals
- default Semi Modal footers driven by `onOk`, `okText`, and `cancelText`

The default footer pattern makes action buttons feel visually cramped in the affected dialogs.

## Approved Scope

Introduce a shared `ModalActionFooter` component and use it for the first batch of management dialogs:

- `web/src/pages/AdminPermissionTemplatesPageV2/index.jsx`
- `web/src/pages/AdminManagersPageV2/index.jsx`
- `web/src/pages/AdminAgentsPageV2/index.jsx`
- `web/src/pages/AdminUserPermissionsPageV3/CleanPage.jsx`
- `web/src/components/table/users/modals/EnableDisableUserModal.jsx`
- `web/src/components/table/users/modals/AddUserModal.jsx`
- `web/src/components/table/users/modals/EditUserModal.jsx`
- `web/src/pages/User/PermissionManagementTabEnhanced.jsx`
- `web/src/pages/User/AgentManagementTabEnhanced.jsx`
- `web/src/pages/User/ManagedUsersTabEnhanced.jsx`

## Design

- Add `web/src/components/common/modals/ModalActionFooter.jsx`
- Layout remains right-aligned
- Buttons use icon + text with fixed spacing
- Default cancel action uses `IconClose`
- Edit/save flows use `IconSaveStroked`
- Confirm-only stateful actions can override the confirm icon to `IconTickCircle`
- Existing business logic, callbacks, and loading states remain unchanged

## Non-Goals

- No backend changes
- No change to modal body layout
- No rewrite of already-custom complex footers outside the approved scope
- No global styling override of Semi Modal defaults

## Validation

- Add a source-based test that checks the approved first-batch files use `ModalActionFooter`
- Run the targeted node test and verify it fails before implementation
- Run it again after implementation and confirm it passes
- Run the frontend build to catch JSX/import regressions
