const DIGITAL_CHINA_URL = 'https://www.digitalchina.com/';
const CONFIG_SOURCE_DEFAULT = 'default';
const CONFIG_SOURCE_CONFIGURED = 'configured';

export const defaultAboutPageConfig = {
  enabled: true,
  hero: {
    eyebrow: '神州数码集团 · 企业级 AI 能力入口',
    title: '统一接入、分发与治理企业 AI 能力',
    subtitle:
      '面向企业团队与开发者的一站式 AI API 聚合平台，统一模型接入、鉴权、计费与观测能力。',
    primaryActionText: '进入控制台',
    primaryActionUrl: '/console',
    secondaryActionText: '访问集团官网',
    secondaryActionUrl: DIGITAL_CHINA_URL,
  },
  overview: {
    title: 'AI Gateway Overview',
    description: '统一协议、统一鉴权、统一计费，降低多模型接入复杂度。',
    status: '运行中',
    metrics: [
      { value: '40+', label: '上游模型渠道' },
      { value: '99.9%', label: '服务可用性目标' },
      { value: '24/7', label: '企业支持响应' },
    ],
    channels: [
      { name: 'OpenAI', value: 86, status: '健康' },
      { name: 'Claude', value: 78, status: '稳定' },
      { name: 'Gemini', value: 72, status: '可用' },
    ],
  },
  capabilities: [
    {
      icon: 'network',
      title: '多模型聚合',
      description: '统一接入主流大模型服务，减少多供应商集成和维护成本。',
    },
    {
      icon: 'route',
      title: '智能路由分发',
      description: '按模型、渠道、分组和策略灵活调度请求，提升调用稳定性。',
    },
    {
      icon: 'shield',
      title: '企业安全治理',
      description: '集中管理鉴权、额度、分组和审计能力，支撑企业级管控。',
    },
    {
      icon: 'chart',
      title: '用量计费分析',
      description: '沉淀调用、消耗和账单数据，帮助团队持续优化 AI 成本。',
    },
  ],
  group: {
    title: '神州数码集团企业数字化能力支撑',
    description:
      '依托神州数码在企业数字化服务、云计算和生态整合领域的能力，提供稳定可信的 AI API 聚合入口。',
    status: '集团能力支持',
    bullets: [
      '服务企业数字化转型与智能化升级',
      '整合多云、多模型和多场景 AI 能力',
      '面向业务团队提供统一、可治理的接入体验',
    ],
    websiteLabel: 'Digital China',
    websiteUrl: DIGITAL_CHINA_URL,
  },
  contacts: [
    {
      type: 'wechat',
      title: '微信客服',
      description: '扫码添加平台客服，获取接入咨询与使用支持。',
      imageUrl: '',
      fallbackUrl: '',
    },
    {
      type: 'work_wechat',
      title: '企业微信客服',
      description: '通过企业微信联系服务团队，获取企业级支持。',
      imageUrl: '',
      fallbackUrl: '',
    },
  ],
  customContent: '',
};

const hasOwn = (value, key) => Object.prototype.hasOwnProperty.call(value, key);

const isPlainObject = (value) =>
  value !== null && typeof value === 'object' && !Array.isArray(value);

const clone = (value) => JSON.parse(JSON.stringify(value));

// Metadata keys beginning with "__" are helper metadata; renderers and admin
// editors should ignore them when displaying or editing user-facing content.
const withSourceMetadata = (config, source) => ({
  ...config,
  __source: source,
});

const hasMeaningfulVisibleContent = (
  value,
  fallback = defaultAboutPageConfig,
) => {
  if (isPlainObject(value)) {
    return Object.entries(value).some(([key, childValue]) => {
      if (key.startsWith('__')) {
        return false;
      }

      return hasMeaningfulVisibleContent(
        childValue,
        isPlainObject(fallback) ? fallback[key] : undefined,
      );
    });
  }

  if (Array.isArray(value)) {
    if (!Array.isArray(fallback)) {
      return value.length > 0;
    }

    if (value.length !== fallback.length) {
      return true;
    }

    return value.some((childValue, index) =>
      hasMeaningfulVisibleContent(childValue, fallback[index]),
    );
  }

  return value !== fallback;
};

