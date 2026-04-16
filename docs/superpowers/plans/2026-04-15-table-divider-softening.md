# Table Divider Softening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Soften the visual weight of bordered admin tables by replacing the current hard single-color grid treatment with a scoped, variable-driven border hierarchy for `grid-bordered-table`.

**Architecture:** Keep the rollout entirely in shared frontend styling. The implementation updates the shared table CSS contract in `web/src/index.css`, scopes the softer divider treatment to `grid-bordered-table`, and updates the existing source tests so they verify the new variable-based styling contract instead of the old `#34353A` literal.

**Tech Stack:** React 18, Vite, Semi UI tables, global CSS in `web/src/index.css`, `node:test` source tests

---

## File Structure

**Modify**

- `web/src/index.css`
  - Add shared CSS variables for softened table border strength
  - Scope the refined border treatment to `.grid-bordered-table.semi-table-wrapper`
  - Add an optional very light row hover background for the scoped bordered tables
- `web/src/components/common/ui/tableListFrame.source.test.js`
  - Update the shared bordered-table source test to assert the new scoped variable-driven CSS contract
- `web/src/pages/AdminQuotaLedgerPageV2/source.test.js`
  - Update the quota-ledger-specific source test to assert the new scoped variable-driven CSS contract

**Read / verify only**

- `web/src/components/table/tokens/TokensTable.jsx`
  - Already uses `className='grid-bordered-table rounded-xl overflow-hidden'`
- `web/src/components/table/channels/ChannelsTable.jsx`
  - Already uses `className='grid-bordered-table rounded-xl overflow-hidden'`
- `web/src/pages/AdminQuotaLedgerPageV2/index.jsx`
  - Already uses `className='grid-bordered-table'`

No JSX changes are planned in the first pass because the opt-in class is already present on the target surfaces.

### Task 1: Lock the New Styling Contract with Failing Source Tests

**Files:**
- Modify: `web/src/components/common/ui/tableListFrame.source.test.js`
- Modify: `web/src/pages/AdminQuotaLedgerPageV2/source.test.js`
- Test: `web/src/components/common/ui/tableListFrame.source.test.js`
- Test: `web/src/pages/AdminQuotaLedgerPageV2/source.test.js`

- [ ] **Step 1: Write the failing source-test expectations for the new scoped CSS contract**

Update `web/src/components/common/ui/tableListFrame.source.test.js` so the shared contract no longer asserts `#34353A`, and instead checks for the scoped `grid-bordered-table` selectors and CSS variables:

```js
test('shared table border style uses scoped variable-driven softened dividers', () => {
  assert.doesNotMatch(indexCssSource, /\.table-list-frame/);
  assert.doesNotMatch(indexCssSource, /\.quota-ledger-debug-table/);
  assert.match(indexCssSource, /--table-border-outer:/);
  assert.match(indexCssSource, /--table-divider-strong:/);
  assert.match(indexCssSource, /--table-divider-soft:/);
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper \.semi-table-container\s*\{[\s\S]*border:\s*1px solid var\(--table-border-outer\);/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper \.semi-table-container > \.semi-table-header\s*\{[\s\S]*border-bottom:\s*1px solid var\(--table-divider-strong\) !important;/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper[\s\S]*border-right:\s*1px solid var\(--table-divider-soft\) !important;[\s\S]*border-bottom:\s*1px solid var\(--table-divider-soft\) !important;/,
  );
});
```

Update `web/src/pages/AdminQuotaLedgerPageV2/source.test.js` so the quota ledger page keeps relying on the shared scoped bordered-table style:

```js
test('AdminQuotaLedgerPageV2 relies on the scoped softened grid-bordered-table style', () => {
  assert.match(pageSource, /<Table[\s\S]*className='grid-bordered-table'/);
  assert.match(pageSource, /<Table[\s\S]*\bbordered=\{true\}/);
  assert.doesNotMatch(pageSource, /quota-ledger-debug-table/);
  assert.match(indexCssSource, /--table-border-outer:/);
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper \.semi-table-container\s*\{[\s\S]*border:\s*1px solid var\(--table-border-outer\);/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper \.semi-table-container > \.semi-table-header\s*\{[\s\S]*border-bottom:\s*1px solid var\(--table-divider-strong\) !important;/,
  );
});
```

- [ ] **Step 2: Run the source tests to verify they fail for the right reason**

Run:

```bash
node --test web/src/components/common/ui/tableListFrame.source.test.js web/src/pages/AdminQuotaLedgerPageV2/source.test.js
```

Expected: FAIL because `web/src/index.css` still uses the old broad `.semi-table-wrapper` selectors and hard-coded `#34353A` values instead of the new variable-based `grid-bordered-table` contract.

- [ ] **Step 3: Commit the failing-test checkpoint**

```bash
git add web/src/components/common/ui/tableListFrame.source.test.js web/src/pages/AdminQuotaLedgerPageV2/source.test.js
git commit -m "test: lock softened table divider style contract"
```

