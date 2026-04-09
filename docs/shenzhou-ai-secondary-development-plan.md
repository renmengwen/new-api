# 神州 AI 二开实施方案

## 1. 文档目标

本文档用于沉淀基于 `new-api` 项目的“神州 AI V4.3 页面级需求”二开方案，重点覆盖：

- 总体方案选型
- 模块拆分与架构建议
- 总表结构设计
- 需求编号到表/接口/服务/页面的映射
- 数据库 DDL 草案
- 一期详细开发任务清单
- 一期 GORM 模型草案

目标不是直接替代开发设计，而是先把底层边界、数据模型、开发顺序一次定稳，尽量避免后续返工。

## 2. 方案结论

### 2.1 推荐方案

采用“领域扩展型”方案：

- 保留现有 `new-api` 网关主链路，不重写 relay、渠道适配、调用日志主流程。
- 新增“运营平台业务域”表和服务，承接代理商、权限、额度账本、分销、风控、审计、导出、告警等需求。
- 对现有核心表仅做必要增补，不大改原有结构和语义。

### 2.2 不推荐方案

不建议采用“直接在现有表和页面上不断加字段、加按钮”的方式，因为以下几类能力会持续返工：

- 代理商体系
- 额度账本与流水
- 分销与佣金
- 风控与冻结
- 细粒度权限
- 操作审计

## 3. 总体设计原则

### 3.1 数据设计原则

- 新业务优先新增表，不强行污染现有网关核心表。
- 所有额度变动统一进入 `quota_ledger`，禁止业务层直接修改 `users.quota` 作为唯一事实来源。
- 所有后台写操作统一进入 `admin_audit_logs`。
- 所有权限判断统一由服务端执行，前端菜单显隐仅作为体验层，不作为安全边界。
- 所有统计页面优先读取聚合表，不直接扫调用明细表。

### 3.2 技术设计原则

- 继续遵循项目现有分层：`router -> controller -> service -> model`
- 兼容 SQLite / MySQL / PostgreSQL 三库
- JSON 配置字段统一使用 `TEXT`
- 时间统一使用 `BIGINT` Unix 秒
- 复杂业务状态统一用枚举值，不用多个布尔字段拼接
- 额度和金额统一使用整数 Credit，避免浮点误差

## 4. 总体架构

建议按 8 个业务域拆分：

1. 身份与账号域
2. 权限与菜单域
3. 额度账本域
4. 使用统计域
5. 模型与厂商域
6. 分销与佣金域
7. 风控与审计域
8. 系统配置与通知域

### 4.1 现有项目可复用部分

- 用户、登录、注册、管理员基础能力
- 调用日志和基础统计能力
- 模型和厂商元数据管理
- 后台路由、布局、侧边栏和基础表格页面

### 4.2 现有项目核心缺口

- 当前角色体系只有 `guest/common/admin/root`，不足以覆盖“超级管理员 / 管理员 / 代理商 / 普通用户”
- 当前权限更偏页面显隐，不是服务端细粒度 RBAC
- 缺少独立额度账户与账本流水
- 缺少完整分销、佣金、风控业务域
- 缺少统一结构化审计日志

## 5. 业务域拆分

### 5.1 身份与账号域

负责：

- 登录、注册、基础资料
- 用户类型区分
- 用户状态、冻结状态
- 代理商与普通用户归属关系

### 5.2 权限与菜单域

负责：

- 权限模板
- 菜单可见权限
- 动作权限
- 数据范围权限

### 5.3 额度账本域

负责：

- 主体账户
- 单笔调额
- 批量调额
- 代理商给用户充值与回收
- 佣金到账、奖励发放等后续复用能力

### 5.4 使用统计域

负责：

- 小时级、日级聚合
- 概览卡片
- 按用户、模型、日期维度统计
- 导出任务

### 5.5 模型与厂商域

负责：

- 模型清单
- 厂商配置
- 定价、默认佣金比例
- 模型健康快照

### 5.6 分销与佣金域

负责：

- 推广码
- 绑定关系
- 佣金规则
- 佣金流水

### 5.7 风控与审计域

负责：

- 异常事件
- 冻结账号
- 误判与人工处理
- 管理员操作审计

### 5.8 系统配置与通知域

负责：

- 基础配置
- 通知渠道
- 告警规则
- 额度预警
- 下载中心

## 6. 总表结构设计

### 6.1 尽量保留并增补的现有表

#### `users`

用途：全系统账号主表。

建议新增字段：

- `user_type`
- `parent_agent_id`
- `phone`
- `last_active_at`
- `register_ip`
- `source_channel`
- `invited_by_promo_code_id`
- `commission_enabled`
- `freeze_reason`
- `freeze_at`

说明：

- 现有 `role` 不建议一期直接废弃，先做兼容。
- 后续新业务优先依赖 `user_type + 权限模板 + 数据范围`。

#### `models`

建议新增：

- `display_name`
- `vendor_id`
- `upstream_input_price`
- `upstream_output_price`
- `default_commission_rate`
- `health_status`
- `last_called_at`

#### `vendors`

建议新增：

- `vendor_code`
- `api_base_url`
- `api_key_encrypted`
- `default_commission_rate`
- `status`
- `remark`

#### `logs`

