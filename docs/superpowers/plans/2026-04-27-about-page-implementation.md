# Structured About Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a structured, configurable, responsive About page for the Digital China Group AI API aggregation platform.

**Architecture:** Keep `/about` as the public route and `/api/about` as the data source. Store the new structured page data in the existing options table under `AboutPageConfig`, keep the legacy `About` field for backward compatibility, and render the modern page only when normalized structured data is available or the install has no legacy content.

**Tech Stack:** Go, Gin, GORM options table, React 18, Vite, Semi UI, i18next, Bun, scoped CSS.

---

## File Structure

- Modify `model/option.go`
  - Add `AboutPageConfig` to the in-memory option map with an empty string default.

- Modify `controller/misc.go`
  - Return `/api/about` data as an object with `legacy` and `config` fields.

- Modify `controller/option.go`
  - Validate `AboutPageConfig` as JSON when non-empty.

- Modify `service/setting_audit_catalog.go`
  - Map `AboutPageConfig` to the same About settings audit action as `About`.

- Create `controller/about_test.go`
  - Verify `/api/about` returns both structured config and legacy content.
  - Verify the endpoint still returns an empty legacy string safely.

- Create `web/src/pages/About/aboutPageConfig.js`
  - Export default About page config.
  - Export `normalizeAboutPageConfig`, `parseAboutResponse`, and `isStructuredAboutEnabled`.
  - Keep all helpers framework-free so they can be tested with Node.

- Create `web/src/pages/About/aboutPageConfig.test.js`
  - Test response parsing, malformed config fallback, array normalization, and legacy compatibility.

- Create `web/src/pages/About/AboutStructuredPage.jsx`
  - Render the hero, overview, capability cards, group backing section, contact QR cards, and custom content slot.

- Create `web/src/pages/About/about.css`
  - Scope styles under `.about-page`.
  - Implement desktop, tablet, and mobile responsive behavior.
  - Add hover, focus-visible, active, image loading, and image failure states.

- Modify `web/src/pages/About/index.jsx`
  - Load the new response shape.
  - Render `AboutStructuredPage` when structured content is enabled.
  - Preserve legacy Markdown and iframe behavior.

- Create `web/src/components/settings/AboutPageSetting.jsx`
  - Provide a structured admin editor for hero, overview, metrics, channels, capabilities, group copy, contact QR cards, and custom content.

- Modify `web/src/components/settings/OtherSetting.jsx`
  - Add `AboutPageConfig` to loaded inputs.
  - Replace the old About textarea area with `AboutPageSetting`, leaving legacy `About` available as an advanced compatibility field inside the new component.

- Modify `web/src/i18n/locales/{zh-CN,zh-TW,en,fr,ru,ja,vi}.json`
  - Add admin labels, fallback states, and UI-only strings used by the structured page.

---

## Task 1: Backend Option And About API

**Files:**
- Modify: `model/option.go`
- Modify: `controller/misc.go`
- Modify: `controller/option.go`
- Modify: `service/setting_audit_catalog.go`
- Create: `controller/about_test.go`

- [ ] **Step 1: Write failing backend tests**

Create `controller/about_test.go` with:

```go
package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type aboutResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Legacy string `json:"legacy"`
		Config string `json:"config"`
	} `json:"data"`
}

func TestGetAboutReturnsLegacyAndStructuredConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousOptionMap := common.OptionMap
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		"About":           "# Legacy about",
		"AboutPageConfig": `{"enabled":true,"hero":{"title":"Structured"}}`,
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/about", nil)

	GetAbout(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response aboutResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, "# Legacy about", response.Data.Legacy)
	require.JSONEq(t, `{"enabled":true,"hero":{"title":"Structured"}}`, response.Data.Config)
}

func TestGetAboutHandlesMissingOptionValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousOptionMap := common.OptionMap
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/about", nil)

	GetAbout(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response aboutResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, "", response.Data.Legacy)
	require.Equal(t, "", response.Data.Config)
}
```

- [ ] **Step 2: Run backend tests and verify they fail**

Run:

```powershell
go test ./controller -run TestGetAbout -count=1
```

Expected: FAIL because `GetAbout` currently returns `data` as a string, not an object with `legacy` and `config`.

- [ ] **Step 3: Add the option key**

In `model/option.go`, inside `InitOptionMap`, add the new default immediately after `About`:

```go
common.OptionMap["About"] = ""
common.OptionMap["AboutPageConfig"] = ""
common.OptionMap["HomePageContent"] = ""
```

- [ ] **Step 4: Return structured About API data**

In `controller/misc.go`, replace `GetAbout` with:

```go
func GetAbout(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"legacy": common.OptionMap["About"],
			"config": common.OptionMap["AboutPageConfig"],
		},
	})
	return
}
```

- [ ] **Step 5: Validate `AboutPageConfig` on option update**

In `controller/option.go`, add a switch case before the final update call:

```go
	case "AboutPageConfig":
		rawValue := strings.TrimSpace(option.Value.(string))
		if rawValue != "" {
			var parsed map[string]any
			if err := common.UnmarshalJsonStr(rawValue, &parsed); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "AboutPageConfig JSON invalid: " + err.Error(),
				})
				return
			}
		}
```

