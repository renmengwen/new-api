import React from 'react';
import { Banner, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { useUserPermissions } from '../../hooks/common/useUserPermissions';
import AdminUserPermissionsPageV3 from '../AdminUserPermissionsPageV3';

const { Text } = Typography;

const AdminUserPermissionsPage = () => {
  const { t } = useTranslation();
  const { loading, hasActionPermission } = useUserPermissions();
  const canRead = hasActionPermission('permission_management', 'read');

  if (loading) {
    return (
      <div className='mt-[60px] px-2'>
        <Text>{t('加载中')}</Text>
      </div>
    );
  }

  if (!canRead) {
    return (
      <div className='mt-[60px] px-2'>
        <Banner
          type='warning'
          closeIcon={null}
          description={t('你没有用户权限管理的查看权限，请为该账号授予权限管理中的查看权限后再访问')}
        />
      </div>
    );
  }

  return <AdminUserPermissionsPageV3 />;
};

export default AdminUserPermissionsPage;
