import test from 'node:test';
import assert from 'node:assert/strict';

import { getSelectedPlayableModel } from './playgroundModelSelection.js';

test('keeps current model when it remains valid', () => {
  const models = ['gpt-4o-mini', 'gpt-5.3'];

  assert.equal(getSelectedPlayableModel(models, 'gpt-5.3'), 'gpt-5.3');
});

test('switches to the first valid model when current model is no longer available', () => {
  const models = ['gpt-4o-mini', 'gpt-5.3'];

  assert.equal(getSelectedPlayableModel(models, 'gemini-2.5-pro'), 'gpt-4o-mini');
});

test('returns empty string when the selected group has no models', () => {
  assert.equal(getSelectedPlayableModel([], 'gpt-5.3'), '');
});