The file already imports `strings` and `common`, so no new imports are required.

- [ ] **Step 6: Add audit catalog mapping**

In `service/setting_audit_catalog.go`, add `AboutPageConfig` to the existing `save_about` registration:

```go
	registerSettingAuditMeta(meta, SettingAuditActionMeta{"setting_misc", "save_about", "系统设置-其他设置-设置关于", "option_key"},
		"About",
		"AboutPageConfig",
	)
```

Keep the existing action module and action type.

- [ ] **Step 7: Run backend tests and commit**

Run:

```powershell
go test ./controller -run "TestGetAbout|TestAuditCatalogMapsNoticeAndSensitiveKeys" -count=1
```

Expected: PASS.

Commit:

```powershell
git add model/option.go controller/misc.go controller/option.go service/setting_audit_catalog.go controller/about_test.go
git commit -m "feat: expose structured about page config"
```

---

## Task 2: Frontend Config Helpers

**Files:**
- Create: `web/src/pages/About/aboutPageConfig.js`
- Create: `web/src/pages/About/aboutPageConfig.test.js`

- [ ] **Step 1: Write failing helper tests**

Create `web/src/pages/About/aboutPageConfig.test.js`:

```js
import test from 'node:test';
import assert from 'node:assert/strict';

import {
  defaultAboutPageConfig,
  isStructuredAboutEnabled,
  normalizeAboutPageConfig,
  parseAboutResponse,
} from './aboutPageConfig.js';

test('parseAboutResponse keeps legacy strings from older API responses', () => {
  assert.deepEqual(parseAboutResponse('# old'), {
    legacy: '# old',
    config: '',
  });
});

test('parseAboutResponse reads the new object response shape', () => {
  assert.deepEqual(
    parseAboutResponse({
      legacy: '# old',
      config: '{"enabled":true}',
    }),
    {
      legacy: '# old',
      config: '{"enabled":true}',
    },
  );
});

test('normalizeAboutPageConfig falls back to default content for empty installs', () => {
  const normalized = normalizeAboutPageConfig('');
  assert.equal(normalized.enabled, true);
  assert.equal(normalized.hero.secondaryActionUrl, 'https://www.digitalchina.com/');
  assert.ok(normalized.capabilities.length >= 4);
  assert.ok(normalized.contacts.length >= 2);
});

test('normalizeAboutPageConfig keeps valid user values and normalizes arrays', () => {
  const normalized = normalizeAboutPageConfig(
    JSON.stringify({
      enabled: true,
      hero: { title: 'Custom title' },
      overview: { metrics: [{ value: '3', label: 'Teams' }] },
      contacts: [{ title: 'Service', imageUrl: 'https://example.com/qr.png' }],
    }),
  );

  assert.equal(normalized.hero.title, 'Custom title');
  assert.equal(normalized.overview.metrics[0].value, '3');
  assert.equal(normalized.contacts[0].imageUrl, 'https://example.com/qr.png');
  assert.equal(normalized.contacts[1].title, defaultAboutPageConfig.contacts[1].title);
});

test('isStructuredAboutEnabled preserves legacy content when config is disabled', () => {
  assert.equal(
    isStructuredAboutEnabled({ enabled: false }, '# legacy'),
    false,
  );
  assert.equal(
    isStructuredAboutEnabled({ enabled: true }, '# legacy'),
    true,
  );
});
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```powershell
Set-Location web
node --test src/pages/About/aboutPageConfig.test.js
```

Expected: FAIL with module not found for `aboutPageConfig.js`.

- [ ] **Step 3: Implement config helpers**

Create `web/src/pages/About/aboutPageConfig.js` with:

```js
export const defaultAboutPageConfig = {
  enabled: true,
  hero: {
    eyebrow: '隶属于神州数码集团 · 企业级 AI 能力入口',
    title: '统一接入、分发与治理企业 AI 能力',
    subtitle:
      '面向企业团队与开发者的一站式 AI API 聚合平台，统一对接主流大模型服务，提供模型路由、用量计量、权限管理、账单分析与稳定性治理。',
    primaryActionText: '进入控制台',
    primaryActionUrl: '/console',
    secondaryActionText: '访问集团官网',
    secondaryActionUrl: 'https://www.digitalchina.com/',
  },
  overview: {
    title: 'AI Gateway Overview',
    description: '统一协议、统一鉴权、统一计费，降低多模型接入复杂度。',
    status: '运行中',
    metrics: [
      { value: '40+', label: '上游模型渠道' },
      { value: '99.9%', label: '服务可用目标' },
      { value: '7x24', label: '支持响应' },
    ],
    channels: [
      { name: 'OpenAI', value: 86, status: '健康' },
      { name: 'Claude', value: 78, status: '健康' },
      { name: 'Gemini', value: 72, status: '健康' },
    ],
  },
  capabilities: [
    {
      icon: 'network',
      title: '多模型聚合',
      description: '统一接入 OpenAI、Claude、Gemini、Azure、AWS Bedrock 等上游能力。',
    },
    {
      icon: 'route',
      title: '统一 API',
      description: '兼容主流调用方式，减少业务侧适配成本，提升模型切换效率。',
    },
    {
      icon: 'shield',
      title: '企业级治理',
      description: '支持账号、分组、权限、限流、审计与安全策略，适配组织化管理。',
    },
    {
      icon: 'chart',
      title: '计量与分析',
      description: '统一记录调用、额度、账单与模型消费，帮助团队做成本透明化管理。',
    },
  ],
  group: {
    title: '神州数码集团企业数字化能力支撑',
    description:
      '平台依托神州数码集团在云、数据、AI 与企业服务领域的长期实践，面向实际业务场景提供可落地、可扩展、可运营的 AI 能力底座。',
    bullets: [
      '服务企业数字化转型与智能化升级',
      '支持多团队、多应用、多模型统一管理',
      '可结合企业合规、安全和成本治理要求配置',
    ],
    websiteLabel: 'Digital China',
    websiteUrl: 'https://www.digitalchina.com/',
  },
  contacts: [
    {
      type: 'wechat',
      title: '微信客服',
      description: '扫码添加平台客服，咨询开通、额度、模型接入与使用问题。',
      imageUrl: '',
      fallbackUrl: '',
    },
    {
      type: 'work_wechat',
      title: '企业微信客服',
      description: '企业用户可通过企业微信联系客户成功团队，获得业务支持与服务响应。',
      imageUrl: '',
      fallbackUrl: '',
    },
  ],
  customContent: '',
};

