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
import { Banner, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import UsersTable from '../../components/table/users';
import ManagedUsersTabEnhanced from './ManagedUsersTabEnhanced';
import { isAdmin, isRoot } from '../../helpers';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';

const { Text } = Typography;

const User = () => {
  const { t } = useTranslation();
  const { loading, hasActionPermission } = useUserPermissions();

  const canReadManagedUsers = hasActionPermission('user_management', 'read');
  const canUpdateUserStatus = hasActionPermission('user_management', 'update_status');
  const canAdjustQuota =
    hasActionPermission('quota_management', 'adjust') ||
    hasActionPermission('quota_management', 'adjust_batch');

  if (loading) {
    return (
      <div className='mt-[60px] px-2'>
        <Text>{t('加载中')}</Text>
      </div>
    );
  }

  if (isRoot() || isAdmin()) {
    return (
      <div className='mt-[60px] px-2'>
        <UsersTable />
      </div>
    );
  }

  if (!canReadManagedUsers) {
    return (
      <div className='mt-[60px] px-2'>
        <Banner
          type='warning'
          closeIcon={null}
          description={t('您无权访问此页面，请联系管理员')}
        />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <ManagedUsersTabEnhanced
        t={t}
        canUpdateUserStatus={canUpdateUserStatus}
        canAdjustQuota={canAdjustQuota}
      />
    </div>
  );
};

export default User;
