import test from 'node:test';
import assert from 'node:assert/strict';

import { buildOptionAuditPayload } from './settingAudit.js';

test('buildOptionAuditPayload includes explicit audit context when provided', () => {
  assert.deepEqual(
    buildOptionAuditPayload({
      key: 'ServerAddress',
      value: 'https://pay.example.com',
      auditModule: 'setting_payment',
      auditType: 'save_payment_general',
      auditDesc: '系统设置-支付设置-通用-更新服务器地址',
    }),
    {
      key: 'ServerAddress',
      value: 'https://pay.example.com',
      audit_module: 'setting_payment',
      audit_type: 'save_payment_general',
      audit_desc: '系统设置-支付设置-通用-更新服务器地址',
    },
  );
});

test('buildOptionAuditPayload falls back to plain option payload without complete audit context', () => {
  assert.deepEqual(
    buildOptionAuditPayload({
      key: 'Notice',
      value: 'hello',
      auditModule: 'setting_misc',
      auditDesc: '系统设置-其他设置-设置公告',
    }),
    {
      key: 'Notice',
      value: 'hello',
    },
  );
});
