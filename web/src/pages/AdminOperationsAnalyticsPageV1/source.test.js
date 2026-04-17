/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const readSource = (fileUrl) =>
  fs.existsSync(fileUrl) ? fs.readFileSync(fileUrl, 'utf8') : '';

const pageSource = readSource(new URL('./index.jsx', import.meta.url));
const hookSource = readSource(
  new URL('../../hooks/operations-analytics/useOperationsAnalyticsData.js', import.meta.url),
);
const summaryCardsSource = readSource(
  new URL('../../components/operations-analytics/SummaryCards.jsx', import.meta.url),
);
const analyticsToolbarSource = readSource(
  new URL('../../components/operations-analytics/AnalyticsToolbar.jsx', import.meta.url),
);
const modelTabSource = readSource(
  new URL('../../components/operations-analytics/ModelAnalyticsTab.jsx', import.meta.url),
);
const userTabSource = readSource(
  new URL('../../components/operations-analytics/UserAnalyticsTab.jsx', import.meta.url),
);
const dailyTabSource = readSource(
  new URL('../../components/operations-analytics/DailyAnalyticsTab.jsx', import.meta.url),
);
const chartsHookSource = readSource(
  new URL('../../hooks/operations-analytics/useOperationsAnalyticsCharts.js', import.meta.url),
);
const appSource = readSource(new URL('../../App.jsx', import.meta.url));
const sidebarSource = readSource(
  new URL('../../components/layout/SiderBar.jsx', import.meta.url),
);
const useSidebarSource = readSource(
  new URL('../../hooks/common/useSidebar.js', import.meta.url),
);
const permissionCatalogSource = readSource(
  new URL('../AdminConsole/permissionCatalog.js', import.meta.url),
);
const permissionCatalogUiSource = readSource(
  new URL('../AdminConsole/permissionCatalogUi.js', import.meta.url),
);
const permissionCatalogUiCleanSource = readSource(
  new URL('../AdminConsole/permissionCatalogUiClean.js', import.meta.url),
);
const adminUserPermissionsCatalogSource = readSource(
  new URL('../AdminUserPermissionsPageV3/catalog.js', import.meta.url),
);

const ANALYTICS_MENU_OPTION_PATTERN =
  /\{\s*sectionKey:\s*'admin',\s*moduleKey:\s*'operations-analytics',\s*label:\s*'运营分析台'\s*\}/;
const ANALYTICS_RESOURCE_PATTERN =
  /\{\s*resourceKey:\s*'analytics_management',\s*label:\s*'运营分析台',[\s\S]*actionKey:\s*'read',\s*label:\s*'查看'[\s\S]*actionKey:\s*'export',\s*label:\s*'导出'[\s\S]*\}/;

test('App.jsx lazy-loads the operations analytics page and exposes /console/operations-analytics', () => {
  assert.match(
    appSource,
    /const AdminOperationsAnalytics = lazy\(\(\) => import\('\.\/pages\/AdminOperationsAnalyticsPageV1'\)\);/,
  );
  assert.match(appSource, /path='\/console\/operations-analytics'/);
  assert.match(appSource, /<AdminPlatformRoute>/);
  assert.match(appSource, /<AdminOperationsAnalytics \/>/);
});

test('SiderBar.jsx exposes operations-analytics in routerMap and admin menu items', () => {
  assert.match(sidebarSource, /'operations-analytics': '\/console\/operations-analytics'/);
  assert.match(
    sidebarSource,
    /text: t\('运营分析台'\),[\s\S]*itemKey: 'operations-analytics'[\s\S]*to: '\/console\/operations-analytics'/,
  );
});

