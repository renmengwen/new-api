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
import React from 'react';
import { Button, DatePicker, Input, Tag } from '@douyinfe/semi-ui';

const datePresetOptions = [
  { key: 'today', label: '今日' },
  { key: 'last7days', label: '近7天' },
  { key: 'custom', label: '自定义' },
];

const renderTabFilterInput = ({ activeTab, draftFilters, updateDraftFilter, t }) => {
  if (activeTab === 'models') {
    return (
      <Input
        style={{ width: 240 }}
        value={draftFilters.modelKeyword}
        placeholder={t('模型名称')}
        onChange={(value) => updateDraftFilter('modelKeyword', value)}
        showClear
      />
    );
  }

  if (activeTab === 'users') {
    return (
      <Input
        style={{ width: 240 }}
        value={draftFilters.usernameKeyword}
        placeholder={t('用户昵称 / 用户名')}
        onChange={(value) => updateDraftFilter('usernameKeyword', value)}
        showClear
      />
    );
  }

  return null;
};

const AnalyticsToolbar = ({
  activeTab,
  datePreset,
  setDatePreset,
  draftFilters,
  updateDraftFilter,
  onReset,
  onApply,
  onExport,
  canExport,
  exportLoading,
  t,
}) => (
  <div
    className='rounded-2xl border px-4 py-4 w-full flex flex-col gap-3'
    style={{
      borderColor: 'var(--semi-color-border)',
      background: 'var(--semi-color-bg-1)',
    }}
  >
    <div className='flex flex-wrap gap-2 items-center'>
      {datePresetOptions.map((option) => (
        <Button
          key={option.key}
          type={datePreset === option.key ? 'primary' : 'tertiary'}
          theme={datePreset === option.key ? 'solid' : 'outline'}
          onClick={() => setDatePreset(option.key)}
        >
          {t(option.label)}
        </Button>
      ))}
      {datePreset === 'last7days' && (
        <Tag color='blue' size='large'>
          {t('自然周同比仅在近7天展示')}
        </Tag>
      )}
    </div>

    <div className='flex flex-wrap gap-2 items-center'>
      {renderTabFilterInput({
        activeTab,
        draftFilters,
        updateDraftFilter,
        t,
      })}

      {datePreset === 'custom' && (
        <>
          <DatePicker
            type='date'
            style={{ width: 180 }}
            value={draftFilters.startDate}
            placeholder={t('开始日期')}
            onChange={(value) => updateDraftFilter('startDate', value || null)}
          />
          <DatePicker
            type='date'
            style={{ width: 180 }}
            value={draftFilters.endDate}
            placeholder={t('结束日期')}
            onChange={(value) => updateDraftFilter('endDate', value || null)}
          />
        </>
      )}

      <Button onClick={onReset}>{t('重置')}</Button>
      <Button theme='solid' type='primary' onClick={onApply}>
        {t('应用')}
      </Button>
      <Button
        disabled={!canExport}
        loading={exportLoading}
        onClick={onExport}
      >
        {t('导出')}
      </Button>
    </div>
  </div>
);

export default AnalyticsToolbar;
