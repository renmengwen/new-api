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

import test from 'node:test';
import assert from 'node:assert/strict';

import {
  resolveInitialVisibleModelNames,
  resolveVisibleModels,
} from './modelPricingVisibility.js';

const models = [
  { name: 'enabled-priced', fixedPrice: '1', inputPrice: '' },
  { name: 'hidden-priced', fixedPrice: '2', inputPrice: '' },
  { name: 'enabled-unset', fixedPrice: '', inputPrice: '' },
];

test('resolveInitialVisibleModelNames keeps only channel-enabled models in enabled mode', () => {
  const result = resolveInitialVisibleModelNames({
    nextModels: models,
    filterMode: 'enabled',
    candidateModelNames: ['enabled-priced', 'enabled-unset'],
    isBasePricingUnset: (model) =>
      model.fixedPrice === '' && model.inputPrice === '',
  });

  assert.deepEqual(result, ['enabled-priced', 'enabled-unset']);
});

test('resolveVisibleModels hides non-candidate models in enabled mode', () => {
  const result = resolveVisibleModels({
    models,
    filterMode: 'enabled',
    candidateModelNames: ['enabled-priced', 'enabled-unset'],
    initialVisibleModelNames: [],
  });

  assert.deepEqual(
    result.map((model) => model.name),
    ['enabled-priced', 'enabled-unset'],
  );
});

test('resolveVisibleModels preserves the existing unset snapshot semantics', () => {
  const result = resolveVisibleModels({
    models,
    filterMode: 'unset',
    candidateModelNames: [],
    initialVisibleModelNames: ['enabled-unset'],
  });

  assert.deepEqual(result.map((model) => model.name), ['enabled-unset']);
});

test('unset mode keeps only channel-enabled models from the initial unset snapshot', () => {
  const result = resolveInitialVisibleModelNames({
    nextModels: models,
    filterMode: 'unset',
    candidateModelNames: ['enabled-unset'],
    isBasePricingUnset: (model) =>
      model.fixedPrice === '' && model.inputPrice === '',
  });

  assert.deepEqual(result, ['enabled-unset']);
});

test('resolveVisibleModels intersects unset models with channel-enabled candidates', () => {
  const result = resolveVisibleModels({
    models,
    filterMode: 'unset',
    candidateModelNames: ['enabled-unset'],
    initialVisibleModelNames: ['enabled-unset', 'hidden-priced'],
  });

  assert.deepEqual(result.map((model) => model.name), ['enabled-unset']);
});