const normalizeString = (value, fallback = '') =>
  typeof value === 'string' ? value : fallback;

const normalizeNumber = (value, fallback = 0) => {
  const numeric = Number(value);
  if (Number.isFinite(numeric)) {
    return Math.max(0, Math.min(100, numeric));
  }
  return fallback;
};

const normalizeArray = (items, fallbackItems, normalizeItem) => {
  const source = Array.isArray(items) && items.length > 0 ? items : fallbackItems;
  const normalized = source.map((item, index) =>
    normalizeItem(item || {}, fallbackItems[index] || fallbackItems[0]),
  );
  if (normalized.length >= fallbackItems.length) {
    return normalized;
  }
  return [
    ...normalized,
    ...fallbackItems.slice(normalized.length).map((item) => normalizeItem(item, item)),
  ];
};

export const parseAboutResponse = (data) => {
  if (typeof data === 'string') {
    return { legacy: data, config: '' };
  }
  if (data && typeof data === 'object') {
    return {
      legacy: normalizeString(data.legacy),
      config: normalizeString(data.config),
    };
  }
  return { legacy: '', config: '' };
};

export const normalizeAboutPageConfig = (rawConfig) => {
  let parsed = {};
  if (typeof rawConfig === 'string' && rawConfig.trim()) {
    try {
      parsed = JSON.parse(rawConfig);
    } catch (error) {
      parsed = {};
    }
  } else if (rawConfig && typeof rawConfig === 'object') {
    parsed = rawConfig;
  }

  const defaults = defaultAboutPageConfig;
  return {
    enabled: parsed.enabled !== false,
    hero: {
      eyebrow: normalizeString(parsed.hero?.eyebrow, defaults.hero.eyebrow),
      title: normalizeString(parsed.hero?.title, defaults.hero.title),
      subtitle: normalizeString(parsed.hero?.subtitle, defaults.hero.subtitle),
      primaryActionText: normalizeString(
        parsed.hero?.primaryActionText,
        defaults.hero.primaryActionText,
      ),
      primaryActionUrl: normalizeString(
        parsed.hero?.primaryActionUrl,
        defaults.hero.primaryActionUrl,
      ),
      secondaryActionText: normalizeString(
        parsed.hero?.secondaryActionText,
        defaults.hero.secondaryActionText,
      ),
      secondaryActionUrl: normalizeString(
        parsed.hero?.secondaryActionUrl,
        defaults.hero.secondaryActionUrl,
      ),
    },
    overview: {
      title: normalizeString(parsed.overview?.title, defaults.overview.title),
      description: normalizeString(
        parsed.overview?.description,
        defaults.overview.description,
      ),
      status: normalizeString(parsed.overview?.status, defaults.overview.status),
      metrics: normalizeArray(
        parsed.overview?.metrics,
        defaults.overview.metrics,
        (item, fallback) => ({
          value: normalizeString(item.value, fallback.value),
          label: normalizeString(item.label, fallback.label),
        }),
      ),
      channels: normalizeArray(
        parsed.overview?.channels,
        defaults.overview.channels,
        (item, fallback) => ({
          name: normalizeString(item.name, fallback.name),
          value: normalizeNumber(item.value, fallback.value),
          status: normalizeString(item.status, fallback.status),
        }),
      ),
    },
    capabilities: normalizeArray(
      parsed.capabilities,
      defaults.capabilities,
      (item, fallback) => ({
        icon: normalizeString(item.icon, fallback.icon),
        title: normalizeString(item.title, fallback.title),
        description: normalizeString(item.description, fallback.description),
      }),
    ),
    group: {
      title: normalizeString(parsed.group?.title, defaults.group.title),
      description: normalizeString(parsed.group?.description, defaults.group.description),
      bullets: normalizeArray(
        parsed.group?.bullets,
        defaults.group.bullets,
        (item, fallback) => normalizeString(item, fallback),
      ),
      websiteLabel: normalizeString(parsed.group?.websiteLabel, defaults.group.websiteLabel),
      websiteUrl: normalizeString(parsed.group?.websiteUrl, defaults.group.websiteUrl),
    },
    contacts: normalizeArray(parsed.contacts, defaults.contacts, (item, fallback) => ({
      type: normalizeString(item.type, fallback.type),
      title: normalizeString(item.title, fallback.title),
      description: normalizeString(item.description, fallback.description),
      imageUrl: normalizeString(item.imageUrl, fallback.imageUrl),
      fallbackUrl: normalizeString(item.fallbackUrl, fallback.fallbackUrl),
    })),
    customContent: normalizeString(parsed.customContent, defaults.customContent),
  };
};

