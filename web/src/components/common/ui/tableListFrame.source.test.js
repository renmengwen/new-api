import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const projectRoot = process.cwd();
const tokensTableSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/components/table/tokens/TokensTable.jsx'),
  'utf8',
);
const channelsTableSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/components/table/channels/ChannelsTable.jsx'),
  'utf8',
);
const cardTableSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/components/common/ui/CardTable.jsx'),
  'utf8',
);
const quotaLedgerSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/pages/AdminQuotaLedgerPageV2/index.jsx'),
  'utf8',
);
const indexCssSource = fs.readFileSync(
  path.join(projectRoot, 'web/src/index.css'),
  'utf8',
);

test('token, channel and quota ledger tables keep native bordered props without debug classes', () => {
  assert.match(cardTableSource, /const \{ bordered = true, \.\.\.desktopTableProps \} = finalTableProps;/);
  assert.match(cardTableSource, /<Table[\s\S]*bordered=\{bordered\}/);
  assert.match(tokensTableSource, /<CardTable[\s\S]*\bbordered\b/);
  assert.match(channelsTableSource, /<CardTable[\s\S]*\bbordered\b/);
  assert.match(quotaLedgerSource, /<Table[\s\S]*\bbordered\b/);
  assert.ok(tokensTableSource.includes("className='grid-bordered-table rounded-xl overflow-hidden'"));
  assert.ok(channelsTableSource.includes("className='grid-bordered-table rounded-xl overflow-hidden'"));
  assert.match(quotaLedgerSource, /<Table[\s\S]*className='grid-bordered-table'/);
  assert.doesNotMatch(quotaLedgerSource, /quota-ledger-debug-table/);
});

test('shared table border style uses the unified #34353A color without debug hooks', () => {
  assert.doesNotMatch(tokensTableSource, /table-list-frame/);
  assert.doesNotMatch(channelsTableSource, /table-list-frame/);
  assert.doesNotMatch(quotaLedgerSource, /table-list-frame/);
  assert.doesNotMatch(indexCssSource, /\.table-list-frame/);
  assert.doesNotMatch(indexCssSource, /\.quota-ledger-debug-table/);
  assert.match(indexCssSource, /\.semi-table-wrapper \.semi-table-container\s*\{[\s\S]*border:\s*1px solid #34353A;/);
  assert.match(indexCssSource, /\.semi-table-wrapper \.semi-table-container > \.semi-table-header\s*\{/);
  assert.match(indexCssSource, /\.semi-table-wrapper \.semi-table-container > \.semi-table-body[\s\S]*> \.semi-table-thead/);
  assert.match(indexCssSource, /\.semi-table-wrapper \.semi-table-container > \.semi-table-body/);
  assert.match(indexCssSource, /\.semi-table-wrapper \.semi-table-container > \.semi-table-header/);
});
