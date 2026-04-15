# Table Divider Softening Design

Date: 2026-04-15

## Goal

Make list and table dividers feel visually softer in the admin UI without reducing information density or changing existing table structure, spacing, pagination, or data behavior.

## Current State

The project currently applies a unified hard border treatment to Semi tables through global styles in `web/src/index.css`.

The current style uses a single strong color value for:

- table outer border
- header separator
- cell right borders
- cell bottom borders

This creates a grid-heavy look, especially on dense admin tables.

There are also source tests that currently lock the styling to `#34353A`, so any visual refinement must update the styling contract and the tests together.

## Approved Scope

The first implementation pass only targets tables explicitly using the `grid-bordered-table` class.

This includes the existing high-signal management tables already styled as bordered grid tables, rather than every `.semi-table-wrapper` in the application.

The change is limited to:

- shared table border styling
- table hover emphasis if needed
- related source tests that currently assert the old hard-coded border color

## Recommended Approach

Use a layered border system instead of one flat border color.

The visual hierarchy should be:

1. outer border: visible but lighter than current
2. header separator: lighter than outer border
3. internal cell dividers: softest

This keeps the existing table structure and readability while making the grid feel less rigid.

## Design

### 1. Limit the first rollout to `grid-bordered-table`

Do not globally weaken all Semi tables in the first pass.

Instead, scope the refined border treatment to the tables that already opt into the current bordered grid pattern through `grid-bordered-table`.

Reason:

- lower regression risk
- easier visual review
- avoids accidental changes to unrelated tables and embedded table-like components

### 2. Introduce shared CSS variables for table border strength

Replace hard-coded border color usage in the refined table style with shared variables.

Suggested variables:

- `--table-border-outer`
- `--table-divider-strong`
- `--table-divider-soft`
- optional: `--table-row-hover-soft`

These variables should be defined in a shared global style location already used by table styles.

Reason:

- future tuning becomes a variable adjustment instead of selector rewrites
- avoids repeating literal color values across selectors
- gives the design a stable contract for later expansion

### 3. Keep the geometry, only soften the visual contrast

Do not change:

- row height
- cell padding
- header layout
- pagination layout
- column alignment
- border presence as a whole

Only change the contrast hierarchy of the lines.

This means:

- the outer frame remains present
- the header still reads as a separated region
- rows and columns remain scannable
- internal cell dividers become less dominant than they are now

### 4. Add a very light row hover layer

If needed, add a subtle row hover background to help row separation rely less on hard borders.

This hover should be minimal and should not compete with selection, status color, or tag color.

The goal is support, not decoration.

### 5. Update tests to verify strategy instead of hard-coded color

Existing source tests should stop asserting `#34353A` directly for shared table borders.

They should instead verify:

- `grid-bordered-table` continues to use the shared border styling path
- the shared styling uses the new variable-based contract
- no temporary debug class path is introduced

This keeps the tests stable while allowing future visual tuning.

## Non-Goals

- no redesign of non-table list components such as sidebar dividers or scroll lists
- no card-style list rewrite
- no zebra-row redesign
- no mobile card layout change
- no table API changes
- no backend changes

## Risks

### 1. Column separation may become too weak

If vertical dividers are softened too aggressively, dense operational tables may become harder to scan.

Mitigation:

- keep outer border strongest
- keep header separator stronger than cell dividers
- review at least one dense table and one lighter table before expanding

### 2. Global selector spillover

If the refined selectors remain too broad, unrelated Semi tables may inherit the new style.

Mitigation:

- scope first pass to `.grid-bordered-table`
- only expand after visual confirmation

### 3. Tests tied to old literal color values

Current tests are coupled to a specific hex color and will fail by design.

Mitigation:

- update tests together with the CSS contract
- assert variable usage or shared style presence rather than a single literal value

## Validation

Validation for the first implementation pass should include:

- source tests covering the shared table style contract
- at least one table component using `CardTable` with `grid-bordered-table`
- at least one direct Semi `Table` page using `grid-bordered-table`
- frontend build to catch CSS and import regressions

Recommended visual review targets:

- token table
- channel table
- admin quota ledger table

## Success Criteria

The design is successful when:

- table dividers look lighter and less grid-heavy
- table readability remains intact on dense admin lists
- the change only affects the scoped bordered-table surfaces in the first pass
- future tuning can be done through shared variables rather than rewriting selectors