export const isStructuredAboutEnabled = (config, legacyContent) => {
  if (!config || config.enabled === false) {
    return false;
  }
  return config.enabled === true || !legacyContent;
};
```

- [ ] **Step 4: Run helper tests and commit**

Run:

```powershell
Set-Location web
node --test src/pages/About/aboutPageConfig.test.js
```

Expected: PASS.

Commit:

```powershell
git add web/src/pages/About/aboutPageConfig.js web/src/pages/About/aboutPageConfig.test.js
git commit -m "feat: add about page config helpers"
```

---

## Task 3: Structured About Page Rendering

**Files:**
- Create: `web/src/pages/About/AboutStructuredPage.jsx`
- Create: `web/src/pages/About/about.css`
- Modify: `web/src/pages/About/index.jsx`

- [ ] **Step 1: Add a source-level rendering guard test**

Create `web/src/pages/About/source.test.js` with:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const aboutDir = dirname(fileURLToPath(import.meta.url));

test('structured about page includes responsive breakpoints and interaction states', () => {
  const css = readFileSync(join(aboutDir, 'about.css'), 'utf8');
  assert.match(css, /@media\s*\(max-width:\s*1024px\)/);
  assert.match(css, /@media\s*\(max-width:\s*640px\)/);
  assert.match(css, /\.about-card:hover/);
  assert.match(css, /\.about-card:focus-visible/);
  assert.match(css, /\.about-qr-card:active/);
  assert.match(css, /grid-template-columns:\s*repeat\(4,\s*minmax\(0,\s*1fr\)\)/);
});

test('about page preserves legacy iframe and markdown branches', () => {
  const source = readFileSync(join(aboutDir, 'index.jsx'), 'utf8');
  assert.match(source, /about\.startsWith\('https:\/\/'\)/);
  assert.match(source, /dangerouslySetInnerHTML/);
  assert.match(source, /AboutStructuredPage/);
});
```

- [ ] **Step 2: Run source test and verify it fails**

Run:

```powershell
Set-Location web
node --test src/pages/About/source.test.js
```

Expected: FAIL because `about.css` and `AboutStructuredPage` are not present.

- [ ] **Step 3: Implement `AboutStructuredPage.jsx`**

Create a component with this public shape:

