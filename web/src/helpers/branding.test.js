import test from 'node:test';
import assert from 'node:assert/strict';

import {
  applyDocumentBranding,
  resolveBrandingState,
} from './branding.js';

test('resolveBrandingState adds a build version to the default logo fallback', () => {
  assert.equal(
    resolveBrandingState({ assetVersion: '20260413' }).logo,
    '/logo.png?v=20260413',
  );
});

test('resolveBrandingState prefers status branding over the cached fallback', () => {
  const branding = resolveBrandingState({
    storedSystemName: 'Cached',
    storedLogo: '/logo.png',
    statusData: {
      system_name: 'Server',
      logo: '/brand/logo.png',
    },
    assetVersion: '20260413',
  });

  assert.equal(branding.systemName, 'Server');
  assert.equal(branding.logo, '/brand/logo.png?v=20260413');
});

test('applyDocumentBranding updates the favicon href after status data arrives', () => {
  const linkElement = { href: '/logo.png' };
  const doc = {
    title: 'Old title',
    querySelector(selector) {
      assert.equal(selector, "link[rel~='icon']");
      return linkElement;
    },
  };

  applyDocumentBranding(doc, {
    systemName: 'Updated title',
    logo: '/logo.png?v=20260413',
  });

  assert.equal(doc.title, 'Updated title');
  assert.equal(linkElement.href, '/logo.png?v=20260413');
});