test('useSidebar default admin config includes operations-analytics', () => {
  assert.match(
    useSidebarSource,
    /admin:\s*\{[\s\S]*'operations-analytics': true/,
  );
});

test('permission catalogs expose analytics_management resource and operations-analytics menu option', () => {
  assert.match(permissionCatalogSource, ANALYTICS_RESOURCE_PATTERN);
  assert.match(permissionCatalogUiSource, ANALYTICS_RESOURCE_PATTERN);
  assert.match(permissionCatalogUiCleanSource, ANALYTICS_RESOURCE_PATTERN);
  assert.match(adminUserPermissionsCatalogSource, ANALYTICS_RESOURCE_PATTERN);

  assert.match(permissionCatalogSource, ANALYTICS_MENU_OPTION_PATTERN);
  assert.match(permissionCatalogUiSource, ANALYTICS_MENU_OPTION_PATTERN);
  assert.match(permissionCatalogUiCleanSource, ANALYTICS_MENU_OPTION_PATTERN);
  assert.match(adminUserPermissionsCatalogSource, ANALYTICS_MENU_OPTION_PATTERN);
});

test('AdminOperationsAnalyticsPageV1 uses dedicated hook/components and preserves permission keys plus three tabs', () => {
  assert.match(
    pageSource,
    /hasActionPermission\('analytics_management', 'read'\)/,
  );
  assert.match(
    pageSource,
    /hasActionPermission\('analytics_management', 'export'\)/,
  );
  assert.match(pageSource, /运营分析台/);
  assert.match(pageSource, /useOperationsAnalyticsData/);
  assert.match(pageSource, /SummaryCards/);
  assert.match(pageSource, /AnalyticsToolbar/);
  assert.match(pageSource, /按模型/);
  assert.match(pageSource, /按用户/);
  assert.match(pageSource, /按日/);
  assert.doesNotMatch(
    pageSource,
    /当前页面仅提供前端骨架，真实统计查询、图表数据和导出能力将在后续任务接入。/,
  );
});

test('SummaryCards component exposes five summary cards and only shows natural-week wow when last7days is active', () => {
  assert.ok(summaryCardsSource, 'SummaryCards.jsx should exist');
  assert.match(summaryCardsSource, /总调用量/);
  assert.match(summaryCardsSource, /总费用/);
  assert.match(summaryCardsSource, /活跃用户/);
  assert.match(summaryCardsSource, /活跃模型/);
  assert.match(summaryCardsSource, /datePreset === 'last7days'/);
  assert.match(summaryCardsSource, /Token/);
  assert.match(summaryCardsSource, /自然周同比/);
  assert.match(summaryCardsSource, /\brenderQuota\(summary\.total_cost\)/);
  assert.doesNotMatch(summaryCardsSource, /formatSummaryValue\(summary\.total_cost\)/);
  assert.doesNotMatch(
    summaryCardsSource,
    /title:\s*t\('总费用'\),[\s\S]*unit:\s*t\('元'\)/,
  );
  assert.match(summaryCardsSource, /wowValue\.previous === 0 && wowValue\.current > 0/);
  assert.match(summaryCardsSource, /wowValue\.previous === 0 && wowValue\.current === 0/);
  assert.match(summaryCardsSource, /自然周同比 新增/);
  assert.match(summaryCardsSource, /自然周同比 -/);
});

test('SummaryCards component adds the fifth total-token card with wow support and Semi icons', () => {
  assert.ok(summaryCardsSource, 'SummaryCards.jsx should exist');
  assert.equal((summaryCardsSource.match(/title:\s*t\('/g) || []).length, 5);
  assert.match(summaryCardsSource, /Token/);
  assert.match(summaryCardsSource, /总 Token/);
  assert.match(summaryCardsSource, /formatSummaryValue\(summary\.total_tokens\)/);
  assert.match(summaryCardsSource, /summary\.wow\?\.total_tokens/);
  assert.match(summaryCardsSource, /@douyinfe\/semi-icons/);
  assert.equal((summaryCardsSource.match(/\bicon:\s*</g) || []).length, 5);
});

test('SummaryCards keeps total-token wow scoped to last7days and avoids fixed light surfaces', () => {
  assert.ok(summaryCardsSource, 'SummaryCards.jsx should exist');
  assert.equal((summaryCardsSource.match(/summary\.wow\?\.total_tokens/g) || []).length, 1);
  assert.match(
    summaryCardsSource,
    /title:\s*t\('[^']*Token'\),[\s\S]*helper:\s*datePreset === 'last7days'\s*\?\s*formatWowText\(t,\s*summary\.wow\?\.total_tokens\)\s*:\s*t\('[^']+'\)/,
  );
  assert.match(
    summaryCardsSource,
    /summary\.wow\?\.total_calls\)\s*:\s*(t\('[^']+'\))[\s\S]*summary\.wow\?\.total_tokens\)\s*:\s*\1/,
  );
  assert.doesNotMatch(summaryCardsSource, /rgba\(255,\s*255,\s*255,\s*0\.\d+\)/);
  assert.match(summaryCardsSource, /var\(--semi-color-bg-0\)/);
  assert.match(summaryCardsSource, /var\(--semi-color-fill-0\)/);
});

test('AnalyticsToolbar component exposes three date presets plus reset apply export and custom date controls', () => {
  assert.ok(analyticsToolbarSource, 'AnalyticsToolbar.jsx should exist');
  assert.match(analyticsToolbarSource, /今日/);
  assert.match(analyticsToolbarSource, /近7天/);
  assert.match(analyticsToolbarSource, /自定义/);
  assert.match(analyticsToolbarSource, /重置/);
  assert.match(analyticsToolbarSource, /应用/);
  assert.match(analyticsToolbarSource, /导出/);
  assert.match(analyticsToolbarSource, /DatePicker/);
  assert.match(analyticsToolbarSource, /开始日期/);
  assert.match(analyticsToolbarSource, /结束日期/);
});

test('useOperationsAnalyticsData hook drives summary loading and export payload from activeTab plus appliedFilters', () => {
  assert.ok(hookSource, 'useOperationsAnalyticsData.js should exist');
  assert.match(hookSource, /\/api\/admin\/analytics\/summary/);
  assert.match(hookSource, /\/api\/admin\/analytics\/export/);
  assert.match(hookSource, /\bdownloadExcelBlob\b/);
  assert.match(hookSource, /const \[activeTab, setActiveTab\] = useState\('models'\)/);
  assert.match(hookSource, /const \[datePreset, setDatePreset\] = useState\('last7days'\)/);
  assert.match(hookSource, /const \[appliedFilters, setAppliedFilters\] = useState/);
  assert.match(
    hookSource,
    /const createAppliedFilters = \(\s*activeTab,\s*datePreset = DEFAULT_DATE_PRESET,/,
  );
  assert.match(
    hookSource,
    /const filterKeywordStateByActiveTab = \(filters,\s*activeTab\) => \(\{/,
  );
  assert.match(
    hookSource,
    /modelKeyword:\s*activeTab === 'models' \? \(filters\.modelKeyword \|\| ''\)\.trim\(\) : ''/,
  );
  assert.match(
    hookSource,
    /activeTab === 'users' \? \(filters\.usernameKeyword \|\| ''\)\.trim\(\) : ''/,
  );
  assert.match(hookSource, /API\.get\('\/api\/admin\/analytics\/summary'/);
  assert.match(hookSource, /view:\s*activeTab/);
  assert.match(
    hookSource,
    /payload:\s*buildOperationsAnalyticsExportPayload\(\{\s*activeTab,\s*datePreset:\s*appliedFilters\.datePreset,\s*filters:\s*appliedFilters,\s*sortState:\s*sortStateByTab\[activeTab\],/,
  );
  assert.match(hookSource, /validateFilters\(appliedFilters,\s*t\)/);
  assert.match(hookSource, /sortStateByTab = \{\}/);
  assert.match(
    hookSource,
    /buildOperationsAnalyticsExportPayload = \(\{\s*activeTab,\s*datePreset,\s*filters,\s*sortState,\s*\}\) =>/,
  );
  assert.match(hookSource, /const getQuotaDisplayType = \(\) => \{/);
  assert.match(hookSource, /payload\.quota_display_type = getQuotaDisplayType\(\);/);
  assert.match(hookSource, /payload\.sort_by = sortState\.sortBy;/);
  assert.match(hookSource, /payload\.sort_order = sortState\.sortOrder;/);
  assert.match(
    hookSource,
    /useEffect\(\(\) => \{\s*setAppliedFilters\(\(currentFilters\) => \{/,
  );
  assert.match(
    hookSource,
    /useEffect\(\(\) => \{[\s\S]*setDraftFilters\(\(currentFilters\) => \{/,
  );
  assert.match(
    hookSource,
    /const FIXED_ANALYTICS_TIMEZONE_OFFSET_MINUTES = 8 \* 60;/,
  );
  assert.match(
    hookSource,
    /const serializeFixedUtc8DayTimestamp = \(value,\s*boundary\) => \{/,
  );
  assert.match(
    hookSource,
    /const FIXED_ANALYTICS_TIMEZONE_OFFSET_SECONDS =\s*FIXED_ANALYTICS_TIMEZONE_OFFSET_MINUTES \* 60;/,
  );
  assert.match(
    hookSource,
    /const utcDayStart =\s*Date\.UTC\([\s\S]*dayValue\.year\(\),[\s\S]*dayValue\.month\(\),[\s\S]*dayValue\.date\(\),?[\s\S]*\)\s*\/\s*1000;/,
  );
  assert.match(
    hookSource,
    /return boundary === 'end'\s*\?\s*utcDayStart - FIXED_ANALYTICS_TIMEZONE_OFFSET_SECONDS \+ 24 \* 60 \* 60 - 1\s*:\s*utcDayStart - FIXED_ANALYTICS_TIMEZONE_OFFSET_SECONDS;/,
  );
  assert.match(
    hookSource,
    /params\.start_timestamp = serializeFixedUtc8DayTimestamp\([\s\S]*appliedFilters\.startDate,[\s\S]*'start',?[\s\S]*\);/,
  );
  assert.match(
    hookSource,
    /params\.end_timestamp = serializeFixedUtc8DayTimestamp\([\s\S]*appliedFilters\.endDate,[\s\S]*'end',?[\s\S]*\);/,
  );
  assert.match(hookSource, /params\.request_ts = Date\.now\(\);/);
  assert.match(
    hookSource,
    /payload\.start_timestamp = serializeFixedUtc8DayTimestamp\([\s\S]*filters\.startDate,[\s\S]*'start',?[\s\S]*\);/,
  );
  assert.match(
    hookSource,
    /payload\.end_timestamp = serializeFixedUtc8DayTimestamp\([\s\S]*filters\.endDate,[\s\S]*'end',?[\s\S]*\);/,
  );
  assert.doesNotMatch(
    hookSource,
    /dayValue\.endOf\('day'\)\.unix\(\)\s*:\s*dayValue\.startOf\('day'\)\.unix\(\)/,
  );
  assert.doesNotMatch(
    hookSource,
    /const exportAnalytics = async \(\) => \{[\s\S]*createAppliedFilters\(datePreset,\s*draftFilters\)/,
  );
  assert.match(hookSource, /useEffect\(\(\) => \{/);
});

test('AdminOperationsAnalyticsPageV1 mounts dedicated model user and daily tab components', () => {
  assert.ok(modelTabSource, 'ModelAnalyticsTab.jsx should exist');
  assert.ok(userTabSource, 'UserAnalyticsTab.jsx should exist');
  assert.ok(dailyTabSource, 'DailyAnalyticsTab.jsx should exist');

  assert.match(pageSource, /ModelAnalyticsTab/);
  assert.match(pageSource, /UserAnalyticsTab/);
  assert.match(pageSource, /DailyAnalyticsTab/);
  assert.match(pageSource, /const \[tabSortState, setTabSortState\] = useState/);
  assert.match(pageSource, /const updateTabSortState = \(tabKey, nextSortState\) => \{/);
  assert.match(pageSource, /<ModelAnalyticsTab[\s\S]*appliedFilters=\{appliedFilters\}[\s\S]*sortState=\{tabSortState\.models\}[\s\S]*onSortStateChange=\{\(nextSortState\) => updateTabSortState\('models', nextSortState\)\}/);
  assert.match(pageSource, /<UserAnalyticsTab[\s\S]*appliedFilters=\{appliedFilters\}[\s\S]*sortState=\{tabSortState\.users\}[\s\S]*onSortStateChange=\{\(nextSortState\) => updateTabSortState\('users', nextSortState\)\}/);
  assert.match(pageSource, /<DailyAnalyticsTab[\s\S]*appliedFilters=\{appliedFilters\}/);
});

test('operations analytics page removes the original placeholder copy', () => {
  assert.doesNotMatch(pageSource, /鎸夋ā鍨嬭〃鏍奸鏋?/);
  assert.doesNotMatch(pageSource, /鎸夌敤鎴疯〃鏍奸鏋?/);
  assert.doesNotMatch(pageSource, /鎸夋棩瓒嬪娍鎶樼嚎鍥?/);
  assert.doesNotMatch(pageSource, /PlaceholderPanel/);
});

test('useOperationsAnalyticsCharts defines line pie and bar specs with semi theme initialization', () => {
  assert.ok(chartsHookSource, 'useOperationsAnalyticsCharts.js should exist');
  assert.match(chartsHookSource, /@visactor\/vchart-semi-theme/);
  assert.match(chartsHookSource, /initVChartSemiTheme/);
  assert.match(chartsHookSource, /type:\s*'line'/);
  assert.match(chartsHookSource, /type:\s*'pie'/);
  assert.match(chartsHookSource, /type:\s*'bar'/);
  assert.match(chartsHookSource, /specLine/);
  assert.match(chartsHookSource, /specPie/);
  assert.match(chartsHookSource, /specBar/);
  assert.match(
    chartsHookSource,
    /const specBar = useCallback\([\s\S]*?valueField\s*=\s*xField[\s\S]*?formatTooltipValue\(valueFormatter,\s*datum\?\.\[valueField\]\)/,
  );
});

test('ModelAnalyticsTab defines sortable analytics columns and VChart usage', () => {
  assert.match(modelTabSource, /@visactor\/react-vchart/);
  assert.match(
    modelTabSource,
    /const ModelAnalyticsTab = \(\{\s*activeTab,\s*appliedFilters,\s*sortState,\s*onSortStateChange,\s*\}\) =>/,
  );
  assert.match(modelTabSource, /dataIndex:\s*'model_name'/);
  assert.match(modelTabSource, /dataIndex:\s*'call_count'/);
  assert.match(modelTabSource, /dataIndex:\s*'prompt_tokens'/);
  assert.match(modelTabSource, /dataIndex:\s*'completion_tokens'/);
  assert.match(modelTabSource, /dataIndex:\s*'total_cost'/);
  assert.match(modelTabSource, /dataIndex:\s*'avg_use_time'/);
  assert.match(modelTabSource, /dataIndex:\s*'success_rate'/);
  assert.match(modelTabSource, /sorter:\s*true/);
  assert.match(modelTabSource, /renderQuota\(value/);
  assert.match(modelTabSource, /sort_by/);
  assert.match(modelTabSource, /sort_order/);
  assert.match(modelTabSource, /const sortBy = sortState\?\.sortBy \|\| '';/);
  assert.match(modelTabSource, /const sortOrder = sortState\?\.sortOrder \|\| '';/);
  assert.match(modelTabSource, /暂无 token 数据/);
  assert.match(
    modelTabSource,
    /const barSpec = useMemo\([\s\S]*?callRankItems\.map[\s\S]*?xField:\s*'model_name'[\s\S]*?yField:\s*'call_count'[\s\S]*?valueField:\s*'call_count'/,
  );
  assert.match(
    modelTabSource,
    /onSortStateChange\(\{\s*sortBy:\s*nextSortBy,\s*sortOrder:\s*nextSortOrder,\s*\}\)/,
  );
  assert.doesNotMatch(modelTabSource, /const \[sortBy, setSortBy\] = useState\(''\);/);
  assert.doesNotMatch(modelTabSource, /const \[sortOrder, setSortOrder\] = useState\(''\);/);
});

test('UserAnalyticsTab defines sortable analytics columns and quota plus timestamp rendering', () => {
  assert.match(userTabSource, /@visactor\/react-vchart/);
  assert.match(
    userTabSource,
    /const UserAnalyticsTab = \(\{\s*activeTab,\s*appliedFilters,\s*sortState,\s*onSortStateChange,\s*\}\) =>/,
  );
  assert.match(userTabSource, /dataIndex:\s*'user_id'/);
  assert.match(userTabSource, /dataIndex:\s*'username'/);
  assert.match(userTabSource, /dataIndex:\s*'call_count'/);
  assert.match(userTabSource, /dataIndex:\s*'model_count'/);
  assert.match(userTabSource, /dataIndex:\s*'total_cost'/);
  assert.match(userTabSource, /dataIndex:\s*'last_called_at'/);
  assert.match(userTabSource, /sorter:\s*true/);
  assert.match(userTabSource, /renderQuota\(value/);
  assert.match(userTabSource, /timestamp2string\(value\)/);
  assert.match(userTabSource, /const sortBy = sortState\?\.sortBy \|\| '';/);
  assert.match(userTabSource, /const sortOrder = sortState\?\.sortOrder \|\| '';/);
  assert.match(
    userTabSource,
    /onSortStateChange\(\{\s*sortBy:\s*nextSortBy,\s*sortOrder:\s*nextSortOrder,\s*\}\)/,
  );
  assert.doesNotMatch(userTabSource, /const \[sortBy, setSortBy\] = useState\(''\);/);
  assert.doesNotMatch(userTabSource, /const \[sortOrder, setSortOrder\] = useState\(''\);/);
});

test('DailyAnalyticsTab loads daily endpoint and renders trends without a table', () => {
  assert.match(dailyTabSource, /@visactor\/react-vchart/);
  assert.match(dailyTabSource, /\/api\/admin\/analytics\/daily/);
  assert.match(dailyTabSource, /bucket_day/);
  assert.match(dailyTabSource, /call_count/);
  assert.match(dailyTabSource, /total_cost/);
  assert.match(dailyTabSource, /active_users/);
  assert.match(dailyTabSource, /active_models/);
  assert.doesNotMatch(dailyTabSource, /<Table/);
});

test('operations analytics tabs remount tab content when applied filters change', () => {
  assert.match(pageSource, /const filtersCacheKey = JSON\.stringify\(appliedFilters\);/);
  assert.match(pageSource, /key=\{`models-\$\{filtersCacheKey\}`\}/);
  assert.match(pageSource, /key=\{`users-\$\{filtersCacheKey\}`\}/);
  assert.match(pageSource, /key=\{`daily-\$\{filtersCacheKey\}`\}/);
});

test('ModelAnalyticsTab separates table and chart loading, splits error state, and invalidates stale requests', () => {
  assert.match(modelTabSource, /const loadTableData = useCallback\(async \(\) => \{/);
  assert.match(modelTabSource, /const loadChartData = useCallback\(async \(\) => \{/);
  assert.match(modelTabSource, /const \[tableError, setTableError\] = useState\(''\);/);
  assert.match(modelTabSource, /const \[chartError, setChartError\] = useState\(''\);/);
  assert.doesNotMatch(modelTabSource, /const \[error, setError\] = useState\(''\);/);
  assert.match(modelTabSource, /useEffect\(\(\) => \{\s*loadTableData\(\);\s*\}, \[loadTableData\]\);/);
  assert.match(modelTabSource, /useEffect\(\(\) => \{\s*loadChartData\(\);\s*\}, \[loadChartData\]\);/);
  assert.match(
    modelTabSource,
    /const loadChartData = useCallback\(async \(\) => \{[\s\S]*?\}, \[activeTab, appliedFilters, t\]\);/,
  );
  assert.doesNotMatch(
    modelTabSource,
    /useEffect\(\(\) => \{\s*setPage\(1\);\s*\}, \[appliedFilters\]\);/,
  );
  assert.match(
    modelTabSource,
    /useEffect\(\(\) => \(\) => \{\s*tableRequestRef\.current \+= 1;\s*chartRequestRef\.current \+= 1;\s*\}, \[\]\);/,
  );
  assert.match(
    modelTabSource,
    /useEffect\(\(\) => \{\s*if \(activeTab !== 'models'\) \{\s*tableRequestRef\.current \+= 1;\s*chartRequestRef\.current \+= 1;\s*setLoading\(false\);\s*\}\s*\}, \[activeTab\]\);/,
  );
});

test('UserAnalyticsTab separates table and chart loading, splits error state, and invalidates stale requests', () => {
  assert.match(userTabSource, /const loadTableData = useCallback\(async \(\) => \{/);
  assert.match(userTabSource, /const loadChartData = useCallback\(async \(\) => \{/);
  assert.match(userTabSource, /const \[tableError, setTableError\] = useState\(''\);/);
  assert.match(userTabSource, /const \[chartError, setChartError\] = useState\(''\);/);
  assert.doesNotMatch(userTabSource, /const \[error, setError\] = useState\(''\);/);
  assert.match(userTabSource, /useEffect\(\(\) => \{\s*loadTableData\(\);\s*\}, \[loadTableData\]\);/);
  assert.match(userTabSource, /useEffect\(\(\) => \{\s*loadChartData\(\);\s*\}, \[loadChartData\]\);/);
  assert.match(
    userTabSource,
    /const loadChartData = useCallback\(async \(\) => \{[\s\S]*?\}, \[activeTab, appliedFilters, t\]\);/,
  );
  assert.doesNotMatch(
    userTabSource,
    /useEffect\(\(\) => \{\s*setPage\(1\);\s*\}, \[appliedFilters\]\);/,
  );
  assert.match(
    userTabSource,
    /useEffect\(\(\) => \(\) => \{\s*tableRequestRef\.current \+= 1;\s*chartRequestRef\.current \+= 1;\s*\}, \[\]\);/,
  );
  assert.match(
    userTabSource,
    /useEffect\(\(\) => \{\s*if \(activeTab !== 'users'\) \{\s*tableRequestRef\.current \+= 1;\s*chartRequestRef\.current \+= 1;\s*setLoading\(false\);\s*\}\s*\}, \[activeTab\]\);/,
  );
});

test('DailyAnalyticsTab exposes a real loading spinner and invalidates stale requests', () => {
  assert.match(dailyTabSource, /import \{ Banner, Empty, Spin \} from '@douyinfe\/semi-ui';/);
  assert.match(dailyTabSource, /<Spin spinning=\{loading\}/);
  assert.match(
    dailyTabSource,
    /useEffect\(\(\) => \(\) => \{\s*requestRef\.current \+= 1;\s*\}, \[\]\);/,
  );
  assert.match(
    dailyTabSource,
    /useEffect\(\(\) => \{\s*if \(activeTab !== 'daily'\) \{\s*requestRef\.current \+= 1;\s*setLoading\(false\);\s*\}\s*\}, \[activeTab\]\);/,
  );
});
