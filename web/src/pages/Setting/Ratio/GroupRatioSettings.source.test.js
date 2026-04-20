import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const wrapperSource = fs.readFileSync(
  new URL('./GroupRatioSettings.jsx', import.meta.url),
  'utf8',
);

const visualSource = fs.readFileSync(
  new URL('./GroupRatioSettingsVisual.jsx', import.meta.url),
  'utf8',
);

test('group ratio settings wrapper delegates to the new visual page component', () => {
  assert.match(
    wrapperSource,
    /import GroupRatioSettingsVisual from '\.\/GroupRatioSettingsVisual';/,
  );
  assert.match(
    wrapperSource,
    /return <GroupRatioSettingsVisual \{\.\.\.props\} \/>;/,
  );
});

test('group ratio settings visual page includes upstream-style dual-mode editors and guide', () => {
  assert.match(
    visualSource,
    /import GroupTable from '\.\/components\/GroupTable';/,
  );
  assert.match(
    visualSource,
    /import AutoGroupList from '\.\/components\/AutoGroupList';/,
  );
  assert.match(
    visualSource,
    /import GroupGroupRatioRules from '\.\/components\/GroupGroupRatioRules';/,
  );
  assert.match(
    visualSource,
    /import GroupSpecialUsableRules from '\.\/components\/GroupSpecialUsableRules';/,
  );
  assert.match(visualSource, /<Radio value='visual'>/);
  assert.match(visualSource, /<Radio value='manual'>/);
  assert.match(visualSource, /title=\{t\('分组设置使用说明'\)\}/);
  assert.match(visualSource, /editMode === 'visual' \? renderVisualMode\(\) : renderManualMode\(\)/);
});
