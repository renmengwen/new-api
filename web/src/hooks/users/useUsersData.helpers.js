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

export const normalizeUserPageData = (data, fallbackPage = 1) => {
  const items = Array.isArray(data?.items) ? data.items : [];
  const page =
    Number.isInteger(data?.page) && data.page > 0 ? data.page : fallbackPage;
  const total =
    typeof data?.total === 'number' && Number.isFinite(data.total)
      ? data.total
      : items.length;

  return {
    items,
    page,
    total,
  };
};

export const shouldUseUserSearch = ({
  isManagedMode,
  searchKeyword = '',
  searchGroup = '',
  searchRole = '',
  searchStatus = '',
}) => {
  const normalizedKeyword = String(searchKeyword || '').trim();

  if (isManagedMode) {
    return normalizedKeyword !== '';
  }

  return (
    normalizedKeyword !== '' ||
    searchGroup !== '' ||
    searchRole !== '' ||
    searchStatus !== ''
  );
};

export const toGroupOptions = (payload) => {
  if (!payload?.success || !Array.isArray(payload?.data)) {
    return [];
  }

  return payload.data.map((group) => ({
    label: group,
    value: group,
  }));
};

export const toManagedGroupOptions = (data, userGroup) => {
  let groupOptions = Object.entries(data || {}).map(([group, info]) => ({
    label: group,
    value: group,
    ratio: info.ratio,
    fullLabel: info.desc,
  }));

  if (groupOptions.length === 0) {
    groupOptions = [
      {
        label: '用户分组',
        value: '',
        ratio: 1,
      },
    ];
  } else if (userGroup) {
    const userGroupIndex = groupOptions.findIndex((g) => g.value === userGroup);
    if (userGroupIndex > -1) {
      const userGroupOption = groupOptions.splice(userGroupIndex, 1)[0];
      groupOptions.unshift(userGroupOption);
    }
  }

  return groupOptions;
};
