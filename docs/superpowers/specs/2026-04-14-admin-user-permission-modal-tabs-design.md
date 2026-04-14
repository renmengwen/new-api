# Admin User Permission Modal Tabs Design

Date: 2026-04-14

## Goal

Improve readability of the user permission configuration modal by keeping template binding at the top and splitting the three override sections into separate tabs.

## Current State

The modal in `web/src/pages/AdminUserPermissionsPageV3/CleanPage.jsx` renders four stacked sections:

1. Template binding
2. Action permission overrides
3. Menu visibility overrides
4. Data scope overrides

This makes the modal tall and harder to scan, especially when action and menu matrices grow.

## Approved Design

- Keep the template binding section in its current top position.
- Replace the three stacked override sections with tabs below it.
- Use exactly three tabs:
  - `action`
  - `menu`
  - `data-scope`
- Keep one shared modal footer with the existing save/cancel behavior.
- Preserve all existing state and payload shapes; this is a presentation-only change.
- Reset the active tab to the action tab when the modal closes so each new edit session starts from the same entry point.

## Non-Goals

- No backend API changes
- No permission model changes
- No new summary badges or counters in tab headers for this iteration
- No redesign of the page outside the modal

## Validation

- Add a lightweight source-based frontend test that asserts the modal uses tabs with the three expected item keys.
- Run the targeted frontend test.
- Run the frontend build to catch JSX/import regressions.