继续作为调用明细主来源，不改造成运营账本。建议补充：

- `success_flag`
- `http_status_code`
- `latency_ms`
- `biz_trace_id`
- `cost_amount`

#### `options`

继续承载轻量配置；复杂配置拆新表，不继续往大 JSON 上堆。

### 6.2 权限域表

#### `permission_profiles`

权限模板主表。一个管理员或代理商可绑定一个权限模板。

#### `permission_profile_items`

权限模板明细，描述资源、动作、范围。

#### `user_permission_bindings`

用户与权限模板绑定关系。

#### `user_data_scopes`

特殊数据范围覆盖，用于控制用户可访问的数据对象集合。

### 6.3 代理商域表

#### `agent_profiles`

代理商扩展资料表。

#### `agent_user_relations`

代理商和普通用户绑定关系表。

#### `agent_quota_policies`

代理商额度操作策略表，用于控制是否允许充值、回收以及单次额度限制。

### 6.4 额度账本域表

#### `quota_accounts`

所有可持有额度主体的账户表。主体可为普通用户、代理商、系统账户。

#### `quota_transfer_orders`

一笔双边额度流转的业务单。

#### `quota_ledger`

额度总账流水，必须保存：

- 变动类型
- 收支方向
- 变动前余额
- 变动后余额
- 操作者
- 原因
- 来源单据

#### `quota_adjustment_batches`

批量调额批次主表。

#### `quota_adjustment_batch_items`

批量调额批次明细表。

### 6.5 使用统计与导出域表

#### `usage_stat_hourly_user_model`

用户-模型-小时聚合表。

#### `usage_stat_daily_user`

用户日报聚合表。

#### `usage_stat_daily_model`

模型日报聚合表。

#### `usage_stat_daily_overview`

平台概览聚合表。

#### `export_jobs`

导出任务表。

### 6.6 模型与健康域表

#### `model_commission_rules`

模型级佣金规则覆盖表。

#### `model_health_snapshots`

模型健康快照表。

#### `model_health_check_tasks`

模型手动检测或定时检测任务记录表。

### 6.7 分销与佣金域表

#### `promotion_codes`

推广码主表。

#### `promotion_code_bindings`

推广码绑定关系表。

#### `commission_rules`

佣金规则表，支持全局、厂商、模型级覆盖。

#### `commission_records`

佣金流水表，最终到账仍通过 `quota_ledger` 体现。

### 6.8 风控域表

#### `risk_rules`

风控规则表。

#### `risk_events`

异常事件主表。

#### `risk_event_targets`

异常事件涉及账号明细表。

#### `frozen_accounts`

冻结账号主表。

#### `risk_case_reviews`

人工审核、误判、解冻处理记录表。

### 6.9 审计域表

#### `admin_audit_logs`

后台管理写操作审计日志表，统一记录：

- 操作者
- 模块
- 动作
- 目标对象
- 修改前
- 修改后
- IP
- 时间

### 6.10 系统配置与通知域表

#### `system_configs`

结构化系统配置主表。

#### `notification_channels`

通知渠道表，支持邮箱、钉钉、飞书、企微等。

#### `notification_time_windows`

通知时段配置表。

#### `alert_rules`

告警规则表。

#### `credit_alert_configs`

额度预警配置表。

#### `credit_alert_templates`

额度预警消息模板表。

#### `credit_alert_send_logs`

额度预警发送记录表。

### 6.11 文档中心域表

#### `document_assets`

下载中心文件记录表。

## 7. 核心关系说明

- `users` 是所有账号根表。
- `agent_profiles` 是 `user_type='agent'` 的扩展资料。
- `agent_user_relations` 描述代理商与普通用户的归属关系。
- `quota_accounts` 负责保存各主体余额。
- `quota_ledger` 是所有额度行为的唯一账本。
- `promotion_codes / promotion_code_bindings / commission_records` 都围绕 `users` 和 `quota_ledger` 建立关系。
- `risk_events / frozen_accounts / risk_case_reviews` 围绕注册、分销、批量绑定等异常行为建模。
- `admin_audit_logs` 是后台所有写操作的统一审计出口。
- `usage_stat_*` 由异步任务从 `logs` 聚合生成。

## 8. 为什么这套设计更不容易返工

- 额度、佣金、奖励统一走 `quota_ledger`，后续新增业务不必重做余额系统。
- 权限先抽象为“模板 + 权限点 + 数据范围”，后续管理员与代理商都能复用。
- 分销、风控、审计彼此独立，但统一通过 `user_id / promo_code_id / quota_ledger_id / risk_event_id` 关联。
- 统计和导出独立建表，不污染在线调用主链路。
- 网关主链路基本不动，二开风险和回归风险都更低。

## 9. 需求编号映射

### 9.1 `AUTH-001`

涉及表：

- `users`
- `agent_user_relations`
- `promotion_codes`
- `promotion_code_bindings`
- `user_permission_bindings`

建议接口：

- `POST /api/user/login`
- `POST /api/user/register`
- `GET /api/promotion-codes/validate`

建议服务：

- `auth_service`
- `distribution_service`

前端落位：

- 复用现有登录、注册页组件
- 增加邀请码实时校验
- 登录后按 `user_type + 权限` 分流

### 9.2 `USR-001`、`USR-002`

