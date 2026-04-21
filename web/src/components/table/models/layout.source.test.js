import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const projectRoot = process.cwd();
const modelsTableSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/components/table/models/ModelsTable.jsx'),
  'utf8',
);
const indexCssSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/index.css'),
  'utf8',
);

test('models table uses dedicated class and no horizontal scroll prop', () => {
  assert.match(
    modelsTableSource,
    /className=['"][^'"]*\bmodels-manage-table\b[^'"]*['"]/,
  );
  assert.doesNotMatch(modelsTableSource, /\bscroll=\{/);
});

test('models table last action cell keeps opaque hover background and stacking context', () => {
  assert.match(
    indexCssSource,
    /\.models-manage-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-cell:last-child\s*\{[\s\S]*position:\s*relative;[\s\S]*z-index:\s*1;[\s\S]*background:\s*var\(--semi-color-bg-2\);/,
  );
  assert.match(
    indexCssSource,
    /\.models-manage-table\.semi-table-wrapper[\s\S]*> \.semi-table-row:hover[\s\S]*> \.semi-table-row-cell:last-child\s*\{[\s\S]*z-index:\s*2;[\s\S]*background:\s*var\(--semi-color-bg-2\);/,
  );
});
