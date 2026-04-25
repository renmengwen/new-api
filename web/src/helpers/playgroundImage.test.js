import test from 'node:test';
import assert from 'node:assert/strict';

import {
  buildImageGenerationPayload,
  buildImageResponseContent,
  isImageGenerationModel,
} from './playgroundImage.js';

test('recognizes gpt-image-2 aliases as image generation models', () => {
  assert.equal(isImageGenerationModel('gpt-image-2'), true);
  assert.equal(isImageGenerationModel('gpt-image-2-plus'), true);
});

test('builds an image generation payload from playground input', () => {
  const payload = buildImageGenerationPayload('draw a tiger', {
    model: 'gpt-image-2-plus',
    group: 'TunnelForTL',
  });

  assert.deepEqual(payload, {
    model: 'gpt-image-2-plus',
    group: 'TunnelForTL',
    prompt: 'draw a tiger',
  });
});

test('converts image generation response data into renderable message content', () => {
  const content = buildImageResponseContent([
    { url: 'https://example.com/a.png' },
    { b64_json: 'abc123', revised_prompt: 'a better prompt' },
  ]);

  assert.deepEqual(content, [
    {
      type: 'image_url',
      image_url: { url: 'https://example.com/a.png' },
    },
    {
      type: 'image_url',
      image_url: { url: 'data:image/png;base64,abc123' },
    },
    {
      type: 'text',
      text: 'a better prompt',
    },
  ]);
});