涉及表：

- `users`
- `agent_profiles`
- `agent_user_relations`
- `quota_accounts`

建议接口：

- `GET /api/admin/users`
- `GET /api/admin/users/:id`

建议服务：

- `user_query_service`
- `permission_service`

前端落位：

- 新建运营版用户管理页
- 复用表格和筛选交互，不直接复用旧的用户管理语义

### 9.3 `USR-003`、`USR-004`、`USR-005`

涉及表：

- `quota_accounts`
- `quota_transfer_orders`
- `quota_ledger`
- `quota_adjustment_batches`
- `quota_adjustment_batch_items`
- `admin_audit_logs`

建议接口：

- `POST /api/admin/quota/adjust`
- `POST /api/admin/quota/adjust/batch`
- `GET /api/admin/quota/ledger`
- `GET /api/admin/users/:id/quota-summary`

建议服务：

- `quota_service`
- `audit_service`

前端落位：

- 用户详情页中提供额度卡片
- 单用户调额弹窗
- 批量调额弹窗
- 额度流水页

### 9.4 `STA-003`、`STA-004`、`STA-005`、`STA-006`

涉及表：

- `logs`
- `usage_stat_hourly_user_model`
- `usage_stat_daily_user`
- `usage_stat_daily_model`
- `usage_stat_daily_overview`

建议接口：

- `GET /api/admin/stats/overview`
- `GET /api/admin/stats/by-model`
- `GET /api/admin/stats/by-user`
- `GET /api/admin/stats/by-day`

建议服务：

- `stats_service`
- `usage_agg_job`

前端落位：

- 新建统计概览页
- 新建按模型、按用户、按时间分析页

### 9.5 `STA-007`

涉及表：

- `export_jobs`

建议接口：

- `POST /api/admin/exports`
- `GET /api/admin/exports`
- `GET /api/admin/exports/:id/download`

建议服务：

- `export_service`

前端落位：

- 任务通知入口
- 导出任务列表页

### 9.6 `DIS-001`

涉及表：

- `promotion_codes`
- `promotion_code_bindings`

建议接口：

- `GET /api/admin/promotion-codes`
- `GET /api/admin/promotion-codes/:id`
- `POST /api/admin/promotion-codes/:id/enable`
- `POST /api/admin/promotion-codes/:id/disable`

建议服务：

- `distribution_service`

前端落位：

- 推广码管理页
- 详情页显示二维码、链接、绑定用户

### 9.7 `DIS-002`

涉及表：

- `commission_rules`
- `risk_rules`
- `system_configs`

建议接口：

- `GET /api/admin/distribution/settings`
- `PUT /api/admin/distribution/settings`

建议服务：

- `distribution_service`
- `risk_service`
- `audit_service`

### 9.8 `DIS-003`

涉及表：

- `risk_rules`
- `risk_events`
- `risk_event_targets`
- `frozen_accounts`
- `risk_case_reviews`

建议接口：

- `GET /api/admin/risk/events`
- `GET /api/admin/risk/frozen-accounts`
- `POST /api/admin/risk/events/:id/freeze`
- `POST /api/admin/risk/events/:id/misjudge`
- `POST /api/admin/frozen-accounts/:id/unfreeze`

建议服务：

- `risk_service`
- `audit_service`

前端落位：

- 风控页
- 冻结账号页

### 9.9 `DIS-004`

涉及表：

- `commission_records`
- `quota_ledger`

建议接口：

- `GET /api/admin/commissions`

建议服务：

- `distribution_service`
- `quota_service`

前端落位：

- 独立佣金流水页

### 9.10 `SUB-001`

涉及表：

- `users`
- `agent_profiles`
- `agent_user_relations`
- `quota_accounts`

建议接口：

- `GET /api/admin/agents`
- `POST /api/admin/agents`
- `GET /api/admin/agents/:id`
- `POST /api/admin/agents/:id/enable`
- `POST /api/admin/agents/:id/disable`

建议服务：

- `agent_service`
- `quota_service`

前端落位：

- 代理商列表页
- 代理商详情页

### 9.11 `MOD-001`

涉及表：

- `models`
- `model_commission_rules`

建议接口：

- `GET /api/admin/model-catalog/models`
- `POST /api/admin/model-catalog/models`
- `PUT /api/admin/model-catalog/models/:id`

建议服务：

- `model_catalog_service`

### 9.12 `MOD-002`

涉及表：

- `vendors`

建议接口：

- `GET /api/admin/model-catalog/vendors`
- `POST /api/admin/model-catalog/vendors`
- `PUT /api/admin/model-catalog/vendors/:id`

建议服务：

- `model_catalog_service`

### 9.13 `MOD-003`

涉及表：

- `model_health_snapshots`
- `model_health_check_tasks`

建议接口：

- `GET /api/admin/model-health`
- `POST /api/admin/model-health/check`
- `POST /api/admin/model-health/check/:modelId`

建议服务：

- `model_health_service`

### 9.14 `SET-001`

涉及表：

- `system_configs`

建议接口：

- `GET /api/admin/settings/basic`
- `PUT /api/admin/settings/basic`

建议服务：

- `setting_service`
- `audit_service`

### 9.15 `SET-002`

涉及表：

- `notification_channels`
- `notification_time_windows`

