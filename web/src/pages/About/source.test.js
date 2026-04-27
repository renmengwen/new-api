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
    /\.about-capability-grid[\s\S]*grid-template-columns:\s*repeat\(4,\s*minmax\(0,\s*1fr\)\)/,
  );
  assert.match(source, /\.about-card:hover/);
  assert.match(source, /\.about-card:focus-visible/);
  assert.match(source, /\.about-qr-card:active/);
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

test('structured about page lazily loads QR images and handles broken images', () => {
  const source = readSource('AboutStructuredPage.jsx');

  assert.match(source, /<img[\s\S]*loading=['"]lazy['"]/);
  assert.match(source, /onError=\{/);
  assert.match(source, /fallbackUrl/);
});
