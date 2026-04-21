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
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  renderQuota,
  renderQuotaWithPrompt,
  getCurrencyConfig,
} from '../../../../helpers';
import { toGroupOptions } from '../../../../hooks/users/useUsersData.helpers';
import {
  quotaToDisplayAmount,
  displayAmountToQuota,
} from '../../../../helpers/quota';
import {
  QUOTA_ADJUST_MODE,
  calculateAdjustedQuota,
  normalizePositiveAdjustmentValue,
  shouldDisableQuotaAdjustmentConfirm,
  shouldDisableQuotaInput,
} from './editUserModalHelpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Button,
  Modal,
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Form,
  Avatar,
  Row,
  Col,
  InputNumber,
  Select,
} from '@douyinfe/semi-ui';
import {
  IconUser,
  IconLink,
  IconUserGroup,
  IconPlus,
} from '@douyinfe/semi-icons';
import UserBindingManagementModal from './UserBindingManagementModal';
import ModalActionFooter from '../../../common/modals/ModalActionFooter';

const { Text, Title } = Typography;
const ADMIN_QUOTA_DISPLAY_DIGITS = 6;

const EditUserModal = (props) => {
  const { t } = useTranslation();
  const userId = props.editingUser.id;
  const [loading, setLoading] = useState(true);
  const [addQuotaModalOpen, setIsModalOpen] = useState(false);
  const [addQuotaLocal, setAddQuotaLocal] = useState('');
  const [addAmountLocal, setAddAmountLocal] = useState('');
  const [quotaAdjustMode, setQuotaAdjustMode] = useState(
    QUOTA_ADJUST_MODE.increase,
  );
  const isMobile = useIsMobile();
  const [groupOptions, setGroupOptions] = useState([]);
  const [bindingModalVisible, setBindingModalVisible] = useState(false);
  const formApiRef = useRef(null);

  const isEdit = Boolean(userId);

  const getInitValues = () => ({
    username: '',
    display_name: '',
    password: '',
    github_id: '',
    oidc_id: '',
    discord_id: '',
    wechat_id: '',
    telegram_id: '',
    linux_do_id: '',
    email: '',
    quota: 0,
    group: 'default',
    allowed_token_groups_enabled: false,
    allowed_token_groups: [],
    remark: '',
  });

  const fetchGroups = async () => {
    if (Array.isArray(props.groupOptions) && props.groupOptions.length > 0) {
      setGroupOptions(props.groupOptions);
      return;
    }
    try {
      let res = await API.get(`/api/group/`);
      setGroupOptions(toGroupOptions(res.data));
    } catch (e) {
      showError(e.message);
    }
  };

  const handleCancel = () => props.handleClose();

  const loadUser = async () => {
    setLoading(true);
    const response = userId
      ? props.loadUserDetail
        ? await props.loadUserDetail(userId)
        : (await API.get(`/api/user/${userId}`)).data
      : (await API.get(`/api/user/self`)).data;
    const { success, message, data } = response;
    if (success) {
      data.password = '';
      formApiRef.current?.setValues({ ...getInitValues(), ...data });
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    loadUser();
    if (userId) fetchGroups();
    setBindingModalVisible(false);
  }, [props.editingUser.id]);

  useEffect(() => {
    if (Array.isArray(props.groupOptions) && props.groupOptions.length > 0) {
      setGroupOptions(props.groupOptions);
    }
  }, [props.groupOptions]);

  const openBindingModal = () => {
    setBindingModalVisible(true);
  };

  const closeBindingModal = () => {
    setBindingModalVisible(false);
  };

  /* ----------------------- submit ----------------------- */
  const submit = async (values) => {
    setLoading(true);
    let payload = { ...values };
    if (typeof payload.quota === 'string')
      payload.quota = parseInt(payload.quota) || 0;
    if (userId) {
      payload.id = parseInt(userId);
    }
    if (props.supportsAllowedTokenGroups) {
      const primaryGroup = payload.group || groupOptions?.[0]?.value || 'default';
      const allowedGroups = Array.isArray(payload.allowed_token_groups)
        ? payload.allowed_token_groups.filter(Boolean)
        : [];
      payload.group = primaryGroup;
      payload.allowed_token_groups = payload.allowed_token_groups_enabled
        ? Array.from(new Set([primaryGroup, ...allowedGroups].filter(Boolean)))
        : allowedGroups;
    }
    const response = props.updateUser
      ? await props.updateUser(userId, payload)
      : (
          await API.put(userId ? `/api/user/` : `/api/user/self`, payload)
        ).data;
    const { success, message } = response;
    if (success) {
      showSuccess(t('用户信息更新成功！'));
      props.refresh();
      props.handleClose();
    } else {
      showError(message);
    }
    setLoading(false);
  };

  /* --------------------- quota helper -------------------- */
  const resetQuotaAdjustModal = () => {
    setIsModalOpen(false);
    setAddQuotaLocal('');
    setAddAmountLocal('');
    setQuotaAdjustMode(QUOTA_ADJUST_MODE.increase);
  };

  const currentQuota = parseInt(formApiRef.current?.getValue('quota'), 10) || 0;
  const adjustedQuota = calculateAdjustedQuota(
    currentQuota,
    addQuotaLocal,
    quotaAdjustMode,
  );
  const quotaAdjustDisabled = shouldDisableQuotaAdjustmentConfirm(
    currentQuota,
    addQuotaLocal,
    quotaAdjustMode,
  );
  const quotaAdjustOperator =
    quotaAdjustMode === QUOTA_ADJUST_MODE.decrease ? '-' : '+';

  const applyQuotaAdjustment = () => {
    if (quotaAdjustDisabled) {
      return;
    }
    formApiRef.current?.setValue('quota', adjustedQuota);
    resetQuotaAdjustModal();
  };

  /* --------------------------- UI --------------------------- */
  return (
    <>
      <SideSheet
        placement='right'
        title={
          <Space>
            <Tag color='blue' shape='circle'>
              {t(isEdit ? '编辑' : '新建')}
            </Tag>
            <Title heading={4} className='m-0'>
              {isEdit ? t('编辑用户') : t('创建用户')}
            </Title>
          </Space>
        }
        bodyStyle={{ padding: 0 }}
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
        onCancel={handleCancel}
      >
        <Spin spinning={loading}>
          <Form
            initValues={getInitValues()}
            getFormApi={(api) => (formApiRef.current = api)}
            onSubmit={submit}
          >
            {({ values }) => (
              <div className='p-2 space-y-3'>
                {/* 基本信息 */}
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='blue'
                      className='mr-2 shadow-md'
                    >
                      <IconUser size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('基本信息')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('用户的基本账户信息')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Input
                        field='username'
                        label={t('用户名')}
                        placeholder={t('请输入新的用户名')}
                        rules={[{ required: true, message: t('请输入用户名') }]}
                        showClear
                      />
                    </Col>

                    <Col span={24}>
                      <Form.Input
                        field='password'
                        label={t('密码')}
                        placeholder={t('请输入新的密码，最短 8 位')}
                        mode='password'
                        showClear
                      />
                    </Col>

                    <Col span={24}>
                      <Form.Input
                        field='display_name'
                        label={t('显示名称')}
                        placeholder={t('请输入新的显示名称')}
                        showClear
                      />
                    </Col>

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

                {/* 权限设置 */}
                {userId && (
                  <Card className='!rounded-2xl shadow-sm border-0'>
                    <div className='flex items-center mb-2'>
                      <Avatar
                        size='small'
                        color='green'
                        className='mr-2 shadow-md'
                      >
                        <IconUserGroup size={16} />
                      </Avatar>
                      <div>
                        <Text className='text-lg font-medium'>
                          {t('权限设置')}
                        </Text>
                        <div className='text-xs text-gray-600'>
                          {t('用户分组和额度管理')}
                        </div>
                      </div>
                    </div>

                    <Row gutter={12}>
                      <Col span={24}>
                        <Form.Select
                          field='group'
                          label={t('分组')}
                          placeholder={t('请选择分组')}
                          optionList={groupOptions}
                          allowAdditions
                          search
                          rules={[{ required: true, message: t('请选择分组') }]}
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
                              optionList={groupOptions}
                              multiple
                              allowAdditions
                              search
                              showClear
                              extraText={t(
                                '仅在开启限制令牌分组后生效，不影响用户主分组和计费语义',
                              )}
                            />
                          </Col>
                        </>
                      ) : null}

                      <Col span={10}>
                        <Form.InputNumber
                          field='quota'
                          label={t('剩余额度')}
                          placeholder={t('请输入新的剩余额度')}
                          step={500000}
                          extraText={renderQuotaWithPrompt(
                            values.quota || 0,
                            ADMIN_QUOTA_DISPLAY_DIGITS,
                          )}
                          rules={[{ required: true, message: t('请输入额度') }]}
                          style={{ width: '100%' }}
                          disabled={shouldDisableQuotaInput(isEdit)}
                        />
                      </Col>

                      <Col span={14}>
                        <Form.Slot
                          label={
                            <span className='invisible'>{t('调整额度')}</span>
                          }
                        >
                          <Button
                            icon={<IconPlus />}
                            onClick={() => setIsModalOpen(true)}
                            className='!inline-flex !items-center !gap-1'
                          >
                            {t('调整额度')}
                          </Button>
                        </Form.Slot>
                      </Col>
                    </Row>
                  </Card>
                )}

                {/* 绑定信息入口 */}
                {userId && props.supportsBindingManagement !== false && (
                  <Card className='!rounded-2xl shadow-sm border-0'>
                    <div className='flex items-center justify-between gap-3'>
                      <div className='flex items-center min-w-0'>
                        <Avatar
                          size='small'
                          color='purple'
                          className='mr-2 shadow-md'
                        >
                          <IconLink size={16} />
                        </Avatar>
                        <div className='min-w-0'>
                          <Text className='text-lg font-medium'>
                            {t('绑定信息')}
                          </Text>
                          <div className='text-xs text-gray-600'>
                            {t('管理用户已绑定的第三方账户，支持筛选与解绑')}
                          </div>
                        </div>
                      </div>
                      <Button
                        type='primary'
                        theme='outline'
                        onClick={openBindingModal}
                      >
                        {t('管理绑定')}
                      </Button>
                    </div>
                  </Card>
                )}
              </div>
            )}
          </Form>
        </Spin>
      </SideSheet>

      <UserBindingManagementModal
        visible={bindingModalVisible}
        onCancel={closeBindingModal}
        userId={userId}
        isMobile={isMobile}
        formApiRef={formApiRef}
      />

      {/* 添加额度模态框 */}
      <Modal
        centered
        visible={addQuotaModalOpen}
        footer={
          <ModalActionFooter
            onConfirm={applyQuotaAdjustment}
            onCancel={resetQuotaAdjustModal}
            confirmText={t('确定')}
            cancelText={t('取消')}
            confirmDisabled={quotaAdjustDisabled}
          />
        }
        onCancel={resetQuotaAdjustModal}
        closable={null}
        title={
          <div className='flex items-center'>
            <IconPlus className='mr-2' />
            {t('调整额度')}
          </div>
        }
      >
        <div className='mb-3'>
          <div className='mb-1'>
            <Text size='small'>{t('操作类型')}</Text>
          </div>
          <Select
            value={quotaAdjustMode}
            style={{ width: '100%' }}
            optionList={[
              { label: t('增加'), value: QUOTA_ADJUST_MODE.increase },
              { label: t('减少'), value: QUOTA_ADJUST_MODE.decrease },
            ]}
            onChange={(value) => setQuotaAdjustMode(value)}
          />
        </div>
        <div className='mb-4'>
          <Text type='secondary' className='block mb-2'>
            {`${t('新额度：')}${renderQuota(currentQuota, ADMIN_QUOTA_DISPLAY_DIGITS)} ${quotaAdjustOperator} ${renderQuota(addQuotaLocal, ADMIN_QUOTA_DISPLAY_DIGITS)} = ${renderQuota(adjustedQuota, ADMIN_QUOTA_DISPLAY_DIGITS)}`}
          </Text>
          {adjustedQuota < 0 ? (
            <Text type='danger' className='block'>
              {t('新额度不能小于 0')}
            </Text>
          ) : null}
        </div>
        {getCurrencyConfig().type !== 'TOKENS' && (
          <div className='mb-3'>
            <div className='mb-1'>
              <Text size='small'>{t('金额')}</Text>
              <Text size='small' type='tertiary'>
                {' '}
                ({t('仅用于换算，实际保存的是额度')})
              </Text>
            </div>
            <InputNumber
              prefix={getCurrencyConfig().symbol}
              placeholder={t('输入金额')}
              value={addAmountLocal}
              precision={2}
              onChange={(val) => {
                const normalizedAmount = normalizePositiveAdjustmentValue(val);
                setAddAmountLocal(normalizedAmount);
                setAddQuotaLocal(
                  normalizedAmount !== ''
                    ? displayAmountToQuota(normalizedAmount)
                    : '',
                );
              }}
              style={{ width: '100%' }}
              showClear
              min={0}
            />
          </div>
        )}
        <div>
          <div className='mb-1'>
            <Text size='small'>{t('额度')}</Text>
          </div>
          <InputNumber
            placeholder={t('输入额度')}
            value={addQuotaLocal}
            onChange={(val) => {
              const normalizedQuota = normalizePositiveAdjustmentValue(val);
              setAddQuotaLocal(normalizedQuota);
              setAddAmountLocal(
                normalizedQuota !== ''
                  ? Number(
                      quotaToDisplayAmount(normalizedQuota).toFixed(2),
                    )
                  : '',
              );
            }}
            style={{ width: '100%' }}
            showClear
            step={500000}
            min={0}
          />
        </div>
      </Modal>
    </>
  );
};

export default EditUserModal;
