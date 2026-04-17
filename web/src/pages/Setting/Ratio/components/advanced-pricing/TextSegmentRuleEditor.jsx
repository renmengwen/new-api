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
import { Card, Input, Radio, RadioGroup } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const { TextArea } = Input;

export default function TextSegmentRuleEditor({
  rule,
  onRuleTypeChange,
  onRuleFieldChange,
}) {
  const { t } = useTranslation();

  return (
    <Card title={t('文本分段规则')}>
      <div className='mb-4'>
        <div className='mb-2 font-medium'>{t('规则类型')}</div>
        <RadioGroup
          type='button'
          value='text_segment'
          onChange={(event) => onRuleTypeChange(event.target.value)}
        >
          <Radio value='text_segment'>{t('文本分段规则')}</Radio>
          <Radio value='media_task'>{t('媒体任务规则')}</Radio>
        </RadioGroup>
      </div>
      <div className='mb-4'>
        <div className='mb-2 font-medium'>{t('规则名称')}</div>
        <Input
          value={rule.display_name || ''}
          placeholder={t('例如：长文本阶梯定价')}
          onChange={(value) => onRuleFieldChange('display_name', value)}
        />
      </div>
      <div className='mb-4'>
        <div className='mb-2 font-medium'>{t('分段依据')}</div>
        <Input
          value={rule.segment_basis || ''}
          placeholder={t('例如：token / 字符 / 秒')}
          onChange={(value) => onRuleFieldChange('segment_basis', value)}
        />
      </div>
      <div className='mb-4'>
        <div className='mb-2 font-medium'>{t('计费单位')}</div>
        <Input
          value={rule.billing_unit || ''}
          placeholder={t('例如：1K tokens')}
          onChange={(value) => onRuleFieldChange('billing_unit', value)}
        />
      </div>
      <div className='mb-4'>
        <div className='mb-2 font-medium'>{t('默认单价')}</div>
        <Input
          value={rule.default_price || ''}
          placeholder={t('例如：0.0012')}
          onChange={(value) => onRuleFieldChange('default_price', value)}
        />
      </div>
      <div className='mb-4'>
        <div className='mb-2 font-medium'>{t('分段示例')}</div>
        <TextArea
          rows={5}
          value={rule.segments_text || ''}
          placeholder={t('例如：0-1000: 0.001\n1000-8000: 0.0008')}
          onChange={(value) => onRuleFieldChange('segments_text', value)}
        />
      </div>
      <div>
        <div className='mb-2 font-medium'>{t('备注')}</div>
        <TextArea
          rows={3}
          value={rule.note || ''}
          placeholder={t('补充当前文本分段规则的适用说明')}
          onChange={(value) => onRuleFieldChange('note', value)}
        />
      </div>
    </Card>
  );
}