建议接口：

- `GET /api/admin/settings/notifications`
- `PUT /api/admin/settings/notifications`
- `POST /api/admin/settings/notifications/test`

建议服务：

- `alert_service`

### 9.16 `SET-003`

涉及表：

- `alert_rules`

建议接口：

- `GET /api/admin/settings/alert-rules`
- `POST /api/admin/settings/alert-rules`
- `PUT /api/admin/settings/alert-rules/:id`
- `DELETE /api/admin/settings/alert-rules/:id`

建议服务：

- `alert_service`

### 9.17 `SET-004`

涉及表：

- `system_configs`

建议接口：

- `GET /api/admin/settings/security`
- `PUT /api/admin/settings/security`

建议服务：

- `setting_service`

### 9.18 `SET-005`

涉及表：

- `admin_audit_logs`

建议接口：

- `GET /api/admin/audit-logs`

建议服务：

- `audit_service`

### 9.19 `SET-006`

涉及表：

- `permission_profiles`
- `permission_profile_items`
- `user_permission_bindings`
- `user_data_scopes`

建议接口：

- `GET /api/admin/permission/users`
- `GET /api/admin/permission/profiles`
- `PUT /api/admin/permission/users/:id`

建议服务：

- `permission_service`
- `audit_service`

### 9.20 `SET-007`

涉及表：

- `credit_alert_configs`
- `credit_alert_templates`
- `credit_alert_send_logs`

建议接口：

- `GET /api/admin/settings/credit-alerts`
- `PUT /api/admin/settings/credit-alerts`
- `POST /api/admin/settings/credit-alerts/test`

建议服务：

- `alert_service`

### 9.21 `DOC-001`

涉及表：

- `document_assets`

建议接口：

- `GET /api/admin/documents`
- `GET /api/admin/documents/:id/download`

建议服务：

- `document_service`

### 9.22 `MON-001`、`MON-002`

涉及表：

- `model_health_snapshots`
- `alert_rules`
- `system_configs`

建议接口：

- `GET /api/admin/monitoring/summary`
- `GET /api/admin/monitoring/grafana-links`

建议服务：

- `model_health_service`
- `alert_service`

## 10. 建议的后端模块拆分

建议新增以下控制器和服务，而不是继续把逻辑堆到已有大文件中：

### 10.1 Controller

- `controller/admin_user.go`
- `controller/admin_quota.go`
- `controller/admin_stats.go`
- `controller/admin_distribution.go`
- `controller/admin_agent.go`
- `controller/admin_setting.go`
- `controller/admin_document.go`
- `controller/admin_audit.go`

### 10.2 Service

- `service/auth_service.go`
- `service/permission_service.go`
- `service/agent_service.go`
- `service/quota_service.go`
- `service/stats_service.go`
- `service/export_service.go`
- `service/distribution_service.go`
- `service/risk_service.go`
- `service/audit_service.go`
- `service/alert_service.go`
- `service/model_catalog_service.go`
- `service/model_health_service.go`
- `service/document_service.go`

### 10.3 异步任务

- `service/jobs/usage_agg_job.go`
- `service/jobs/export_job.go`
- `service/jobs/credit_alert_job.go`

## 11. 建议的接口分组

- `/api/admin/users`
- `/api/admin/agents`
- `/api/admin/quota`
- `/api/admin/stats`
- `/api/admin/exports`
- `/api/admin/promotion-codes`
- `/api/admin/commissions`
- `/api/admin/risk`
- `/api/admin/model-catalog`
- `/api/admin/model-health`
- `/api/admin/settings`
- `/api/admin/audit-logs`
- `/api/admin/documents`

## 12. 数据库 DDL 草案

### 12.1 通用约定

- 时间统一使用 `BIGINT` 存 Unix 秒
- JSON 统一使用 `TEXT`
- 布尔统一使用 `SMALLINT` 的 `0/1`
- 金额和额度统一使用整数
- 比例统一使用 `DECIMAL(8,4)` 语义
- 一期优先通过 GORM 管理迁移，少写方言 SQL

### 12.2 `users` 增补字段

```sql
ALTER TABLE users ADD COLUMN user_type VARCHAR(32) DEFAULT 'end_user';
ALTER TABLE users ADD COLUMN parent_agent_id INT DEFAULT 0;
ALTER TABLE users ADD COLUMN phone VARCHAR(32) DEFAULT '';
ALTER TABLE users ADD COLUMN last_active_at BIGINT DEFAULT 0;
ALTER TABLE users ADD COLUMN register_ip VARCHAR(64) DEFAULT '';
ALTER TABLE users ADD COLUMN source_channel VARCHAR(64) DEFAULT '';
ALTER TABLE users ADD COLUMN invited_by_promo_code_id INT DEFAULT 0;
ALTER TABLE users ADD COLUMN commission_enabled SMALLINT DEFAULT 1;
ALTER TABLE users ADD COLUMN freeze_reason VARCHAR(255) DEFAULT '';
ALTER TABLE users ADD COLUMN freeze_at BIGINT DEFAULT 0;
```

```sql
CREATE INDEX idx_users_user_type ON users(user_type);
CREATE INDEX idx_users_parent_agent_id ON users(parent_agent_id);
CREATE INDEX idx_users_status_user_type ON users(status, user_type);
CREATE INDEX idx_users_last_active_at ON users(last_active_at);
```