const hasExplicitVisibleConfig = (value) => {
  if (Array.isArray(value)) {
    return value.some(hasExplicitVisibleConfig);
  }

  if (isPlainObject(value)) {
    return Object.entries(value).some(
      ([key, childValue]) =>
        !key.startsWith('__') && hasExplicitVisibleConfig(childValue),
    );
  }

  return value !== null && typeof value !== 'undefined';
};

const resolveParsedSource = (config) => {
  if (!isPlainObject(config)) {
    return CONFIG_SOURCE_DEFAULT;
  }

  if (config.__source === CONFIG_SOURCE_CONFIGURED) {
    return CONFIG_SOURCE_CONFIGURED;
  }

  if (config.__source === CONFIG_SOURCE_DEFAULT) {
    return hasMeaningfulVisibleContent(config)
      ? CONFIG_SOURCE_CONFIGURED
      : CONFIG_SOURCE_DEFAULT;
  }

  return hasExplicitVisibleConfig(config)
    ? CONFIG_SOURCE_CONFIGURED
    : CONFIG_SOURCE_DEFAULT;
};

const normalizeString = (source, key, fallback) =>
  isPlainObject(source) &&
  hasOwn(source, key) &&
  typeof source[key] === 'string'
    ? source[key]
    : fallback;

const normalizeObjectStrings = (source, fallback) =>
  Object.keys(fallback).reduce((result, key) => {
    result[key] = normalizeString(source, key, fallback[key]);
    return result;
  }, {});

const clampChannelValue = (value, fallback) => {
  if (typeof value === 'string' && value.trim() === '') {
    return fallback;
  }

  const numericValue =
    typeof value === 'number' || typeof value === 'string'
      ? Number(value)
      : Number.NaN;

  if (!Number.isFinite(numericValue)) {
    return fallback;
  }

  return Math.min(100, Math.max(0, numericValue));
};

const normalizeMetric = (metric, fallback) =>
  normalizeObjectStrings(metric, fallback);

const normalizeChannel = (channel, fallback) => ({
  name: normalizeString(channel, 'name', fallback.name),
  value: clampChannelValue(channel?.value, fallback.value),
  status: normalizeString(channel, 'status', fallback.status),
});

const normalizeCapability = (capability, fallback) =>
  normalizeObjectStrings(capability, fallback);

const normalizeContact = (contact, fallback) =>
  normalizeObjectStrings(contact, fallback);

const normalizeStringArray = (values, fallback) => {
  if (!Array.isArray(values)) {
    return [...fallback];
  }

  const targetLength = Math.max(values.length, fallback.length);

  return Array.from({ length: targetLength }, (_, index) =>
    typeof values[index] === 'string'
      ? values[index]
      : fallback[index] ?? fallback[fallback.length - 1],
  );
};

const normalizeArray = (values, fallback, normalizeItem) => {
  const source = Array.isArray(values) ? values : [];
  const targetLength = Math.max(source.length, fallback.length);

  return Array.from({ length: targetLength }, (_, index) =>
    normalizeItem(
      source[index],
      fallback[index] ?? fallback[fallback.length - 1],
    ),
  );
};

const parseConfigInput = (input) => {
  if (typeof input === 'string') {
    if (input.trim() === '') {
      return { config: null, source: CONFIG_SOURCE_DEFAULT };
    }

    try {
      const parsed = JSON.parse(input);
      const config = isPlainObject(parsed) ? parsed : null;

      return {
        config,
        source: resolveParsedSource(config),
      };
    } catch {
      return { config: null, source: CONFIG_SOURCE_DEFAULT };
    }
  }

  const config = isPlainObject(input) ? input : null;

  return {
    config,
    source: resolveParsedSource(config),
  };
};

export const parseAboutResponse = (data) => {
  if (typeof data === 'string') {
    return { legacy: data, config: '' };
  }

  if (!isPlainObject(data)) {
    return { legacy: '', config: '' };
  }

  return {
    legacy:
      typeof data.legacy === 'string'
        ? data.legacy
        : typeof data.data === 'string'
        ? data.data
        : '',
    config: typeof data.config === 'string' ? data.config : '',
  };
};

