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
import { useCallback, useEffect } from 'react';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import { CHART_CONFIG } from '../../constants/dashboard.constants';
import { modelColorMap, modelToColor, renderNumber, renderQuota } from '../../helpers';

const FALLBACK_COLORS = [
  '#2563eb',
  '#16a34a',
  '#f59e0b',
  '#dc2626',
  '#7c3aed',
  '#0891b2',
  '#ea580c',
  '#4f46e5',
];

const normalizeSeriesKeys = (keys = []) =>
  Array.from(new Set(keys.filter((key) => key !== null && key !== undefined && key !== '')));

const createSpecifiedColors = (keys = [], initialMap = {}) => {
  const specified = { ...initialMap };
  normalizeSeriesKeys(keys).forEach((key, index) => {
    if (specified[key]) {
      return;
    }
    specified[key] = modelColorMap[key] || modelToColor(key) || FALLBACK_COLORS[index % FALLBACK_COLORS.length];
  });
  return specified;
};

const formatTooltipValue = (valueFormatter, value) => {
  if (typeof valueFormatter === 'function') {
    return valueFormatter(value);
  }
  if (typeof value === 'number') {
    return renderNumber(value);
  }
  return String(value ?? '');
};

export const useOperationsAnalyticsCharts = ({ t }) => {
  useEffect(() => {
    initVChartSemiTheme({
      isWatchingThemeSwitch: true,
    });
  }, []);

  const specLine = useCallback(
    ({
      data = [],
      title,
      subtext = '',
      xField,
      yField,
      seriesField,
      valueFormatter,
      colorMap = {},
      legendVisible = true,
    }) => ({
      type: 'line',
      data: [
        {
          id: 'lineData',
          values: data,
        },
      ],
      xField,
      yField,
      seriesField,
      legends: {
        visible: legendVisible,
      },
      point: {
        visible: true,
        style: {
          size: 5,
        },
      },
      title: {
        visible: true,
        text: title,
        subtext,
      },
      tooltip: {
        mark: {
          content: [
            {
              key: (datum) =>
                seriesField ? datum?.[seriesField] || title : t('指标'),
              value: (datum) =>
                formatTooltipValue(valueFormatter, datum?.[yField]),
            },
          ],
        },
      },
      color: {
        specified: createSpecifiedColors(
          data.map((item) => item?.[seriesField]).filter(Boolean),
          colorMap,
        ),
      },
    }),
    [t],
  );

  const specBar = useCallback(
    ({
      data = [],
      title,
      subtext = '',
      xField,
      yField,
      seriesField,
      categoryField = seriesField || yField,
      valueField = xField,
      valueFormatter,
      colorMap = {},
      legendVisible = false,
    }) => ({
      type: 'bar',
      data: [
        {
          id: 'barData',
          values: data,
        },
      ],
      xField,
      yField,
      seriesField: seriesField || yField,
      legends: {
        visible: legendVisible,
      },
      title: {
        visible: true,
        text: title,
        subtext,
      },
      bar: {
        state: {
          hover: {
            stroke: '#000',
            lineWidth: 1,
          },
        },
      },
      tooltip: {
        mark: {
          content: [
            {
              key: (datum) =>
                seriesField
                  ? datum?.[seriesField] || datum?.[categoryField] || title
                  : datum?.[categoryField] || title,
              value: (datum) =>
                formatTooltipValue(valueFormatter, datum?.[valueField]),
            },
          ],
        },
      },
      color: {
        specified: createSpecifiedColors(
          data.map((item) => item?.[seriesField || categoryField]).filter(Boolean),
          colorMap,
        ),
      },
    }),
    [],
  );

  const specPie = useCallback(
    ({
      data = [],
      title,
      subtext = '',
      categoryField = 'type',
      valueField = 'value',
      valueFormatter,
      colorMap = {},
    }) => ({
      type: 'pie',
      data: [
        {
          id: 'pieData',
          values: data,
        },
      ],
      valueField,
      categoryField,
      outerRadius: 0.8,
      innerRadius: 0.48,
      padAngle: 0.6,
      pie: {
        style: {
          cornerRadius: 8,
        },
      },
      title: {
        visible: true,
        text: title,
        subtext,
      },
      legends: {
        visible: true,
        orient: 'left',
      },
      label: {
        visible: true,
      },
      tooltip: {
        mark: {
          content: [
            {
              key: (datum) => datum?.[categoryField],
              value: (datum) =>
                formatTooltipValue(valueFormatter, datum?.[valueField]),
            },
          ],
        },
      },
      color: {
        specified: createSpecifiedColors(
          data.map((item) => item?.[categoryField]).filter(Boolean),
          colorMap,
        ),
      },
    }),
    [],
  );

  return {
    chartOption: CHART_CONFIG,
    specLine,
    specBar,
    specPie,
  };
};

export default useOperationsAnalyticsCharts;
