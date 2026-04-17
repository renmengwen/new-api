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

import React from 'react';
import { Card, Empty } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

import { useIsMobile } from '../../../hooks/common/useIsMobile';
import AdvancedPricingModelList from './components/advanced-pricing/AdvancedPricingModelList';
import AdvancedPricingSummary from './components/advanced-pricing/AdvancedPricingSummary';
import AdvancedPricingPreview from './components/advanced-pricing/AdvancedPricingPreview';
import MediaTaskRuleEditor from './components/advanced-pricing/MediaTaskRuleEditor';
import TextSegmentRuleEditor from './components/advanced-pricing/TextSegmentRuleEditor';
import useAdvancedPricingRulesState from './hooks/useAdvancedPricingRulesState';

const RULE_TYPE_TEXT_SEGMENT = 'text_segment';
const RULE_TYPE_MEDIA_TASK = 'media_task';

export default function AdvancedPricingRulesPage({
  options,
  refresh,
  initialModelName = '',
  initialModelSelectionKey = 0,
}) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const {
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
  } = useAdvancedPricingRulesState({
    options,
    refresh,
    t,
    initialModelName,
    initialModelSelectionKey,
  });

  return (
    <div
      style={{
        display: 'grid',
        gap: 16,
        gridTemplateColumns: isMobile
          ? 'minmax(0, 1fr)'
          : 'minmax(280px, 320px) minmax(0, 1fr)',
      }}
    >
      <AdvancedPricingModelList
        models={filteredModels}
        searchText={searchText}
        onSearchTextChange={setSearchText}
        selectedModelName={selectedModelName}
        onSelectModel={setSelectedModelName}
      />
      <div style={{ display: 'grid', gap: 16 }}>
        <AdvancedPricingSummary
          selectedModel={selectedModel}
          currentBillingMode={currentBillingMode}
          draftBillingMode={draftBillingMode}
          currentRuleType={currentRuleType}
          onBillingModeChange={updateSelectedBillingMode}
          onSave={saveSelectedRule}
          saving={saving}
        />
        {!selectedModel ? (
          <Card>
            <Empty
              title={t('尚未选择模型')}
              description={t('从左侧选择模型后，这里会显示规则编辑器')}
            />
          </Card>
        ) : currentRuleType === RULE_TYPE_MEDIA_TASK ? (
          <MediaTaskRuleEditor
            rule={selectedRule}
            onRuleTypeChange={updateSelectedRuleType}
            onRuleFieldChange={updateSelectedRuleField}
          />
        ) : (
          <TextSegmentRuleEditor
            rule={selectedRule}
            onRuleTypeChange={updateSelectedRuleType}
            onRuleFieldChange={updateSelectedRuleField}
          />
        )}
        <AdvancedPricingPreview
          selectedModel={selectedModel}
          previewPayload={previewPayload}
        />
      </div>
    </div>
  );
}
