import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(new URL('./index.jsx', import.meta.url), 'utf8');

test('permission templates page wires guarded delete action into the admin table', () => {
  assert.match(source, /\bPopconfirm\b/);
  assert.match(source, /\bIconDelete\b/);
  assert.match(source, /const handleDelete = async \(templateId\) =>/);
  assert.match(source, /API\.delete\(`\/api\/admin\/permission-templates\/\$\{templateId\}`\)/);
  assert.match(
    source,
    /<Popconfirm[\s\S]*onConfirm=\{\(\) => handleDelete\(record\.id\)\}[\s\S]*>/,
  );
});
