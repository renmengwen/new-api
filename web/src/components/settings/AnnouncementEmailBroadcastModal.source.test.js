import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(
  new URL('./AnnouncementEmailBroadcastModal.jsx', import.meta.url),
  'utf8',
);

test('announcement email broadcast modal exposes editable target title and body', () => {
  assert.match(source, /Form\.Select[\s\S]*field='target'/);
  assert.match(source, /Form\.Input[\s\S]*field='title'/);
  assert.match(source, /Form\.TextArea[\s\S]*field='content'/);
});

test('announcement email broadcast modal renders edited content as email html before submit', () => {
  assert.match(source, /import\s+\{\s*marked\s*\}\s+from\s+'marked'/);
  assert.match(source, /marked\.parse\(values\.content\s*\|\|\s*''\)/);
  assert.match(source, /API\.post\('\/api\/notice\/email-broadcast'/);
});
