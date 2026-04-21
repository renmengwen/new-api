import test from 'node:test';
import assert from 'node:assert/strict';

import {
  AI_MODEL_DOC_DEFAULT_ID,
  buildAiModelDocRoute,
  createAiModelDocSelectionHandler,
  expandAiModelDocGroups,
  getAiModelDocById,
  getAiModelDocDisplayState,
  resolveAiModelDocPageState,
} from './catalog.js';

test('ApiDocsPageV1 redirects invalid route params to the default document', () => {
  const missingDocState = resolveAiModelDocPageState('ai-model', undefined);
  assert.equal(missingDocState.shouldRedirect, true);
  assert.equal(
    missingDocState.redirectTo,
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

test('DocsSidebar auto-expands the active doc group without dropping existing state', () => {
  const nextGroups = expandAiModelDocGroups(['chat'], 'videos-sora');

  assert.deepEqual(nextGroups, ['chat', 'videos']);
});
