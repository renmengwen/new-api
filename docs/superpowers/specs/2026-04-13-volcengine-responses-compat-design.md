# 火山方舟白名单 Responses 兼容设计

## 背景

当前下游通常只会把网关的 Base URL 配成 `https://www.a.com/v1`，然后继续按 OpenAI Chat Completions 接口方式调用。这个方式对大多数上游渠道是成立的，但火山方舟里有一部分模型实际要求走 Responses API，而不是 Chat Completions。

目前已经确认有两个具体问题：

1. 部分火山方舟模型应当走 `/v1/responses`，但网关当前仍会把它们当作普通的 `chat/completions` 请求处理，除非显式做特殊配置。
2. 火山方舟 Responses API 要求 `input.content` 使用 typed content 结构，例如：

```json
[
  {
    "type": "input_text",
    "text": "hello"
  }
]
```

而当前兼容转换逻辑以及渠道测试请求构造逻辑，仍可能生成普通字符串形式的 `content`，从而被火山方舟拒绝，并报出 `MissingParameter: input.content`。

## 目标

让下游客户端在继续调用 `POST /v1/chat/completions` 的前提下，也能透明地访问那些实际上要求使用 Responses API 的火山方舟模型。

## 非目标

- 不把整个火山方舟渠道类型统一强制切到 Responses API。
- 不在本次改动中把操练场重构为完整的多端点通用界面。
- 不引入数据库结构变更。

## 需求

### 功能需求

1. 只有白名单内的火山方舟模型，才会自动从 `chat/completions` 转到 `responses`。
2. 白名单规则必须可配置，且不依赖代码改动。
3. 下游直接调用 `/v1/responses` 时，同一批火山模型也必须能正常工作。
4. 渠道测试弹框的自动检测行为必须与正式 relay 行为一致。

### 兼容性要求

1. 现有本来就能正常使用 `chat/completions` 的火山模型，例如 `Doubao-pro-*`、`Doubao-lite-*`，必须继续走原有 chat 路径。
2. 非火山方舟渠道不能受到火山专用 Responses 规范化逻辑的影响。
3. 已经是合法 typed content 数组的请求内容，不能重复改写。

## 选型结论

白名单配置直接复用现有全局配置项 `global.chat_completions_to_responses_policy`，并在火山方舟适配器中增加一层 VolcEngine 专用的 Responses 请求规范化。

选择这个方案的原因：

- 正式 `/v1/chat/completions` 链路里已经存在 `chat -> responses` 的兼容转换入口。
- 现有配置结构已经支持 `channel_types` 和 `model_patterns` 两个维度。
- 路由控制可以继续集中在同一处，不需要再新增第二套重复开关。

## 设计方案

### 1. 白名单配置

白名单仍放在 `global.chat_completions_to_responses_policy` 中，不新增新的配置面。

推荐配置示例：

```json
{
  "enabled": true,
  "channel_types": [45],
  "model_patterns": [
    "^doubao-seed-translation-.*$",
    "^doubao-seed-1-6-thinking-.*$"
  ]
}
```

含义：

- `45` 代表 `ChannelTypeVolcEngine`
- 只有命中这些模型名的请求，才会在内部从 `chat/completions` 转成 `responses`

### 2. 正式 relay 路由转换

正式链路已经支持：

`/v1/chat/completions` -> 兼容转换 -> `/v1/responses`

因此本次不新增新的路由机制，重点是让转换后的 Responses 请求满足火山方舟的要求。

### 3. 火山方舟 Responses 输入规范化

在 `relay/channel/volcengine/adaptor.go` 的 `ConvertOpenAIResponsesRequest` 中增加火山方舟专用规范化逻辑。

规则如下：

1. 解析 `request.Input`
2. 对每个 input item 进行检查：
   - 如果 `content` 是字符串，则改写为：
     ```json
     [
       {
         "type": "input_text",
         "text": "<原始字符串>"
       }
     ]
     ```
   - 如果 `content` 已经是数组，则保持不变
   - 如果 `content` 缺失或为 null，则维持当前行为
3. 该逻辑仅作用于火山方舟的 Responses 请求

这层规范化必须同时覆盖两种入口：

- 下游直接调用 `/v1/responses`
- 下游调用 `/v1/chat/completions`，但被内部自动转换为 `responses`

### 4. 渠道测试对齐

需要修改渠道测试逻辑，让火山方舟白名单模型在“自动检测”下也被当作 Responses 模型处理。

涉及范围：

- `normalizeChannelTestEndpoint`
- 自动检测时的请求路径选择
- Responses 测试请求构造

对于火山方舟白名单模型，测试请求应使用 typed content，例如：

```json
{
  "model": "<model>",
  "input": [
    {
      "role": "user",
      "content": [
        {
          "type": "input_text",
          "text": "hi"
        }
      ]
    }
  ]
}
```

这样可以避免当前出现的假失败：正式 relay 逻辑兼容后可以正常调用，但后台测试弹框仍然因为请求体格式不对而失败。

### 5. 共享模型分类辅助函数

增加一个火山方舟 Responses 白名单模型识别 helper，用于复用同一套匹配规则。

首批匹配前缀建议只包含：

- `doubao-seed-translation-`
- `doubao-seed-1-6-thinking-`

该 helper 后续可被以下位置共同使用：

- 正式 relay 逻辑的默认判定或辅助逻辑
- 渠道测试自动检测逻辑

这样后续扩展新的火山 Responses-only 模型时，只需要补前缀，不需要重新设计整套路由架构。

## 错误处理

### 预期改善

- 白名单模型通过 `/v1/chat/completions` 调用时，不再出现 `does not support this api`
- 网关构造的火山方舟 Responses 请求，不再出现 `MissingParameter input.content`

### 有意保留的限制

- 未进入白名单的模型，网关仍保持当前行为，不做猜测性转换
- 如果上游模型除 typed text content 之外还要求额外参数，网关应继续原样透出上游校验错误，而不是擅自伪造字段

## 测试方案

### 单元测试

1. 火山方舟白名单模型匹配测试
2. 火山方舟 Responses 规范化测试：
   - 字符串 content -> typed content 数组
   - 已有 typed content 数组保持不变
   - 非火山渠道不受影响
3. 渠道测试自动检测对白名单火山模型的识别测试

### 集成风格测试

1. `chat/completions` 请求命中火山白名单模型后，正确触发 `chat -> responses`
2. 直接 `/v1/responses` 请求白名单模型时，合法 typed content 保持不变

### 人工验证

1. 下游配置 `https://www.a.com/v1`，调用 `chat/completions`，使用白名单火山模型，能够成功
2. 下游调用 `chat/completions`，使用普通 `Doubao-pro-*` 模型，仍然继续走原来的 chat 路径并成功
3. 后台渠道测试弹框在“自动检测”下测试白名单火山模型时能够成功

## 实施步骤

1. 增加火山方舟白名单模型识别 helper
2. 在火山方舟 adaptor 中增加 Responses 输入规范化
3. 让渠道测试自动检测与请求构造复用同一套白名单识别逻辑
4. 补充聚焦的转换与检测测试

## 风险

1. 某些火山模型可能同时支持 chat 和 responses，如果白名单范围过大，可能会把本不需要转换的模型也路由到 responses
2. 某些多模态或工具调用模型未来可能还需要额外的火山专用规范化

当前风险控制方式：

- 初始白名单保持严格收敛
- 仅使用显式前缀匹配
- 仅在火山方舟 Responses 路径下做内容规范化
