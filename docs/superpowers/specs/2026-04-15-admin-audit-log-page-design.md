# Admin Audit Log Page Design

Date: 2026-04-15

## Goal

补齐后台“审计日志”页面的最小可用闭环，使现有后端审计日志能力能够在前端被访问和使用。

本次范围只覆盖：

- 后台菜单入口
- 前端页面与路由
- 页面级动作权限控制
- 列表查询、筛选、分页

本次明确不覆盖：

- 审计详情弹窗
- `before_json` / `after_json` 可视化展示
- 新增后端筛选条件
- 审计日志导出
- 审计日志写入逻辑扩展

## Current State

当前代码已经具备以下基础能力：

- 后端存在审计日志数据模型 `AdminAuditLog`
- 数据表已纳入 `AutoMigrate`
- 后端已提供 `GET /api/admin/audit-logs`
- 接口权限已绑定到 `audit_management.read`
- 多个后台写操作服务已经调用统一审计写入服务
- 前端权限配置弹框中已经存在“审计日志”动作权限

当前缺失点是：

- 前端没有审计日志页面组件
- 前端没有 `/console/audit-logs` 路由
- 侧边栏没有审计日志菜单项
- sidebar 默认配置和权限菜单配置中没有审计日志模块
- 前端没有任何代码调用 `/api/admin/audit-logs`

因此，当前状态不是“页面已实现但被隐藏”，而是“后端能力已存在，前端展示层未接通”。

## Scope

### In Scope

- 新增独立后台页：`/console/audit-logs`
- 在后台管理分组中新增“审计日志”菜单项
- 基于 `audit_management.read` 做页面访问控制
- 接入已有接口 `GET /api/admin/audit-logs`
- 支持以下筛选：
  - `action_module`
  - `operator_user_id`
- 支持分页、刷新、重置
- 展示基础审计字段
- 补齐最小测试

### Out of Scope

- 新增或修改后端 API
- 菜单可见性以外的权限模型重构
- 审计日志详情查看
- JSON diff、格式化查看、字段高亮
- 时间范围筛选
- `action_type`、`target_id` 等更多高级筛选

## Chosen Approach

采用“独立审计日志页”的方式实现，保持与现有后台列表页一致的交互模型。

原因：

- 与现有 `额度流水`、`权限模板管理`、`用户权限管理` 页的结构一致，维护成本最低
- 不需要改动后端接口协议
- 用户能通过菜单直接发现功能，信息架构清晰
- 最适合当前“补齐已有能力缺口”的目标

不采用以下方案：

- 挂到现有页面内部：会降低可发现性，也会让页面职责变杂
- 只补隐藏路由不补菜单：不能解决“菜单中没有”的实际问题

## UI Design

页面采用当前后台通用列表页模式：

- 外层使用 `CardPro`
- 顶部包含说明区、操作区、搜索区、分页区
- 主体使用 `Table`
- 无权限时显示 `Banner`
- 列表加载失败时显示 `Banner + 重新加载`

### Page Title and Copy

- 页面名称：`审计日志`
- 页面说明：用于查看后台管理写操作的审计记录

### Search Controls

只保留与现有后端接口完全对齐的两个筛选项：

- `动作模块 action_module`
- `操作人 ID operator_user_id`

交互行为：

- 点击“查询”时，从第一页开始加载
- 点击“重置”时，清空两个筛选条件并回到第一页
- 点击“刷新”时，按当前筛选和分页状态重新加载

`action_module` 先使用普通输入框，不在本次范围内维护固定选项字典，避免和实际审计写入模块值脱节。

## Table Design

最小表格列如下：

- `ID`
- `操作人 ID`
- `操作人类型`
- `动作模块`
- `动作类型`
- `目标类型`
- `目标 ID`
- `IP`
- `时间`

显示规则：

- `created_at` 使用现有 `timestamp2string`
- 空值统一展示 `-`
- 不在本次范围内显示 `before_json`、`after_json`、`action_desc`

## Routing and Navigation Design

### Route

在前端新增独立路由：

- `path`: `/console/audit-logs`

接入方式与现有后台页一致：

- 在 `web/src/App.jsx` 中新增 lazy import
- 使用 `PrivateRoute` 包裹
- 组件命名风格与现有后台页保持一致

### Sidebar

在后台管理分组新增菜单项：

- `itemKey`: `audit-logs`
- `to`: `/console/audit-logs`
- `text`: `审计日志`