### Task 2: Implement Scoped Softened Divider Styling in Shared CSS

**Files:**
- Modify: `web/src/index.css`
- Test: `web/src/components/common/ui/tableListFrame.source.test.js`
- Test: `web/src/pages/AdminQuotaLedgerPageV2/source.test.js`

- [ ] **Step 1: Add shared CSS variables near the existing `:root` block**

Insert the new table-divider variables into `web/src/index.css` near the existing root custom properties:

```css
:root {
  --sidebar-width: 180px;
  --sidebar-width-collapsed: 60px;
  --sidebar-current-width: var(--sidebar-width);
  --table-border-outer: rgba(52, 53, 58, 0.72);
  --table-divider-strong: rgba(52, 53, 58, 0.48);
  --table-divider-soft: rgba(52, 53, 58, 0.26);
  --table-row-hover-soft: rgba(255, 255, 255, 0.025);
}
```

This keeps future visual tuning isolated to variables instead of repeated selector edits.

- [ ] **Step 2: Replace the current broad hard-coded table border selectors with scoped `grid-bordered-table` selectors**

Update the shared table border block in `web/src/index.css` from the current global `.semi-table-wrapper` selectors to the scoped `grid-bordered-table` version:

```css
.grid-bordered-table.semi-table-wrapper .semi-table-container {
  border: 1px solid var(--table-border-outer);
  border-right: 0;
  border-bottom: 0;
}

.grid-bordered-table.semi-table-wrapper .semi-table-container > .semi-table-header {
  border-bottom: 1px solid var(--table-divider-strong) !important;
}

.grid-bordered-table.semi-table-wrapper .semi-table-container > .semi-table-body
  > .semi-table
  > .semi-table-thead
  > .semi-table-row
  > .semi-table-row-head,
.grid-bordered-table.semi-table-wrapper .semi-table-container > .semi-table-body
  > .semi-table
  > .semi-table-tbody
  > .semi-table-row
  > .semi-table-row-cell,
.grid-bordered-table.semi-table-wrapper .semi-table-container > .semi-table-header
  > .semi-table
  > .semi-table-thead
  > .semi-table-row
  > .semi-table-row-head,
.grid-bordered-table.semi-table-wrapper .semi-table-container > .semi-table-body
  > .semi-table-placeholder {
  border-right: 1px solid var(--table-divider-soft) !important;
  border-bottom: 1px solid var(--table-divider-soft) !important;
}
```

This preserves the geometry while softening the internal grid contrast.

- [ ] **Step 3: Add a minimal row hover support layer for the scoped bordered tables**

Append a hover rule below the scoped border block:

```css
.grid-bordered-table.semi-table-wrapper .semi-table-container > .semi-table-body
  > .semi-table
  > .semi-table-tbody
  > .semi-table-row:hover
  > .semi-table-row-cell {
  background: var(--table-row-hover-soft);
}
```

Keep this intentionally subtle so it helps row scanning without changing the table’s interaction model.

- [ ] **Step 4: Run the focused source tests to verify the new styling contract passes**

Run:

```bash
node --test web/src/components/common/ui/tableListFrame.source.test.js web/src/pages/AdminQuotaLedgerPageV2/source.test.js
```

Expected: PASS

- [ ] **Step 5: Commit the CSS implementation**

```bash
git add web/src/index.css
git commit -m "style: soften bordered table dividers"
```

### Task 3: Run Frontend Verification and Final Review

**Files:**
- Verify: `web/src/index.css`
- Verify: `web/src/components/common/ui/tableListFrame.source.test.js`
- Verify: `web/src/pages/AdminQuotaLedgerPageV2/source.test.js`
- Verify visually: `web/src/components/table/tokens/TokensTable.jsx`
- Verify visually: `web/src/components/table/channels/ChannelsTable.jsx`
- Verify visually: `web/src/pages/AdminQuotaLedgerPageV2/index.jsx`

- [ ] **Step 1: Run the targeted frontend build**

Run:

```bash
npm run build
```

Working directory:

```bash
cd web
```

Expected: Vite production build succeeds. Existing chunk-size warnings are acceptable if they are unchanged from the current baseline.

- [ ] **Step 2: Manually inspect the three target surfaces**

Open and review:

- token table
- channel table
- admin quota ledger table

Check for:

- outer border still visible
- header separation still readable
- inner cell dividers visibly softer than before
- hover effect subtle and not noisy
- no broken rounded corners or clipped border edges

- [ ] **Step 3: Stage the verification-related updates and commit the completed pass**

```bash
git add web/src/components/common/ui/tableListFrame.source.test.js web/src/pages/AdminQuotaLedgerPageV2/source.test.js web/src/index.css
git commit -m "test: verify softened bordered table styling"
```

- [ ] **Step 4: Capture the outcome for handoff**

Summarize:

- which selectors were scoped
- which CSS variables control the new border hierarchy
- which tables were visually reviewed
- whether any follow-up rollout to non-`grid-bordered-table` tables is still pending
