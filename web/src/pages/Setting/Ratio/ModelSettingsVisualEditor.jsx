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
import { Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../../helpers';
import ModelPricingEditor from './components/ModelPricingEditor';

const FALLBACK_MODEL_OPTION_KEYS = [
  'AdvancedPricingMode',
  'AdvancedPricingRules',
  'ModelPrice',
  'ModelRatio',
  'CompletionRatio',
  'CompletionRatioMeta',
  'CacheRatio',
  'CreateCacheRatio',
  'ImageRatio',
  'AudioRatio',
  'AudioCompletionRatio',
];

const parseOptionMap = (rawValue) => {
  if (!rawValue || typeof rawValue !== 'string') {
    return {};
  }

  try {
    const parsedValue = JSON.parse(rawValue);
    return parsedValue &&
      typeof parsedValue === 'object' &&
      !Array.isArray(parsedValue)
      ? parsedValue
      : {};
  } catch {
    return {};
  }
};

const buildFallbackEnabledModelNames = ({ options, initialModelName = '' }) => {
  const names = new Set();

  if (initialModelName) {
    names.add(initialModelName);
  }

  const parsedAdvancedPricingConfig = parseOptionMap(options?.AdvancedPricingConfig);
  const canonicalBillingModeMap =
    parsedAdvancedPricingConfig.billing_mode &&
    typeof parsedAdvancedPricingConfig.billing_mode === 'object' &&
    !Array.isArray(parsedAdvancedPricingConfig.billing_mode)
      ? parsedAdvancedPricingConfig.billing_mode
      : {};
  const canonicalRulesMap =
    parsedAdvancedPricingConfig.rules &&
    typeof parsedAdvancedPricingConfig.rules === 'object' &&
    !Array.isArray(parsedAdvancedPricingConfig.rules)
      ? parsedAdvancedPricingConfig.rules
      : {};

  Object.keys(canonicalBillingModeMap).forEach((modelName) => names.add(modelName));
  Object.keys(canonicalRulesMap).forEach((modelName) => names.add(modelName));

  FALLBACK_MODEL_OPTION_KEYS.forEach((key) => {
    Object.keys(parseOptionMap(options?.[key])).forEach((modelName) =>
      names.add(modelName),
    );
  });

  return Array.from(names)
    .filter(Boolean)
    .sort((leftName, rightName) => leftName.localeCompare(rightName));
};

export default function ModelSettingsVisualEditor(props) {
  const { t } = useTranslation();
  const [enabledModels, setEnabledModels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [shouldUseFallbackEnabledModels, setShouldUseFallbackEnabledModels] =
    useState(false);
  const resolvedEnabledModels = shouldUseFallbackEnabledModels
    ? buildFallbackEnabledModelNames({
        options: props.options,
        initialModelName: props.initialModelName,
      })
    : enabledModels;

  useEffect(() => {
    let active = true;

    const loadEnabledModels = async () => {
      const fallbackEnabledModels = buildFallbackEnabledModelNames({
        options: props.options,
        initialModelName: props.initialModelName,
      });
      setLoading(true);
      try {
        const res = await API.get('/api/channel/models_enabled');
        const { success, message, data } = res.data;
        if (!active) {
          return;
        }
        if (success) {
          setShouldUseFallbackEnabledModels(false);
          setEnabledModels(Array.isArray(data) ? data : []);
        } else {
          setShouldUseFallbackEnabledModels(true);
          setEnabledModels(fallbackEnabledModels);
          showError(message);
        }
      } catch (error) {
        if (!active) {
          return;
        }
        setShouldUseFallbackEnabledModels(true);
        setEnabledModels(fallbackEnabledModels);
        console.error(t('获取启用模型失败:'), error);
        showError(t('获取启用模型失败'));
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };

    loadEnabledModels();

    return () => {
      active = false;
    };
  }, [props.initialModelName, props.options, t]);

  if (loading) {
    return (
      <Spin spinning={true}>
        <div style={{ minHeight: 160 }} />
      </Spin>
    );
  }

  return (
    <ModelPricingEditor
      options={props.options}
      refresh={props.refresh}
      candidateModelNames={resolvedEnabledModels}
      filterMode='enabled'
      initialSelectedModelName={props.initialModelName}
      initialSelectionVersion={props.initialModelSelectionKey}
      allowAddModel={false}
      onEditAdvancedRules={(model) => props.onOpenAdvancedPricingRules?.(model)}
      listDescription={t(
        '此页面仅显示渠道管理中已配置且已启用的模型，未启用模型的价格配置会继续保留。',
      )}
      emptyTitle={t('没有已启用的模型')}
      emptyDescription={t('当前渠道管理中没有已配置且已启用的模型')}
    />
  );
}
