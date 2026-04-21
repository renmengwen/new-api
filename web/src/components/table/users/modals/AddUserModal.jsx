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

import React, { useEffect, useState, useRef } from 'react';
import { API, showError, showSuccess } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Avatar,
  Form,
  Row,
  Col,
} from '@douyinfe/semi-ui';
import { IconUserAdd } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import ModalActionFooter from '../../../common/modals/ModalActionFooter';

const { Text, Title } = Typography;

const AddUserModal = (props) => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const isMobile = useIsMobile();

  const getDefaultGroupValue = () => props.groupOptions?.[0]?.value || '';
  const getDefaultAllowedTokenGroups = () => {
    const defaultGroup = getDefaultGroupValue();
    return defaultGroup ? [defaultGroup] : [];
  };

  const getInitValues = () => ({
    username: '',
    display_name: '',
    password: '',
    remark: '',
    group: getDefaultGroupValue(),
    allowed_token_groups_enabled: false,
    allowed_token_groups: getDefaultAllowedTokenGroups(),
  });

  useEffect(() => {
    if (!props.visible) {
      return;
    }
    const nextDefaultGroup = getDefaultGroupValue();
    if (!nextDefaultGroup) {
      return;
    }
    const currentGroup = formApiRef.current?.getValue('group');
    if (!currentGroup) {
      formApiRef.current?.setValue('group', nextDefaultGroup);
    }
    if (props.supportsAllowedTokenGroups) {
      const currentAllowedGroups =
        formApiRef.current?.getValue('allowed_token_groups') || [];
      if (!Array.isArray(currentAllowedGroups) || currentAllowedGroups.length === 0) {
        formApiRef.current?.setValue('allowed_token_groups', [nextDefaultGroup]);
      }
    }
  }, [props.visible, props.groupOptions]);

  const submit = async (values) => {
    setLoading(true);
    const payload = { ...values };
    if (props.supportsAllowedTokenGroups) {
      const primaryGroup = payload.group || getDefaultGroupValue();
      const allowedGroups = Array.isArray(payload.allowed_token_groups)
        ? payload.allowed_token_groups.filter(Boolean)
        : [];
      payload.group = primaryGroup;
      payload.allowed_token_groups = payload.allowed_token_groups_enabled
        ? Array.from(new Set([primaryGroup, ...allowedGroups].filter(Boolean)))
        : allowedGroups;
    }
    const response = props.createUser
      ? await props.createUser(payload)
      : (await API.post(`/api/user/`, payload)).data;
    const { success, message } = response;
    if (success) {
      showSuccess(t('用户账户创建成功！'));
      formApiRef.current?.setValues(getInitValues());
      props.refresh();
      props.handleClose();
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const handleCancel = () => {
    formApiRef.current?.setValues(getInitValues());
    props.handleClose();
  };

  return (
    <>
      <SideSheet
        placement={'left'}
        title={
          <Space>
            <Tag color='green' shape='circle'>
              {t('新建')}
            </Tag>
            <Title heading={4} className='m-0'>
              {t('添加用户')}
            </Title>
          </Space>
        }
        bodyStyle={{ padding: '0' }}
        visible={props.visible}
        width={isMobile ? '100%' : 600}
        footer={
          <ModalActionFooter
            onConfirm={() => formApiRef.current?.submitForm()}
            onCancel={handleCancel}
            confirmText={t('提交')}
            cancelText={t('取消')}
            confirmLoading={loading}
          />
        }
        closeIcon={null}
        onCancel={() => handleCancel()}
      >
        <Spin spinning={loading}>
          <Form
            initValues={getInitValues()}
            getFormApi={(api) => (formApiRef.current = api)}
            onSubmit={submit}
            onSubmitFail={(errs) => {
              const first = Object.values(errs)[0];
              if (first) showError(Array.isArray(first) ? first[0] : first);
              formApiRef.current?.scrollToError();
            }}
          >
            <div className='p-2'>
              <Card className='!rounded-2xl shadow-sm border-0'>
                <div className='flex items-center mb-2'>
                  <Avatar size='small' color='blue' className='mr-2 shadow-md'>
                    <IconUserAdd size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>{t('用户信息')}</Text>
                    <div className='text-xs text-gray-600'>
                      {t('创建新用户账户')}
                    </div>
                  </div>
                </div>

                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Input
                      field='username'
                      label={t('用户名')}
                      placeholder={t('请输入用户名')}
                      rules={[{ required: true, message: t('请输入用户名') }]}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    <Form.Input
                      field='display_name'
                      label={t('显示名称')}
                      placeholder={t('请输入显示名称')}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    <Form.Input
                      field='password'
                      label={t('密码')}
                      type='password'
                      placeholder={t('请输入密码')}
                      rules={[{ required: true, message: t('请输入密码') }]}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    <Form.Select
                      field='group'
                      label={t('分组')}
                      placeholder={t('请选择分组')}
                      optionList={props.groupOptions}
                      rules={[{ required: true, message: t('请选择分组') }]}
                      search
                    />
                  </Col>
                  {props.supportsAllowedTokenGroups ? (
                    <>
                      <Col
                        span={24}
                        style={props.hideAllowedTokenGroupFields ? { display: 'none' } : undefined}
                      >
                        <Form.Switch
                          field='allowed_token_groups_enabled'
                          label={t('限制令牌分组')}
                          checkedText={t('开')}
                          uncheckedText={t('关')}
                          extraText={t(
                            '开启后，用户创建令牌时只能选择下列分组，主分组会自动纳入',
                          )}
                        />
                      </Col>
                      <Col
                        span={24}
                        style={props.hideAllowedTokenGroupFields ? { display: 'none' } : undefined}
                      >
                        <Form.Select
                          field='allowed_token_groups'
                          label={t('可创建令牌分组')}
                          placeholder={t('请选择可创建令牌的分组')}
                          optionList={props.groupOptions}
                          multiple
                          search
                          showClear
                          extraText={t(
                            '仅在开启限制令牌分组后生效，不影响用户主分组和计费语义',
                          )}
                        />
                      </Col>
                    </>
                  ) : null}
                  <Col span={24}>
                    <Form.Input
                      field='remark'
                      label={t('备注')}
                      placeholder={t('请输入备注（仅管理员可见）')}
                      showClear
                    />
                  </Col>
                </Row>
              </Card>
            </div>
          </Form>
        </Spin>
      </SideSheet>
    </>
  );
};

export default AddUserModal;
