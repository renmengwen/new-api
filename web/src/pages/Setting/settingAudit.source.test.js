import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';

const readSource = (fileUrl) =>
  fs.existsSync(fileUrl) ? fs.readFileSync(fileUrl, 'utf8') : '';

const systemSettingSource = readSource(
  new URL('../../components/settings/SystemSetting.jsx', import.meta.url),
);
const paymentGeneralSource = readSource(
  new URL('./Payment/SettingsGeneralPayment.jsx', import.meta.url),
);

test('ServerAddress save actions send explicit audit context for system and payment settings', () => {
  assert.match(systemSettingSource, /save_server_url/);
  assert.match(
    systemSettingSource,
    /系统设置-系统设置-更新服务器地址/,
  );
  assert.match(paymentGeneralSource, /save_payment_general/);
  assert.match(
    paymentGeneralSource,
    /系统设置-支付设置-通用-更新服务器地址/,
  );
});
