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
  buildAdvancedPricingSavePayload,
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
  const dirtyRuleModelNamesRef = useRef(new Set());
  const dirtyBillingModeModelNamesRef = useRef(new Set());
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
      preserveDraftRuleModelNames: dirtyRuleModelNamesRef.current,
      preserveDraftBillingModeModelNames: dirtyBillingModeModelNamesRef.current,
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
  const previewPayload = useMemo(() => {
    if (!selectedModel) {
      return null;
    }

    try {
      return buildAdvancedPricingSavePayload({
        modelName: selectedModel.name,
        billingMode: draftBillingMode,
        draftRule: selectedRule,
      }).previewPayload;
    } catch (error) {
      return null;
    }
  }, [draftBillingMode, selectedModel, selectedRule]);

  const updateSelectedRuleType = (ruleType) => {
    if (!selectedModelName) {
      return;
    }

    dirtyRuleModelNamesRef.current.add(selectedModelName);

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

    dirtyRuleModelNamesRef.current.add(selectedModelName);

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

    dirtyBillingModeModelNamesRef.current.add(selectedModelName);

    const nextDraftBillingModes = {
      ...draftBillingModesRef.current,
      [selectedModelName]: billingMode,
    };

    draftBillingModesRef.current = nextDraftBillingModes;
    setDraftBillingModes(nextDraftBillingModes);
  };

  const saveSelectedRule = async () => {
    if (!selectedModel) {
      showError(t('请先选择模型！'));
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
        throw new Error(latestOptionsMessage || t('保存失败，请重试'));
      }

      const latestOptionsByKey = reduceOptionsByKey(latestOptionsData);
      const savePayload = buildAdvancedPricingSavePayload({
        modelName: selectedModel.name,
        billingMode: draftBillingMode,
        draftRule: selectedRule,
        latestModeMap: parseOptionJSON(latestOptionsByKey.AdvancedPricingMode),
        latestRulesMap: parseOptionJSON(latestOptionsByKey.AdvancedPricingRules),
      });

      const saveRes = await API.put('/api/option/', {
        key: 'AdvancedPricingConfig',
        value: savePayload.optionValue,
      });

      if (!saveRes?.data?.success) {
        throw new Error(saveRes?.data?.message || t('保存失败，请重试'));
      }

      showSuccess(t('保存成功'));
      dirtyRuleModelNamesRef.current.delete(selectedModel.name);
      dirtyBillingModeModelNamesRef.current.delete(selectedModel.name);
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
