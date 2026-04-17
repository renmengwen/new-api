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

export default function AdvancedPricingPreview({ selectedModel, previewPayload }) {
  const { t } = useTranslation();

  return (
    <Card title={t('保存预览')}>
      {!selectedModel || !previewPayload ? (
        <Empty
          title={t('暂无预览')}
          description={t('选择模型后即可查看将写入的 AdvancedPricingMode 与 AdvancedPricingRules')}
        />
      ) : (
        <>
          <div className='text-sm text-gray-500 mb-3'>
            {t('下方预览的是当前模型保存后会写入的配置片段。')}
          </div>
          <pre
            style={{
              margin: 0,
              padding: 16,
              borderRadius: 12,
              background: 'var(--semi-color-fill-0)',
              border: '1px solid var(--semi-color-border)',
              overflowX: 'auto',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}
          >
            {JSON.stringify(previewPayload, null, 2)}
          </pre>
        </>
      )}
    </Card>
  );
}
