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

const appSource = readSource(new URL('../../App.jsx', import.meta.url));
const dockerIgnoreSource = readSource(
  new URL('../../../../.dockerignore', import.meta.url),
);
const deployWorkflowSource = readSource(
  new URL('../../../../.github/workflows/deploy.yml', import.meta.url),
);
const siderBarSource = readSource(
  new URL('../../components/layout/SiderBar.jsx', import.meta.url),
);
const useSidebarSource = readSource(
  new URL('../../hooks/common/useSidebar.js', import.meta.url),
);
const renderHelperSource = readSource(
  new URL('../../helpers/render.jsx', import.meta.url),
);
const settingsSidebarModulesAdminSource = readSource(
  new URL('../Setting/Operation/SettingsSidebarModulesAdmin.jsx', import.meta.url),
);
const pageSource = readSource(new URL('./index.jsx', import.meta.url));
const sidebarSource = readSource(new URL('./DocsSidebar.jsx', import.meta.url));
const contentSource = readSource(new URL('./DocContent.jsx', import.meta.url));

const rawTextPlugin = {
  name: 'raw-text',
  setup(build) {
    build.onResolve({ filter: /\?raw$/ }, (args) => ({
      path: path.resolve(args.resolveDir, args.path.replace(/\?raw$/, '')),
      namespace: 'raw-text',
    }));

    build.onLoad({ filter: /.*/, namespace: 'raw-text' }, (args) => ({
      contents: `export default ${JSON.stringify(fs.readFileSync(args.path, 'utf8'))};`,
      loader: 'js',
    }));
  },
};

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
      plugins: [rawTextPlugin],
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
  const markdownState = getAiModelDocDisplayState(
    getAiModelDocById('videos-seedance'),
  );

  assert.equal(audioState.kind, 'doc');
  assert.equal(chatState.kind, 'doc');
  assert.notEqual(audioState.title, chatState.title);
  assert.notEqual(audioState.path, chatState.path);

  assert.equal(placeholderState.kind, 'placeholder');
  assert.match(placeholderState.message, /尚未补全|补充/);
  assert.equal(placeholderState.path, '/v1/files');

  assert.equal(markdownState.kind, 'markdown');
  assert.equal(markdownState.path, '/v1/video/generations');
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
  assert.match(pageSource, /pt-16/);
  assert.match(pageSource, /sticky top-\[80px\]/);
  assert.match(pageSource, /max-h-\[calc\(100vh-96px\)\]/);
  assert.match(pageSource, /\[scrollbar-width:thin\]/);
  assert.match(pageSource, /\[&::-webkit-scrollbar\]:w-1\.5/);
  assert.match(pageSource, /overflow-y-auto/);
  assert.doesNotMatch(pageSource, /<aside className='[^']*overflow-y-auto/);
  assert.doesNotMatch(pageSource, /<main className='[^']*overflow-y-auto/);

  assert.match(sidebarSource, /expandAiModelDocGroups/);
  assert.match(sidebarSource, /useEffect/);
  assert.match(sidebarSource, /setExpandedGroups\(\(current\) => expandAiModelDocGroups\(current, activeDocId\)\)/);
});

