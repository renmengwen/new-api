# Admin List Excel Export Design

Date: 2026-04-16

## Goal

为以下三个列表补齐 Excel 导出能力：

- `使用日志`
- `额度流水`
- `审计日志`

导出目标不是“当前页数据”，而是“当前表格已提交查询条件命中的全部数据”，并遵循统一上限：

- 命中 `0` 条：不给下载，直接提示无可导出数据
- 命中 `1..2000` 条：全部导出
- 命中 `> 2000` 条：提示“仅导出最近 2000 条”，确认后导出最近 2000 条

本次交付要求导出为原生 `.xlsx` 文件，而不是 `.csv`。

## Current State

当前三个页面都只有分页列表能力，没有现成 Excel 导出链路。

### 审计日志

- 前端页面已存在：`web/src/pages/AdminAuditLogsPageV1/index.jsx`
- 后端列表接口已存在：`GET /api/admin/audit-logs`
- 当前筛选条件：
  - `action_module`
  - `operator_user_id`
- 页面内部已经区分：
  - 草稿筛选 `draftFilters`
  - 已提交查询 `committedRequest`

这意味着审计日志天然适合按“当前已提交查询条件”导出。

### 额度流水

- 前端页面已存在：`web/src/pages/AdminQuotaLedgerPageV2/index.jsx`
- 后端列表接口已存在：`GET /api/admin/quota/ledger`
- 当前筛选条件：
  - `user_id`
  - `entry_type`
- 当前页面只有输入态，没有单独维护“已提交查询快照”

这意味着如果直接读取输入框当前值，可能出现“用户改了表单但没点查询，导出结果和表格不一致”的问题。

### 使用日志

- 前端页面已存在：`web/src/components/table/usage-logs/index.jsx`
- 数据状态集中在：`web/src/hooks/usage-logs/useUsageLogsData.jsx`
- 后端列表接口已存在：
  - admin: `GET /api/log/`
  - self: `GET /api/log/self/`
- 当前筛选条件来自表单：
  - `dateRange`
  - `token_name`
  - `model_name`
  - `group`
  - `request_id`
  - admin 额外有 `channel`、`username`
  - `logType`
- 使用日志还有“列设置”，表格列是动态可见的

这意味着使用日志导出不能只传查询条件，还必须传“当前可见列 key 和顺序”。

## Scope

### In Scope

- 三个页面都增加 `导出 Excel` 按钮
- 导出基于“当前已提交查询结果”，不是当前页
- 导出支持筛选条件透传
- 导出统一上限 `2000`
- 超上限时先提示，再导出最近 `2000` 条
- 使用日志导出列严格按“当前页面可见列”导出
- 后端生成 `.xlsx` 文件流返回
- 补齐前后端最小测试

### Out of Scope

- PDF 导出
- CSV 导出
- 后台异步导出任务
- 导出历史记录
- 自定义导出列选择弹窗
- 导出 expanded row 的完整详情内容

## Chosen Approach

采用“后端生成 `.xlsx`，前端发起下载”的方案。

不采用前端循环翻页本地拼 Excel，原因如下：

- 当前三个列表都已经有成熟的后端筛选逻辑，复用后端查询更稳
- 2000 条导出会涉及多次分页请求，前端拼装更重，也更容易与列表逻辑不一致
- 使用日志存在 admin/self 两套接口和权限边界，由后端统一收口更安全
- 文件名、下载头、Excel 格式都更适合由后端统一输出

后端 Excel 生成建议新增依赖：

- `github.com/xuri/excelize/v2`

## User-Facing Behavior

### Common Rules

三个页面的导出行为统一如下：

1. 点击 `导出 Excel`
2. 以前端当前“已提交查询条件”作为导出条件
3. 若当前命中总数为 `0`，提示：
   - `当前筛选条件下无可导出数据`
4. 若当前命中总数为 `1..2000`，直接导出全部命中数据
5. 若当前命中总数大于 `2000`，提示：
   - `当前命中 X 条，仅导出最近 2000 条`
6. 用户确认后发起导出下载

### Definition of “Current Query”

本次明确采用“当前表格正在显示的查询结果”作为导出基准，而不是未提交的草稿输入。

这意味着：

- 审计日志：直接复用现有 `committedRequest`
- 额度流水：新增 `committed filters`
- 使用日志：新增“最近一次已提交查询快照”

### Definition of “Latest 2000 Rows”

“最近 2000 条”不是按时间字段重新解释，而是严格按当前列表实际排序导出前 `2000` 条。

当前三个列表的排序基准分别为：

- 审计日志：`id desc`
- 额度流水：`quota_ledgers.id desc`
- 使用日志：`logs.id desc`

因此“最近 2000 条”在本次实现中是明确且稳定的。

## API Design

### Endpoints

新增以下导出接口：

- `POST /api/admin/audit-logs/export`
- `POST /api/admin/quota/ledger/export`
- `POST /api/log/export`
- `POST /api/log/self/export`

