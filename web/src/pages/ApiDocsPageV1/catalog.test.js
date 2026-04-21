import test from 'node:test';
import assert from 'node:assert/strict';

import {
  AI_MODEL_DOC_DEFAULT_ID,
  AI_MODEL_DOC_ITEMS,
  AI_MODEL_DOC_GROUPS,
  buildAiModelDocTree,
  getAiModelDocById,
  resolveAiModelDocId,
  buildAiModelDocRoute,
} from './catalog.js';

const REQUIRED_DOC_IDS = [
  'audio-native-gemini',
  'audio-native-openai',
  'chat-native-claude',
  'chat-gemini-media-recognition',
  'chat-gemini-text-chat',
  'chat-openai-chat-completions',
  'chat-openai-responses',
  'completions-native-openai',
  'embeddings-native-openai',
  'embeddings-native-gemini',
  'images-gemini-native',
  'images-gemini-openai-chat',
  'images-openai-edit',
  'images-openai-generate',
  'images-qwen-generate',
  'images-qwen-edit',
  'models-native-openai',
  'models-native-gemini',
  'moderations-native-openai',
  'realtime-native-openai',
  'rerank-document',
  'unimplemented-files',
  'unimplemented-fine-tuning',
  'videos-create-task',
  'videos-get-task',
  'videos-jimeng',
  'videos-kling',
  'videos-sora',
];

const VALID_GROUP_KEYS = new Set(AI_MODEL_DOC_GROUPS.map((group) => group.key));

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

test('catalog data stays internally consistent and groups render in approved order', () => {
  const ids = AI_MODEL_DOC_ITEMS.map((item) => item.id);
  const uniqueIds = new Set(ids);

  assert.equal(uniqueIds.size, AI_MODEL_DOC_ITEMS.length);
  assert.equal(
    AI_MODEL_DOC_ITEMS.map((item) => item.groupKey).every((groupKey) => VALID_GROUP_KEYS.has(groupKey)),
    true,
  );

  REQUIRED_DOC_IDS.forEach((docId) => {
    assert.ok(uniqueIds.has(docId), `missing required doc: ${docId}`);
  });

  const tree = buildAiModelDocTree();
  assert.deepEqual(
    tree.map((group) => group.key),
    AI_MODEL_DOC_GROUPS.map((group) => group.key),
  );
  assert.equal(tree.length, AI_MODEL_DOC_GROUPS.length);
  assert.ok(tree.every((group) => group.items.length > 0));

  const chatGroup = tree.find((group) => group.key === 'chat');
  assert.ok(chatGroup);
  assert.ok(chatGroup.items.length >= 5);

  AI_MODEL_DOC_ITEMS.forEach((item) => {
    assert.ok(VALID_GROUP_KEYS.has(item.groupKey), `invalid group key: ${item.groupKey}`);
    assert.ok(item.summary.length > 0);
    assert.ok(item.description.length > 0);
    assert.ok(item.requestExample.length > 0);
    assert.ok(item.responseExample.length > 0);
    assert.ok(!item.requestExample.includes('https://api.newapi.pro'));

    if (item.method === 'GET') {
      assert.ok(!item.requestExample.includes('Content-Type: application/json'));
      assert.ok(!item.requestExample.includes('-d '));
    }

    if (item.transport === 'multipart') {
      assert.ok(item.requestExample.includes('-F '));
      assert.ok(!item.requestExample.includes('Content-Type: application/json'));
    }
  });

  assert.equal(resolveAiModelDocId('missing-doc'), AI_MODEL_DOC_DEFAULT_ID);
  assert.equal(getAiModelDocById('models-native-openai').method, 'GET');
  assert.equal(
    buildAiModelDocRoute('chat-openai-chat-completions'),
    '/console/docs/ai-model/chat-openai-chat-completions',
  );
});
