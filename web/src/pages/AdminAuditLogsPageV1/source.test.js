import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const readSource = (fileUrl) =>
  fs.existsSync(fileUrl) ? fs.readFileSync(fileUrl, 'utf8') : '';

const pageSource = readSource(new URL('./index.jsx', import.meta.url));
const appSource = readSource(new URL('../../App.jsx', import.meta.url));
const sidebarSource = readSource(new URL('../../components/layout/SiderBar.jsx', import.meta.url));
const useSidebarSource = readSource(new URL('../../hooks/common/useSidebar.js', import.meta.url));
const permissionCatalogSource = readSource(new URL('../AdminConsole/permissionCatalog.js', import.meta.url));
const permissionCatalogUiSource = readSource(new URL('../AdminConsole/permissionCatalogUi.js', import.meta.url));
const permissionCatalogUiCleanSource = readSource(new URL('../AdminConsole/permissionCatalogUiClean.js', import.meta.url));
const adminUserPermissionsCatalogSource = readSource(new URL('../AdminUserPermissionsPageV3/catalog.js', import.meta.url));

const AUDIT_LOGS_MENU_OPTION_PATTERN = /\{\s*sectionKey:\s*'admin',\s*moduleKey:\s*'audit-logs',\s*label:\s*'审计日志'\s*\}/;

test('AdminAuditLogsPageV1 uses audit_management.read and calls the audit logs endpoint', () => {
  assert.match(pageSource, /hasActionPermission\('audit_management', 'read'\)/);
  assert.match(pageSource, /\/api\/admin\/audit-logs/);
  assert.match(pageSource, /action_module/);
  assert.match(pageSource, /operator_user_id/);
});

test('AdminAuditLogsPageV1 builds moduleOptions from AUDIT_LOG_COVERAGE and filters modules with Select', () => {
  assert.match(pageSource, /AUDIT_LOG_COVERAGE/);
  assert.match(pageSource, /const moduleOptions =/);
  assert.match(pageSource, /AUDIT_LOG_COVERAGE\.map\(\(\{ module \}\) => \(\{/);
  assert.match(pageSource, /label: getAuditLogModuleLabel\(module\)/);
  assert.match(pageSource, /value: module/);
  assert.match(pageSource, /<Select[\s\S]*optionList=\{moduleOptions\}/);
  assert.doesNotMatch(pageSource, /<Input[\s\S]*?value=\{actionModule\}/);
});

test('AdminAuditLogsPageV1 renders enriched audit log display fields through display helpers', () => {
  assert.match(pageSource, /from '\.\/display'/);
  assert.match(pageSource, /getAuditLogModuleLabel/);
  assert.match(pageSource, /getAuditLogActionLabel/);
  assert.match(pageSource, /formatAuditIdentity/);
  assert.match(pageSource, /formatAuditTarget/);
  assert.match(pageSource, /title: t\('操作人'\)/);
  assert.match(pageSource, /title: t\('目标'\)/);
  assert.equal(/title: t\('操作人 ID'\)/.test(pageSource), false);
  assert.equal(/title: t\('目标 ID'\)/.test(pageSource), false);
  assert.match(pageSource, /operator_username/);
  assert.match(pageSource, /operator_display_name/);
  assert.match(pageSource, /target_username/);
  assert.match(pageSource, /target_display_name/);
});

test('AdminAuditLogsPageV1 wires Excel export from the committed request and guards empty or capped exports', () => {
  assert.match(pageSource, /downloadExcelBlob/);
  assert.match(pageSource, /\/api\/admin\/audit-logs\/export/);
  assert.match(pageSource, /payload:\s*\{/);
  assert.match(pageSource, /committedRequest\.actionModule/);
  assert.match(pageSource, /committedRequest\.operatorUserId/);
  assert.match(pageSource, /limit:\s*MAX_EXCEL_EXPORT_ROWS/);
  assert.match(pageSource, /showInfo\(t\('无可导出数据'\)\)/);
  assert.match(pageSource, /Modal\.confirm/);
  assert.match(pageSource, /导出 Excel/);
});

test('App.jsx lazy-loads the audit logs page and exposes /console/audit-logs', () => {
  assert.match(appSource, /const AdminAuditLogs = lazy\(\(\) => import\('\.\/pages\/AdminAuditLogsPageV1'\)\);/);
  assert.match(appSource, /path='\/console\/audit-logs'/);
  assert.match(appSource, /<AdminAuditLogs \/>/);
});

test('SiderBar.jsx exposes audit-logs in routerMap and admin menu items', () => {
  assert.match(sidebarSource, /'audit-logs': '\/console\/audit-logs'/);
  assert.match(sidebarSource, /text: t\('审计日志'\),[\s\S]*itemKey: 'audit-logs'[\s\S]*to: '\/console\/audit-logs'/);
});

test('useSidebar default admin config includes audit-logs', () => {
  assert.match(useSidebarSource, /admin:\s*\{[\s\S]*'audit-logs': true/);
});

test('permission catalogs expose 审计日志 as an admin menu option', () => {
  assert.match(permissionCatalogSource, AUDIT_LOGS_MENU_OPTION_PATTERN);
  assert.match(permissionCatalogUiSource, AUDIT_LOGS_MENU_OPTION_PATTERN);
  assert.match(permissionCatalogUiCleanSource, AUDIT_LOGS_MENU_OPTION_PATTERN);
  assert.match(adminUserPermissionsCatalogSource, AUDIT_LOGS_MENU_OPTION_PATTERN);
});
