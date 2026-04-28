# 额度成本汇总设计

日期：2026-04-28

## 背景

现有额度流水页面展示的是账户额度变动明细，数据主源是 `quota_ledgers`。它适合追踪调额、充值、消费、退款等账户流水，但不适合展示按业务维度聚合后的模型成本。

用户提供的目标表头属于成本汇总报表，核心粒度是：

`日期 + 模型名称 + 供应商名称`

同一天、同模型、同供应商聚合成一行，展示调用次数、输入/输出 token、cache token、分项费用、总费用、折扣和实付金额。

## 目标

1. 在额度流水页面新增一个“成本汇总”视图或 Tab。
2. 成本汇总列表按 `日期 + 模型名称 + 供应商名称` 聚合。
3. 成本汇总列表支持更多查询条件，并保证列表和导出口径一致。
4. 成本汇总导出使用同一套后端汇总服务，避免页面和 Excel 数据不一致。
5. 保留现有“流水明细”列表和导出行为，不改变原有额度流水语义。

## 非目标

本期不做以下内容：

- 不把成本汇总字段硬塞进现有额度流水明细行。
- 不新增预聚合表、定时任务或历史回填任务。
- 不重构使用日志页面或运营分析页面。
- 不改变现有额度扣费逻辑。
- 不承诺历史日志都能精确拆出 input/output/cache 分项费用；缺少分项信息的历史数据按可得信息降级展示。

## 用户体验

额度流水页面新增两个 Tab：

1. `流水明细`
   - 保持现有列表、筛选、分页、导出不变。
2. `成本汇总`
   - 展示按天、模型、供应商聚合后的成本数据。
   - 提供独立筛选、分页和导出按钮。
   - 导出条件来自当前已提交查询条件，而不是未提交的表单输入。

默认进入页面仍展示 `流水明细`，避免改变现有用户工作流。

## 成本汇总筛选条件

首版支持以下查询条件：

- 日期范围：默认最近 7 天，最大 90 天。
- 模型名称：模糊搜索。
- 供应商：按供应商筛选。
- 用户：支持用户 ID 或用户名。
- 令牌：支持令牌名称。
- 渠道：支持渠道 ID。
- 分组：按用户分组筛选。
- 最小调用次数：过滤低频聚合行。
- 最小实付金额：过滤低消费聚合行。

导出接口接收同一套筛选条件。

## 列定义

成本汇总列表和导出使用同一组字段：

| 字段 | 说明 |
| --- | --- |
| 日期 | 按 UTC+8 自然日展示，格式 `YYYY-MM-DD` |
| 模型名称 | `logs.model_name`，为空时显示 `-` |
| 供应商名称 | 来自模型元数据 `models.vendor_id -> vendors.name`，未匹配时显示 `未知供应商` |
| 结算含税价 input | 聚合行内 input 单价，价格不一致时显示加权平均 |
| 结算含税价 output | 聚合行内 output 单价，价格不一致时显示加权平均 |
| 输入 tokens | `SUM(logs.prompt_tokens)` |
| 输出 tokens | `SUM(logs.completion_tokens)` |
| 调用次数 | 消费日志条数 |
| input 费用 | 估算 input 分项费用 |
| output 费用 | 估算 output 分项费用 |
| 缓存创建 | cache creation token 数 |
| 缓存读取 | cache read token 数 |
| 缓存创建单价 | 聚合行内 cache creation 单价，价格不一致时显示加权平均 |
| 缓存读取单价 | 聚合行内 cache read 单价，价格不一致时显示加权平均 |
| cache 的 token | 缓存创建与缓存读取 token 合计 |
| cache 的金额 | cache creation 与 cache read 费用合计 |
| 总费用 USD | input、output、cache 分项费用合计 |
| 折扣 | `max(总费用 USD - 实付金额 USD, 0)` |
| 实付金额 USD | 实际扣费 `SUM(logs.quota) / common.QuotaPerUnit` |

## 数据源与口径

主数据源是 `logs` 中的消费日志：

- 只统计 `logs.type = consume`。
- 日期、模型、用户、令牌、渠道、分组等筛选直接作用于日志查询。
- 分页前按聚合维度进行汇总，再应用最小调用次数和最小实付金额过滤。

供应商名称通过模型元数据补齐：

1. 汇总查询得到模型名称集合。
2. 从主库批量查询 `models` 和 `vendors`。
3. 在 Go 层把模型映射到供应商名称。

这样避免 `LOG_DB` 与主业务库分离时做跨库 join。

供应商筛选需要在日志查询前处理：

1. 先从主库按供应商名称解析出对应模型名称集合。
2. 再把模型名称集合转成 `logs.model_name IN ?` 条件。
3. 如果供应商下没有模型，则直接返回空结果。

## 费用拆分规则

实付金额以 `logs.quota` 为准，这是实际扣除的额度。

分项费用按日志中已有 token 和 `other` 信息估算：

