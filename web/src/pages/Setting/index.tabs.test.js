import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

test('system settings page does not include model monitor tab', () => {
  const source = readFileSync(resolve(__dirname, 'index.jsx'), 'utf8');

  assert.equal(source.includes("itemKey: 'model-monitor'"), false);
  assert.equal(
    source.includes("import ModelMonitorCenter from './ModelMonitor/ModelMonitorCenter'"),
    false,
  );
});

test('standalone model monitor console route remains registered', () => {
  const appSource = readFileSync(resolve(__dirname, '../../App.jsx'), 'utf8');

  assert.equal(appSource.includes("path='/console/model-monitor'"), true);
});
