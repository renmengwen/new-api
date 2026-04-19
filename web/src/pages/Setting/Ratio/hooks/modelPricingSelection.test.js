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
  resolveModelPricingBridgeSelection,
  resolveModelPricingSelectedModelName,
  resolveModelPricingSelectionPage,
} from './modelPricingSelection.js';

test('resolveModelPricingSelectedModelName applies the initial selection only once per version', () => {
  assert.deepEqual(
    resolveModelPricingSelectedModelName({
      currentSelectedModelName: 'alpha',
      modelNames: ['alpha', 'beta'],
      initialSelectedModelName: 'beta',
      initialSelectionVersion: 2,
      lastAppliedInitialSelectionVersion: 1,
    }),
    {
      nextSelectedModelName: 'beta',
      nextAppliedInitialSelectionVersion: 2,
      shouldSyncSelection: true,
    },
  );

  assert.deepEqual(
    resolveModelPricingSelectedModelName({
      currentSelectedModelName: 'alpha',
      modelNames: ['alpha', 'beta'],
      initialSelectedModelName: 'beta',
      initialSelectionVersion: 2,
      lastAppliedInitialSelectionVersion: 2,
    }),
    {
      nextSelectedModelName: 'alpha',
      nextAppliedInitialSelectionVersion: 2,
      shouldSyncSelection: false,
    },
  );
});

test('resolveModelPricingSelectedModelName preserves the current selection or falls back to the first visible model', () => {
  assert.deepEqual(
    resolveModelPricingSelectedModelName({
      currentSelectedModelName: 'beta',
      modelNames: ['alpha', 'beta', 'gamma'],
      initialSelectedModelName: '',
      initialSelectionVersion: 0,
      lastAppliedInitialSelectionVersion: null,
    }),
    {
      nextSelectedModelName: 'beta',
      nextAppliedInitialSelectionVersion: null,
      shouldSyncSelection: false,
    },
  );

  assert.deepEqual(
    resolveModelPricingSelectedModelName({
      currentSelectedModelName: 'missing',
      modelNames: ['alpha', 'beta', 'gamma'],
      initialSelectedModelName: '',
      initialSelectionVersion: 0,
      lastAppliedInitialSelectionVersion: null,
    }),
    {
      nextSelectedModelName: 'alpha',
      nextAppliedInitialSelectionVersion: null,
      shouldSyncSelection: false,
    },
  );
});

test('resolveModelPricingSelectionPage returns the selected model page', () => {
  assert.equal(
    resolveModelPricingSelectionPage({
      modelNames: ['alpha', 'beta', 'gamma', 'delta', 'epsilon'],
      selectedModelName: 'delta',
      pageSize: 2,
    }),
    2,
  );

  assert.equal(
    resolveModelPricingSelectionPage({
      modelNames: ['alpha', 'beta'],
      selectedModelName: 'missing',
      pageSize: 10,
    }),
    1,
  );
});

test('resolveModelPricingBridgeSelection clears active filters and keeps the target page pending when bridge sync hits', () => {
  assert.deepEqual(
    resolveModelPricingBridgeSelection({
      shouldSyncSelection: true,
      modelNames: ['alpha', 'beta', 'gamma', 'delta'],
      selectedModelName: 'delta',
      pageSize: 2,
      searchText: 'del',
      conflictOnly: true,
    }),
    {
      shouldResetSearchText: true,
      shouldResetConflictOnly: true,
      pendingSelectionPage: 2,
      nextCurrentPage: null,
    },
  );
});

test('resolveModelPricingBridgeSelection preserves uncontrolled filter behavior when bridge sync does not hit', () => {
  assert.deepEqual(
    resolveModelPricingBridgeSelection({
      shouldSyncSelection: false,
      modelNames: ['alpha', 'beta', 'gamma', 'delta'],
      selectedModelName: 'delta',
      pageSize: 2,
      searchText: 'del',
      conflictOnly: true,
    }),
    {
      shouldResetSearchText: false,
      shouldResetConflictOnly: false,
      pendingSelectionPage: null,
      nextCurrentPage: null,
    },
  );
});
