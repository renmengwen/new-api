import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const readSource = (fileUrl) =>
  fs.existsSync(fileUrl) ? fs.readFileSync(fileUrl, 'utf8') : '';

const pageSource = readSource(new URL('./index.jsx', import.meta.url));
const sidebarSource = readSource(new URL('./DocsSidebar.jsx', import.meta.url));
const contentSource = readSource(new URL('./DocContent.jsx', import.meta.url));

test('ApiDocsPageV1 normalizes missing params and uses a mobile SideSheet', () => {
  assert.match(pageSource, /useParams/);
  assert.match(pageSource, /<Navigate to=\{buildAiModelDocRoute\(AI_MODEL_DOC_DEFAULT_ID\)\} replace \/>/);
  assert.match(pageSource, /useIsMobile/);
  assert.match(pageSource, /SideSheet/);
  assert.match(pageSource, /<DocsSidebar/);
  assert.match(pageSource, /<DocContent/);
});

test('DocsSidebar renders grouped docs and keeps expanded group state', () => {
  assert.match(sidebarSource, /buildAiModelDocTree/);
  assert.match(sidebarSource, /expandedGroups/);
  assert.match(sidebarSource, /onSelectDoc/);
});

test('DocContent renders the lightweight documentation sections', () => {
  assert.match(contentSource, /接口概览/);
  assert.match(contentSource, /请求路径/);
  assert.match(contentSource, /鉴权方式/);
  assert.match(contentSource, /请求示例/);
  assert.match(contentSource, /响应示例/);
  assert.match(contentSource, /methodColorMap/);
});