export const normalizeAboutPageConfig = (input) => {
  const { config, source } = parseConfigInput(input);

  if (!config) {
    return withSourceMetadata(
      clone(defaultAboutPageConfig),
      CONFIG_SOURCE_DEFAULT,
    );
  }

  const defaults = defaultAboutPageConfig;

  const normalizedConfig = {
    enabled:
      typeof config.enabled === 'boolean' ? config.enabled : defaults.enabled,
    hero: {
      ...normalizeObjectStrings(config.hero, defaults.hero),
    },
    overview: {
      title: normalizeString(config.overview, 'title', defaults.overview.title),
      description: normalizeString(
        config.overview,
        'description',
        defaults.overview.description,
      ),
      status: normalizeString(
        config.overview,
        'status',
        defaults.overview.status,
      ),
      metrics: normalizeArray(
        config.overview?.metrics,
        defaults.overview.metrics,
        normalizeMetric,
      ),
      channels: normalizeArray(
        config.overview?.channels,
        defaults.overview.channels,
        normalizeChannel,
      ),
    },
    capabilities: normalizeArray(
      config.capabilities,
      defaults.capabilities,
      normalizeCapability,
    ),
    group: {
      title: normalizeString(config.group, 'title', defaults.group.title),
      description: normalizeString(
        config.group,
        'description',
        defaults.group.description,
      ),
      status: normalizeString(config.group, 'status', defaults.group.status),
      bullets: normalizeStringArray(
        config.group?.bullets,
        defaults.group.bullets,
      ),
      websiteLabel: normalizeString(
        config.group,
        'websiteLabel',
        defaults.group.websiteLabel,
      ),
      websiteUrl: normalizeString(
        config.group,
        'websiteUrl',
        defaults.group.websiteUrl,
      ),
    },
    contacts: normalizeArray(
      config.contacts,
      defaults.contacts,
      normalizeContact,
    ),
    customContent: normalizeString(
      config,
      'customContent',
      defaults.customContent,
    ),
  };

  return withSourceMetadata(normalizedConfig, source);
};

const translateText = (value, translate) =>
  typeof value === 'string' && value !== '' ? translate(value) : value;

export const translateAboutPageConfig = (config, translate) => {
  const t = typeof translate === 'function' ? translate : (text) => text;
  const safeConfig = isPlainObject(config) ? config : {};
  const hero = safeConfig.hero || {};
  const overview = safeConfig.overview || {};
  const group = safeConfig.group || {};

  return {
    ...safeConfig,
    hero: {
      ...hero,
      eyebrow: translateText(hero.eyebrow, t),
      title: translateText(hero.title, t),
      subtitle: translateText(hero.subtitle, t),
      primaryActionText: translateText(hero.primaryActionText, t),
      secondaryActionText: translateText(hero.secondaryActionText, t),
    },
    overview: {
      ...overview,
      title: translateText(overview.title, t),
      description: translateText(overview.description, t),
      status: translateText(overview.status, t),
      metrics: Array.isArray(overview.metrics)
        ? overview.metrics.map((metric) => ({
            ...metric,
            label: translateText(metric.label, t),
          }))
        : [],
      channels: Array.isArray(overview.channels)
        ? overview.channels.map((channel) => ({
            ...channel,
            status: translateText(channel.status, t),
          }))
        : [],
    },
    capabilities: Array.isArray(safeConfig.capabilities)
      ? safeConfig.capabilities.map((capability) => ({
          ...capability,
          title: translateText(capability.title, t),
          description: translateText(capability.description, t),
        }))
      : [],
    group: {
      ...group,
      title: translateText(group.title, t),
      description: translateText(group.description, t),
      status: translateText(group.status, t),
      bullets: Array.isArray(group.bullets)
        ? group.bullets.map((bullet) => translateText(bullet, t))
        : [],
      websiteLabel: translateText(group.websiteLabel, t),
    },
    contacts: Array.isArray(safeConfig.contacts)
      ? safeConfig.contacts.map((contact) => ({
          ...contact,
          title: translateText(contact.title, t),
          description: translateText(contact.description, t),
        }))
      : [],
  };
};

export const isStructuredAboutEnabled = (config) => {
  if (config?.enabled !== true) {
    return false;
  }

  return (
    !config.__source ||
    config.__source === CONFIG_SOURCE_CONFIGURED ||
    hasMeaningfulVisibleContent(config)
  );
};
