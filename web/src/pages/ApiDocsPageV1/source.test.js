import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { createRequire } from 'node:module';
import { fileURLToPath, pathToFileURL } from 'node:url';

import esbuild from 'esbuild';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

import {
  AI_MODEL_DOC_DEFAULT_ID,
  AI_MODEL_DOC_ITEMS,
  buildAiModelDocRoute,
  createAiModelDocSelectionHandler,
  expandAiModelDocGroups,
  getAiModelDocById,
  getAiModelDocDisplayState,
  resolveAiModelDocPageState,
} from './catalog.js';

const require = createRequire(import.meta.url);
require.extensions['.css'] = () => {};

const readSource = (fileUrl) =>
  fs.existsSync(fileUrl) ? fs.readFileSync(fileUrl, 'utf8') : '';

const pageSource = readSource(new URL('./index.jsx', import.meta.url));
const sidebarSource = readSource(new URL('./DocsSidebar.jsx', import.meta.url));
const contentSource = readSource(new URL('./DocContent.jsx', import.meta.url));

const buildRenderedModule = async (entryUrl) => {
  const entryPath = fileURLToPath(entryUrl);
  const tempDir = fs.mkdtempSync(path.join(path.dirname(entryPath), '.tmp-doc-render-'));
  const outfile = path.join(tempDir, 'bundle.mjs');

  try {
    await esbuild.build({
      entryPoints: [entryPath],
      bundle: true,
      format: 'esm',
      platform: 'node',
      outfile,
      external: [
        'react',
        'react/jsx-runtime',
        'react-dom',
        'react-dom/server',
        '@douyinfe/semi-icons',
      ],
      logLevel: 'silent',
    });

    return await import(`${pathToFileURL(outfile).href}?v=${Date.now()}`);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
};

const docContentModulePromise = buildRenderedModule(
  new URL('./DocContent.jsx', import.meta.url),
);

test('ApiDocsPageV1 redirects invalid route params to the default document', () => {
  const missingDocState = resolveAiModelDocPageState('ai-model', undefined);
  assert.equal(missingDocState.shouldRedirect, true);
  assert.equal(
    missingDocState.redirectTo,
    buildAiModelDocRoute(AI_MODEL_DOC_DEFAULT_ID),
  );

  const invalidCategoryState = resolveAiModelDocPageState(
    'chat',
    'chat-openai-chat-completions',
  );
  assert.equal(invalidCategoryState.shouldRedirect, true);
  assert.equal(
    invalidCategoryState.redirectTo,
    buildAiModelDocRoute(AI_MODEL_DOC_DEFAULT_ID),
  );

  const invalidDocState = resolveAiModelDocPageState('ai-model', 'missing-doc');
  assert.equal(invalidDocState.shouldRedirect, true);
  assert.equal(invalidDocState.docId, AI_MODEL_DOC_DEFAULT_ID);
  assert.equal(
    invalidDocState.redirectTo,
    buildAiModelDocRoute(AI_MODEL_DOC_DEFAULT_ID),
  );
});

test('ApiDocsPageV1 selection closes the side sheet and navigates to the chosen doc', () => {
  const calls = [];
  let sidebarVisible = true;

  const selectDoc = createAiModelDocSelectionHandler(
    (nextRoute) => calls.push(nextRoute),
    () => {
      sidebarVisible = false;
    },
  );

  selectDoc('chat-openai-chat-completions');

  assert.equal(sidebarVisible, false);
  assert.deepEqual(calls, [
    '/console/docs/ai-model/chat-openai-chat-completions',
  ]);
});

test('ApiDocsPageV1 doc display state switches between complete and placeholder docs', () => {
  const audioState = getAiModelDocDisplayState(
    getAiModelDocById('audio-native-gemini'),
  );
  const chatState = getAiModelDocDisplayState(
    getAiModelDocById('chat-openai-chat-completions'),
  );
  const placeholderState = getAiModelDocDisplayState(
    getAiModelDocById('unimplemented-files'),
  );

  assert.equal(audioState.kind, 'doc');
  assert.equal(chatState.kind, 'doc');
  assert.notEqual(audioState.title, chatState.title);
  assert.notEqual(audioState.path, chatState.path);

  assert.equal(placeholderState.kind, 'placeholder');
  assert.match(placeholderState.message, /尚未补全|补充/);
  assert.equal(placeholderState.path, '/v1/files');
});

test('placeholder catalog entries are marked consistently', () => {
  const placeholderIds = new Set([
    'unimplemented-files',
    'unimplemented-fine-tuning',
    'videos-jimeng',
    'videos-kling',
    'videos-sora',
  ]);

  AI_MODEL_DOC_ITEMS.filter((item) => placeholderIds.has(item.id)).forEach((item) => {
    assert.equal(item.status, 'placeholder', item.id);
    assert.ok(item.placeholderMessage.length > 0, item.id);
  });
});

test('DocsSidebar auto-expands the active doc group without dropping existing state', () => {
  const nextGroups = expandAiModelDocGroups(['chat'], 'videos-sora');

  assert.deepEqual(nextGroups, ['chat', 'videos']);
});

test('DocsSidebar and index.jsx stay wired to the route helpers', () => {
  assert.match(pageSource, /resolveAiModelDocPageState/);
  assert.match(pageSource, /createAiModelDocSelectionHandler/);
  assert.match(pageSource, /if \(routeState\.shouldRedirect\)/);

  assert.match(sidebarSource, /expandAiModelDocGroups/);
  assert.match(sidebarSource, /useEffect/);
  assert.match(sidebarSource, /setExpandedGroups\(\(current\) => expandAiModelDocGroups\(current, activeDocId\)\)/);
});

test('DocContent renders full docs and placeholder panels differently', async () => {
  const { default: DocContent } = await docContentModulePromise;

  const completeDoc = getAiModelDocById('chat-openai-chat-completions');
  const placeholderDoc = getAiModelDocById('videos-sora');

  const completeHtml = renderToStaticMarkup(
    React.createElement(DocContent, { doc: completeDoc }),
  );
  const placeholderHtml = renderToStaticMarkup(
    React.createElement(DocContent, { doc: placeholderDoc }),
  );

  assert.match(completeHtml, new RegExp(completeDoc.title));
  assert.match(completeHtml, new RegExp(completeDoc.path.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  assert.match(completeHtml, /请求示例/);
  assert.match(completeHtml, /响应示例/);

  assert.match(placeholderHtml, /占位文档/);
  assert.match(placeholderHtml, new RegExp(placeholderDoc.placeholderMessage));
  assert.doesNotMatch(placeholderHtml, /接口概览/);
  assert.doesNotMatch(placeholderHtml, /请求示例/);
  assert.doesNotMatch(placeholderHtml, /响应示例/);
});
