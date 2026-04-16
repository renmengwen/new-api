# 权限模板删除保护设计

## 背景

当前“权限模板管理”列表页只提供编辑操作，不提供删除操作。模板数据存储在 `permission_profiles` 表，模板项存储在 `permission_profile_items` 表。

模板会被用户权限绑定使用。当前生效绑定记录存储在 `user_permission_bindings` 表，其中：

- `profile_id` 指向权限模板
- `status = 1` 表示当前仍生效
- `status = 0` 表示历史或已失效绑定

本次需求需要在列表中增加删除入口，同时防止误删仍在使用中的模板。

## 目标

为“权限模板管理”列表的操作栏增加“删除”按钮，并在删除前检查该模板是否被当前生效绑定引用。

如果存在 `user_permission_bindings.profile_id = 模板ID AND status = 1` 的记录，则不允许删除，并向前端返回明确提示。

如果不存在当前生效绑定，则允许删除模板。

## 非目标

- 不检查历史绑定记录 `status = 0`
- 不新增模板回收站、软删除恢复或批量删除
- 不改变模板编辑、新增、绑定逻辑
- 不扩展新的权限动作定义

## 最终决策

采用“后端强校验 + 前端删除入口”的最小改动方案。

- 后端新增删除接口，删除前只检查当前生效绑定
- 前端在列表中增加删除按钮和确认弹窗
- 删除权限复用现有 `permission_management.bind_profile`

不新增 `permission_management.delete` 权限动作，避免扩大本次改动范围到权限目录、模板默认值和已有管理员授权数据。

## 行为定义

### 删除可见性

在“权限模板管理”列表页中：

- 具备当前编辑权限的用户继续看到“编辑”按钮
- 同一权限条件下增加“删除”按钮

本次删除按钮与编辑按钮共用当前页面的 `canEdit` 判断。

### 删除确认

点击“删除”后，前端弹出确认框，提示用户该操作会删除模板及其模板项，删除后不可恢复。

用户确认后才调用删除接口。

### 删除阻断规则

后端删除时执行引用检查：

- 查询 `user_permission_bindings`
- 条件为 `profile_id = 当前模板ID AND status = 1`

如果引用数量大于 0：

- 返回业务错误
- 不删除 `permission_profiles`
- 不删除 `permission_profile_items`

如果引用数量等于 0：

- 删除该模板对应的 `permission_profile_items`
- 删除该模板对应的 `permission_profiles`

### 错误提示

如果模板仍被引用，前端展示后端返回的明确提示，例如：

`该模板正在被 X 个账号使用，无法删除`

如果删除成功，前端提示删除成功并刷新列表。

## 后端设计

### 路由

在现有 `admin/permission-templates` 路由组下新增：

- `DELETE /api/admin/permission-templates/:id`

### 控制器

新增控制器方法：

- `DeletePermissionTemplate`

行为：

- 复用当前模板管理的权限校验方式，即 `permission_management.bind_profile`
- 解析模板 ID
- 调用 service 删除方法
- 根据结果返回成功或业务错误

### 服务层

新增 service 方法：

- `DeletePermissionTemplate(profileId int) error`

行为：

1. 校验模板是否存在
2. 查询当前生效绑定数量
3. 若引用数大于 0，返回业务错误
4. 若无引用，开启事务删除模板项和模板记录

### 数据访问

引用检查只依赖当前现有表：

- `permission_profiles`
- `permission_profile_items`
- `user_permission_bindings`

不新增字段，不新增表。

## 前端设计

### 列表操作列

在 `AdminPermissionTemplatesPageV2` 的操作列中增加“删除”按钮，与“编辑”并列显示。

### 删除交互

删除流程：

1. 点击删除
2. 弹出确认框
3. 确认后调用 `DELETE /api/admin/permission-templates/:id`
4. 成功后提示并刷新当前列表页
5. 失败后展示后端错误消息

### 列表刷新

删除成功后复用现有 `loadTemplates(page, pageSize, profileTypeFilter, keyword)` 重新加载列表。

## 测试策略

### 后端测试

新增控制器或 service 用例，至少覆盖：

1. 模板存在当前生效绑定时删除失败
2. 模板无当前生效绑定时删除成功
3. 删除成功后模板项也被一并删除

### 前端测试

增加最小范围测试，至少覆盖：

1. 列表操作列存在“删除”按钮
2. 删除行为调用正确的删除接口

如果继续采用源码契约测试，则锁定：

- `API.delete('/api/admin/permission-templates/...')`
- 操作列包含删除按钮

## 风险与控制

### 风险 1：误把历史绑定当成当前引用

控制方式：删除检查必须显式限定 `status = 1`。

### 风险 2：前端隐藏按钮但后端未校验

控制方式：以后端删除校验为准，前端按钮只负责交互，不负责约束。

### 风险 3：模板主表被删但模板项残留

控制方式：删除逻辑必须放在事务中，先删模板项，再删模板记录。

## 验收标准

满足以下条件即视为完成：

1. 权限模板管理列表操作栏出现“删除”按钮
2. 当前生效绑定引用模板时，删除被阻止并提示原因
3. 仅有历史绑定或完全无绑定时，模板允许删除
4. 删除成功后列表刷新，模板及模板项都已移除
5. 不新增新的权限动作配置要求