使用 `POST` 的原因：

- 查询字段较多
- 使用日志需要额外传 `column_keys`
- 请求体更适合表达复杂筛选和列顺序

### Request Body

#### Audit Logs

```json
{
  "action_module": "quota",
  "operator_user_id": "123",
  "limit": 2000
}
```

#### Quota Ledger

```json
{
  "user_id": "123",
  "entry_type": "adjust",
  "limit": 2000
}
```

#### Usage Logs

admin:

```json
{
  "type": 2,
  "username": "alice",
  "token_name": "demo",
  "model_name": "gpt-4.1",
  "start_timestamp": 1713196800,
  "end_timestamp": 1713283200,
  "channel": "18",
  "group": "default",
  "request_id": "req_xxx",
  "column_keys": ["time", "username", "model", "cost", "details"],
  "limit": 2000
}
```

self:

```json
{
  "type": 2,
  "token_name": "demo",
  "model_name": "gpt-4.1",
  "start_timestamp": 1713196800,
  "end_timestamp": 1713283200,
  "group": "default",
  "request_id": "req_xxx",
  "column_keys": ["time", "token", "model", "cost", "details"],
  "limit": 2000
}
```

### Response

成功时直接返回文件流：

- `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `Content-Disposition: attachment; filename="..."` 

失败时继续走现有 JSON 错误响应结构。

## Backend Design

### Shared Excel Service

新增一个共享导出服务，例如：

- `service/export_excel_service.go`

职责：

- 创建 workbook
- 写表头
- 写二维表格数据
- 处理基础列宽
- 输出二进制内容和文件名

此服务不关心业务筛选，只负责把“列定义 + 行数据”写成 `.xlsx`。

### Business Export Flow

每个导出 controller 负责：

1. 权限校验
2. 解析请求体
3. 规范化筛选参数
4. 根据现有列表排序查询数据
5. 统计总数
6. 根据总数决定实际导出条数：
   - `0`
   - `<= 2000`
   - `> 2000` 时强制截断为 `2000`
7. 将业务数据映射为导出列文本
8. 调用共享 Excel service 输出文件流

### Query Reuse

本次不新增完全独立的导出查询逻辑，尽量复用现有列表查询。

#### Audit Logs

复用 `service.ListAdminAuditLogs` 的筛选条件与排序语义。

建议补一个无分页/自定义 limit 的导出查询版本，避免伪造超大分页对象。

#### Quota Ledger

复用 `service.ListQuotaLedger` 的权限作用域、筛选逻辑和排序语义。

建议同样补一个导出查询版本，支持自定义 `limit`。

#### Usage Logs

复用：

- `model.GetAllLogs`
- `model.GetUserLogs`

并保留当前 admin/self 的作用域差异。

### Hard Limit

后端必须有硬限制：

- 最大导出 `2000` 条

即使前端没做提示或被绕过，后端也只导出最近 `2000` 条。

## Frontend Design

### Shared Download Helper

新增一个通用 helper，例如：

- `web/src/helpers/exportExcel.js`

职责：

- 发起 `POST`
- 读取 blob 响应
- 解析 `Content-Disposition` 文件名
- 触发浏览器下载

### Audit Logs Page

页面：

- `web/src/pages/AdminAuditLogsPageV1/index.jsx`

在 `actionsArea` 新增：

- `导出 Excel`

导出请求严格使用当前 `committedRequest`，不读取草稿筛选框。

### Quota Ledger Page

页面：

- `web/src/pages/AdminQuotaLedgerPageV2/index.jsx`

在 `actionsArea` 新增：

- `导出 Excel`

同时新增与审计日志类似的已提交查询状态，保证：

- 表格查询
- 刷新
- 分页
- 导出

都基于同一份 committed filters。

### Usage Logs Page

页面入口：

- `web/src/components/table/usage-logs/index.jsx`
- `web/src/components/table/usage-logs/UsageLogsFilters.jsx`
- `web/src/hooks/usage-logs/useUsageLogsData.jsx`

在筛选区按钮组中新增：

- `导出 Excel`

并在 `useUsageLogsData.jsx` 中新增：

- 已提交查询快照
- 当前可见列 key 的有序数组导出能力

使用日志导出时必须发送：

- 当前已提交筛选条件
- `column_keys`

## Export Column Rules

### Audit Logs

导出列固定为当前页面列：

- `ID`
- `操作人`
- `动作模块`
- `动作类型`
- `目标`
- `IP`
- `时间`

导出值规则：

- `操作人`：`用户名（显示名） [ID:123]`
- `动作模块`：中文标签
- `动作类型`：中文标签
- `目标`：与页面显示语义一致
- `时间`：格式化后的字符串

### Quota Ledger

导出列固定为当前页面列：

- `流水号`
- `账户 ID`
- `被操作账号`
- `类型`
- `方向`
- `金额`
- `变动前`
- `变动后`
- `操作人`
- `原因`
- `时间`

导出值规则：

- `类型`：中文
- `方向`：`入账` / `出账`
- 金额相关列沿用页面当前格式化语义
- `时间`：格式化后的字符串

### Usage Logs

导出列严格按当前页面可见列和顺序决定。

列来源：

- `web/src/components/table/usage-logs/UsageLogsColumnDefs.jsx`

导出规则：

- 当前显示列才导出
- 当前隐藏列不导出
- 顺序与当前表格顺序一致

导出值使用纯文本，不导出 UI 组件本身。关键规则：

- `类型`：导出中文
- `模型`：如存在映射模型，导出 `请求模型 -> 实际模型`
- `用时/首字`：导出纯文本摘要
- `花费`：导出纯文本
- `渠道`：导出可读摘要文本
- `详情`：只导出当前列表的摘要文本，不导出 expanded row 的完整详情

## File Naming

统一命名规则：

- `审计日志_YYYY-MM-DD_HH-mm-ss.xlsx`
- `额度流水_YYYY-MM-DD_HH-mm-ss.xlsx`
- `使用日志_YYYY-MM-DD_HH-mm-ss.xlsx`

## Permissions

导出权限与列表读取权限保持完全一致，不单独增加新权限。

### Audit Logs

- `audit_management.read`
- 并保留现有 root/admin 角色兜底限制

### Quota Ledger

- `quota_management.ledger_read`

### Usage Logs

- admin 导出：沿用 `GET /api/log/` 对应的 admin 权限边界
- self 导出：沿用 `GET /api/log/self/` 对应的用户作用域

## Files To Change

Backend:

- `go.mod`
- `router/api-router.go`
- `controller/admin_audit.go`
- `controller/admin_quota.go`
- `controller/log.go`
- `service/export_excel_service.go` 或等价共享导出文件
- 审计/额度/日志对应 service 或 model 查询文件

Frontend:

- `web/src/helpers/exportExcel.js`
- `web/src/pages/AdminAuditLogsPageV1/index.jsx`
- `web/src/pages/AdminQuotaLedgerPageV2/index.jsx`
- `web/src/components/table/usage-logs/UsageLogsFilters.jsx`
- `web/src/components/table/usage-logs/UsageLogsActions.jsx` 或 `index.jsx`
- `web/src/hooks/usage-logs/useUsageLogsData.jsx`

Tests:

- 审计日志 controller / page tests
- 额度流水 controller / page tests
- 使用日志 controller / hook / source tests

## Testing Strategy

### Backend Tests

至少覆盖：

- 无筛选条件时，命中 `123` 条则导出 `123` 条
- 有筛选条件时，命中 `88` 条则导出 `88` 条
- 命中 `> 2000` 条时，只导出最近 `2000` 条
- 权限不足时拒绝导出
- 使用日志 admin/self 作用域正确
- `Content-Disposition` 文件名存在

### Frontend Tests

至少覆盖：

- 三个页面存在 `导出 Excel` 按钮
- 审计日志导出使用 `committedRequest`
- 额度流水导出使用 committed filters
- 使用日志导出带当前已提交筛选条件
- 使用日志导出带当前可见列 key
- 总数超过 `2000` 时先弹提示

### Manual Verification

至少手工验证以下场景：

1. 审计日志无筛选、总数 `123`，导出 `123` 条
2. 额度流水筛选 `类型=调额`、总数 `88`，导出 `88` 条
3. 使用日志命中 `2500` 条，提示后只导出最近 `2000` 条
4. 使用日志隐藏若干列后，导出文件只包含当前可见列
5. 三个文件都能被 Excel 正常打开

## Risks and Mitigations

### Risk 1: 导出条件和当前表格不一致

原因：

- 页面存在草稿输入但尚未查询

应对：

- 三个页面统一引入 committed query 概念
- 导出仅使用 committed query

### Risk 2: 使用日志列过于动态

原因：

- 列显示受前端控制

应对：

- 前端传 `column_keys`
- 后端只按 key 映射表头和文本

### Risk 3: 导出详情列过长

原因：

- 使用日志 expanded row 信息很多

应对：

- 本次只导出列表摘要文本
- 不导出展开区完整内容

### Risk 4: 大数据量导出影响响应

原因：

- 同步导出会占用一次请求

应对：

- 统一上限 `2000`
- 后端硬限制兜底

## Acceptance Criteria

满足以下条件即视为本次任务完成：

- 三个页面都有 `导出 Excel` 按钮
- 导出按当前已提交查询条件工作，不按当前页工作
- 无筛选 `123` 条时导出 `123` 条
- 有筛选 `88` 条时导出 `88` 条
- 命中超过 `2000` 条时提示，并只导出最近 `2000` 条
- 使用日志按当前可见列导出
- 导出文件为 `.xlsx`
- 权限边界与原列表一致
- 相关测试通过
