import test from 'node:test';
import assert from 'node:assert/strict';

import {
  OP_ADD,
  OP_APPEND,
  OP_REMOVE,
  flattenGroupGroupRatioRules,
  flattenGroupSpecialUsableRules,
  serializeAutoGroups,
  serializeGroupGroupRatioRules,
  serializeGroupSpecialUsableRules,
  serializeGroupTableRows,
} from './groupSettingsSerialization.js';

test('serializeGroupTableRows writes both group ratio and user-usable maps', () => {
  const result = serializeGroupTableRows([
    {
      name: 'default',
      ratio: 1,
      selectable: false,
      description: '',
    },
    {
      name: 'vip',
      ratio: 0.5,
      selectable: true,
      description: 'VIP 用户',
    },
  ]);

  assert.deepEqual(JSON.parse(result.GroupRatio), {
    default: 1,
    vip: 0.5,
  });
  assert.deepEqual(JSON.parse(result.UserUsableGroups), {
    vip: 'VIP 用户',
  });
  assert.deepEqual(serializeGroupTableRows([]), {
    GroupRatio: '{}',
    UserUsableGroups: '{}',
  });
});

test('serializeAutoGroups preserves order and drops empty names', () => {
  assert.equal(
    serializeAutoGroups([
      { name: 'default' },
      { name: '' },
      { name: 'vip' },
    ]),
    '["default","vip"]',
  );
  assert.equal(serializeAutoGroups([]), '[]');
});

test('group-group-ratio rules flatten and serialize nested maps', () => {
  const rules = flattenGroupGroupRatioRules({
    vip: {
      default: 0.8,
      premium: 0.3,
    },
  });

  assert.equal(rules.length, 2);
  assert.equal(rules[0].userGroup, 'vip');
  assert.equal(rules[0].usingGroup, 'default');
  assert.equal(rules[0].ratio, 0.8);

  assert.deepEqual(
    JSON.parse(
      serializeGroupGroupRatioRules([
        { userGroup: 'vip', usingGroup: 'default', ratio: 0.8 },
        { userGroup: 'vip', usingGroup: 'premium', ratio: 0.3 },
      ]),
    ),
    {
      vip: {
        default: 0.8,
        premium: 0.3,
      },
    },
  );
  assert.equal(serializeGroupGroupRatioRules([]), '{}');
});

test('special-usable rules flatten and serialize prefix operations', () => {
  const rules = flattenGroupSpecialUsableRules({
    vip: {
      '+:exclusive': '专属分组',
      '-:default': '默认分组',
      premium: '高级套餐',
    },
  });

  assert.deepEqual(
    rules.map((rule) => ({
      userGroup: rule.userGroup,
      op: rule.op,
      targetGroup: rule.targetGroup,
      description: rule.description,
    })),
    [
      {
        userGroup: 'vip',
        op: OP_ADD,
        targetGroup: 'exclusive',
        description: '专属分组',
      },
      {
        userGroup: 'vip',
        op: OP_REMOVE,
        targetGroup: 'default',
        description: '默认分组',
      },
      {
        userGroup: 'vip',
        op: OP_APPEND,
        targetGroup: 'premium',
        description: '高级套餐',
      },
    ],
  );

  assert.deepEqual(
    JSON.parse(
      serializeGroupSpecialUsableRules([
        {
          userGroup: 'vip',
          op: OP_ADD,
          targetGroup: 'exclusive',
          description: '专属分组',
        },
        {
          userGroup: 'vip',
          op: OP_REMOVE,
          targetGroup: 'default',
          description: '默认分组',
        },
        {
          userGroup: 'vip',
          op: OP_APPEND,
          targetGroup: 'premium',
          description: '高级套餐',
        },
      ]),
    ),
    {
      vip: {
        '+:exclusive': '专属分组',
        '-:default': '默认分组',
        premium: '高级套餐',
      },
    },
  );
  assert.equal(serializeGroupSpecialUsableRules([]), '{}');
});
