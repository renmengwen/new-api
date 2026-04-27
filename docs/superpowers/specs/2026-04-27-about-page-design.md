# About Page Design

## Goal

Build a polished, modern About page for the platform at `/about`. The page presents the product as the Digital China Group AI API aggregation platform, supports internationalization, and exposes all visible content through admin configuration.

The implementation must keep the existing route and navigation behavior. It should upgrade the current single-field About content into structured content while preserving backward compatibility with the existing `About` option.

## User-Approved Direction

Use the enterprise product-page style from the demo:

- Calm enterprise visual tone, aligned with a B2B AI infrastructure platform.
- First viewport identifies the platform clearly.
- Content emphasizes unified AI API access, model aggregation, governance, billing, and observability.
- The Digital China Group website link defaults to `https://www.digitalchina.com/`.
- Customer service contact area includes WeChat and Enterprise WeChat QR codes.
- Images, QR cards, capability cards, and buttons should include subtle transitions and hover interactions.

## Page Structure

The page renders these sections in order:

1. Hero
   - Badge or eyebrow text.
   - Main title.
   - Subtitle/description.
   - Primary action, usually console entry.
   - Secondary action, usually official group website.
   - Right-side platform overview panel with configurable status text, metrics, and channel rows.

2. Capability Cards
   - Four default cards:
     - Multi-model aggregation.
     - Unified API.
     - Enterprise governance.
     - Metering and analytics.
   - Card titles, descriptions, and icon identifiers are configurable.
   - Cards use hover elevation, border-color transition, and a small icon movement.

3. Group Backing
   - Configurable title and body copy.
   - Configurable bullet list.
   - Configurable official website title and URL.

4. Contact
   - WeChat service QR card.
   - Enterprise WeChat service QR card.
   - Each QR card has configurable title, description, image URL, and fallback link.
   - QR image hover can slightly lift/scale while preserving layout stability.
   - Missing image URL renders a clean placeholder state instead of a broken image.

5. Optional Custom Content
   - Keep a Markdown/HTML custom content field as an advanced supplement.
   - It is appended below the structured page when configured.
   - If the new structured config is absent, the existing legacy About renderer continues to work.

## Configuration Model

Add a new option key, `AboutPageConfig`, stored as JSON text through the existing options table. The backend returns this config through `/api/about`, together with the existing legacy `About` value.

Recommended shape:

```json
{
  "enabled": true,
  "hero": {
    "eyebrow": "隶属于神州数码集团 · 企业级 AI 能力入口",
    "title": "统一接入、分发与治理企业 AI 能力",
    "subtitle": "面向企业团队与开发者的一站式 AI API 聚合平台...",
    "primaryActionText": "进入控制台",
    "primaryActionUrl": "/console",
    "secondaryActionText": "访问集团官网",
    "secondaryActionUrl": "https://www.digitalchina.com/"
  },
  "overview": {
    "title": "AI Gateway Overview",
    "description": "统一协议、统一鉴权、统一计费，降低多模型接入复杂度。",
    "status": "运行中",
    "metrics": [
      { "value": "40+", "label": "上游模型渠道" }
    ],
    "channels": [
      { "name": "OpenAI", "value": 86, "status": "健康" }
    ]
  },
  "capabilities": [
    { "icon": "network", "title": "多模型聚合", "description": "..." }
  ],
  "group": {
    "title": "神州数码集团企业数字化能力支撑",
    "description": "...",
    "bullets": ["服务企业数字化转型与智能化升级"],
    "websiteLabel": "Digital China",
    "websiteUrl": "https://www.digitalchina.com/"
  },
  "contacts": [
    {
      "type": "wechat",
      "title": "微信客服",
      "description": "扫码添加平台客服...",
      "imageUrl": "",
      "fallbackUrl": ""
    }
  ],
  "customContent": ""
}
```

The frontend should normalize missing or malformed arrays to safe defaults. Admin-entered text is treated as data and rendered through React, except `customContent`, which keeps the existing Markdown/HTML behavior.

## Admin Editing

In `OtherSetting`, replace the single About textarea with a structured form:

- Basic fields for hero and action links.
- Editable metric rows.
- Editable capability rows.
- Editable group backing copy and bullet rows.
- Two contact QR sections for WeChat and Enterprise WeChat.
- Advanced custom Markdown/HTML supplement.
- A legacy compatibility note for existing `About` content.

The form should avoid full-file rewrites and preserve readable Chinese text. The UI can use Semi `Card`, `Form`, `Input`, `TextArea`, `ArrayField`-style local state, and buttons consistent with existing settings pages.

## Internationalization

Frontend visible fallback strings use `t('中文key')`. Configured content is already user-provided copy and should render as configured. The default seed content should be present in zh-CN and translated in en, zh-TW, fr, ru, ja, and vi locale files only for UI labels and fallback states.

Admin form labels and validation messages must be added to i18n resources.

## Error Handling

- If `/api/about` fails, show the existing localized error state.
- If structured config JSON is invalid, fall back to legacy `About`.
- If both structured config and legacy `About` are empty, show a polished empty state.
- If QR image fails to load, replace it with a placeholder block and keep the contact card visible.
- External links open in a new tab with safe `rel` attributes.

## Compatibility

- Existing `About` Markdown/HTML and `https://` iframe behavior must keep working when `AboutPageConfig` is absent or disabled.
- Database changes must use the existing options table and GORM-backed option update flow. No migration table is required.
- JSON marshal/unmarshal in Go must use wrappers from `common/json.go`.
- Do not modify protected project identifiers or existing license headers.

## Verification

Run at least:

- `go test ./...` if backend code changes.
- `bun run build` in `web/` for frontend build validation.
- `bun run i18n:lint` or the closest available i18n validation if locale files change.
- Manual browser review for desktop and mobile widths, including hover/focus behavior and missing QR image placeholders.
