import test from 'node:test';
import assert from 'node:assert/strict';

import {
  defaultAboutPageConfig,
  isStructuredAboutEnabled,
  normalizeAboutPageConfig,
  parseAboutResponse,
} from './aboutPageConfig.js';

test('parseAboutResponse keeps old string responses as legacy content', () => {
  assert.deepEqual(parseAboutResponse('legacy markdown'), {
    legacy: 'legacy markdown',
    config: '',
  });
});

test('parseAboutResponse reads legacy and config strings from the new response object', () => {
  assert.deepEqual(
    parseAboutResponse({
      data: 'legacy fallback',
      legacy: 'legacy html',
      config: '{"enabled":true}',
    }),
    {
      legacy: 'legacy html',
      config: '{"enabled":true}',
    },
  );
});

test('normalizeAboutPageConfig returns the demo defaults for empty config', () => {
  const config = normalizeAboutPageConfig('');

  assert.equal(config.enabled, true);
  assert.equal(config.hero.eyebrow, '神州数码集团 · 企业级 AI 能力入口');
  assert.equal(config.hero.primaryActionText, '进入控制台');
  assert.equal(config.hero.primaryActionUrl, '/console');
  assert.equal(config.hero.secondaryActionText, '访问集团官网');
  assert.equal(
    config.hero.secondaryActionUrl,
    'https://www.digitalchina.com/',
  );
  assert.equal(config.overview.metrics.length, 3);
  assert.equal(config.overview.channels.length, 3);
  assert.deepEqual(
    config.capabilities.map((capability) => capability.icon),
    ['network', 'route', 'shield', 'chart'],
  );
  assert.equal(config.group.websiteLabel, 'Digital China');
  assert.equal(config.group.websiteUrl, 'https://www.digitalchina.com/');
  assert.deepEqual(
    config.contacts.map((contact) => contact.type),
    ['wechat', 'work_wechat'],
  );
  assert.equal(config.customContent, '');
  assert.equal(isStructuredAboutEnabled(config), true);
});

test('normalizeAboutPageConfig preserves user values and fills short arrays', () => {
  const config = normalizeAboutPageConfig({
    hero: {
      title: 'Custom title',
      primaryActionUrl: '/custom-console',
    },
    overview: {
      metrics: [{ value: '12', label: 'custom metric' }],
      channels: [{ name: 'Custom provider', value: 55, status: 'ok' }],
    },
    capabilities: [
      {
        icon: 'network',
        title: 'Custom capability',
        description: 'Custom description',
      },
    ],
    contacts: [
      {
        type: 'wechat',
        title: 'Custom contact',
        description: 'Scan code',
        imageUrl: '/wechat.png',
      },
    ],
    customContent: '## Custom',
  });

  assert.equal(config.hero.title, 'Custom title');
  assert.equal(config.hero.primaryActionUrl, '/custom-console');
  assert.equal(config.hero.secondaryActionUrl, 'https://www.digitalchina.com/');
  assert.equal(config.overview.metrics[0].value, '12');
  assert.equal(config.overview.metrics.length, 3);
  assert.equal(config.overview.channels[0].name, 'Custom provider');
  assert.equal(config.overview.channels.length, 3);
  assert.equal(config.capabilities[0].title, 'Custom capability');
  assert.equal(config.capabilities.length, 4);
  assert.equal(config.contacts[0].title, 'Custom contact');
  assert.equal(config.contacts.length, 2);
  assert.equal(config.customContent, '## Custom');
});

test('normalizeAboutPageConfig falls back to defaults for malformed JSON', () => {
  const config = normalizeAboutPageConfig('{bad json');

  assert.deepEqual(config, defaultAboutPageConfig);
});

test('normalizeAboutPageConfig clamps channel values into the 0 to 100 range', () => {
  const config = normalizeAboutPageConfig(
    JSON.stringify({
      overview: {
        channels: [
          { name: 'Too high', value: 140, status: 'hot' },
          { name: 'Too low', value: -10, status: 'cold' },
          { name: 'Text value', value: '68', status: 'ok' },
        ],
      },
    }),
  );

  assert.equal(config.overview.channels[0].value, 100);
  assert.equal(config.overview.channels[1].value, 0);
  assert.equal(config.overview.channels[2].value, 68);
});

test('structured about is disabled only when explicitly configured false', () => {
  const disabledConfig = normalizeAboutPageConfig({ enabled: false });

  assert.equal(disabledConfig.enabled, false);
  assert.equal(isStructuredAboutEnabled(disabledConfig), false);
  assert.equal(isStructuredAboutEnabled(), false);
  assert.equal(isStructuredAboutEnabled(null), false);
  assert.equal(isStructuredAboutEnabled({ enabled: true }), true);
});
