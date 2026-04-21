import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const modalSource = fs.readFileSync(
  path.join(process.cwd(), 'web/src/components/table/users/modals/EditUserModal.jsx'),
  'utf8',
);

test('edit user quota adjustment trigger uses a labeled button', () => {
  assert.ok(modalSource.includes("icon={<IconPlus />}"));
  assert.ok(!modalSource.includes("label={t('添加额度')}"));
  assert.ok(!modalSource.includes("<Form.Slot label=' '>"));
  assert.ok(modalSource.includes("className='invisible'"));
  assert.match(
    modalSource,
    /<Form\.Slot[\s\S]*label=\{[\s\S]*<span className='invisible'>\{t\('调整额度'\)\}<\/span>[\s\S]*}\s*[\s\S]*<Button[\s\S]*icon=\{<IconPlus \/>}[\s\S]*onClick=\{\(\) => setIsModalOpen\(true\)\}[\s\S]*>\s*\{t\('调整额度'\)\}\s*<\/Button>/,
  );
});

test('edit user quota adjustment modal uses shared footer and mode selector copy', () => {
  assert.ok(modalSource.includes('<ModalActionFooter'));
  assert.ok(modalSource.includes("{t('调整额度')}"));
  assert.ok(modalSource.includes("{t('操作类型')}"));
  assert.ok(modalSource.includes("label: t('增加')"));
  assert.ok(modalSource.includes("label: t('减少')"));
});

test('edit user quota display keeps six decimal places for admin adjustments', () => {
  assert.ok(modalSource.includes('const ADMIN_QUOTA_DISPLAY_DIGITS = 6;'));
  assert.ok(
    modalSource.includes(
      'renderQuotaWithPrompt(\n                            values.quota || 0,\n                            ADMIN_QUOTA_DISPLAY_DIGITS,',
    ),
  );
  assert.ok(
    modalSource.includes(
      'renderQuota(currentQuota, ADMIN_QUOTA_DISPLAY_DIGITS)',
    ),
  );
  assert.ok(
    modalSource.includes(
      'renderQuota(addQuotaLocal, ADMIN_QUOTA_DISPLAY_DIGITS)',
    ),
  );
  assert.ok(
    modalSource.includes(
      'renderQuota(adjustedQuota, ADMIN_QUOTA_DISPLAY_DIGITS)',
    ),
  );
});

test('edit user modal keeps allowed token groups logic while hiding the controls', () => {
  assert.ok(modalSource.includes('props.supportsAllowedTokenGroups'));
  assert.ok(modalSource.includes("field='allowed_token_groups_enabled'"));
  assert.ok(modalSource.includes("field='allowed_token_groups'"));
  assert.ok(modalSource.includes('optionList={groupOptions}'));
  assert.ok(modalSource.includes('multiple'));
  assert.ok(
    modalSource.includes(
      "style={props.hideAllowedTokenGroupFields ? { display: 'none' } : undefined}",
    ),
  );
});