### 12.3 权限域表

```sql
CREATE TABLE permission_profiles (
  id INT PRIMARY KEY,
  profile_name VARCHAR(128) NOT NULL,
  profile_type VARCHAR(32) NOT NULL,
  is_builtin SMALLINT NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  description TEXT,
  created_at BIGINT NOT NULL DEFAULT 0,
  updated_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_permission_profiles_name_type
ON permission_profiles(profile_name, profile_type);
CREATE INDEX idx_permission_profiles_type_status
ON permission_profiles(profile_type, status);
```

```sql
CREATE TABLE permission_profile_items (
  id INT PRIMARY KEY,
  profile_id INT NOT NULL,
  resource_key VARCHAR(64) NOT NULL,
  action_key VARCHAR(64) NOT NULL,
  allowed SMALLINT NOT NULL DEFAULT 1,
  scope_type VARCHAR(32) NOT NULL DEFAULT 'all',
  extra_scope_json TEXT,
  created_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_ppi_profile_resource_action
ON permission_profile_items(profile_id, resource_key, action_key);
CREATE INDEX idx_ppi_profile_id
ON permission_profile_items(profile_id);
```

```sql
CREATE TABLE user_permission_bindings (
  id INT PRIMARY KEY,
  user_id INT NOT NULL,
  profile_id INT NOT NULL,
  effective_from BIGINT NOT NULL DEFAULT 0,
  effective_to BIGINT NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX idx_upb_user_id_status
ON user_permission_bindings(user_id, status);
CREATE INDEX idx_upb_profile_id
ON user_permission_bindings(profile_id);
```

```sql
CREATE TABLE user_data_scopes (
  id INT PRIMARY KEY,
  user_id INT NOT NULL,
  scope_type VARCHAR(32) NOT NULL,
  target_type VARCHAR(32) NOT NULL,
  target_id INT NOT NULL,
  created_at BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX idx_uds_user_id
ON user_data_scopes(user_id);
CREATE INDEX idx_uds_target_type_target_id
ON user_data_scopes(target_type, target_id);
```

### 12.4 代理商域表

```sql
CREATE TABLE agent_profiles (
  id INT PRIMARY KEY,
  user_id INT NOT NULL,
  agent_name VARCHAR(128) NOT NULL,
  company_name VARCHAR(128) DEFAULT '',
  contact_phone VARCHAR(32) DEFAULT '',
  remark TEXT,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at BIGINT NOT NULL DEFAULT 0,
  updated_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_agent_profiles_user_id
ON agent_profiles(user_id);
CREATE INDEX idx_agent_profiles_status
ON agent_profiles(status);
CREATE INDEX idx_agent_profiles_agent_name
ON agent_profiles(agent_name);
```

```sql
CREATE TABLE agent_user_relations (
  id INT PRIMARY KEY,
  agent_user_id INT NOT NULL,
  end_user_id INT NOT NULL,
  bind_source VARCHAR(32) NOT NULL DEFAULT 'manual',
  bind_at BIGINT NOT NULL DEFAULT 0,
  unbind_at BIGINT NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_aur_agent_end_user_status
ON agent_user_relations(agent_user_id, end_user_id, status);
CREATE INDEX idx_aur_end_user_id
ON agent_user_relations(end_user_id);
CREATE INDEX idx_aur_agent_user_id_status
ON agent_user_relations(agent_user_id, status);
```

```sql
CREATE TABLE agent_quota_policies (
  id INT PRIMARY KEY,
  agent_user_id INT NOT NULL,
  allow_recharge_user SMALLINT NOT NULL DEFAULT 1,
  allow_reclaim_quota SMALLINT NOT NULL DEFAULT 1,
  max_single_adjust_amount INT NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  updated_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_aqp_agent_user_id
ON agent_quota_policies(agent_user_id);
```

### 12.5 额度账本域表

```sql
CREATE TABLE quota_accounts (
  id INT PRIMARY KEY,
  owner_type VARCHAR(32) NOT NULL,
  owner_id INT NOT NULL,
  balance INT NOT NULL DEFAULT 0,
  frozen_balance INT NOT NULL DEFAULT 0,
  total_recharged INT NOT NULL DEFAULT 0,
  total_consumed INT NOT NULL DEFAULT 0,
  total_adjusted_in INT NOT NULL DEFAULT 0,
  total_adjusted_out INT NOT NULL DEFAULT 0,
  version INT NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at BIGINT NOT NULL DEFAULT 0,
  updated_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_quota_accounts_owner
ON quota_accounts(owner_type, owner_id);
CREATE INDEX idx_quota_accounts_status
ON quota_accounts(status);
```

```sql
CREATE TABLE quota_transfer_orders (
  id INT PRIMARY KEY,
  order_no VARCHAR(64) NOT NULL,
  from_account_id INT NOT NULL,
  to_account_id INT NOT NULL,
  transfer_type VARCHAR(32) NOT NULL,
  amount INT NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  operator_user_id INT NOT NULL DEFAULT 0,
  operator_user_type VARCHAR(32) NOT NULL DEFAULT '',
  reason TEXT,
  remark TEXT,
  created_at BIGINT NOT NULL DEFAULT 0,
  completed_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_qto_order_no
ON quota_transfer_orders(order_no);
CREATE INDEX idx_qto_from_account_id
ON quota_transfer_orders(from_account_id);
CREATE INDEX idx_qto_to_account_id
ON quota_transfer_orders(to_account_id);
CREATE INDEX idx_qto_operator_user_id
ON quota_transfer_orders(operator_user_id);
CREATE INDEX idx_qto_status_created_at
ON quota_transfer_orders(status, created_at);
```

