# 编辑用户调整额度弹框设计

## 背景

编辑用户弹框中的额度操作入口和二级弹框存在两个问题：

- 入口文案与当前能力不匹配，实际上需要支持增加与减少
- 二级弹框仍使用默认 footer，样式与编辑用户主弹框不一致

## 目标

- 将入口文案从“添加额度”调整为“调整额度”
- 弹框内默认选择“增加”，并允许切换到“减少”
- 金额与额度输入框只允许输入正数
- 当减少后新额度小于 0 时，不允许提交
- 弹框 footer 统一为 `ModalActionFooter`

## 设计

- 保留现有二级弹框结构与前端本地改值逻辑，不修改后端接口
- 新增前端操作类型状态：`increase | decrease`
- 通过 helper 统一处理：
  - 正数输入归一化
  - 新额度计算
  - 提交可用性判断
- 顶部预览文案改成：`当前额度 ± 调整额度 = 新额度`
- 当结果为负时，显示提示并禁用确认按钮

## 范围

- `web/src/components/table/users/modals/EditUserModal.jsx`
- `web/src/components/table/users/modals/editUserModalHelpers.js`
- 相关测试文件