- input token 来自 `logs.prompt_tokens`。
- output token 来自 `logs.completion_tokens`。
- cache read token 优先取 `other.cache_tokens`。
- cache creation token 优先取 `other.cache_creation_tokens`，并兼容 `cache_creation_tokens_5m`、`cache_creation_tokens_1h`。
- 价格优先取高级计费快照中的 input/output/cache 单价；没有快照时按 legacy 的 `model_ratio`、`completion_ratio`、`cache_ratio`、`cache_creation_ratio` 和 `QuotaPerUnit` 推导。

当单条日志缺少足够信息拆分分项费用时：

- `实付金额 USD` 仍然计入。
- 无法可靠拆分的分项费用记为 0。
- 聚合行的 `总费用 USD` 只汇总可拆分部分。
- 折扣计算仍使用 `max(总费用 USD - 实付金额 USD, 0)`，避免出现负数折扣。

## 后端接口

新增接口，权限沿用 `quota_management:ledger_read`：

- `GET /api/admin/quota/cost-summary`
- `POST /api/admin/quota/cost-summary/export-auto`
- 如触发后台导出，复用现有 async export job 查询和下载机制。

列表请求参数：

- `start_timestamp`
- `end_timestamp`
- `model_name`
- `vendor`
- `user`
- `token_name`
- `channel`
- `group`
- `min_call_count`
- `min_paid_usd`
- `p`
- `page_size`
- `sort_by`
- `sort_order`

其中 `sort_by` 首版支持：

- `date`
- `model_name`
- `vendor_name`
- `call_count`
- `input_tokens`
- `output_tokens`
- `paid_usd`

默认排序为 `date desc, paid_usd desc, call_count desc`。

导出请求体使用同样字段，并增加：

- `limit`

## 后端实现结构

建议新增服务文件：

- `service/quota_cost_summary.go`
- `service/async_export_quota_cost_summary.go`

建议新增 DTO：

- `dto.AdminQuotaCostSummaryQuery`
- `dto.AdminQuotaCostSummaryExportRequest`
- `dto.AdminQuotaCostSummaryItem`

主要服务函数：

- `ListQuotaCostSummary(query, pageInfo, requesterUserID, requesterRole)`
- `ListQuotaCostSummaryForExport(query, requesterUserID, requesterRole, limit)`
- `CreateQuotaCostSummaryExportJob(...)`
- `executeQuotaCostSummaryExportJob(...)`

内部实现分两步：

1. 在日期范围、权限范围和查询条件内分批读取符合条件的日志，Go 层解析 `Other` 并按日期、模型、供应商聚合。
2. 对聚合结果应用最小调用次数、最小实付金额、排序和分页。

为控制首版复杂度，日期范围最大 90 天；同步导出超过阈值时走现有后台导出。

## 前端实现结构

在 `web/src/pages/AdminQuotaLedgerPageV2/index.jsx` 中新增 Tab 状态。

推荐把成本汇总逻辑拆到独立文件，避免页面继续膨胀：

- `web/src/pages/AdminQuotaLedgerPageV2/CostSummaryTab.jsx`
- `web/src/pages/AdminQuotaLedgerPageV2/costSummaryRequestState.js`

成本汇总 Tab 负责：

- 筛选表单。
- 提交查询和重置。
- 表格列定义。
- 分页。
- 导出按钮。

导出继续使用现有 `runSmartExport`。

## 错误处理

- 日期范围超过 90 天时后端返回错误。
- 无权限时沿用现有权限错误。
- 供应商未匹配时不报错，显示 `未知供应商`。
- `logs.other` JSON 解析失败时忽略该条日志的分项拆分字段，但保留 token、调用次数和实付金额。
- `QuotaPerUnit <= 0` 时避免除零，金额字段降级为 0。

## 性能考虑

首版不做预聚合表，但要限制查询范围。

列表查询会比普通额度流水更重，因为需要解析 `logs.other` 并聚合。为降低风险：

- 默认最近 7 天。
- 最大 90 天。
- 分页大小沿用现有页面设置。
- 先按日志筛选缩小范围，再做 Go 层聚合。
- 大导出走后台任务。

如果后续日志量过大，再引入按日预聚合表作为二期优化。

## 测试

后端测试：

- 按 `日期 + 模型 + 供应商` 聚合。
- 模型到供应商映射。
- 用户、令牌、渠道、分组、模型、供应商筛选。
- 最小调用次数和最小实付金额过滤。
- cache read/cache creation token 与费用汇总。
- `logs.other` 解析失败时不影响主汇总。
- 导出表头和行数据。
- 权限范围沿用额度管理读权限。

前端测试：

- 成本汇总 Tab 使用新接口。
- 查询条件提交后才影响列表和导出。
- 导出 payload 与当前已提交查询一致。
- 无数据、加载中、错误状态显示正常。

## 验收标准

1. 额度流水页面有 `流水明细` 和 `成本汇总` 两个视图。
2. `流水明细` 行为保持不变。
3. `成本汇总` 可按确认的查询条件筛选。
4. `成本汇总` 列表按 `日期 + 模型名称 + 供应商名称` 聚合。
5. `成本汇总` 导出字段与列表字段一致。
6. 导出数据与当前已提交查询条件一致。
7. 后端测试和前端相关测试通过。
