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
import {
  Button,
  Space,
  Tag,
  Tooltip,
  Progress,
  Popover,
  Typography,
  Dropdown,
} from '@douyinfe/semi-ui';
import { IconMore } from '@douyinfe/semi-icons';
import { renderGroup, renderNumber, renderQuota } from '../../../helpers';
import { isUserDeleted } from './statusHelpers';
import { getInviteOwnerName } from './inviteHelpers';
import { resolveDisplayUserRole } from './roleHelpers';

const renderRole = (role, userType, t) => {
  switch (resolveDisplayUserRole(role, userType)) {
    case 'agent':
      return (
        <Tag color='purple' shape='circle'>
          {t('代理商')}
        </Tag>
      );
    case 'admin':
      return (
        <Tag color='yellow' shape='circle'>
          {t('管理员')}
        </Tag>
      );
    case 'root':
      return (
        <Tag color='orange' shape='circle'>
          {t('超级管理员')}
        </Tag>
      );
    case 'end_user':
      return (
        <Tag color='blue' shape='circle'>
          {t('普通用户')}
        </Tag>
      );
    default:
      return (
        <Tag color='red' shape='circle'>
          {t('未知身份')}
        </Tag>
      );
  }
};

const renderUsername = (text, record) => {
  const remark = record.remark;
  if (!remark) {
    return <span>{text}</span>;
  }
  const maxLen = 10;
  const displayRemark =
    remark.length > maxLen ? remark.slice(0, maxLen) + '…' : remark;
  return (
    <Space spacing={2}>
      <span>{text}</span>
      <Tooltip content={remark} position='top' showArrow>
        <Tag color='white' shape='circle' className='!text-xs'>
          <div className='flex items-center gap-1'>
            <div
              className='w-2 h-2 flex-shrink-0 rounded-full'
              style={{ backgroundColor: '#10b981' }}
            />
            {displayRemark}
          </div>
        </Tag>
      </Tooltip>
    </Space>
  );
};

const renderStatistics = (text, record, showEnableDisableModal, t) => {
  const isDeleted = isUserDeleted(record);

  let tagColor = 'grey';
  let tagText = t('未知状态');
  if (isDeleted) {
    tagColor = 'red';
    tagText = t('已注销');
  } else if (record.status === 1) {
    tagColor = 'green';
    tagText = t('已启用');
  } else if (record.status === 2) {
    tagColor = 'red';
    tagText = t('已禁用');
  }

  const content = (
    <Tag color={tagColor} shape='circle' size='small'>
      {tagText}
    </Tag>
  );

  const tooltipContent = (
    <div className='text-xs'>
      <div>
        {t('调用次数')}: {renderNumber(record.request_count)}
      </div>
    </div>
  );

  return (
    <Tooltip content={tooltipContent} position='top'>
      {content}
    </Tooltip>
  );
};

const renderQuotaUsage = (text, record, t) => {
  const { Paragraph } = Typography;
  const quotaDigits = 6;
  const used = parseInt(record.used_quota) || 0;
  const remain = parseInt(record.quota) || 0;
  const total = used + remain;
  const percent = total > 0 ? (remain / total) * 100 : 0;
  const popoverContent = (
    <div className='text-xs p-2'>
      <Paragraph copyable={{ content: renderQuota(used, quotaDigits) }}>
        {t('已用额度')}: {renderQuota(used, quotaDigits)}
      </Paragraph>
      <Paragraph copyable={{ content: renderQuota(remain, quotaDigits) }}>
        {t('剩余额度')}: {renderQuota(remain, quotaDigits)} ({percent.toFixed(0)}%)
      </Paragraph>
      <Paragraph copyable={{ content: renderQuota(total, quotaDigits) }}>
        {t('总额度')}: {renderQuota(total, quotaDigits)}
      </Paragraph>
    </div>
  );
  return (
    <Popover content={popoverContent} position='top'>
      <Tag color='white' shape='circle'>
        <div className='flex flex-col items-end'>
          <span className='text-xs leading-none'>{`${renderQuota(remain, quotaDigits)} / ${renderQuota(total, quotaDigits)}`}</span>
          <Progress
            percent={percent}
            aria-label='quota usage'
            format={() => `${percent.toFixed(0)}%`}
            style={{ width: '100%', marginTop: '1px', marginBottom: 0 }}
          />
        </div>
      </Tag>
    </Popover>
  );
};

