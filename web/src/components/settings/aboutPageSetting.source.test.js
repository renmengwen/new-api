import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const readSource = (path) => {
  const url = new URL(path, import.meta.url);

  return fs.existsSync(url) ? fs.readFileSync(url, 'utf8') : '';
};

const aboutPageSettingSource = readSource('./AboutPageSetting.jsx');
const otherSettingSource = readSource('./OtherSetting.jsx');
const updateOptionSource =
  otherSettingSource.match(
    /const updateOption = async[\s\S]*?const \[loadingInput/,
  )?.[0] || '';

test('about page setting saves structured and legacy about page options', () => {
  assert.match(aboutPageSettingSource, /AboutPageConfig/);
  assert.match(aboutPageSettingSource, /JSON\.stringify/);
  assert.match(aboutPageSettingSource, /updateOption\('AboutPageConfig'/);
  assert.match(aboutPageSettingSource, /updateOption\('About'/);
  assert.match(aboutPageSettingSource, /微信客服/);
  assert.match(aboutPageSettingSource, /企业微信客服/);
  assert.match(aboutPageSettingSource, /二维码图片地址/);
  assert.match(aboutPageSettingSource, /备用链接/);
});

test('about page setting strips metadata keys before persisting config', () => {
  assert.match(aboutPageSettingSource, /stripConfigMetadata/);
  assert.match(aboutPageSettingSource, /startsWith\('__'\)/);
  assert.match(aboutPageSettingSource, /delete\s+cleaned\[key\]/);
});

test('other setting wires about page config into personalization settings', () => {
  assert.match(
    otherSettingSource,
    /import AboutPageSetting from ['"]\.\/AboutPageSetting['"]/,
  );
  assert.match(otherSettingSource, /AboutPageConfig:\s*''/);
  assert.match(otherSettingSource, /AboutPageConfig:\s*false/);
  assert.match(otherSettingSource, /<AboutPageSetting[\s\S]*inputs=\{inputs\}/);
});

test('other setting propagates failed option saves to callers', () => {
  assert.match(updateOptionSource, /throw new Error\(/);
  assert.doesNotMatch(updateOptionSource, /showError\(message\)/);
});

test('about page setting uses responsive columns on narrow admin screens', () => {
  assert.match(
    aboutPageSettingSource,
    /const quarterColProps = \{[\s\S]*xs: 24,[\s\S]*sm: 12,[\s\S]*lg: 6,/,
  );
  assert.match(
    aboutPageSettingSource,
    /const thirdColProps = \{[\s\S]*xs: 24,[\s\S]*sm: 12,[\s\S]*lg: 8,/,
  );
  assert.match(
    aboutPageSettingSource,
    /const halfColProps = \{[\s\S]*xs: 24,[\s\S]*sm: 12,[\s\S]*lg: 12,/,
  );
  assert.doesNotMatch(aboutPageSettingSource, /<Col\s+span=\{(?:6|8|12)\}/);
});
