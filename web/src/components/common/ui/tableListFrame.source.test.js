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
  assert.match(tokensTableSource, /<CardTable[\s\S]*className=['"][^'"]*\bgrid-bordered-table\b[^'"]*['"]/);
  assert.match(channelsTableSource, /<CardTable[\s\S]*className=['"][^'"]*\bgrid-bordered-table\b[^'"]*['"]/);
  assert.match(quotaLedgerSource, /<Table[\s\S]*className=['"][^'"]*\bgrid-bordered-table\b[^'"]*['"]/);
  assert.doesNotMatch(quotaLedgerSource, /<Table[\s\S]*className=['"][^'"]*\bquota-ledger-table\b[^'"]*['"]/);
  assert.doesNotMatch(quotaLedgerSource, /quota-ledger-debug-table/);
});

test('CardTable desktop bordered path opts into grid-bordered-table by default', () => {
  assert.match(cardTableSource, /const \{ bordered = true, \.\.\.desktopTableProps \} = finalTableProps;/);
  assert.match(
    cardTableSource,
    /<Table[\s\S]*\bclassName=(?:\{[\s\S]*\bgrid-bordered-table\b[\s\S]*\}|['"][^'"]*\bgrid-bordered-table\b[^'"]*['"])/,
  );
});

test('shared table border style uses the softened list contract without debug hooks', () => {
  assert.doesNotMatch(tokensTableSource, /table-list-frame/);
  assert.doesNotMatch(channelsTableSource, /table-list-frame/);
  assert.doesNotMatch(quotaLedgerSource, /table-list-frame/);
  assert.doesNotMatch(indexCssSource, /\.table-list-frame/);
  assert.doesNotMatch(indexCssSource, /\.quota-ledger-debug-table/);
  assert.doesNotMatch(indexCssSource, /(?:^|\r?\n)\.semi-table-wrapper \.semi-table-container\s*\{/);
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper \.semi-table-container\s*\{[\s\S]*border:\s*0;[\s\S]*background:\s*transparent;[\s\S]*border-radius:\s*16px;/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper \.semi-table-container > \.semi-table-header,[\s\S]*background:\s*transparent;/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-head[\s\S]*border-right:\s*0 !important;[\s\S]*border-bottom:\s*1px solid #EDEDEE !important;/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-cell[\s\S]*border-right:\s*0 !important;[\s\S]*border-bottom:\s*1px solid #EDEDEE !important;/,
  );
  assert.match(
    indexCssSource,
    /html\.dark \.grid-bordered-table\.semi-table-wrapper \.semi-table-container\s*\{[\s\S]*border:\s*1px solid rgba\(255,\s*255,\s*255,\s*0\.06\);[\s\S]*background:\s*rgba\(24,\s*27,\s*32,\s*0\.88\);/,
  );
  assert.match(
    indexCssSource,
    /html\.dark \.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-head[\s\S]*border-bottom:\s*1px solid #34353A !important;/,
  );
  assert.match(
    indexCssSource,
    /html\.dark \.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-cell[\s\S]*border-bottom:\s*1px solid #34353A !important;/,
  );
  assert.match(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row:hover[\s\S]*> \.semi-table-row-cell\s*\{[\s\S]*background:\s*rgba\(15,\s*23,\s*42,\s*0\.03\);/,
  );
  assert.match(
    indexCssSource,
    /html\.dark \.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row:hover[\s\S]*> \.semi-table-row-cell\s*\{[\s\S]*background:\s*rgba\(255,\s*255,\s*255,\s*0\.04\);/,
  );
  assert.doesNotMatch(
    indexCssSource,
    /\.grid-bordered-table\.semi-table-wrapper[\s\S]*> \.semi-table-row-(?:head|cell)[\s\S]*border-right:\s*1px solid/,
  );
});
