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

import React, { useMemo } from 'react';
import { Banner, Card, Empty, Tabs } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import useUserPermissions from '../../hooks/common/useUserPermissions';
import PermissionManagementTab from './PermissionManagementTabEnhanced';
import AgentManagementTab from './AgentManagementTabEnhanced';
import ManagedUsersTab from './ManagedUsersTabEnhanced';
import QuotaLedgerTab from './QuotaLedgerTabEnhanced';
import { ADMIN_ACTION_KEYS, ADMIN_PERMISSION_KEYS } from './permissionKeys';

const User = () => {
  const { t } = useTranslation();
  const { loading, hasActionPermission, hasAnyActionPermission } =
    useUserPermissions();

  const tabs = useMemo(() => {
    const items = [
      {
        key: 'managed-users',
        label: t('运营用户'),
        visible: hasActionPermission(
          ADMIN_PERMISSION_KEYS.userManagement,
          ADMIN_ACTION_KEYS.read,
        ),
        content: (
          <ManagedUsersTab
            t={t}
            canUpdateUserStatus={hasActionPermission(
              ADMIN_PERMISSION_KEYS.userManagement,
              ADMIN_ACTION_KEYS.updateStatus,
            )}
            canAdjustQuota={hasActionPermission(
              ADMIN_PERMISSION_KEYS.quotaManagement,
              ADMIN_ACTION_KEYS.adjust,
            )}
          />
        ),
      },
      {
        key: 'agents',
        label: t('代理商'),
        visible: hasAnyActionPermission([
          {
            resource: ADMIN_PERMISSION_KEYS.agentManagement,
            action: ADMIN_ACTION_KEYS.read,
          },
          {
            resource: ADMIN_PERMISSION_KEYS.agentManagement,
            action: ADMIN_ACTION_KEYS.create,
          },
        ]),
        content: (
          <AgentManagementTab
            t={t}
            canCreateAgent={hasActionPermission(
              ADMIN_PERMISSION_KEYS.agentManagement,
              ADMIN_ACTION_KEYS.create,
            )}
            canUpdateAgentStatus={hasActionPermission(
              ADMIN_PERMISSION_KEYS.agentManagement,
              ADMIN_ACTION_KEYS.updateStatus,
            )}
          />
        ),
      },
      {
        key: 'permissions',
        label: t('权限'),
        visible: hasAnyActionPermission([
          {
            resource: ADMIN_PERMISSION_KEYS.permissionManagement,
            action: ADMIN_ACTION_KEYS.read,
          },
          {
            resource: ADMIN_PERMISSION_KEYS.permissionManagement,
            action: ADMIN_ACTION_KEYS.bindProfile,
          },
        ]),
        content: (
          <PermissionManagementTab
            t={t}
            canBindProfile={hasActionPermission(
              ADMIN_PERMISSION_KEYS.permissionManagement,
              ADMIN_ACTION_KEYS.bindProfile,
            )}
          />
        ),
      },
      {
        key: 'quota-ledger',
        label: t('额度流水'),
        visible: hasActionPermission(
          ADMIN_PERMISSION_KEYS.quotaManagement,
          ADMIN_ACTION_KEYS.ledgerRead,
        ),
        content: <QuotaLedgerTab t={t} />,
      },
    ];

    return items.filter((item) => item.visible);
  }, [hasActionPermission, hasAnyActionPermission, t]);

  if (loading) {
    return (
      <div className='mt-[60px] px-2'>
        <Card loading style={{ minHeight: 480 }} />
      </div>
    );
  }

  if (tabs.length === 0) {
    return (
      <div className='mt-[60px] px-2'>
        <Banner type='warning' description={t('您无权访问此页面，请联系管理员')} />
        <div style={{ marginTop: 16 }}>
          <Empty description={t('当前账号未分配运营后台权限')} />
        </div>
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <Card className='!rounded-2xl'>
        <Tabs type='card' defaultActiveKey={tabs[0]?.key}>
          {tabs.map((tab) => (
            <Tabs.TabPane tab={tab.label} itemKey={tab.key} key={tab.key}>
              <div style={{ paddingTop: 8 }}>{tab.content}</div>
            </Tabs.TabPane>
          ))}
        </Tabs>
      </Card>
    </div>
  );
};

export default User;
