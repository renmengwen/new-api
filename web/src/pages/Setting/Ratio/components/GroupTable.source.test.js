import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(
  new URL('./GroupTable.jsx', import.meta.url),
  'utf8',
);

test('group table emits serialized row changes from a synchronous latest-row ref', () => {
  assert.match(source, /const rowsRef = useRef\(rows\);/);
  assert.match(source, /const previousRows = rowsRef\.current;/);
  assert.match(source, /rowsRef\.current = nextRows;/);
  assert.match(
    source,
    /setRows\(nextRows\);\s*onChangeRef\.current\?\.\(serializeGroupTableRows\(nextRows\)\);/s,
  );
  assert.doesNotMatch(
    source,
    /setRows\(\(previousRows\) => \{[\s\S]*onChangeRef\.current\?\./,
  );
});
