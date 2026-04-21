# Console AI Model Docs Center Design

Date: 2026-04-21

## Goal

Add an in-app documentation center for AI model APIs inside the current system.
The first release is a lightweight skeleton, not a full external-docs replacement.

The page must:

- live inside the current console experience
- require login
- use static frontend configuration as the content source
- cover only the `AI model API` category in the first release
- provide a left navigation tree and a right detail panel

## Confirmed Scope

This design is based on the confirmed constraints from the review conversation:

- The docs center is an in-site page, not an external redirect.
- Access is login-required and should reuse existing console guards.
- Content is frontend static configuration in the first phase.
- Only the `AI model API` category is included now.
- The first phase is a lightweight skeleton instead of a fully interactive API explorer.

## Product Intent

The page should give logged-in users a stable place to browse supported AI API formats without leaving the system.
It is not meant to replicate every capability from the external reference site yet.

The first release should optimize for:

- discoverability
- clear information hierarchy
- clean route structure
- easy future migration from static config to backend-driven content

## Entry And Access

The docs center should be added as a new console entry using the existing API documentation label in the current i18n/navigation system.

Access rules:

- The page lives under `/console`.
- It is guarded by the same login-required route strategy used by other console pages.
- Unauthenticated users should be redirected by the existing `PrivateRoute` flow.

Recommended top-level entry:

- `/console/docs`

Recommended first content route family:

- `/console/docs/ai-model/:docId`

## Route Design

The route design should remain simple in phase one:

1. `/console/docs`
   Redirect to the default first AI model API document.
2. `/console/docs/ai-model/:docId`
   Render the docs workspace with the left tree and the selected detail page.

This avoids a blank landing page and makes every document directly linkable.

## Page Architecture

The page should reuse the existing global app shell:

- top header stays unchanged
- global console sidebar stays unchanged
- the docs page is rendered inside the existing console content area

Inside the docs page itself, add a dedicated docs workspace:

1. Local page header
   Show the page title and a short explanatory sentence.
2. Docs navigation panel
   Show the `AI model API` document tree.
3. Docs detail panel
   Show the selected document content.

This keeps the docs center visually integrated with the current system while still providing a documentation-style reading layout.

## Information Architecture

Phase one includes only the `AI model API` category.
The left navigation tree should be organized by group, then by document item.

Recommended groups for the first release:

- Audio
- Chat
- Completions
- Embeddings
- Images
- Models
- Moderations
- Realtime
- Rerank
- Unimplemented
- Videos

Each group contains document items such as:

- native Gemini format
- Gemini text chat
- ChatCompletions format
- Responses format
- image edit
- image generation
- model list

The exact item list should come from the static config source and can mirror the provided reference menu for the `AI model API` category only.

## Static Content Model

Use one frontend static source of truth to drive both the left menu and the right detail view.
Do not maintain menu definitions and page content in separate structures.

Recommended structure:

```js
{
  groups: [
    {
      key: 'chat',
      title: 'Chat',
    },
  ],
  docs: [
    {
      id: 'gemini-text-chat',
      groupKey: 'chat',
      title: 'Gemini Text Chat',
      method: 'POST',
      path: '/v1beta/models/{model}:generateContent',
      summary: 'Short one-line purpose statement',
      description: 'Longer overview for the lightweight detail panel',
      auth: {
        type: 'bearer',
        location: 'header',
        example: 'Authorization: Bearer sk-xxxxxx',
      },
      requestExample: 'curl ...',
      responseExample: '{ ... }',
    },
  ],
}
```

Required fields for phase one:

- `id`
- `groupKey`
- `title`
- `method`
- `path`
- `summary`
- `description`
- `auth`
- `requestExample`
- `responseExample`

Optional fields that should not be required in phase one:

- `status`
- `tags`
- `requestFields`
- `responseFields`
- `languageExamples`

## Detail Page Template

Each document detail page should use the same lightweight template.

Recommended sections:

1. Breadcrumb
   Example: `API docs / AI model API / Chat`
2. Title row
   Show the document title and HTTP method badge.
3. Summary block
   Short statement of what the endpoint is for.
4. Request path block
   Show the path clearly in monospace style.
5. Authorization block
   Explain bearer token usage in simple terms.
6. Request example block
   Phase one only needs `cURL`.
7. Response example block
   Show a representative JSON example.

If a document is listed but not fully written yet, render a clear placeholder message instead of leaving the detail area empty.

## Interaction Design

Desktop behavior:

- The left docs tree stays visible beside the detail panel.
- Selecting a doc updates the route and the detail content.
- The selected item is highlighted.
- The parent group of the selected item is expanded automatically.

Mobile behavior:

- The left docs tree should not stay permanently visible.
- Replace the docs tree with a drawer triggered from the page header.
- Selecting an item closes the drawer and updates the detail panel.

Default state:

- Visiting `/console/docs` should redirect to the first configured document.
- There should be no empty default detail page.

Error and empty states:

- unknown `docId` should redirect to the default first document or a controlled not-found state inside the docs page
- missing content should show a placeholder panel

## Visual Direction

Do not copy the external docs site one-to-one.
This page should look like a documentation workspace inside the current console.

Visual guidance:

- keep the existing top header and console shell
- use a white or neutral content workspace inside the current page area
- use clear card sections for each document block
- use method badges with obvious color distinction such as blue for `POST` and green for `GET`
- keep spacing and typography readable, but avoid building a new standalone site theme

## Implementation Boundaries

Phase one should include:

- new console route
- new console sidebar entry
- local docs page layout
- static config source for `AI model API`
- left docs tree
- right detail template
- mobile drawer behavior for the local docs tree

Phase one should explicitly exclude:

- full-text search
- online request debugging
- multi-language code tabs
- backend-managed document content
- automatic OpenAPI generation
- editing tools in the UI

## Testing Strategy

The first release should verify the smallest complete loop.

Required checks:

1. Route behavior
   `/console/docs` resolves to the default first document.
2. Access control
   Unauthenticated users are still blocked by the existing login guard.
3. Static config mapping
   The left groups and document items render correctly from the config source.
4. Selection behavior
   Clicking a document item updates the route and the visible detail content.
5. Mobile behavior
   The local docs menu drawer can open and close correctly.

Recommended implementation-time tests:

- source-level tests for static config mapping helpers
- route-level tests for default redirection behavior
- lightweight rendering tests for detail template sections

## Future Evolution

This phase should leave a clean upgrade path for:

- backend-driven document content
- code language tabs
- request and response field tables
- search and filtering
- additional top-level documentation categories beyond `AI model API`

The most important rule is to keep the content contract stable so a later API response can replace the local static config without rewriting the page architecture.

## Success Criteria

The design is successful when:

- logged-in users can open an in-app docs center from the console
- the first release clearly presents the `AI model API` category
- the page uses a docs-style left tree plus right content layout
- routes are stable and directly linkable
- the implementation can start with static data without blocking future backend integration