同时补齐：

- `routerMap`
- `adminItems`

## Permission Design

### Action Permission

页面访问权限只认：

- resource: `audit_management`
- action: `read`

行为规则：

- 无权限：不发起列表请求，只显示无权限提示
- 有权限：允许进入页面并调用接口

### Sidebar Permission

为了让菜单可见性控制能够覆盖新页面，需要把 `audit-logs` 视为后台 sidebar 模块的一部分。

需要同时补齐两个层面：

- 前端 sidebar 默认配置
- 后端 `adminSidebarModuleCatalog`

如果只改前端、不改后端，权限快照里不会包含这个模块，菜单显示会和其它模块行为不一致。

### Permission Configuration UI

本次不新增新的动作权限资源，因为 `audit_management.read` 已存在。

但需要把“审计日志”加入菜单可见性配置项中，使其能在：

- 权限模板管理
- 用户权限管理

里作为可配置菜单模块出现。

## Data Flow

页面初始化流程：

1. 页面读取当前用户权限
2. 若 `audit_management.read` 为 `false`，直接显示无权限提示
3. 若有权限，则请求 `/api/admin/audit-logs?p=1&page_size=10`
4. 服务端返回分页数据后渲染表格

查询流程：

1. 读取筛选条件
2. 构造查询参数
3. 调用 `/api/admin/audit-logs`
4. 更新表格、页码、总数

分页流程：

1. 保留当前筛选条件
2. 切页或切换分页大小
3. 重新请求接口

## Files To Change

前端：

- `web/src/App.jsx`
- `web/src/components/layout/SiderBar.jsx`
- `web/src/hooks/common/useSidebar.js`
- `web/src/pages/AdminAuditLogsPageV1/index.jsx`
- `web/src/pages/AdminConsole/permissionCatalog.js`
- `web/src/pages/AdminConsole/permissionCatalogUi.js`
- `web/src/pages/AdminConsole/permissionCatalogUiClean.js`
- `web/src/pages/AdminUserPermissionsPageV3/catalog.js`

后端：

- `service/admin_action_permission_service.go`

测试：

- 新增前端 source test，覆盖页面接线和接口调用
- 新增后端测试，覆盖 sidebar 模块快照包含 `audit-logs`

## Testing Strategy

本次实现遵循最小 TDD 闭环。

### Backend Tests

新增或扩展测试，确认：

- `BuildUserPermissions` 返回的后台 sidebar 模块包含 `audit-logs`
- root / admin 在对应场景下能拿到该模块

### Frontend Tests

新增 source-level 最小测试，确认：

- `App.jsx` 中存在 `/console/audit-logs` 路由
- `SiderBar.jsx` 中存在 `audit-logs` 路由映射和菜单项
- 审计日志页会调用 `/api/admin/audit-logs`
- 审计日志页使用 `audit_management.read` 做权限判断

### Manual Verification

实现完成后需要手工验证：

1. 有 `audit_management.read` 权限的账号能在菜单中看到“审计日志”
2. 点击菜单能进入 `/console/audit-logs`
3. 默认能加载第一页数据
4. `action_module` 和 `operator_user_id` 筛选生效
5. 无权限账号进入页面时只看到无权限提示

## Risks and Mitigations

### Risk 1: Sidebar Permission Snapshot Inconsistency

风险：

- 只改前端菜单，不改后端 sidebar 模块目录，会导致模块权限快照缺项

应对：

- 同步修改后端 `adminSidebarModuleCatalog`

### Risk 2: Action Module Input Free Text

风险：

- `action_module` 没有固定选项，用户需要知道模块值

应对：

- 最小版先保持自由输入，避免硬编码错误映射
- 后续如需要可升级为接口返回的枚举筛选

### Risk 3: Missing Audit Detail View

风险：

- 最小版不能直接查看变更前后内容

应对：

- 明确将详情查看留在后续增强版，不混入本次交付

## Acceptance Criteria

满足以下条件时，本次任务视为完成：

- 后台菜单中出现“审计日志”
- 菜单可见性配置中可以配置“审计日志”菜单
- 前端存在 `/console/audit-logs` 页面
- 页面权限受 `audit_management.read` 控制
- 页面能调用已有 `/api/admin/audit-logs`
- 页面支持按 `action_module` 和 `operator_user_id` 查询
- 页面支持刷新、重置、分页
- 相关新增测试通过

