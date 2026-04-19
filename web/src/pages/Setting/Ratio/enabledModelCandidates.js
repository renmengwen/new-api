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

export const buildFallbackEnabledModelNames = ({
  options,
  initialModelName = '',
}) => {
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
