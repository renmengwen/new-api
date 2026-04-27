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
  assert.equal(config.hero.secondaryActionUrl, 'https://www.digitalchina.com/');
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

test('empty default config does not override existing legacy content', () => {
  assert.equal(
    isStructuredAboutEnabled(normalizeAboutPageConfig(''), '# legacy'),
    false,
  );
});

test('empty object config does not override existing legacy content', () => {
  assert.equal(
    isStructuredAboutEnabled(normalizeAboutPageConfig('{}'), '# legacy'),
    false,
  );
});

test('malformed config does not override existing legacy content', () => {
  assert.equal(
    isStructuredAboutEnabled(normalizeAboutPageConfig('{bad json'), '# legacy'),
    false,
  );
});

test('empty default config is enabled for empty installs', () => {
  assert.equal(
    isStructuredAboutEnabled(normalizeAboutPageConfig(''), ''),
    true,
  );
});

test('empty object config is enabled for empty installs', () => {
  assert.equal(
    isStructuredAboutEnabled(normalizeAboutPageConfig('{}'), ''),
    true,
  );
});

test('fallback metadata survives spread and JSON round trip', () => {
  const config = normalizeAboutPageConfig('');
  const spreadConfig = { ...config };
  const serializedConfig = JSON.parse(JSON.stringify(config));

  assert.equal(config.__source, 'default');
  assert.equal(spreadConfig.__source, 'default');
  assert.equal(serializedConfig.__source, 'default');
  assert.equal(isStructuredAboutEnabled(spreadConfig, '# legacy'), false);
  assert.equal(isStructuredAboutEnabled(serializedConfig, '# legacy'), false);
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
  const { __source, ...visibleConfig } = config;

  assert.equal(__source, 'default');
  assert.deepEqual(visibleConfig, defaultAboutPageConfig);
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

test('normalizeAboutPageConfig treats blank channel values as fallback values', () => {
  const config = normalizeAboutPageConfig({
    overview: {
      channels: [{ name: 'Blank value', value: ' ', status: 'idle' }],
    },
  });

  assert.equal(
    config.overview.channels[0].value,
    defaultAboutPageConfig.overview.channels[0].value,
  );
});

test('normalizeAboutPageConfig preserves bullet positions when filling invalid entries', () => {
  const config = normalizeAboutPageConfig({
    group: {
      bullets: [123, 'second'],
    },
  });

  assert.equal(
    config.group.bullets[0],
    defaultAboutPageConfig.group.bullets[0],
  );
  assert.equal(config.group.bullets[1], 'second');
});

test('normalizeAboutPageConfig does not let normalized changes mutate defaults', () => {
  const config = normalizeAboutPageConfig('');

  config.hero.title = 'Changed title';
  config.overview.metrics[0].value = '0';
  config.group.bullets[0] = 'Changed bullet';

  assert.equal(
    defaultAboutPageConfig.hero.title,
    '统一接入、分发与治理企业 AI 能力',
  );
  assert.equal(defaultAboutPageConfig.overview.metrics[0].value, '40+');
  assert.equal(
    defaultAboutPageConfig.group.bullets[0],
    '服务企业数字化转型与智能化升级',
  );
});

test('structured about is disabled only when explicitly configured false', () => {
  const disabledConfig = normalizeAboutPageConfig({ enabled: false });

  assert.equal(disabledConfig.enabled, false);
  assert.equal(isStructuredAboutEnabled(disabledConfig), false);
  assert.equal(isStructuredAboutEnabled(), false);
  assert.equal(isStructuredAboutEnabled(null), false);
  assert.equal(isStructuredAboutEnabled({ enabled: true }), true);
});
