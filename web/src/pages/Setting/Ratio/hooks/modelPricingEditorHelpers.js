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

export const BILLING_MODE_PER_TOKEN = 'per_token';
export const BILLING_MODE_PER_REQUEST = 'per_request';
export const BILLING_MODE_ADVANCED = 'advanced';
export const BILLING_MODE_CHANGE_CONFIRM_TITLE = '确认切换当前生效模式？';
export const BILLING_MODE_CHANGE_CONFIRM_CONTENT =
  '切换不会删除另一套配置；保存后，新请求会按新的计费模式结算，历史日志和账单不会回滚。';

const VALID_BILLING_MODES = new Set([
  BILLING_MODE_PER_TOKEN,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_ADVANCED,
]);

export const hasValue = (value) =>
  value !== '' && value !== null && value !== undefined && value !== false;

export const resolveBillingMode = ({
  explicitMode,
  fixedPrice,
  advancedRuleType,
}) => {
  const hasInvalidExplicitAdvancedMode =
    explicitMode === BILLING_MODE_ADVANCED && !hasValue(advancedRuleType);
  const hasExplicitBillingMode =
    VALID_BILLING_MODES.has(explicitMode) && !hasInvalidExplicitAdvancedMode;
  const billingMode = hasExplicitBillingMode
    ? explicitMode
    : hasValue(fixedPrice)
      ? BILLING_MODE_PER_REQUEST
      : BILLING_MODE_PER_TOKEN;

  return {
    billingMode,
    explicitBillingMode: hasExplicitBillingMode ? explicitMode : '',
    hasExplicitBillingMode,
    hasInvalidExplicitAdvancedMode,
  };
};

export const canUseAdvancedBilling = (model) =>
  hasValue(model?.advancedRuleType);

export const hasEditableFixedPricingConfig = (model) =>
  Boolean(
    model &&
      [
        model.fixedPrice,
        model.inputPrice,
        model.completionPrice,
        model.cachePrice,
        model.createCachePrice,
        model.imagePrice,
        model.audioInputPrice,
        model.audioOutputPrice,
      ].some(hasValue),
  );

export const isBasePricingUnset = (model) =>
  !hasValue(model?.fixedPrice) &&
  !hasValue(model?.inputPrice) &&
  !canUseAdvancedBilling(model);

export const resolveBatchBillingModeConfirmation = ({
  selectedModel,
  selectedModelNames = [],
  models = [],
}) => {
  const normalizedModelNames = Array.isArray(selectedModelNames)
    ? selectedModelNames
    : [];
  const normalizedModels = Array.isArray(models) ? models : [];

  if (!selectedModel?.billingMode || normalizedModelNames.length === 0) {
    return {
      requiresConfirmation: false,
      changedModelNames: [],
      title: BILLING_MODE_CHANGE_CONFIRM_TITLE,
      content: BILLING_MODE_CHANGE_CONFIRM_CONTENT,
    };
  }

  const changedModelNames = normalizedModelNames.filter((modelName) => {
    const targetModel = normalizedModels.find((model) => model.name === modelName);
    return targetModel && targetModel.billingMode !== selectedModel.billingMode;
  });

  return {
    requiresConfirmation: changedModelNames.length > 0,
    changedModelNames,
    title: BILLING_MODE_CHANGE_CONFIRM_TITLE,
    content: BILLING_MODE_CHANGE_CONFIRM_CONTENT,
  };
};

export const shouldPersistAdvancedPricingMode = ({
  model,
  dirtyModeNames = [],
}) => {
  if (!model?.name) {
    return false;
  }

  const dirtySet =
    dirtyModeNames instanceof Set ? dirtyModeNames : new Set(dirtyModeNames);

  return Boolean(model.hasExplicitBillingMode) || dirtySet.has(model.name);
};

export const buildAdvancedPricingModePayload = ({
  latestModeMap,
  latestRulesMap,
  models,
  dirtyModeNames = [],
}) => {
  const dirtySet =
    dirtyModeNames instanceof Set ? dirtyModeNames : new Set(dirtyModeNames);
  const normalizedLatestModeMap =
    latestModeMap &&
    typeof latestModeMap === 'object' &&
    !Array.isArray(latestModeMap)
      ? { ...latestModeMap }
      : {};
  const normalizedLatestRulesMap =
    latestRulesMap &&
    typeof latestRulesMap === 'object' &&
    !Array.isArray(latestRulesMap)
      ? latestRulesMap
      : {};

  models.forEach((model) => {
    const latestAdvancedModeIsValid = canUseAdvancedBilling({
      advancedRuleType: normalizedLatestRulesMap[model.name]?.rule_type,
    });
    const latestExplicitMode = normalizedLatestModeMap[model.name];

    if (
      latestExplicitMode === BILLING_MODE_ADVANCED &&
      !latestAdvancedModeIsValid
    ) {
      delete normalizedLatestModeMap[model.name];
    }

    if (!shouldPersistAdvancedPricingMode({ model, dirtyModeNames: dirtySet })) {
      return;
    }

    if (model.billingMode === BILLING_MODE_ADVANCED) {
      if (latestAdvancedModeIsValid) {
        normalizedLatestModeMap[model.name] = BILLING_MODE_ADVANCED;
      }
      return;
    }

    if (
      !dirtySet.has(model.name) &&
      Object.prototype.hasOwnProperty.call(normalizedLatestModeMap, model.name)
    ) {
      return;
    }

    normalizedLatestModeMap[model.name] = model.billingMode;
  });

  return normalizedLatestModeMap;
};
