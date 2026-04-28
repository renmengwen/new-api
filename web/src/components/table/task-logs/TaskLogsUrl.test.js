import test from 'node:test';
import assert from 'node:assert/strict';

import { isTaskResultPreviewUrl } from './TaskLogsUrl.js';

test('task result preview accepts absolute and local content URLs', () => {
  assert.equal(isTaskResultPreviewUrl('https://example.com/result.png'), true);
  assert.equal(isTaskResultPreviewUrl('http://example.com/result.png'), true);
  assert.equal(
    isTaskResultPreviewUrl('/v1/images/generations/task_123/content'),
    true,
  );
  assert.equal(isTaskResultPreviewUrl('/v1/videos/task_123/content'), true);
});

test('task result preview rejects raw image payloads and empty values', () => {
  assert.equal(isTaskResultPreviewUrl(''), false);
  assert.equal(isTaskResultPreviewUrl(null), false);
  assert.equal(isTaskResultPreviewUrl('iVBORw0KGgoAAAANSUhEUgAAAAE='), false);
  assert.equal(isTaskResultPreviewUrl('/api/task/self'), false);
});
