const createIdFactory = (prefix) => {
  let counter = 0;
  return () => `${prefix}_${++counter}`;
};

const nextGroupRowId = createIdFactory('gr');
const nextAutoGroupId = createIdFactory('ag');
const nextGroupGroupRatioRuleId = createIdFactory('ggr');
const nextGroupSpecialUsableRuleId = createIdFactory('gsu');

const hasOwn = (object, key) =>
  Object.prototype.hasOwnProperty.call(object || {}, key);

export const OP_ADD = 'add';
export const OP_REMOVE = 'remove';
export const OP_APPEND = 'append';

export const parseJSONSafe = (jsonString, fallback) => {
  if (typeof jsonString !== 'string' || jsonString.trim() === '') {
    return fallback;
  }

  try {
    return JSON.parse(jsonString);
  } catch {
    return fallback;
  }
};

export const buildGroupTableRows = (
  groupRatioString,
  userUsableGroupsString,
) => {
  const groupRatio = parseJSONSafe(groupRatioString, {});
  const userUsableGroups = parseJSONSafe(userUsableGroupsString, {});
  const rowNames = [];

  Object.keys(groupRatio).forEach((name) => {
    rowNames.push(name);
  });

  Object.keys(userUsableGroups).forEach((name) => {
    if (!rowNames.includes(name)) {
      rowNames.push(name);
    }
  });

  return rowNames.map((name) => ({
    _id: nextGroupRowId(),
    name,
    ratio:
      typeof groupRatio[name] === 'number' && Number.isFinite(groupRatio[name])
        ? groupRatio[name]
        : 1,
    selectable: hasOwn(userUsableGroups, name),
    description:
      typeof userUsableGroups[name] === 'string' ? userUsableGroups[name] : '',
    ratioMissing: !hasOwn(groupRatio, name),
  }));
};

export const serializeGroupTableRows = (rows = []) => {
  const groupRatio = {};
  const userUsableGroups = {};

  rows.forEach((row) => {
    if (!row?.name) {
      return;
    }

    if (row.ratioMissing !== true) {
      const ratio =
        typeof row.ratio === 'number' && Number.isFinite(row.ratio)
          ? row.ratio
          : Number(row.ratio);
      groupRatio[row.name] = Number.isFinite(ratio) ? ratio : 1;
    }

    if (row.selectable) {
      userUsableGroups[row.name] =
        typeof row.description === 'string' ? row.description : '';
    }
  });

  return {
    GroupRatio: JSON.stringify(groupRatio, null, 2),
    UserUsableGroups: JSON.stringify(userUsableGroups, null, 2),
  };
};

export const parseAutoGroups = (autoGroupsString) => {
  const parsed = parseJSONSafe(autoGroupsString, []);
  if (!Array.isArray(parsed)) {
    return [];
  }

  return parsed
    .filter((item) => typeof item === 'string')
    .map((name) => ({
      _id: nextAutoGroupId(),
      name,
    }));
};

export const serializeAutoGroups = (items = []) =>
  JSON.stringify(
    items.map((item) => item?.name).filter((name) => typeof name === 'string' && name),
  );

export const flattenGroupGroupRatioRules = (groupGroupRatio = {}) => {
  const rules = [];

  Object.entries(groupGroupRatio).forEach(([userGroup, usingGroupMap]) => {
    if (
      !usingGroupMap ||
      typeof usingGroupMap !== 'object' ||
      Array.isArray(usingGroupMap)
    ) {
      return;
    }

    Object.entries(usingGroupMap).forEach(([usingGroup, ratio]) => {
      rules.push({
        _id: nextGroupGroupRatioRuleId(),
        userGroup,
        usingGroup,
        ratio:
          typeof ratio === 'number' && Number.isFinite(ratio) ? ratio : 1,
      });
    });
  });

  return rules;
};

export const serializeGroupGroupRatioRules = (rules = []) => {
  const groupGroupRatio = {};

  rules.forEach((rule) => {
    if (!rule?.userGroup || !rule?.usingGroup) {
      return;
    }

    if (!groupGroupRatio[rule.userGroup]) {
      groupGroupRatio[rule.userGroup] = {};
    }

    const ratio =
      typeof rule.ratio === 'number' && Number.isFinite(rule.ratio)
        ? rule.ratio
        : Number(rule.ratio);
    groupGroupRatio[rule.userGroup][rule.usingGroup] = Number.isFinite(ratio)
      ? ratio
      : 1;
  });

  return JSON.stringify(groupGroupRatio, null, 2);
};

export const parseSpecialUsableGroupKey = (rawKey) => {
  if (rawKey.startsWith('+:')) {
    return {
      op: OP_ADD,
      targetGroup: rawKey.slice(2),
    };
  }

  if (rawKey.startsWith('-:')) {
    return {
      op: OP_REMOVE,
      targetGroup: rawKey.slice(2),
    };
  }

  return {
    op: OP_APPEND,
    targetGroup: rawKey,
  };
};

export const buildSpecialUsableGroupKey = (op, targetGroup) => {
  if (op === OP_ADD) {
    return `+:${targetGroup}`;
  }

  if (op === OP_REMOVE) {
    return `-:${targetGroup}`;
  }

  return targetGroup;
};

export const flattenGroupSpecialUsableRules = (specialUsableGroups = {}) => {
  const rules = [];

  Object.entries(specialUsableGroups).forEach(([userGroup, rawRules]) => {
    if (!rawRules || typeof rawRules !== 'object' || Array.isArray(rawRules)) {
      return;
    }

    Object.entries(rawRules).forEach(([rawKey, description]) => {
      const { op, targetGroup } = parseSpecialUsableGroupKey(rawKey);
      rules.push({
        _id: nextGroupSpecialUsableRuleId(),
        userGroup,
        op,
        targetGroup,
        description:
          typeof description === 'string'
            ? description
            : description == null
              ? ''
              : String(description),
      });
    });
  });

  return rules;
};

export const serializeGroupSpecialUsableRules = (rules = []) => {
  const specialUsableGroups = {};

  rules.forEach((rule) => {
    if (!rule?.userGroup || !rule?.targetGroup) {
      return;
    }

    if (!specialUsableGroups[rule.userGroup]) {
      specialUsableGroups[rule.userGroup] = {};
    }

    specialUsableGroups[rule.userGroup][
      buildSpecialUsableGroupKey(rule.op, rule.targetGroup)
    ] = typeof rule.description === 'string' ? rule.description : '';
  });

  return JSON.stringify(specialUsableGroups, null, 2);
};
