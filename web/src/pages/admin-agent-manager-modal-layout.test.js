import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const readPageSource = (relativePath) =>
  fs.readFileSync(path.join(process.cwd(), 'web/src/pages', relativePath), 'utf8');

test('agent and manager modals use merged section layouts', () => {
  const agentsSource = readPageSource('AdminAgentsPageV2/index.jsx');
  const managersSource = readPageSource('AdminManagersPageV2/index.jsx');

  for (const source of [agentsSource, managersSource]) {
    assert.ok(source.includes('const mergedSectionStyle = {'));
    assert.ok(source.includes('const sectionBlockStyle = {'));
    assert.ok(source.includes('<div style={mergedSectionStyle}>'));
  }
});

test('agent and manager detail descriptions use localized keys', () => {
  const agentsSource = readPageSource('AdminAgentsPageV2/index.jsx');
  const managersSource = readPageSource('AdminManagersPageV2/index.jsx');

  assert.ok(agentsSource.includes("{ key: t('显示名称'), value: detailData.display_name || '-' }"));
  assert.ok(agentsSource.includes("{ key: t('代理商名称'), value: detailData.agent_name || '-' }"));
  assert.ok(agentsSource.includes("{ key: t('公司名称'), value: detailData.company_name || '-' }"));
  assert.ok(agentsSource.includes("{ key: t('联系电话'), value: detailData.contact_phone || '-' }"));
  assert.ok(managersSource.includes("{ key: t('登录用户名'), value: detailData.username || '-' }"));
  assert.ok(managersSource.includes("{ key: t('显示名称'), value: detailData.display_name || '-' }"));
  assert.ok(managersSource.includes("key: t('最后活跃')"));
});

test('agent create modal keeps required agent name and token group fields visible', () => {
  const agentsSource = readPageSource('AdminAgentsPageV2/index.jsx');

  assert.doesNotMatch(
    agentsSource,
    /display: 'none'[\s\S]*<Text type='tertiary'>\{t\('代理商名称'\)\}<\/Text>/,
  );
  assert.doesNotMatch(
    agentsSource,
    /display: 'none'[\s\S]*<Text type='tertiary'>\{t\('限制令牌分组'\)\}<\/Text>/,
  );
  assert.doesNotMatch(
    agentsSource,
    /display: 'none'[\s\S]*<Text type='tertiary'>\{t\('可创建令牌分组'\)\}<\/Text>/,
  );
});
