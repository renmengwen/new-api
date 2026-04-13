# 操练场 VolcEngine Responses 兼容设计

日期：2026-04-13

## 背景

当前操练场前端固定请求 `/pg/chat/completions`，后端 `controller.Playground` 也固定按 `RelayFormatOpenAI` 进入 relay。  
这导致 VolcEngine 中实际要求 Responses API 的模型在操练场里仍按 chat/completions 路径处理，表现为模型不可用或请求体格式不兼容。

正式链路已经补齐了这类兼容能力：

- 白名单模型识别：
  - `doubao-seed-translation-*`
  - `doubao-seed-1-6-thinking-*`
- VolcEngine `/responses` 请求的 `input.content` typed content 规范化
- `model_mapping` 别名映射识别
- `compact` 后缀优先级控制

本次目标是把同类兼容能力补到操练场，而不是重做一套新的前后端协议。

## 目标

1. 操练场用户继续使用现有界面与 `/pg/chat/completions`。
2. 当模型命中 VolcEngine Responses 白名单时，后端自动把该请求切到 Responses 转换链。
3. 普通 chat 模型保持原有行为不变。
4. 不新增新的操练场前端 endpoint，不改自定义请求体模式。

## 非目标

1. 不在本次引入 `/pg/responses` 新接口。
2. 不在本次重做前端调试面板的“最终请求预览”展示。
3. 不扩展白名单范围，仍只支持当前两类 VolcEngine 模型。

## 方案选择

### 方案 A：后端无感兼容，前端不改交互

前端继续请求 `/pg/chat/completions`，后端在操练场入口或兼容转换链中，对 VolcEngine 白名单模型自动切换到 Responses。

优点：

- 改动集中，前端风险最低
- 与“下游统一只配 `/v1`，网关内部自动转 Responses”的主目标一致
- 可直接复用现有白名单判断与 chat-to-responses 转换能力

缺点：

- 调试面板里看到的原始请求仍然是 chat 风格，不会显式展示最终上游是 Responses

### 方案 B：前后端都显式支持 Responses

前端根据模型切到 `/pg/responses` 并构造 `input`，后端新增对应入口。

优点：

- 语义最清晰

缺点：

- 改动大，需要同步调整非流式、流式、自定义请求体、调试面板
- 回归风险明显高于方案 A

### 结论

采用方案 A。

## 设计

### 1. 路由与入口

操练场仍保留单一路径 `/pg/chat/completions`。  
`controller.Playground` 不再简单固定为 `RelayFormatOpenAI` 的“纯 chat 路径”，而是在读取请求后判断：

- 如果模型命中 VolcEngine Responses 白名单，则进入与正式链路一致的 chat-to-responses 兼容链
- 否则维持原有 chat/completions 行为

### 2. 模型识别

操练场必须复用已有白名单 helper：

- `common.IsVolcEngineResponsesModel`

如果请求模型经过 `model_mapping` 重写，还要使用已补齐的映射解析：

- `helper.ResolveMappedModelName`

这样操练场与正式 relay、渠道测试三处行为保持一致，避免白名单漂移。

### 3. 请求转换

对白名单模型：

1. 继续接收前端发送的 chat/completions payload
2. 在后端复用现有 chat-to-responses 转换逻辑
3. 最终发往 VolcEngine adaptor 时，继续走已经存在的 typed content 规范化逻辑

这样可以覆盖：

- 前端普通消息输入
- `model_mapping` 别名
- 非流式与流式共用同一模型识别边界

### 4. 风险控制

以下行为必须保持不变：

1. 非白名单模型不自动转 Responses
2. 非 VolcEngine 渠道模型不受影响
3. `-openai-compact` 后缀优先级继续高于 VolcEngine Responses 白名单

## 测试

本次至少补 focused tests，覆盖：

1. 操练场白名单模型会命中 Responses 转换路径
2. 普通 VolcEngine chat 模型不会被误转
3. `model_mapping` 把别名映射到白名单模型时，操练场仍会正确命中 Responses
4. `compact` 后缀优先级不被破坏

## 实施范围

预期主要改动文件：

- `controller/playground.go`
- 如有需要，补充相关 helper 或测试文件
- 若前端确实无需改动，则不修改 `web/src`

## 验收标准

1. 操练场选择 `doubao-seed-translation-*` 可正常请求，不再因 chat/completions 路径报错。
2. 操练场选择 `doubao-seed-1-6-thinking-*` 可进入正确转换路径。
3. 操练场选择普通 `Doubao-pro/lite/vision` 模型时，行为与当前一致。
4. 现有 focused regression 继续通过。