test('ApiDocsPageV1 raw markdown imports stay inside the Docker web build context', () => {
  assert.match(contentSource, /seedance-video-task-apis\.md\?raw/);
  assert.match(contentSource, /from '\.\/seedance-video-task-apis\.md\?raw'/);
  assert.doesNotMatch(contentSource, /\.\.\/\.\.\/\.\.\/\.\.\/docs\//);
  assert.match(
    dockerIgnoreSource,
    /^!web\/src\/pages\/ApiDocsPageV1\/seedance-video-task-apis\.md$/m,
  );
});

test('deploy image build does not fail on GitHub Actions cache exporter errors', () => {
  assert.match(deployWorkflowSource, /docker\/build-push-action@v6/);
  assert.doesNotMatch(deployWorkflowSource, /cache-from:\s*type=gha/);
  assert.doesNotMatch(deployWorkflowSource, /cache-to:\s*type=gha/);
});

test('App registers the console docs page behind a private route', () => {
  assert.match(
    appSource,
    /const ApiDocsPage = lazy\(\(\) => import\('\.\/pages\/ApiDocsPageV1'\)\);/,
  );
  assert.match(
    appSource,
    /<Route\s+path='\/console\/docs\/:category\?\/:docId\?'\s+element=\{\s*<PrivateRoute>[\s\S]*?<ApiDocsPage \/>[\s\S]*?<\/PrivateRoute>\s*\}\s*\/>/,
  );
});

test('SiderBar exposes docs as a console entry and keeps nested docs routes highlighted', () => {
  assert.match(siderBarSource, /docs: '\/console\/docs'/);
  assert.match(siderBarSource, /text: t\('API 文档'\),[\s\S]*?itemKey: 'docs',[\s\S]*?to: '\/console\/docs'/);
  assert.match(
    siderBarSource,
    /if \(!matchingKey && currentPath\.startsWith\('\/console\/docs'\)\) \{\s*matchingKey = 'docs';\s*\}/,
  );
});

test('sidebar defaults and admin settings enable console docs by default', () => {
  assert.match(
    useSidebarSource,
    /console:\s*\{[\s\S]*?enabled: true,[\s\S]*?detail: true,[\s\S]*?token: true,[\s\S]*?log: true,[\s\S]*?midjourney: true,[\s\S]*?task: true,[\s\S]*?docs: true,[\s\S]*?\}/,
  );

  const docsEnabledMatches = settingsSidebarModulesAdminSource.match(/docs: true/g) ?? [];
  assert.ok(docsEnabledMatches.length >= 3);
  assert.match(
    settingsSidebarModulesAdminSource,
    /key: 'docs',\s*title: t\('API 文档'\),\s*description: t\('站内接口文档中心'\)/,
  );
});

test('legacy sidebar admin configs still gain console docs when loaded in settings', async () => {
  assert.match(useSidebarSource, /export const mergeAdminConfig = \(savedConfig\) => \{/);
  assert.match(
    useSidebarSource,
    /merged\[sectionKey\] = \{ \.\.\.merged\[sectionKey\], \.\.\.sectionConfig \};/,
  );
  assert.match(
    settingsSidebarModulesAdminSource,
    /import \{ mergeAdminConfig \} from '\.\.\/\.\.\/\.\.\/hooks\/common\/useSidebar';/,
  );
  assert.match(
    settingsSidebarModulesAdminSource,
    /const modules = JSON\.parse\(props\.options\.SidebarModulesAdmin\);\s*setSidebarModulesAdmin\(mergeAdminConfig\(modules\)\);/,
  );
});

test('render helper maps docs to the BookOpen icon', () => {
  assert.match(renderHelperSource, /BookOpen,/);
  assert.match(
    renderHelperSource,
    /case 'docs':\s*return <BookOpen \{\.\.\.commonProps\} color=\{iconColor\} \/>;/,
  );
});

test('DocContent renders full docs and placeholder panels differently', async () => {
  const { default: DocContent } = await docContentModulePromise;

  const completeDoc = getAiModelDocById('chat-openai-chat-completions');
  const placeholderDoc = getAiModelDocById('videos-sora');
  const markdownDoc = getAiModelDocById('videos-seedance');

  const completeHtml = renderToStaticMarkup(
    React.createElement(DocContent, { doc: completeDoc }),
  );
  const placeholderHtml = renderToStaticMarkup(
    React.createElement(DocContent, { doc: placeholderDoc }),
  );
  const markdownHtml = renderToStaticMarkup(
    React.createElement(DocContent, { doc: markdownDoc }),
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

  assert.match(markdownHtml, /Seedance 视频任务接口文档/);
  assert.match(markdownHtml, /POST \/v1\/video\/generations/);
  assert.match(markdownHtml, /metadata\.input_video=true/);
});