```jsx
import React, { useState } from 'react';
import { marked } from 'marked';
import {
  IconActivity,
  IconBarChartVStroked,
  IconBranch,
  IconShield,
} from '@douyinfe/semi-icons';
import './about.css';

const iconMap = {
  network: <IconBranch />,
  route: <IconActivity />,
  shield: <IconShield />,
  chart: <IconBarChartVStroked />,
};

const SafeLink = ({ href, className, children }) => {
  if (!href) {
    return null;
  }
  const isExternal = href.startsWith('http://') || href.startsWith('https://');
  return (
    <a
      className={className}
      href={href}
      target={isExternal ? '_blank' : undefined}
      rel={isExternal ? 'noopener noreferrer' : undefined}
    >
      {children}
    </a>
  );
};

const QRImage = ({ imageUrl, title }) => {
  const [failed, setFailed] = useState(false);
  if (!imageUrl || failed) {
    return <div className='about-qr-placeholder'>{title}</div>;
  }
  return (
    <img
      className='about-qr-image'
      src={imageUrl}
      alt={title}
      loading='lazy'
      onError={() => setFailed(true)}
    />
  );
};

const AboutStructuredPage = ({ config }) => {
  const customContent = config.customContent?.trim()
    ? marked.parse(config.customContent)
    : '';

  return (
    <main className='about-page'>
      <div className='about-shell'>
        <section className='about-hero'>
          <div className='about-hero-copy'>
            <div className='about-eyebrow'>{config.hero.eyebrow}</div>
            <h1>{config.hero.title}</h1>
            <p>{config.hero.subtitle}</p>
            <div className='about-actions'>
              <SafeLink className='about-button about-button-primary' href={config.hero.primaryActionUrl}>
                {config.hero.primaryActionText}
              </SafeLink>
              <SafeLink className='about-button about-button-secondary' href={config.hero.secondaryActionUrl}>
                {config.hero.secondaryActionText}
              </SafeLink>
            </div>
          </div>
          <aside className='about-overview-panel'>
            <div className='about-overview-card'>
              <div className='about-overview-head'>
                <div>
                  <h2>{config.overview.title}</h2>
                  <p>{config.overview.description}</p>
                </div>
                <span>{config.overview.status}</span>
              </div>
              <div className='about-metrics'>
                {config.overview.metrics.map((metric, index) => (
                  <div className='about-metric' key={`${metric.label}-${index}`}>
                    <strong>{metric.value}</strong>
                    <span>{metric.label}</span>
                  </div>
                ))}
              </div>
              <div className='about-channel-list'>
                {config.overview.channels.map((channel, index) => (
                  <div className='about-channel' key={`${channel.name}-${index}`}>
                    <span>{channel.name}</span>
                    <div className='about-channel-bar'>
                      <i style={{ width: `${channel.value}%` }} />
                    </div>
                    <b>{channel.status}</b>
                  </div>
                ))}
              </div>
            </div>
          </aside>
        </section>

        <section className='about-section'>
          <div className='about-section-title'>
            <h2>为企业 AI 应用提供可控、可观测、可运营的基础设施</h2>
          </div>
          <div className='about-capabilities'>
            {config.capabilities.map((item, index) => (
              <article className='about-card' tabIndex={0} key={`${item.title}-${index}`}>
                <div className='about-card-icon'>{iconMap[item.icon] || iconMap.network}</div>
                <h3>{item.title}</h3>
                <p>{item.description}</p>
              </article>
            ))}
          </div>
        </section>

        <section className='about-group-band'>
          <div>
            <h2>{config.group.title}</h2>
            <p>{config.group.description}</p>
            <div className='about-bullets'>
              {config.group.bullets.map((item, index) => (
                <div key={`${item}-${index}`}>✓ {item}</div>
              ))}
            </div>
          </div>
          <div className='about-official-card'>
            <span>集团官网</span>
            <strong>{config.group.websiteLabel}</strong>
            <SafeLink href={config.group.websiteUrl}>www.digitalchina.com</SafeLink>
          </div>
        </section>

        <section className='about-section'>
          <div className='about-section-title'>
            <h2>联系平台客服</h2>
          </div>
          <div className='about-contact-grid'>
            {config.contacts.map((contact, index) => (
              <article className='about-qr-card' tabIndex={0} key={`${contact.type}-${index}`}>
                <QRImage imageUrl={contact.imageUrl} title={contact.title} />
                <div>
                  <h3>{contact.title}</h3>
                  <p>{contact.description}</p>
                  <SafeLink className='about-contact-link' href={contact.fallbackUrl}>
                    {contact.fallbackUrl}
                  </SafeLink>
                </div>
              </article>
            ))}
          </div>
        </section>

        {customContent && (
          <section
            className='about-custom-content'
            dangerouslySetInnerHTML={{ __html: customContent }}
          />
        )}
      </div>
    </main>
  );
};

export default AboutStructuredPage;
```

During implementation, replace the hard-coded section titles `为企业 AI 应用提供可控、可观测、可运营的基础设施`, `集团官网`, and `联系平台客服` with configurable fields or localized fallback strings if the config shape is extended. Do not leave them as unconfigurable production copy.

- [ ] **Step 4: Implement scoped responsive CSS**

Create `web/src/pages/About/about.css` using the demo as the visual base. Include these required selectors exactly so the source test passes:

```css
.about-page {
  min-height: calc(100vh - 60px);
  margin-top: 60px;
  background:
    linear-gradient(135deg, rgba(31, 98, 255, 0.1), rgba(8, 166, 106, 0.06) 38%, rgba(255, 255, 255, 0) 68%),
    var(--semi-color-bg-0);
  color: var(--semi-color-text-0);
  overflow-x: hidden;
}

.about-shell {
  width: min(1180px, calc(100% - 40px));
  margin: 0 auto;
}

.about-hero {
  padding: 42px 0 34px;
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(360px, 0.92fr);
  gap: 32px;
  align-items: stretch;
}

.about-card,
.about-qr-card,
.about-button,
.about-qr-image {
  transition:
    transform 160ms ease,
    border-color 160ms ease,
    box-shadow 160ms ease,
    background-color 160ms ease;
}

.about-card:hover,
.about-card:focus-visible {
  transform: translateY(-4px);
  border-color: rgba(var(--semi-blue-5), 0.45);
  box-shadow: 0 16px 40px rgba(16, 24, 40, 0.12);
  outline: none;
}

.about-qr-card:active,
.about-button:active {
  transform: translateY(1px);
}

.about-capabilities {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

@media (max-width: 1024px) {
  .about-hero {
    grid-template-columns: 1fr;
  }

  .about-capabilities {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .about-shell {
    width: min(100% - 24px, 1180px);
  }

  .about-capabilities,
  .about-contact-grid,
  .about-metrics {
    grid-template-columns: 1fr;
  }

  .about-qr-card {
    grid-template-columns: 1fr;
  }
}
```