const renderInviteInfo = (text, record, t) => {
  const inviteOwnerName = getInviteOwnerName(record);
  return (
    <div>
      <Space spacing={1}>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('邀请')}: {renderNumber(record.aff_count)}
        </Tag>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('收益')}: {renderQuota(record.aff_history_quota)}
        </Tag>
        <Tag color='white' shape='circle' className='!text-xs'>
          {inviteOwnerName === ''
            ? t('无邀请人')
            : `${t('邀请人')}: ${inviteOwnerName}`}
        </Tag>
      </Space>
    </div>
  );
};

const renderOperations = (
  text,
  record,
  {
    capabilities,
    setEditingUser,
    setShowEditUser,
    showEnableDisableModal,
    showDeleteModal,
    showResetPasskeyModal,
    showResetTwoFAModal,
    showUserSubscriptionsModal,
    t,
  },
) => {
  if (isUserDeleted(record)) {
    return <></>;
  }

  const moreMenu = [];
  if (capabilities.canManageSubscriptions) {
    moreMenu.push({
      node: 'item',
      name: t('订阅管理'),
      onClick: () => showUserSubscriptionsModal(record),
    });
  }
  if (capabilities.canResetPasskey) {
    moreMenu.push({
      node: 'item',
      name: t('重置 Passkey'),
      onClick: () => showResetPasskeyModal(record),
    });
  }
  if (capabilities.canResetTwoFA) {
    moreMenu.push({
      node: 'item',
      name: t('重置 2FA'),
      onClick: () => showResetTwoFAModal(record),
    });
  }
  if (capabilities.canDeleteUser) {
    moreMenu.push({
      node: 'item',
      name: t('注销'),
      type: 'danger',
      onClick: () => showDeleteModal(record),
    });
  }

  return (
    <Space>
      {capabilities.canUpdateUserStatus &&
        (record.status === 1 ? (
          <Button
            type='danger'
            size='small'
            onClick={() => showEnableDisableModal(record, 'disable')}
          >
            {t('禁用')}
          </Button>
        ) : (
          <Button
            size='small'
            onClick={() => showEnableDisableModal(record, 'enable')}
          >
            {t('启用')}
          </Button>
        ))}
      {capabilities.canUpdateUser && (
        <Button
          type='tertiary'
          size='small'
          onClick={() => {
            setEditingUser(record);
            setShowEditUser(true);
          }}
        >
          {t('编辑')}
        </Button>
      )}
      {moreMenu.length > 0 && (
        <Dropdown menu={moreMenu} trigger='click' position='bottomRight'>
          <Button type='tertiary' size='small' icon={<IconMore />} />
        </Dropdown>
      )}
    </Space>
  );
};

export const getUsersColumns = ({
  capabilities,
  t,
  setEditingUser,
  setShowEditUser,
  showEnableDisableModal,
  showDeleteModal,
  showResetPasskeyModal,
  showResetTwoFAModal,
  showUserSubscriptionsModal,
}) => {
  return [
    {
      title: 'ID',
      dataIndex: 'id',
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      render: (text, record) => renderUsername(text, record),
    },
    {
      title: t('状态'),
      dataIndex: 'info',
      render: (text, record) =>
        renderStatistics(text, record, showEnableDisableModal, t),
    },
    {
      title: t('剩余额度/总额度'),
      key: 'quota_usage',
      render: (text, record) => renderQuotaUsage(text, record, t),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      render: (text) => {
        return <div>{renderGroup(text)}</div>;
      },
    },
    {
      title: t('角色'),
      dataIndex: 'role',
      render: (text, record) => {
        return <div>{renderRole(text, record.user_type, t)}</div>;
      },
    },
    {
      title: t('邀请信息'),
      dataIndex: 'invite',
      render: (text, record) => renderInviteInfo(text, record, t),
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      width: 200,
      render: (text, record) =>
        renderOperations(text, record, {
          capabilities,
          setEditingUser,
          setShowEditUser,
          showEnableDisableModal,
          showDeleteModal,
          showResetPasskeyModal,
          showResetTwoFAModal,
          showUserSubscriptionsModal,
          t,
        }),
    },
  ];
};
