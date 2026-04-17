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
const hasOwn = (value, key) =>
  value &&
  typeof value === 'object' &&
  Object.prototype.hasOwnProperty.call(value, key);

export const isRootPermissionUser = (permissions, userLike) => {
  if (permissions?.profile_type === 'root') return true;
  if (userLike?.user_type === 'root') return true;
  return typeof userLike?.role === 'number' && userLike.role >= 100;
};

export const hasPermissionAction = (
  permissions,
  resourceKey,
  actionKey,
  userLike,
) => {
  const actions = permissions?.actions;
  if (actions?.[`${resourceKey}.${actionKey}`] === true) {
    return true;
  }
  return isRootPermissionUser(permissions, userLike);
};

export const shouldUseStrictSidebarSnapshot = (responseData) => {
  const permissions = responseData?.permissions;
  if (!hasOwn(permissions, 'sidebar_modules')) {
    return false;
  }
  return !isRootPermissionUser(permissions, responseData);
};
