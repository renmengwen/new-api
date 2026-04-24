import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

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
  'videos-seedance',
  'videos-sora',
];

const VALID_GROUP_KEYS = new Set(AI_MODEL_DOC_GROUPS.map((group) => group.key));
const USER_FACING_STRING_FIELDS = ['title', 'summary', 'description', 'requestExample', 'responseExample'];
const relayOpenApi = JSON.parse(
  fs.readFileSync(new URL('../../../../docs/openapi/relay.json', import.meta.url), 'utf8'),
);

const hasPlaceholderPattern = (value) => value.includes('??');

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
    assert.ok(item.title.length > 0);
    assert.ok(item.summary.length > 0);
    assert.ok(item.description.length > 0);
    assert.ok(item.requestExample.length > 0);
    assert.ok(item.responseExample.length > 0);
    if (item.transport === 'websocket') {
      assert.ok(item.requestExample.includes('{{base_ws_url}}'));
    } else {
      assert.ok(item.requestExample.includes('{{base_url}}'));
    }
    assert.ok(item.auth.example.length > 0);

    USER_FACING_STRING_FIELDS.forEach((field) => {
      assert.ok(!hasPlaceholderPattern(item[field]), `${item.id}.${field} still contains placeholder text`);
    });
    assert.ok(!hasPlaceholderPattern(item.auth.example), `${item.id}.auth.example still contains placeholder text`);

    if (item.transport === 'get') {
      assert.ok(!item.requestExample.includes('Content-Type: application/json'));
      assert.ok(!item.requestExample.includes('-d '));
    }

    if (item.transport === 'multipart') {
      assert.ok(item.requestExample.includes('-F '));
      assert.ok(!item.requestExample.includes('Content-Type: application/json'));
    }

    if (item.transport === 'json') {
      assert.ok(item.requestExample.includes("-H 'Content-Type: application/json'") || item.requestExample.includes('-d '));
    }

    if (item.transport === 'websocket') {
      assert.ok(item.requestExample.includes('wss://'));
      assert.ok(item.requestExample.includes('Sec-WebSocket-Protocol: realtime'));
      assert.ok(!item.requestExample.includes('curl '));
    }
  });

  assert.equal(resolveAiModelDocId('missing-doc'), AI_MODEL_DOC_DEFAULT_ID);
  assert.equal(getAiModelDocById('models-native-openai').method, 'GET');
  assert.equal(
    buildAiModelDocRoute('chat-openai-chat-completions'),
    '/console/docs/ai-model/chat-openai-chat-completions',
  );
});

test('relay openapi contract exposes the corrected realtime and video endpoints', () => {
  const { paths } = relayOpenApi;

  assert.ok(paths['/v1/videos']);
  assert.ok(paths['/v1/videos'].post);
  assert.ok(paths['/v1/videos'].post.requestBody.content['multipart/form-data']);

  assert.ok(paths['/v1/videos/{task_id}']);
  assert.ok(paths['/v1/videos/{task_id}'].get);

  assert.ok(paths['/jimeng/']);
  assert.ok(paths['/jimeng/'].post);

  assert.ok(paths['/kling/v1/videos/text2video']);
  assert.ok(paths['/kling/v1/videos/text2video'].post);

  assert.ok(paths['/v1/realtime']);
  assert.ok(paths['/v1/realtime'].get);
  assert.match(paths['/v1/realtime'].get.description, /WebSocket|wss:\/\//);
});

test('catalog video and realtime docs stay aligned with the relay contract', () => {
  const createTaskDoc = getAiModelDocById('videos-create-task');
  const getTaskDoc = getAiModelDocById('videos-get-task');
  const jimengDoc = getAiModelDocById('videos-jimeng');
  const klingDoc = getAiModelDocById('videos-kling');
  const seedanceDoc = getAiModelDocById('videos-seedance');
  const soraDoc = getAiModelDocById('videos-sora');
  const realtimeDoc = getAiModelDocById('realtime-native-openai');

  assert.equal(createTaskDoc.path, '/v1/videos');
  assert.equal(createTaskDoc.transport, 'multipart');
  assert.match(createTaskDoc.requestExample, /-F '/);

  assert.equal(getTaskDoc.path, '/v1/videos/{task_id}');
  assert.equal(getTaskDoc.transport, 'get');

  assert.equal(jimengDoc.path, '/jimeng/');
  assert.equal(klingDoc.path, '/kling/v1/videos/text2video');
  assert.equal(seedanceDoc.path, '/v1/video/generations');
  assert.equal(seedanceDoc.contentType, 'markdown');
  assert.equal(soraDoc.path, '/v1/videos');

  assert.equal(realtimeDoc.path, '/v1/realtime');
  assert.equal(realtimeDoc.transport, 'websocket');
  assert.match(realtimeDoc.requestExample, /wss:\/\/.*\/v1\/realtime\?model=/);
  assert.match(realtimeDoc.requestExample, /Sec-WebSocket-Protocol: realtime, openai-insecure-api-key\.sk-xxxxxxxx, openai-beta\.realtime-v1/);
  assert.doesNotMatch(realtimeDoc.requestExample, /curl '/);
});
