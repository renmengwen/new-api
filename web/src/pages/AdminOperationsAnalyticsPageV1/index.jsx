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
import React, { useEffect, useState } from 'react';
import { Banner, Empty, Space, TabPane, Tabs, Tag, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import CardPro from '../../components/common/ui/CardPro';
import AnalyticsToolbar from '../../components/operations-analytics/AnalyticsToolbar';
import DailyAnalyticsTab from '../../components/operations-analytics/DailyAnalyticsTab';
import ModelAnalyticsTab from '../../components/operations-analytics/ModelAnalyticsTab';
import SummaryCards from '../../components/operations-analytics/SummaryCards';
import UserAnalyticsTab from '../../components/operations-analytics/UserAnalyticsTab';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import { useOperationsAnalyticsData } from '../../hooks/operations-analytics/useOperationsAnalyticsData';

const { Text, Title } = Typography;

const createEmptyTabSortState = () => ({
  models: {
    sortBy: '',
    sortOrder: '',
  },
  users: {
    sortBy: '',
    sortOrder: '',
  },
});

const pageCardStyle = {
  marginTop: 48,
};

const AdminOperationsAnalyticsPageV1 = () => {
  const { t } = useTranslation();
  const { loading: permissionLoading, hasActionPermission } = useUserPermissions();
  const canRead = hasActionPermission('analytics_management', 'read');
  const canExport = hasActionPermission('analytics_management', 'export');
  const [tabSortState, setTabSortState] = useState(() => createEmptyTabSortState());
  const {
    activeTab,
    setActiveTab,
    datePreset,
    setDatePreset,
    draftFilters,
    updateDraftFilter,
    appliedFilters,
    applyFilters,
    resetFilters,
    summary,
    summaryLoading,
    summaryError,
    exportLoading,
    exportAnalytics,
  } = useOperationsAnalyticsData({
    canRead,
    canExport,
    sortStateByTab: tabSortState,
    t,
  });
  const filtersCacheKey = JSON.stringify(appliedFilters);

  useEffect(() => {
    setTabSortState(createEmptyTabSortState());
  }, [filtersCacheKey]);

  const updateTabSortState = (tabKey, nextSortState) => {
    setTabSortState((currentState) => ({
      ...currentState,
      [tabKey]: {
        sortBy: nextSortState?.sortBy || '',
        sortOrder: nextSortState?.sortOrder || '',
      },
    }));
  };

  const descriptionArea = (
    <div className='flex flex-col gap-1'>
      <Title heading={4} style={{ margin: 0 }}>
        {t('运营分析台')}
      </Title>
      <Text type='tertiary'>
        {t('面向运营后台的调用分析台，支持按模型、按用户、按日三个维度的聚合分析。')}
      </Text>
    </div>
  );

  const actionsArea =
    appliedFilters.datePreset === 'last7days' ? (
      <Tag color='blue' size='large'>
        {t('自然周同比已启用')}
      </Tag>
    ) : null;

  if (permissionLoading) {
    return (
      <CardPro
        type='type1'
        descriptionArea={descriptionArea}
        actionsArea={actionsArea}
        t={t}
        style={pageCardStyle}
      >
        <Empty
          title={t('加载中')}
          description={t('正在校验运营分析台权限与页面配置。')}
        />
      </CardPro>
    );
  }

  if (!canRead) {
    return (
      <CardPro type='type1' descriptionArea={descriptionArea} t={t} style={pageCardStyle}>
        <Banner
          type='warning'
          description={t(
            '当前账号没有运营分析台查看权限，请联系管理员分配 analytics_management.read。',
          )}
        />
        <Empty
          title={t('暂无访问权限')}
          description={t(
            '如需访问运营分析台，请在权限模板或用户权限中开启对应资源动作。',
          )}
        />
      </CardPro>
    );
  }

  return (
    <CardPro
      type='type1'
      descriptionArea={descriptionArea}
      actionsArea={actionsArea}
      t={t}
      style={pageCardStyle}
    >
      <Space vertical align='start' spacing='medium' style={{ width: '100%' }}>
        <Banner
          type='info'
          description={t(
            '页面图表、表格与导出都会实时跟随顶部筛选条件和当前 Tab 维度变化。',
          )}
        />

        {summaryError ? <Banner type='warning' description={summaryError} /> : null}

        <SummaryCards
          datePreset={appliedFilters.datePreset}
          summary={summary}
          summaryLoading={summaryLoading}
          t={t}
        />

        <AnalyticsToolbar
          activeTab={activeTab}
          datePreset={datePreset}
          setDatePreset={setDatePreset}
          draftFilters={draftFilters}
          updateDraftFilter={updateDraftFilter}
          onReset={resetFilters}
          onApply={applyFilters}
          onExport={exportAnalytics}
          canExport={canExport}
          exportLoading={exportLoading}
          t={t}
        />

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          type='line'
          style={{ width: '100%' }}
        >
          <TabPane tab={t('按模型')} itemKey='models'>
            <ModelAnalyticsTab
              key={`models-${filtersCacheKey}`}
              activeTab={activeTab}
              appliedFilters={appliedFilters}
              sortState={tabSortState.models}
              onSortStateChange={(nextSortState) => updateTabSortState('models', nextSortState)}
            />
          </TabPane>

          <TabPane tab={t('按用户')} itemKey='users'>
            <UserAnalyticsTab
              key={`users-${filtersCacheKey}`}
              activeTab={activeTab}
              appliedFilters={appliedFilters}
              sortState={tabSortState.users}
              onSortStateChange={(nextSortState) => updateTabSortState('users', nextSortState)}
            />
          </TabPane>

          <TabPane tab={t('按日')} itemKey='daily'>
            <DailyAnalyticsTab
              key={`daily-${filtersCacheKey}`}
              activeTab={activeTab}
              appliedFilters={appliedFilters}
            />
          </TabPane>
        </Tabs>
      </Space>
    </CardPro>
  );
};

export default AdminOperationsAnalyticsPageV1;