```sql
CREATE TABLE quota_ledger (
  id INT PRIMARY KEY,
  biz_no VARCHAR(64) NOT NULL,
  account_id INT NOT NULL,
  transfer_order_id INT NOT NULL DEFAULT 0,
  entry_type VARCHAR(32) NOT NULL,
  direction VARCHAR(16) NOT NULL,
  amount INT NOT NULL,
  balance_before INT NOT NULL,
  balance_after INT NOT NULL,
  source_type VARCHAR(32) NOT NULL DEFAULT '',
  source_id INT NOT NULL DEFAULT 0,
  operator_user_id INT NOT NULL DEFAULT 0,
  operator_user_type VARCHAR(32) NOT NULL DEFAULT '',
  reason TEXT,
  remark TEXT,
  created_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_ql_biz_no
ON quota_ledger(biz_no);
CREATE INDEX idx_ql_account_id_created_at
ON quota_ledger(account_id, created_at);
CREATE INDEX idx_ql_operator_user_id_created_at
ON quota_ledger(operator_user_id, created_at);
CREATE INDEX idx_ql_entry_type_created_at
ON quota_ledger(entry_type, created_at);
CREATE INDEX idx_ql_source_type_source_id
ON quota_ledger(source_type, source_id);
```

```sql
CREATE TABLE quota_adjustment_batches (
  id INT PRIMARY KEY,
  batch_no VARCHAR(64) NOT NULL,
  operator_user_id INT NOT NULL,
  operation_type VARCHAR(16) NOT NULL,
  target_count INT NOT NULL DEFAULT 0,
  amount INT NOT NULL DEFAULT 0,
  reason TEXT,
  remark TEXT,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at BIGINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX uk_qab_batch_no
ON quota_adjustment_batches(batch_no);
CREATE INDEX idx_qab_operator_user_id_created_at
ON quota_adjustment_batches(operator_user_id, created_at);
```

```sql
CREATE TABLE quota_adjustment_batch_items (
  id INT PRIMARY KEY,
  batch_id INT NOT NULL,
  target_user_id INT NOT NULL,
  quota_account_id INT NOT NULL,
  quota_ledger_id INT NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  error_message TEXT,
  created_at BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX idx_qabi_batch_id
ON quota_adjustment_batch_items(batch_id);
CREATE INDEX idx_qabi_target_user_id
ON quota_adjustment_batch_items(target_user_id);
```

### 12.6 审计域表

```sql
CREATE TABLE admin_audit_logs (
  id INT PRIMARY KEY,
  operator_user_id INT NOT NULL,
  operator_user_type VARCHAR(32) NOT NULL DEFAULT '',
  action_module VARCHAR(64) NOT NULL,
  action_type VARCHAR(64) NOT NULL,
  action_desc TEXT,
  target_type VARCHAR(32) NOT NULL DEFAULT '',
  target_id INT NOT NULL DEFAULT 0,
  before_json TEXT,
  after_json TEXT,
  ip VARCHAR(64) DEFAULT '',
  created_at BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX idx_aal_operator_user_id_created_at
ON admin_audit_logs(operator_user_id, created_at);
CREATE INDEX idx_aal_action_module_created_at
ON admin_audit_logs(action_module, created_at);
CREATE INDEX idx_aal_target_type_target_id
ON admin_audit_logs(target_type, target_id);
CREATE INDEX idx_aal_created_at
ON admin_audit_logs(created_at);
```

### 12.7 二期三期预留核心表

为避免一期接口与服务层返工，建议同步预留以下表结构：

- `promotion_codes`
- `promotion_code_bindings`
- `commission_rules`
- `commission_records`
- `risk_rules`
- `risk_events`
- `risk_event_targets`
- `frozen_accounts`
- `risk_case_reviews`
- `usage_stat_hourly_user_model`
- `usage_stat_daily_user`
- `usage_stat_daily_model`
- `usage_stat_daily_overview`
- `export_jobs`
- `notification_channels`
- `notification_time_windows`
- `alert_rules`
- `credit_alert_configs`
- `credit_alert_templates`
- `credit_alert_send_logs`
- `document_assets`

### 12.8 跨库兼容注意事项

- 不依赖 PostgreSQL `JSONB`
- 不依赖 MySQL 专有函数
- 避免 SQLite 不支持的 `ALTER COLUMN`
- 唯一索引不依赖部分索引或表达式索引
- 优先使用 GORM 抽象，减少原生 SQL

## 13. 一期详细开发任务清单

### 13.1 一期目标

一期优先完成 4 个底座：

- 角色与权限体系
- 代理商体系
- 额度账户与额度总账
- 审计日志体系

一期完成后，能够稳定支撑：

