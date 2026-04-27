import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(new URL('./OtherSetting.jsx', import.meta.url), 'utf8');

test('other setting prompts optional email broadcast after notice save', () => {
  assert.match(source, /AnnouncementEmailBroadcastModal/);
  assert.match(source, /setNoticeEmailConfirmVisible\(true\)/);
  assert.match(source, /defaultTitle=\{t\('系统通知'\)\}/);
  assert.match(source, /defaultContent=\{noticeEmailDraft\}/);
  assert.match(source, /source='notice'/);
});
