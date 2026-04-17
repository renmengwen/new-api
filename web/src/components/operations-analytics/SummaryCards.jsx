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
import {
  IconCoinMoneyStroked,
  IconHistogram,
  IconLayers,
  IconTextStroked,
  IconUserGroup,
} from '@douyinfe/semi-icons';
import { Typography } from '@douyinfe/semi-ui';
import { renderQuota } from '../../helpers';

const { Text, Title } = Typography;

const summaryCardThemes = [
  {
    borderColor: 'var(--semi-color-blue-3)',
    background:
      'linear-gradient(135deg, var(--semi-color-blue-0) 0%, rgba(255, 255, 255, 0.96) 100%)',
    iconBackground: 'rgba(59, 130, 246, 0.16)',
    iconColor: 'var(--semi-color-blue-6)',
    shadow: '0 18px 40px rgba(59, 130, 246, 0.12)',
  },
  {
    borderColor: 'var(--semi-color-orange-3)',
    background:
      'linear-gradient(135deg, var(--semi-color-orange-0) 0%, rgba(255, 255, 255, 0.96) 100%)',
    iconBackground: 'rgba(249, 115, 22, 0.16)',
    iconColor: 'var(--semi-color-orange-6)',
    shadow: '0 18px 40px rgba(249, 115, 22, 0.12)',
  },
  {
    borderColor: 'var(--semi-color-green-3)',
    background:
      'linear-gradient(135deg, var(--semi-color-green-0) 0%, rgba(255, 255, 255, 0.96) 100%)',
    iconBackground: 'rgba(16, 185, 129, 0.16)',
    iconColor: 'var(--semi-color-green-6)',
    shadow: '0 18px 40px rgba(16, 185, 129, 0.12)',
  },
  {
    borderColor: 'var(--semi-color-violet-3)',
    background:
      'linear-gradient(135deg, var(--semi-color-violet-0) 0%, rgba(255, 255, 255, 0.96) 100%)',
    iconBackground: 'rgba(124, 58, 237, 0.16)',
    iconColor: 'var(--semi-color-violet-6)',
    shadow: '0 18px 40px rgba(124, 58, 237, 0.12)',
  },
  {
    borderColor: 'var(--semi-color-cyan-3)',
    background:
      'linear-gradient(135deg, var(--semi-color-cyan-0) 0%, rgba(255, 255, 255, 0.96) 100%)',
    iconBackground: 'rgba(6, 182, 212, 0.16)',
    iconColor: 'var(--semi-color-cyan-6)',
    shadow: '0 18px 40px rgba(6, 182, 212, 0.12)',
  },
];

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
    icon: <IconHistogram size={18} />,
    value: summaryLoading ? '--' : formatSummaryValue(summary.total_calls),
    unit: t('次'),
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.total_calls)
        : t('已按当前筛选范围汇总'),
  },
  {
    title: t('总费用'),
    icon: <IconCoinMoneyStroked size={18} />,
    value: summaryLoading ? '--' : renderQuota(summary.total_cost),
    unit: '',
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.total_cost)
        : t('已按当前筛选范围汇总'),
  },
  {
    title: t('活跃用户'),
    icon: <IconUserGroup size={18} />,
    value: summaryLoading ? '--' : formatSummaryValue(summary.active_users),
    unit: t('人'),
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.active_users)
        : t('已按当前筛选范围汇总'),
  },
  {
    title: t('活跃模型'),
    icon: <IconLayers size={18} />,
    value: summaryLoading ? '--' : formatSummaryValue(summary.active_models),
    unit: t('个'),
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.active_models)
        : t('当前时间范围内有调用的模型数'),
  },
  {
    title: t('总 Token'),
    icon: <IconTextStroked size={18} />,
    value: summaryLoading ? '--' : formatSummaryValue(summary.total_tokens),
    unit: '',
    helper:
      datePreset === 'last7days'
        ? formatWowText(t, summary.wow?.total_tokens)
        : t('已按当前筛选范围汇总'),
  },
];

const SummaryCards = ({ datePreset, summary, summaryLoading, t }) => (
  <div className='flex flex-wrap gap-3 w-full'>
    {buildSummaryCards({
      summary,
      summaryLoading,
      datePreset,
      t,
    }).map((card, index) => {
      const theme = summaryCardThemes[index % summaryCardThemes.length];

      return (
        <div
          key={card.title}
          className='relative flex min-w-[180px] flex-1 flex-col gap-4 overflow-hidden rounded-[24px] border p-4'
          style={{
            borderColor: theme.borderColor,
            background: theme.background,
            boxShadow: theme.shadow,
          }}
        >
          <div
            className='pointer-events-none absolute right-0 top-0 h-20 w-20 translate-x-4 -translate-y-4 rounded-full blur-2xl'
            style={{ background: theme.iconBackground }}
          />
          <div className='relative z-10 flex items-start justify-between gap-3'>
            <div className='flex flex-col gap-1'>
              <Text strong>{card.title}</Text>
              <div className='flex items-end gap-2'>
                <Title heading={3} style={{ margin: 0 }}>
                  {card.value}
                </Title>
                {card.unit ? <Text type='tertiary'>{card.unit}</Text> : null}
              </div>
            </div>
            <div
              className='flex h-11 w-11 items-center justify-center rounded-2xl border'
              style={{
                background: 'rgba(255, 255, 255, 0.78)',
                borderColor: theme.iconBackground,
                color: theme.iconColor,
              }}
            >
              {card.icon}
            </div>
          </div>
          <div
            className='relative z-10 rounded-2xl border px-3 py-2'
            style={{
              background: 'rgba(255, 255, 255, 0.78)',
              borderColor: theme.iconBackground,
            }}
          >
            <Text type='tertiary' size='small'>
              {card.helper}
            </Text>
          </div>
        </div>
      );
    })}
  </div>
);

export default SummaryCards;
