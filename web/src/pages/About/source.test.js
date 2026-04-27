import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const readSource = (fileName) => {
  const filePath = path.join(__dirname, fileName);

  return fs.existsSync(filePath) ? fs.readFileSync(filePath, 'utf8') : '';
};

test('about CSS defines responsive breakpoints and interactive states', () => {
  const source = readSource('about.css');

  assert.match(source, /@media\s*\(\s*max-width:\s*1024px\s*\)/);
  assert.match(source, /@media\s*\(\s*max-width:\s*640px\s*\)/);
  assert.match(
    source,
    /\.about-page \.about-capability-grid[\s\S]*grid-template-columns:\s*repeat\(4,\s*minmax\(0,\s*1fr\)\)/,
  );
  assert.match(source, /\.about-page \.about-hero/);
  assert.match(source, /\.about-page \.about-contact-grid/);
  assert.match(source, /\.about-page \.about-card:hover/);
  assert.match(source, /\.about-page \.about-action:focus-visible/);
  assert.match(source, /\.about-page \.about-qr-card:active/);
  assert.doesNotMatch(source, /^\s*\.about-(?!page(?:[\s,{:#.*>+~]|$))/m);
});

test('about CSS lets long status and channel text wrap', () => {
  const source = readSource('about.css');

  assert.doesNotMatch(source, /white-space:\s*nowrap/);
  assert.match(
    source,
    /\.about-page \.about-status\s*\{[\s\S]*min-width:\s*0[\s\S]*overflow-wrap:\s*anywhere[\s\S]*white-space:\s*normal/,
  );
  assert.match(
    source,
    /\.about-page \.about-channel-label\s*\{[\s\S]*min-width:\s*0[\s\S]*overflow-wrap:\s*anywhere/,
  );
  assert.match(
    source,
    /\.about-page \.about-channel-label span\s*\{[\s\S]*min-width:\s*0[\s\S]*overflow-wrap:\s*anywhere/,
  );
});

test('about CSS lets configured action and group bullet text wrap', () => {
  const source = readSource('about.css');

  assert.match(
    source,
    /\.about-page \.about-action span\s*\{[\s\S]*min-width:\s*0[\s\S]*overflow-wrap:\s*anywhere[\s\S]*white-space:\s*normal[\s\S]*line-height:/,
  );
  assert.match(
    source,
    /\.about-page \.about-group-list span\s*\{[\s\S]*min-width:\s*0[\s\S]*overflow-wrap:\s*anywhere[\s\S]*white-space:\s*normal[\s\S]*line-height:/,
  );
});

test('about CSS lets configured eyebrow and group status text wrap', () => {
  const source = readSource('about.css');

  assert.match(
    source,
    /\.about-page \.about-eyebrow\s*\{[\s\S]*min-width:\s*0[\s\S]*overflow-wrap:\s*anywhere[\s\S]*white-space:\s*normal[\s\S]*line-height:/,
  );
  assert.match(
    source,
    /\.about-page \.about-group-status\s*\{[\s\S]*min-width:\s*0[\s\S]*overflow-wrap:\s*anywhere[\s\S]*white-space:\s*normal[\s\S]*line-height:/,
  );
});

test('about CSS follows the global dark theme class', () => {
  const source = readSource('about.css');

  assert.match(
    source,
    /html\.dark \.about-page\s*\{[\s\S]*--about-text:[\s\S]*--about-panel:[\s\S]*background-color:/,
  );
  assert.match(
    source,
    /html\.dark \.about-page \.about-card,\s*html\.dark \.about-page \.about-qr-card\s*\{/,
  );
  assert.match(source, /html\.dark \.about-page \.about-action-secondary/);
  assert.match(source, /html\.dark \.about-page \.about-attribution/);
});

test('about index wires structured page while preserving legacy render paths', () => {
  const source = readSource('index.jsx');

  assert.match(source, /AboutStructuredPage/);
  assert.match(source, /parseAboutResponse/);
  assert.match(source, /normalizeAboutPageConfig/);
  assert.match(source, /isStructuredAboutEnabled/);
  assert.match(source, /about\.startsWith\('https:\/\/'\)/);
  assert.match(source, /dangerouslySetInnerHTML/);
});

test('about index clears cached structured config when the API request fails', () => {
  const source = readSource('index.jsx');

  assert.match(
    source,
    /else\s*\{[\s\S]*setAboutConfig\(null\)[\s\S]*removeLocalStorage\(ABOUT_CONFIG_CACHE_KEY\)/,
  );
});

test('about index clears cached structured config when the API request rejects', () => {
  const source = readSource('index.jsx');

  assert.match(
    source,
    /try\s*\{[\s\S]*API\.get\('\/api\/about'\)[\s\S]*\}\s*catch\s*(?:\([^)]*\))?\s*\{[\s\S]*showError[\s\S]*setAbout\(t\([\s\S]*setAboutConfig\(null\)[\s\S]*removeLocalStorage\(ABOUT_CONFIG_CACHE_KEY\)/,
  );
});

test('structured about page lazily loads QR images and handles broken images', () => {
  const source = readSource('AboutStructuredPage.jsx');

  assert.match(source, /<img[\s\S]*loading=['"]lazy['"]/);
  assert.match(source, /onError=\{/);
  assert.doesNotMatch(
    source,
    /const\s+\{[^}]*fallbackUrl[^}]*}\s*=\s*contact[\s\S]*imageSources/,
  );
});

test('structured about page renders contact fallback URL as a safe link', () => {
  const source = readSource('AboutStructuredPage.jsx');

  assert.match(source, /const SafeLink/);
  assert.match(source, /href=\{contact\.fallbackUrl\}/);
  assert.match(source, /about-contact-fallback/);
});

test('structured about display cards are not keyboard tab stops', () => {
  const source = readSource('AboutStructuredPage.jsx');

  assert.doesNotMatch(source, /tabIndex=\{0\}/);
});

test('structured about page guards optional aria-labelledby headings', () => {
  const source = readSource('AboutStructuredPage.jsx');

  assert.doesNotMatch(
    source,
    /className='about-hero'\s+aria-labelledby='about-hero-title'/,
  );
  assert.doesNotMatch(
    source,
    /className='about-group-section'\s+aria-labelledby='about-group-title'/,
  );
  assert.match(
    source,
    /hasText\(hero\.title\)[\s\S]*'aria-labelledby': 'about-hero-title'/,
  );
  assert.match(
    source,
    /hasText\(group\.title\)[\s\S]*'aria-labelledby': 'about-group-title'/,
  );
});

test('structured about page translates configured default display copy', () => {
  const source = readSource('AboutStructuredPage.jsx');

  assert.match(source, /translateAboutPageConfig/);
  assert.match(
    source,
    /useMemo\([\s\S]*translateAboutPageConfig\(config,\s*t\)/,
  );
});

test('structured about page renders protected project attribution content', () => {
  const source = readSource('AboutStructuredPage.jsx');
  const indexSource = readSource('index.jsx');

  assert.match(source, /protectedAttribution/);
  assert.match(source, /about-attribution/);
  assert.match(indexSource, /protectedAttribution=\{customDescription\}/);
});