Add the rest of the CSS from the demo with these constraints:

- `.about-contact-grid` is two columns on desktop.
- `.about-qr-image` and `.about-qr-placeholder` use stable `144px` width and height on desktop and do not exceed container width on mobile.
- Text containers use `min-width: 0`, `overflow-wrap: anywhere`, and normal line wrapping.
- Use `@media (hover: hover)` only for hover-only transforms that should not run on touch devices.

- [ ] **Step 5: Wire the renderer into `index.jsx`**

Modify `web/src/pages/About/index.jsx`:

```jsx
import AboutStructuredPage from './AboutStructuredPage';
import {
  isStructuredAboutEnabled,
  normalizeAboutPageConfig,
  parseAboutResponse,
} from './aboutPageConfig';
```

In `displayAbout`, parse the API response:

```jsx
const { legacy, config } = parseAboutResponse(data);
const normalizedConfig = normalizeAboutPageConfig(config);
if (isStructuredAboutEnabled(normalizedConfig, legacy)) {
  setAboutConfig(normalizedConfig);
  setAbout('');
  localStorage.setItem('about_page_config', JSON.stringify(normalizedConfig));
  localStorage.removeItem('about');
  setAboutLoaded(true);
  return;
}
```

Keep the existing legacy branch after this block:

```jsx
let aboutContent = legacy;
if (!legacy.startsWith('https://')) {
  aboutContent = marked.parse(legacy);
}
setAbout(aboutContent);
localStorage.setItem('about', aboutContent);
```

Render structured content first:

```jsx
{aboutConfig ? (
  <AboutStructuredPage config={aboutConfig} />
) : aboutLoaded && about === '' ? (
  renderEmptyAbout()
) : (
  renderLegacyAbout()
)}
```

Define `renderEmptyAbout` by moving the existing `<Empty>` branch into a local function without changing its content. Define `renderLegacyAbout` by moving the existing iframe and `dangerouslySetInnerHTML` branch into a local function without changing legacy behavior.

- [ ] **Step 6: Run page tests and commit**

Run:

```powershell
Set-Location web
node --test src/pages/About/aboutPageConfig.test.js src/pages/About/source.test.js
```

Expected: PASS.

Commit:

```powershell
git add web/src/pages/About
git commit -m "feat: render structured about page"
```

---

## Task 4: Admin Structured About Editor

**Files:**
- Create: `web/src/components/settings/AboutPageSetting.jsx`
- Modify: `web/src/components/settings/OtherSetting.jsx`

- [ ] **Step 1: Add a source-level settings test**

Create `web/src/components/settings/aboutPageSetting.source.test.js`:

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const settingsDir = dirname(fileURLToPath(import.meta.url));

