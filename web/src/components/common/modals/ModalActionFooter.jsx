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
import { Button, Space } from '@douyinfe/semi-ui';
import { IconClose, IconSave, IconTickCircle } from '@douyinfe/semi-icons';

const roundedButtonClass = '!rounded-lg';

const ModalActionFooter = ({
  onConfirm,
  onCancel,
  confirmText,
  cancelText = '取消',
  confirmLoading = false,
  confirmDisabled = false,
  cancelDisabled = false,
  confirmIcon = <IconSave />,
  cancelIcon = <IconClose />,
  showCancel = true,
  confirmProps = {},
  cancelProps = {},
}) => {
  const mergedConfirmProps = {
    theme: 'solid',
    type: 'primary',
    className: roundedButtonClass,
    onClick: onConfirm,
    loading: confirmLoading,
    disabled: confirmDisabled,
    icon: confirmIcon,
    ...confirmProps,
  };

  const mergedCancelProps = {
    theme: 'light',
    type: 'primary',
    className: roundedButtonClass,
    onClick: onCancel,
    disabled: cancelDisabled,
    icon: cancelIcon,
    ...cancelProps,
  };

  return (
    <div className='flex justify-end bg-white'>
      <Space spacing={12}>
        {confirmText ? <Button {...mergedConfirmProps}>{confirmText}</Button> : null}
        {showCancel ? <Button {...mergedCancelProps}>{cancelText}</Button> : null}
      </Space>
    </div>
  );
};

export const defaultConfirmIcon = <IconTickCircle />;

export default ModalActionFooter;
