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

export const resolveDisplayUserRole = (role, userType) => {
  if (userType === 'agent') {
    return 'agent';
  }
  if (role === 100 || userType === 'root') {
    return 'root';
  }
  if (role === 10 || userType === 'admin') {
    return 'admin';
  }
  if (role === 1 || userType === 'end_user' || userType === '' || !userType) {
    return 'end_user';
  }
  return 'unknown';
};
