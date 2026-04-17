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

import { useEffect, useMemo, useRef, useState } from 'react';

import { API, showError, showSuccess } from '../../../../helpers';
import { BILLING_MODE_PER_TOKEN } from './modelPricingEditorHelpers';
import {
  RULE_TYPE_TEXT_SEGMENT,
  buildAdvancedPricingState,
  buildRuleDraft,
  parseOptionJSON,
  reduceOptionsByKey,
} from './advancedPricingRulesStateHelpers';

export default function useAdvancedPricingRulesState({
  options,
  refresh,
  t,
  initialModelName = '',
  initialModelSelectionKey = 0,
}) {
  const [enabledModelNames, setEnabledModelNames] = useState([]);
  const [launchModelName, setLaunchModelName] = useState('');
  const [models, setModels] = useState([]);
  const [searchText, setSearchText] = useState('');
  const [selectedModelName, setSelectedModelName] = useState('');
  const [draftRules, setDraftRules] = useState({});
  const [draftBillingModes, setDraftBillingModes] = useState({});
  const [saving, setSaving] = useState(false);
  const draftRulesRef = useRef({});
  const draftBillingModesRef = useRef({});
  const selectedModelNameRef = useRef('');

  useEffect(() => {
    draftRulesRef.current = draftRules;
  }, [draftRules]);

  useEffect(() => {
    draftBillingModesRef.current = draftBillingModes;
  }, [draftBillingModes]);

  useEffect(() => {
    selectedModelNameRef.current = selectedModelName;
  }, [selectedModelName]);

  useEffect(() => {
    let active = true;

    const loadEnabledModels = async () => {
      try {
        const res = await API.get('/api/channel/models_enabled');
        const { success, message, data } = res?.data || {};
        if (!active) {
          return;
        }
        if (success) {
          setEnabledModelNames(Array.isArray(data) ? data.filter(Boolean) : []);
          return;
        }
        showError(message || 'Failed to load enabled models');
        setEnabledModelNames([]);
      } catch (error) {
        if (!active) {
          return;
        }
        console.error('Failed to load enabled models:', error);
        showError('Failed to load enabled models');
        setEnabledModelNames([]);
      }
    };

    loadEnabledModels();

    return () => {
      active = false;
    };
  }, [t]);

  useEffect(() => {
    if (!initialModelSelectionKey) {
      return;
    }
    setLaunchModelName(initialModelName || '');
  }, [initialModelName, initialModelSelectionKey]);

  useEffect(() => {
    if (launchModelName && selectedModelName === launchModelName) {
      setLaunchModelName('');
    }
  }, [launchModelName, selectedModelName]);

  useEffect(() => {
    const nextState = buildAdvancedPricingState({
      options,
      enabledModelNames,
      launchModelName,
      previousDraftRules: draftRulesRef.current,
      previousDraftBillingModes: draftBillingModesRef.current,
      previousSelectedModelName: selectedModelNameRef.current,
    });

    draftRulesRef.current = nextState.draftRules;
    draftBillingModesRef.current = nextState.draftBillingModes;
    selectedModelNameRef.current = nextState.selectedModelName;

    setModels(nextState.models);
    setDraftRules(nextState.draftRules);
    setDraftBillingModes(nextState.draftBillingModes);
    setSelectedModelName(nextState.selectedModelName);
  }, [enabledModelNames, launchModelName, options]);

  const filteredModels = useMemo(() => {
    const keyword = searchText.trim().toLowerCase();

    return models.filter((model) => {
      if (!keyword) {
        return true;
      }
      return model.name.toLowerCase().includes(keyword);
    });
  }, [models, searchText]);

  const selectedModel = useMemo(
    () => models.find((model) => model.name === selectedModelName) || null,
    [models, selectedModelName],
  );

  const selectedRule = draftRules[selectedModelName] || buildRuleDraft(RULE_TYPE_TEXT_SEGMENT);
  const currentRuleType = selectedRule.rule_type || RULE_TYPE_TEXT_SEGMENT;
  const currentBillingMode = selectedModel?.billingMode || BILLING_MODE_PER_TOKEN;
  const draftBillingMode = draftBillingModes[selectedModelName] || currentBillingMode;

  const previewPayload = selectedModel
    ? {
        AdvancedPricingMode: {
          [selectedModel.name]: draftBillingMode,
        },
        AdvancedPricingRules: {
          [selectedModel.name]: selectedRule,
        },
      }
    : null;

  const updateSelectedRuleType = (ruleType) => {
    if (!selectedModelName) {
      return;
    }

    const nextDraftRules = {
      ...draftRulesRef.current,
      [selectedModelName]: buildRuleDraft(ruleType, draftRulesRef.current[selectedModelName]),
    };

    draftRulesRef.current = nextDraftRules;
    setDraftRules(nextDraftRules);
  };

  const updateSelectedRuleField = (field, value) => {
    if (!selectedModelName) {
      return;
    }

    const nextDraftRules = {
      ...draftRulesRef.current,
      [selectedModelName]: {
        ...buildRuleDraft(currentRuleType, draftRulesRef.current[selectedModelName]),
        [field]: value,
      },
    };

    draftRulesRef.current = nextDraftRules;
    setDraftRules(nextDraftRules);
  };

  const updateSelectedBillingMode = (billingMode) => {
    if (!selectedModelName) {
      return;
    }

    const nextDraftBillingModes = {
      ...draftBillingModesRef.current,
      [selectedModelName]: billingMode,
    };

    draftBillingModesRef.current = nextDraftBillingModes;
    setDraftBillingModes(nextDraftBillingModes);
  };

  const saveSelectedRule = async () => {
    if (!selectedModel) {
      showError(t('请先选择一个模型'));
      return false;
    }

    setSaving(true);
    try {
      const latestOptionsRes = await API.get('/api/option/');
      const {
        success: latestOptionsSuccess,
        message: latestOptionsMessage,
        data: latestOptionsData,
      } = latestOptionsRes?.data || {};

      if (!latestOptionsSuccess) {
        throw new Error(latestOptionsMessage || t('获取配置失败'));
      }

      const latestOptionsByKey = reduceOptionsByKey(latestOptionsData);
      const nextModeMap = parseOptionJSON(latestOptionsByKey.AdvancedPricingMode);
      const nextRulesMap = parseOptionJSON(latestOptionsByKey.AdvancedPricingRules);

      nextModeMap[selectedModel.name] = draftBillingMode;
      nextRulesMap[selectedModel.name] = selectedRule;

      const results = await Promise.all([
        API.put('/api/option/', {
          key: 'AdvancedPricingMode',
          value: JSON.stringify(nextModeMap, null, 2),
        }),
        API.put('/api/option/', {
          key: 'AdvancedPricingRules',
          value: JSON.stringify(nextRulesMap, null, 2),
        }),
      ]);

      for (const res of results) {
        if (!res?.data?.success) {
          throw new Error(res?.data?.message || t('保存失败，请重试'));
        }
      }

      showSuccess(t('高级定价规则已保存'));
      await refresh();
      return true;
    } catch (error) {
      console.error('保存高级定价规则失败:', error);
      showError(error.message || t('保存失败，请重试'));
      return false;
    } finally {
      setSaving(false);
    }
  };

  return {
    models,
    filteredModels,
    searchText,
    setSearchText,
    selectedModel,
    selectedModelName,
    setSelectedModelName,
    selectedRule,
    currentRuleType,
    currentBillingMode,
    draftBillingMode,
    updateSelectedRuleType,
    updateSelectedRuleField,
    updateSelectedBillingMode,
    previewPayload,
    saveSelectedRule,
    saving,
  };
}
