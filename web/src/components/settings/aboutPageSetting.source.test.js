import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const readSource = (path) => {
  const url = new URL(path, import.meta.url);

  return fs.existsSync(url) ? fs.readFileSync(url, 'utf8') : '';
};

const aboutPageSettingSource = readSource('./AboutPageSetting.jsx');
const otherSettingSource = readSource('./OtherSetting.jsx');

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
