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

const toNameSet = (candidateModelNames = []) => new Set(candidateModelNames);

export const resolveInitialVisibleModelNames = ({
  nextModels,
  filterMode,
  candidateModelNames = [],
  isBasePricingUnset,
}) => {
  if (filterMode === 'unset') {
    const enabledNames = toNameSet(candidateModelNames);
    return nextModels
      .filter(
        (model) =>
          isBasePricingUnset(model) &&
          (!candidateModelNames.length || enabledNames.has(model.name)),
      )
      .map((model) => model.name);
  }

  if (filterMode === 'enabled') {
    const enabledNames = toNameSet(candidateModelNames);
    return nextModels
      .filter((model) => enabledNames.has(model.name))
      .map((model) => model.name);
  }

  return nextModels.map((model) => model.name);
};

export const resolveVisibleModels = ({
  models,
  filterMode,
  candidateModelNames = [],
  initialVisibleModelNames = [],
}) => {
  if (filterMode === 'unset') {
    const enabledNames = toNameSet(candidateModelNames);
    return models.filter(
      (model) =>
        initialVisibleModelNames.includes(model.name) &&
        (!candidateModelNames.length || enabledNames.has(model.name)),
    );
  }

  if (filterMode === 'enabled') {
    const enabledNames = toNameSet(candidateModelNames);
    return models.filter((model) => enabledNames.has(model.name));
  }

  return models;
};
