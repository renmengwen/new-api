# 公告邮件群发设计

## 背景

当前有两类站内公告入口：

- `其他设置 -> 通用设置 -> 公告` 保存 `Notice`，展示在公告弹框的“通知”页签。
- `仪表盘设置 -> 系统公告管理` 保存 `console_setting.announcements`，展示在公告弹框和仪表盘的“系统公告”区域。

两者保存后都会立即更新站内展示。新增需求是在保存成功后，允许管理员选择是否把本次通知/公告以邮件形式发送给指定用户范围。邮件内容可以在发送前单独编辑，编辑结果只影响本次邮件，不回写站内公告内容。

## 目标

1. 保存通知或系统公告后，弹出是否邮件发送的确认。
2. 点击“否”时完全沿用老逻辑，只保存站内公告。
3. 点击“是”时进入邮件发送弹窗。
4. 邮件弹窗支持选择收件范围：代理商、普通用户、全量用户。
5. 邮件弹窗展示可编辑的邮件标题和邮件正文。
6. 邮件发送使用弹窗内最终编辑后的标题和正文。
7. 邮件草稿不影响 `Notice` 或 `console_setting.announcements` 的保存值。

## 非目标

- 不新增邮件模板管理。
- 不新增异步任务队列或发送进度页。
- 不向管理员和超管发送“全量用户”邮件。
- 不修改现有站内公告展示逻辑。
- 不改变用户个人通知偏好；本功能是 root 管理员主动群发邮件。

## 权限与收件范围

新增接口使用 root 权限，和 `/api/option` 设置保存保持一致。

收件用户限定为启用状态且邮箱非空：

- `代理商`：`user_type = agent`。
- `普通用户`：`role < admin` 且不是 `agent`；兼容历史空 `user_type`。
- `全量用户`：代理商 + 普通用户。

全量用户不包含 `root` 和 `admin`。软删除用户由 GORM 默认查询自动排除。

## 前端交互

### 通知入口

文件：`web/src/components/settings/OtherSetting.jsx`

`submitNotice` 保存 `Notice` 成功后，显示确认弹窗：

- “否”：关闭确认弹窗，流程结束。
- “是”：打开邮件发送弹窗。

邮件草稿默认值：

- 标题：`系统通知`
- 正文：当前保存的 `inputs.Notice`

### 系统公告入口

文件：`web/src/pages/Setting/Dashboard/SettingsAnnouncements.jsx`

新增或编辑公告后，组件记录最近一次新增/编辑的公告作为邮件候选。点击“保存设置”并保存成功后：

- 如果最近一次操作是新增或编辑公告，则显示邮件确认弹窗。
- 如果本次只删除或批量删除公告，则不显示邮件确认，因为没有明确的“此次公告”可发送。

邮件草稿默认值：

- 标题：`系统公告`
- 正文：最近一次新增或编辑公告的 `content`。

### 邮件发送弹窗

建议抽出共享组件：

`web/src/components/settings/AnnouncementEmailBroadcastModal.jsx`

弹窗字段：

- 接收用户：必选，选项为代理商、普通用户、全量用户。
- 邮件标题：必填，可编辑。
- 邮件正文：必填，多行文本框，可编辑，默认填充本次公告内容。

正文输入延续项目现有习惯，支持 Markdown/HTML。点击发送时，前端使用已有 `marked.parse()` 将编辑后的正文转为 HTML，再提交后端。这样邮件展示和站内公告解析方式尽量一致，也避免后端新增 Markdown 依赖。

发送成功后展示发送统计：

- 已发送数量
- 跳过数量
- 失败数量

发送失败不回滚公告保存。

## 后端 API

新增接口：

`POST /api/notice/email-broadcast`

鉴权：

- `middleware.RootAuth()`

请求体：

```json
{
  "source": "notice",
  "target": "all",
  "title": "系统通知",
  "content": "<p>邮件正文 HTML</p>"
}
```

字段说明：

- `source`: `notice` 或 `announcement`，用于日志和默认语义校验。
- `target`: `agent`、`end_user` 或 `all`。
- `title`: 邮件标题，去除首尾空白后不能为空。
- `content`: 邮件 HTML 正文，去除首尾空白后不能为空。

响应：

```json
{
  "success": true,
  "message": "",
  "data": {
    "sent_count": 12,
    "skipped_count": 3,
    "failed_count": 1
  }
}
```

## 后端服务

建议新增服务文件：

`service/announcement_email_broadcast.go`

核心职责：

1. 校验 `source`、`target`、`title`、`content`。
2. 基于 `target` 查询候选用户。
3. 跳过邮箱为空的用户。
4. 对每个收件人调用现有 `common.SendEmail(title, email, content)`。
5. 返回统计结果。
6. 使用 `common.SysLog` 记录失败用户 ID 和最终统计。

为便于测试，服务层使用可替换的发送函数变量，例如：

```go
var sendAnnouncementBroadcastEmail = common.SendEmail
```

测试中替换该变量收集收件人，避免真实发信。

## 错误处理

- 参数非法：返回 `success=false`，提示具体错误。
- SMTP 未配置或部分用户发送失败：接口仍返回统计；如果全部失败，返回 `success=false` 并带统计数据。
- 邮件发送过程中单个用户失败不影响其他用户。
- 邮件群发不会回滚站内公告保存。

## 数据与兼容性

不新增数据库表，不新增迁移。

查询使用 GORM，兼容 SQLite、MySQL、PostgreSQL。

JSON 请求解析使用 `common.DecodeJson`，响应使用 Gin JSON；不在业务代码中直接调用 `encoding/json` 进行解析。

## 测试

后端单元测试：

- `target=agent` 只发送给启用代理商。
- `target=end_user` 只发送给启用普通用户，兼容空 `user_type`。
- `target=all` 发送给代理商和普通用户，不发送 admin/root。
- 邮箱为空的用户被跳过。
- 单个发送失败时继续发送后续用户，并返回失败计数。
- 非法 `target`、空标题、空正文返回错误。

前端源码测试：

- 通知保存成功后触发邮件确认流程。
- 系统公告新增/编辑保存后触发邮件确认流程。
- 删除公告保存不触发邮件确认。
- 邮件弹窗包含接收用户、邮件标题、邮件正文三个可编辑字段。
- 发送时正文使用 `marked.parse` 转换后提交，站内公告保存值不被邮件草稿改写。

## 实施顺序

1. 后端先写失败测试，覆盖收件范围和统计。
2. 实现服务层和控制器接口。
3. 注册路由。
4. 前端新增共享邮件弹窗组件和源码测试。
5. 接入 `OtherSetting`。
6. 接入 `SettingsAnnouncements`。
7. 运行后端和前端相关测试。

## 自检

- 需求完整，范围明确。
- 邮件草稿和站内公告内容互不影响。
- 全量用户的定义明确排除管理员和超管。
- 不需要数据库迁移。
- 大批量邮件采用同步逐个发送，符合本次最小实现；后续如果需要进度和重试，再单独设计异步任务。
