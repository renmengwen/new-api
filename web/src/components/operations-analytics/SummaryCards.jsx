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
import { Typography } from '@douyinfe/semi-ui';
import { renderQuota } from '../../helpers';

const { Text, Title } = Typography;

const summaryCardThemes = ['blue', 'orange', 'green', 'violet'];

const formatSummaryValue = (value) =>
  new Intl.NumberFormat('zh-CN').format(Number(value || 0));

const formatWowText = (t, wowValue) => {
  if (!wowValue) {
    return t('自然周同比待接入');
  }

  if (wowValue.previous === 0 && wowValue.current > 0) {
    return t('自然周同比 新增');
  }

  if (wowValue.previous === 0 && wowValue.current === 0) {
    return t('自然周同比 -');
  }

  const ratio = `${Math.abs(Number(wowValue.change_ratio || 0) * 100).toFixed(1)}%`;

  if (wowValue.trend === 'up') {
    return `${t('自然周同比')} +${ratio}`;
  }

  if (wowValue.trend === 'down') {
    return `${t('自然周同比')} -${ratio}`;
  }

  return `${t('自然周同比')} 0.0%`;
};

const buildSummaryCards = ({ summary, summaryLoading, datePreset, t }) => [
  {
    title: t('总调用量'),
    value: summaryLoading ? '--' : formatSummaryValue(summary.total_calls),
    unit: t('次'),
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.total_calls)
        : t('已按当前筛选范围汇总'),
  },
  {
    title: t('总费用'),
    value: summaryLoading ? '--' : renderQuota(summary.total_cost),
    unit: '',
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.total_cost)
        : t('已按当前筛选范围汇总'),
  },
  {
    title: t('活跃用户'),
    value: summaryLoading ? '--' : formatSummaryValue(summary.active_users),
    unit: t('人'),
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.active_users)
        : t('已按当前筛选范围汇总'),
  },
  {
    title: t('活跃模型'),
    value: summaryLoading ? '--' : formatSummaryValue(summary.active_models),
    unit: t('个'),
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.active_models)
        : t('当前时间范围内有调用的模型数'),
  },
];

const SummaryCards = ({ datePreset, summary, summaryLoading, t }) => (
  <div className='flex flex-wrap gap-3 w-full'>
    {buildSummaryCards({
      summary,
      summaryLoading,
      datePreset,
      t,
    }).map((card, index) => (
      <div
        key={card.title}
        className='flex flex-col gap-2 rounded-2xl border p-4 min-w-[180px] flex-1'
        style={{
          borderColor: `var(--semi-color-${summaryCardThemes[index]}-3)`,
          background: `var(--semi-color-${summaryCardThemes[index]}-0)`,
        }}
      >
        <Text strong>{card.title}</Text>
        <div className='flex items-end gap-2'>
          <Title heading={3} style={{ margin: 0 }}>
            {card.value}
          </Title>
          {card.unit ? <Text type='tertiary'>{card.unit}</Text> : null}
        </div>
        <Text type='tertiary'>{card.helper}</Text>
      </div>
    ))}
  </div>
);

export default SummaryCards;