test('about page setting editor persists structured config and legacy about content', () => {
  const source = readFileSync(join(settingsDir, 'AboutPageSetting.jsx'), 'utf8');
  assert.match(source, /AboutPageConfig/);
  assert.match(source, /JSON\.stringify/);
  assert.match(source, /updateOption\('AboutPageConfig'/);
  assert.match(source, /updateOption\('About'/);
  assert.match(source, /微信客服/);
  assert.match(source, /企业微信客服/);
});

test('other settings integrates the structured about editor', () => {
  const source = readFileSync(join(settingsDir, 'OtherSetting.jsx'), 'utf8');
  assert.match(source, /AboutPageSetting/);
  assert.match(source, /AboutPageConfig/);
});
```

- [ ] **Step 2: Run settings source test and verify it fails**

Run:

```powershell
Set-Location web
node --test src/components/settings/aboutPageSetting.source.test.js
```

Expected: FAIL because `AboutPageSetting.jsx` does not exist.

- [ ] **Step 3: Create the settings component**

Create `web/src/components/settings/AboutPageSetting.jsx` with these responsibilities:

- Import `defaultAboutPageConfig` and `normalizeAboutPageConfig` from `../../pages/About/aboutPageConfig`.
- Accept props:

```jsx
const AboutPageSetting = ({
  inputs,
  loadingInput,
  updateOption,
  setInputs,
  setLoadingInput,
  t,
}) => {
  return null;
};
```

Replace the temporary `return null` in the following steps with the state, handlers, and Semi form sections described below.

- Initialize local state from `inputs.AboutPageConfig`:

```jsx
const [config, setConfig] = useState(() =>
  normalizeAboutPageConfig(inputs.AboutPageConfig || ''),
);

useEffect(() => {
  setConfig(normalizeAboutPageConfig(inputs.AboutPageConfig || ''));
}, [inputs.AboutPageConfig]);
```

- Provide these update helpers:

```jsx
const updateHero = (key, value) => {
  setConfig((prev) => ({
    ...prev,
    hero: { ...prev.hero, [key]: value },
  }));
};

const updateOverview = (key, value) => {
  setConfig((prev) => ({
    ...prev,
    overview: { ...prev.overview, [key]: value },
  }));
};

const updateListItem = (section, index, key, value) => {
  setConfig((prev) => ({
    ...prev,
    [section]: prev[section].map((item, itemIndex) =>
      itemIndex === index ? { ...item, [key]: value } : item,
    ),
  }));
};

const updateNestedListItem = (section, listKey, index, key, value) => {
  setConfig((prev) => ({
    ...prev,
    [section]: {
      ...prev[section],
      [listKey]: prev[section][listKey].map((item, itemIndex) =>
        itemIndex === index ? { ...item, [key]: value } : item,
      ),
    },
  }));
};
```

- Save both structured and legacy fields:

```jsx
const submitAboutPageConfig = async () => {
  try {
    setLoadingInput((state) => ({ ...state, AboutPageConfig: true }));
    const value = JSON.stringify(config);
    await updateOption('AboutPageConfig', value);
    setInputs((state) => ({ ...state, AboutPageConfig: value }));
    showSuccess(t('关于页面配置已更新'));
  } catch (error) {
    console.error('AboutPageConfig update failed', error);
    showError(t('关于页面配置更新失败'));
  } finally {
    setLoadingInput((state) => ({ ...state, AboutPageConfig: false }));
  }
};
```

- Keep legacy Markdown/HTML as advanced compatibility:

```jsx
const submitLegacyAbout = async () => {
  try {
    setLoadingInput((state) => ({ ...state, About: true }));
    await updateOption('About', inputs.About);
    showSuccess(t('关于兼容内容已更新'));
  } catch (error) {
    console.error('Legacy About update failed', error);
    showError(t('关于兼容内容更新失败'));
  } finally {
    setLoadingInput((state) => ({ ...state, About: false }));
  }
};
```

- Render Semi form fields in these groups:
  - `关于页面开关`
  - `首屏内容`
  - `平台概览`
  - `能力卡片`
  - `集团背书`
  - `客服二维码`
  - `高级兼容内容`

Do not place cards inside cards. Use `Form.Section`, `Row`, `Col`, `Button`, `Divider`, `Banner`, and plain bordered divs for repeated row groups.

- [ ] **Step 4: Integrate into `OtherSetting.jsx` with localized edits**

Modify imports:

```jsx
import AboutPageSetting from './AboutPageSetting';
```

Add `AboutPageConfig` to `inputs`:

```jsx
About: '',
AboutPageConfig: '',
HomePageContent: '',
```

Add `AboutPageConfig` to `loadingInput`:

```jsx
About: false,
AboutPageConfig: false,
Footer: false,
```

Replace the old About textarea and button with:

```jsx
<AboutPageSetting
  inputs={inputs}
  loadingInput={loadingInput}
  updateOption={updateOption}
  setInputs={setInputs}
  setLoadingInput={setLoadingInput}
  t={t}
/>
```

Keep the existing `submitAbout` function only if it is passed into the new component. If the new component saves legacy content itself, remove `submitAbout` in a separate small patch after verifying the import and JSX compile.

- [ ] **Step 5: Run settings source test and commit**

Run:

```powershell
Set-Location web
node --test src/components/settings/aboutPageSetting.source.test.js
```

Expected: PASS.

Commit:

```powershell
git add web/src/components/settings/AboutPageSetting.jsx web/src/components/settings/OtherSetting.jsx web/src/components/settings/aboutPageSetting.source.test.js
git commit -m "feat: add structured about page settings"
```

---

## Task 5: Internationalization

**Files:**
- Modify: `web/src/i18n/locales/zh-CN.json`
- Modify: `web/src/i18n/locales/zh-TW.json`
- Modify: `web/src/i18n/locales/en.json`
- Modify: `web/src/i18n/locales/fr.json`
- Modify: `web/src/i18n/locales/ru.json`
- Modify: `web/src/i18n/locales/ja.json`
- Modify: `web/src/i18n/locales/vi.json`
- Modify: `web/src/i18n/localeKeys.test.js`

- [ ] **Step 1: Extend locale key test**

In `web/src/i18n/localeKeys.test.js`, add this array and test:

```js
const requiredAboutPageCopyKeys = [
  '关于页面配置',
  '关于页面配置已更新',
  '关于页面配置更新失败',
  '关于兼容内容已更新',
  '关于兼容内容更新失败',
  '首屏内容',
  '平台概览',
  '能力卡片',
  '集团背书',
  '客服二维码',
  '高级兼容内容',
  '微信客服',
  '企业微信客服',
  '二维码图片链接',
  '备用联系链接',
  '保存关于页面配置',
];

test('all locales include structured about page copy keys', () => {
  const localeFiles = readdirSync(localeDir).filter((file) =>
    file.endsWith('.json'),
  );

  assert.ok(localeFiles.length > 0);
  for (const localeFile of localeFiles) {
    const localeFileContent = JSON.parse(
      readFileSync(join(localeDir, localeFile), 'utf8'),
    );
    const locale = localeFileContent.translation || localeFileContent;
    for (const key of requiredAboutPageCopyKeys) {
      assert.equal(
        Object.prototype.hasOwnProperty.call(locale, key),
        true,
        `${localeFile} missing ${key}`,
      );
    }
  }
});
```

- [ ] **Step 2: Run locale test and verify it fails**

Run:

```powershell
Set-Location web
node --test src/i18n/localeKeys.test.js
```

Expected: FAIL listing the missing About page keys.

- [ ] **Step 3: Add locale entries with localized values**

Add the required keys to all seven locale JSON files. Use readable UTF-8 text and minimal localized patches. For `zh-CN.json`, values should match keys:

```json
"关于页面配置": "关于页面配置",
"关于页面配置已更新": "关于页面配置已更新",
"关于页面配置更新失败": "关于页面配置更新失败",
"关于兼容内容已更新": "关于兼容内容已更新",
"关于兼容内容更新失败": "关于兼容内容更新失败",
"首屏内容": "首屏内容",
"平台概览": "平台概览",
"能力卡片": "能力卡片",
"集团背书": "集团背书",
"客服二维码": "客服二维码",
"高级兼容内容": "高级兼容内容",
"微信客服": "微信客服",
"企业微信客服": "企业微信客服",
"二维码图片链接": "二维码图片链接",
"备用联系链接": "备用联系链接",
"保存关于页面配置": "保存关于页面配置"
```

For `en.json`, use:

```json
"关于页面配置": "About page configuration",
"关于页面配置已更新": "About page configuration updated",
"关于页面配置更新失败": "Failed to update About page configuration",
"关于兼容内容已更新": "Legacy About content updated",
"关于兼容内容更新失败": "Failed to update legacy About content",
"首屏内容": "Hero content",
"平台概览": "Platform overview",
"能力卡片": "Capability cards",
"集团背书": "Group backing",
"客服二维码": "Customer service QR codes",
"高级兼容内容": "Advanced compatibility content",
"微信客服": "WeChat support",
"企业微信客服": "Enterprise WeChat support",
"二维码图片链接": "QR code image URL",
"备用联系链接": "Fallback contact link",
"保存关于页面配置": "Save About page configuration"
```

For non-Chinese locales, provide direct translations or clear English fallback if a localized translation is uncertain. Do not leave keys missing.

- [ ] **Step 4: Run i18n tests and commit**

Run:

```powershell
Set-Location web
node --test src/i18n/localeKeys.test.js
bun run i18n:lint
```

Expected: PASS for the Node test. `bun run i18n:lint` should complete without missing-key errors.

Commit:

```powershell
git add web/src/i18n/localeKeys.test.js web/src/i18n/locales
git commit -m "feat: add about page i18n copy"
```

---

## Task 6: Full Verification And Mobile Review

**Files:**
- Modify only if verification finds a defect in files touched by Tasks 1-5.

- [ ] **Step 1: Run backend tests**

Run:

```powershell
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run frontend targeted tests**

Run:

```powershell
Set-Location web
node --test src/pages/About/aboutPageConfig.test.js src/pages/About/source.test.js src/components/settings/aboutPageSetting.source.test.js src/i18n/localeKeys.test.js
```

Expected: PASS.

- [ ] **Step 3: Build frontend**

Run:

```powershell
Set-Location web
bun run build
```

Expected: build completes successfully and Vite emits production assets.

- [ ] **Step 4: Start local frontend preview for visual review**

Run:

```powershell
Set-Location web
bun run dev -- --host 127.0.0.1
```

Expected: Vite prints a local URL such as `http://127.0.0.1:5173/`.

- [ ] **Step 5: Manual responsive checks**

Open `/about` and verify:

- Desktop width around `1440px`: hero is two columns, cards are four columns, QR cards are two columns.
- Tablet width around `768px`: hero stacks cleanly, cards are two columns, no horizontal scrolling appears.
- Mobile width around `390px`: all sections are one column, QR images fit, buttons wrap cleanly, text is not clipped.
- Hover-capable device: cards and QR images lift subtly on hover.
- Keyboard navigation: cards and buttons show visible focus states.
- Touch simulation: active feedback appears without hover-only dependency.
- Empty QR image URL: placeholder renders and card text remains readable.
- Broken QR image URL: placeholder replaces the failed image.

- [ ] **Step 6: Commit verification fixes**

If verification required changes, run targeted tests again and commit:

```powershell
git add web/src/pages/About web/src/components/settings web/src/i18n controller model service
git commit -m "fix: polish about page responsive behavior"
```

If verification required no changes, do not create an empty commit.

---

## Self-Review

- Spec coverage:
  - Structured config is covered by Tasks 1, 2, and 4.
  - Modern page rendering is covered by Task 3.
  - WeChat and Enterprise WeChat QR cards are covered by Tasks 3 and 4.
  - Hover, focus, active, image failure, and mobile behavior are covered by Tasks 3 and 6.
  - Legacy About compatibility is covered by Tasks 1, 2, and 3.
  - i18n is covered by Task 5.

- Placeholder scan:
  - The plan contains no `TBD`, `TODO`, or undefined implementation phases.
  - Every test command includes an expected result.

- Type consistency:
  - Backend option key is consistently `AboutPageConfig`.
  - Frontend config helpers consistently use `legacy` and `config`.
  - Contact items consistently use `type`, `title`, `description`, `imageUrl`, and `fallbackUrl`.
