import test from 'node:test';
import assert from 'node:assert/strict';

import {
  AI_MODEL_DOC_DEFAULT_ID,
  AI_MODEL_DOC_GROUPS,
  buildAiModelDocTree,
  getAiModelDocById,
  resolveAiModelDocId,
  buildAiModelDocRoute,
} from './catalog.js';

test('AI model docs catalog keeps the approved group order and default doc id', () => {
  assert.equal(AI_MODEL_DOC_DEFAULT_ID, 'audio-native-gemini');
  assert.deepEqual(
    AI_MODEL_DOC_GROUPS.map((group) => group.key),
    [
      'audio',
      'chat',
      'completions',
      'embeddings',
      'images',
      'models',
      'moderations',
      'realtime',
      'rerank',
      'unimplemented',
      'videos',
    ],
  );
});

test('catalog groups items and resolves unknown doc ids back to the default doc', () => {
  const tree = buildAiModelDocTree();
  const chatGroup = tree.find((group) => group.key === 'chat');

  assert.ok(chatGroup);
  assert.ok(chatGroup.items.length >= 5);
  assert.equal(resolveAiModelDocId('missing-doc'), AI_MODEL_DOC_DEFAULT_ID);
  assert.equal(getAiModelDocById('models-native-openai').method, 'GET');
  assert.equal(
    buildAiModelDocRoute('chat-openai-chat-completions'),
    '/console/docs/ai-model/chat-openai-chat-completions',
  );
});