- `AUTH-001`
- `USR-001`
- `USR-002`
- `USR-003`
- `USR-004`
- `USR-005`
- `SUB-001`
- `SET-005`
- `SET-006`

### 13.2 一期先建表

- `users` 增补字段
- `permission_profiles`
- `permission_profile_items`
- `user_permission_bindings`
- `user_data_scopes`
- `agent_profiles`
- `agent_user_relations`
- `agent_quota_policies`
- `quota_accounts`
- `quota_transfer_orders`
- `quota_ledger`
- `quota_adjustment_batches`
- `quota_adjustment_batch_items`
- `admin_audit_logs`

### 13.3 一期后端开发任务

#### 任务 1：定义新领域模型与迁移框架

目标：

- 完成一期表结构对应的 GORM Model
- 完成 `AutoMigrate` 注册
- 确保三库可迁移

建议文件：

- `model/permission_profile.go`
- `model/agent_profile.go`
- `model/quota_account.go`
- `model/admin_audit_log.go`
- `model/user.go`
- `model/main.go`

#### 任务 2：补强用户类型与账号扩展

目标：

- 让用户主表支持 `root/admin/agent/end_user`
- `GetSelf` 返回用户类型与基础权限信息
- 登录后支持按用户类型分流

建议修改：

- `model/user.go`
- `controller/user.go`
- `middleware/auth.go`

#### 任务 3：落地权限模板与动作校验底座

目标：

- 服务端具备“资源 + 动作 + 数据范围”的检查能力
- 不再只依赖前端菜单控制

建议新增：

- `service/permission_service.go`
- `controller/admin_permission.go`

建议接口：

- `GET /api/admin/permission/users`
- `GET /api/admin/permission/profiles`
- `PUT /api/admin/permission/users/:id`

#### 任务 4：落地代理商资料与归属关系

目标：

- 能创建代理商账号
- 能管理代理商资料
- 能维护代理商与用户归属关系

建议新增：

- `service/agent_service.go`
- `controller/admin_agent.go`

建议接口：

- `GET /api/admin/agents`
- `POST /api/admin/agents`
- `GET /api/admin/agents/:id`
- `POST /api/admin/agents/:id/enable`
- `POST /api/admin/agents/:id/disable`

#### 任务 5：建立额度账户初始化与读模型

目标：

- 所有用户都有 `quota_account`
- 新用户创建自动初始化额度账户
- 历史用户支持补齐账户

建议新增：

- `service/quota_service.go`

建议接口：

- `GET /api/admin/users/:id/quota-summary`

#### 任务 6：实现单用户额度调整

目标：

- 管理员调额必须生成转账单和总账流水
- 同时更新冗余余额展示字段
- 自动写审计日志

建议接口：

- `POST /api/admin/quota/adjust`

#### 任务 7：实现批量额度调整

目标：

- 支持批量增减额度
- 支持失败明细追踪

建议接口：

- `POST /api/admin/quota/adjust/batch`

#### 任务 8：实现代理商给普通用户充值与回收

目标：

- 代理商出账、用户入账
- 用户回收、代理商入账
- 不允许越权操作非归属用户

#### 任务 9：实现额度流水查询

目标：

- 支持按用户、操作人、时间、类型筛选

建议接口：

- `GET /api/admin/quota/ledger`

#### 任务 10：实现统一审计服务

目标：

- 用户管理
- 代理商管理
- 权限配置
- 额度调整

以上写操作统一落 `admin_audit_logs`

建议新增：

- `service/audit_service.go`
- `controller/admin_audit.go`

建议接口：

- `GET /api/admin/audit-logs`

### 13.4 一期前端开发任务

#### 任务 11：登录注册增强

目标：

- 登录按用户类型和权限跳转
- 注册支持邀请码校验

建议修改：

- `web/src/components/auth/LoginForm.jsx`
- `web/src/components/auth/RegisterForm.jsx`
- `web/src/App.jsx`

#### 任务 12：权限管理页

建议新增：

- `web/src/pages/AdminPermission/index.jsx`
- `web/src/components/admin-permission/*`

#### 任务 13：代理商管理页

建议新增：

- `web/src/pages/Agent/index.jsx`
- `web/src/components/agent/*`

#### 任务 14：运营版用户管理页

建议新增：

- `web/src/pages/AdminUser/index.jsx`
- `web/src/components/admin-user/*`

页面内容包括：

- 用户列表
- 详情弹窗
- 单用户调额
- 批量调额
- 额度卡片
- 流水查看

#### 任务 15：审计日志页

建议新增：

- `web/src/pages/AuditLog/index.jsx`
- `web/src/components/audit-log/*`

### 13.5 一期接口清单

- `GET /api/user/self`
- `GET /api/admin/permission/users`
- `GET /api/admin/permission/profiles`
- `PUT /api/admin/permission/users/:id`
- `GET /api/admin/agents`
- `POST /api/admin/agents`
- `GET /api/admin/agents/:id`
- `POST /api/admin/agents/:id/enable`
- `POST /api/admin/agents/:id/disable`
- `GET /api/admin/users`
- `GET /api/admin/users/:id`
- `POST /api/admin/users/:id/enable`
- `POST /api/admin/users/:id/disable`
- `GET /api/admin/users/:id/quota-summary`
- `POST /api/admin/quota/adjust`
- `POST /api/admin/quota/adjust/batch`
- `GET /api/admin/quota/ledger`
- `GET /api/admin/audit-logs`
- `GET /api/promotion-codes/validate`

