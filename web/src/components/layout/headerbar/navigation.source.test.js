import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const readSource = (fileUrl) =>
  fs.existsSync(fileUrl) ? fs.readFileSync(fileUrl, 'utf8') : '';

const navigationSource = readSource(
  new URL('./Navigation.jsx', import.meta.url),
);
const headerBarSource = readSource(new URL('./index.jsx', import.meta.url));
const useNavigationSource = readSource(
  new URL('../../../hooks/common/useNavigation.js', import.meta.url),
);

test('top docs navigation points to the public Gemini audio docs page', () => {
  assert.match(
    useNavigationSource,
    /HEADER_DOCS_ROUTE\s*=\s*'\/docs\/ai-model\/audio-native-gemini'/,
  );
  assert.match(
    useNavigationSource,
    /text:\s*t\('文档'\),[\s\S]*?itemKey:\s*'docs',[\s\S]*?to:\s*HEADER_DOCS_ROUTE/,
  );
  assert.doesNotMatch(
    useNavigationSource,
    /itemKey:\s*'docs'[\s\S]*?isExternal:\s*true/,
  );
  assert.doesNotMatch(headerBarSource, /useNavigation\(t,\s*docsLink,/);
  assert.doesNotMatch(
    navigationSource,
    /link\.itemKey === 'docs' && !userState\.user/,
  );
});
