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

export function resolveModelPricingSelectedModelName({
  currentSelectedModelName,
  modelNames,
  initialSelectedModelName = '',
  initialSelectionVersion = 0,
  lastAppliedInitialSelectionVersion = null,
}) {
  if (!modelNames.length) {
    return {
      nextSelectedModelName: '',
      nextAppliedInitialSelectionVersion: lastAppliedInitialSelectionVersion,
      shouldSyncSelection: false,
    };
  }

  if (
    initialSelectedModelName &&
    initialSelectionVersion !== lastAppliedInitialSelectionVersion &&
    modelNames.includes(initialSelectedModelName)
  ) {
    return {
      nextSelectedModelName: initialSelectedModelName,
      nextAppliedInitialSelectionVersion: initialSelectionVersion,
      shouldSyncSelection: true,
    };
  }

  if (modelNames.includes(currentSelectedModelName)) {
    return {
      nextSelectedModelName: currentSelectedModelName,
      nextAppliedInitialSelectionVersion: lastAppliedInitialSelectionVersion,
      shouldSyncSelection: false,
    };
  }

  return {
    nextSelectedModelName: modelNames[0],
    nextAppliedInitialSelectionVersion: lastAppliedInitialSelectionVersion,
    shouldSyncSelection: false,
  };
}

export function resolveModelPricingSelectionPage({
  modelNames,
  selectedModelName,
  pageSize,
}) {
  const selectedIndex = modelNames.indexOf(selectedModelName);

  if (selectedIndex === -1) {
    return 1;
  }

  return Math.floor(selectedIndex / pageSize) + 1;
}

export function resolveModelPricingBridgeSelection({
  shouldSyncSelection,
  modelNames,
  selectedModelName,
  pageSize,
  searchText = '',
  conflictOnly = false,
}) {
  if (!shouldSyncSelection) {
    return {
      shouldResetSearchText: false,
      shouldResetConflictOnly: false,
      pendingSelectionPage: null,
      nextCurrentPage: null,
    };
  }

  const nextSelectionPage = resolveModelPricingSelectionPage({
    modelNames,
    selectedModelName,
    pageSize,
  });

  if (searchText || conflictOnly) {
    return {
      shouldResetSearchText: Boolean(searchText),
      shouldResetConflictOnly: Boolean(conflictOnly),
      pendingSelectionPage: nextSelectionPage,
      nextCurrentPage: null,
    };
  }

  return {
    shouldResetSearchText: false,
    shouldResetConflictOnly: false,
    pendingSelectionPage: null,
    nextCurrentPage: nextSelectionPage,
  };
}
