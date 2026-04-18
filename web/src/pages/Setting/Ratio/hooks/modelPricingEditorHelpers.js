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

export const isBasePricingUnset = (model) =>
  !hasValue(model?.fixedPrice) &&
  !hasValue(model?.inputPrice) &&
  !canUseAdvancedBilling(model);

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