### 13.6 一期开发顺序

1. 表结构与迁移
2. 用户类型与权限底座
3. 代理商体系
4. 额度账户与总账
5. 审计日志
6. 前端页面
7. 联调与修正

### 13.7 一期建议工期

按 1 后端 + 1 前端 + 1 联调测试的最低配置估算：

- 第 1 周：表结构、用户类型、权限底座
- 第 2 周：代理商、额度账户、单用户调额
- 第 3 周：批量调额、额度流水、审计
- 第 4 周：前端页面、联调、修 bug

## 14. 一期 GORM 模型草案

### 14.1 建议文件

- `model/admin_domain_types.go`
- `model/permission_profile.go`
- `model/agent_profile.go`
- `model/quota_account.go`
- `model/admin_audit_log.go`
- `model/user.go`
- `model/main.go`

### 14.2 常量定义草案

```go
const (
	UserTypeRoot    = "root"
	UserTypeAdmin   = "admin"
	UserTypeAgent   = "agent"
	UserTypeEndUser = "end_user"
)

const (
	CommonStatusDisabled = 0
	CommonStatusEnabled  = 1
)

const (
	ScopeTypeAll       = "all"
	ScopeTypeSelf      = "self"
	ScopeTypeAgentOnly = "agent_only"
	ScopeTypeAssigned  = "assigned"
)

const (
	TransferTypeAdminAdjust   = "admin_adjust"
	TransferTypeAgentRecharge = "agent_recharge"
	TransferTypeAgentReclaim  = "agent_reclaim"
)

const (
	LedgerEntryAdjust     = "adjust"
	LedgerEntryRecharge   = "recharge"
	LedgerEntryReclaim    = "reclaim"
	LedgerEntryConsume    = "consume"
	LedgerEntryRefund     = "refund"
	LedgerEntryCommission = "commission"
	LedgerEntryReward     = "reward"
)

const (
	LedgerDirectionIn  = "in"
	LedgerDirectionOut = "out"
)
```

### 14.3 核心 Model 草案

以下模型建议作为一期第一批落地：

- `PermissionProfile`
- `PermissionProfileItem`
- `UserPermissionBinding`
- `UserDataScope`
- `AgentProfile`
- `AgentUserRelation`
- `AgentQuotaPolicy`
- `QuotaAccount`
- `QuotaTransferOrder`
- `QuotaLedger`
- `QuotaAdjustmentBatch`
- `QuotaAdjustmentBatchItem`
- `AdminAuditLog`

### 14.4 `User` 结构建议新增字段

```go
UserType             string `gorm:"type:varchar(32);default:'end_user';index"`
ParentAgentId        int    `gorm:"default:0;index"`
Phone                string `gorm:"type:varchar(32);default:''"`
LastActiveAt         int64  `gorm:"bigint;default:0;index"`
RegisterIP           string `gorm:"type:varchar(64);default:''"`
SourceChannel        string `gorm:"type:varchar(64);default:''"`
InvitedByPromoCodeId int    `gorm:"default:0"`
CommissionEnabled    bool   `gorm:"default:true"`
FreezeReason         string `gorm:"type:varchar(255);default:''"`
FreezeAt             int64  `gorm:"bigint;default:0"`
```

### 14.5 迁移注册草案

在 `model/main.go` 中加入：

```go
err = DB.AutoMigrate(
	&PermissionProfile{},
	&PermissionProfileItem{},
	&UserPermissionBinding{},
	&UserDataScope{},
	&AgentProfile{},
	&AgentUserRelation{},
	&AgentQuotaPolicy{},
	&QuotaAccount{},
	&QuotaTransferOrder{},
	&QuotaLedger{},
	&QuotaAdjustmentBatch{},
	&QuotaAdjustmentBatchItem{},
	&AdminAuditLog{},
)
```

### 14.6 落地建议

- 一期先不要强依赖数据库级外键
- 一期先不在 Model 上叠过多 Hook
- 时间字段建议手动赋值，不依赖 `gorm.Model`
- 业务号如 `order_no / biz_no / batch_no` 建议统一生成规则

## 15. 实施阶段建议

### 15.1 第一期

- 角色/权限
- 代理商
- 额度调整
- 额度流水
- 审计日志

### 15.2 第二期

- 使用统计
- 模型/厂商管理重构
- 模型健康
- 告警与通知
- 下载中心

### 15.3 第三期

- 推广码
- 佣金流水
- 分销规则
- 风控冻结
- 额度预警
- 外部监控联动

## 16. 项目级约束建议

建议从一开始就确定以下三条规则，避免后续反复返工：

- 所有额度变化必须进入 `quota_ledger`
- 所有后台写操作必须进入 `admin_audit_logs`
- 所有后台接口必须通过 `permission_service` 做动作和范围校验

## 17. 结论

本次二开适合采用“保留网关主链路、扩展运营平台业务域”的方案，不适合简单按页面直接改 UI。

如果先把以下四个底座立住，后面需求可以稳定扩展：

- 权限
- 代理商
- 额度账本
- 审计

这四部分是整份需求避免返工的关键。
