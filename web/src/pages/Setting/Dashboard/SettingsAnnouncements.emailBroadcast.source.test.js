import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const source = fs.readFileSync(
  new URL('./SettingsAnnouncements.jsx', import.meta.url),
  'utf8',
);

test('settings announcements tracks latest add or edit for optional email broadcast', () => {
  assert.match(source, /AnnouncementEmailBroadcastModal/);
  assert.match(source, /setLatestEmailAnnouncementDraft/);
  assert.match(source, /setAnnouncementEmailConfirmVisible\(true\)/);
  assert.match(source, /defaultTitle=\{t\('系统公告'\)\}/);
  assert.match(source, /source='announcement'/);
});

test('settings announcements delete paths clear email draft instead of prompting', () => {
  assert.match(source, /setLatestEmailAnnouncementDraft\(null\)/);
});
