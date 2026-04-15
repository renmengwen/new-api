import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const projectRoot = process.cwd();
const indexCssSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/index.css'),
  'utf8',
);
const loginFormSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/components/auth/LoginForm.jsx'),
  'utf8',
);

test('login form continues to use Semi inputs that rely on shared focus styling', () => {
  assert.match(loginFormSource, /<Form\.Input[\s\S]*field='username'/);
  assert.match(loginFormSource, /<Form\.Input[\s\S]*field='password'/);
});

test('shared input styles reset Firefox focus outlines for Semi input controls', () => {
  assert.match(indexCssSource, /\.semi-input-wrapper input:-moz-focusring/);
  assert.match(indexCssSource, /\.semi-input-wrapper input:focus/);
  assert.match(indexCssSource, /\.semi-input-textarea-wrapper textarea:-moz-focusring/);
  assert.match(indexCssSource, /\.semi-select input:-moz-focusring/);
  assert.match(indexCssSource, /outline:\s*none !important;/);
});
