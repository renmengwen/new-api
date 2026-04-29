# Seedance 视频任务接口文档

## 接口列表

- `POST /v1/video/generations`
- `GET /v1/video/generations/{task_id}`
- `GET /v1/video/generations`
- `DELETE /v1/video/generations/{task_id}`

---

## 通用说明

### 请求头

```http
Authorization: Bearer sk-xxxxxx
Content-Type: application/json
```

### 通用约定

- `task_id` 为网关生成的公网任务 ID，格式通常为 `task_xxx`
- 查询接口当前读取的是本地任务快照
- 任务成功后，`content.video_url` 通常为上游火山返回的视频地址
- 当前状态值：
  - `pending`
  - `processing`
  - `succeeded`
  - `cancelled`
  - `failed`

### 测试环境变量示例

```text
base_url = https://www-test.linkaihub.com
token    = sk-xxxxxx
task_id  = task_xxx
```

---

# 1. 创建视频生成任务

## 接口

```http
POST /v1/video/generations
```

## 请求示例

```bash
curl -X POST "{{base_url}}/v1/video/generations" \
  -H "Authorization: Bearer {{token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-2-0-260128",
    "prompt": "第一人称果茶广告，明亮商业短片风格，镜头稳定，结尾举杯展示产品。",
    "size": "1280x720",
    "metadata": {
      "aspect_ratio": "16:9",
      "resolution": "720p",
      "duration": 5,
      "input_video": false,
      "watermark": false
    }
  }'
```

## 请求字段说明

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `model` | string | 是 | 视频模型名称 |
| `prompt` | string | 是 | 文生视频提示词 |
| `images` | array[string] | 否 | 图片引用列表。如果只传一张图，也请放在数组中 |
| `size` | string | 否 | 输出尺寸，例如 `1280x720`。高级定价匹配时建议与 `metadata.aspect_ratio`、`metadata.resolution` 一起传，避免宽高比推导缺失 |
| `metadata` | object | 否 | Seedance 扩展参数 |

## `metadata` 常用字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `ratio` | string | 否 | 画面比例，如 `16:9` |
| `aspect_ratio` | string | 否 | 画面比例，如 `16:9`。**启用高级定价规则匹配时建议传；当前匹配同时兼容 `aspect_ratio` 与 `ratio`** |
| `duration` | int | 否 | 视频时长，单位秒 |
| `watermark` | bool | 否 | 是否添加水印 |
| `generate_audio` | bool | 否 | 是否生成音频，模型相关 |
| `resolution` | string | 否 | 分辨率，模型相关，部分模型或模式不支持 |
| `service_tier` | string | 否 | 服务等级，模型相关 |
| `input_video` | bool | 否 | 是否显式声明“本次请求带参考视频”。**命中高级定价规则时建议显式传；无输入视频传 `false`，输入含视频传 `true`** |
| `input_video_duration` | int | 否 | 输入参考视频时长，单位秒。**命中高级定价中的输入视频时长区间时必须显式传递** |
| `image_roles` | array[string] | 否 | 与 `images` 按顺序一一对应的图片角色列表。常用值：`reference_image`、`first_frame`、`last_frame` |
| `videos` | array[string] | 否 | 参考视频 URL 列表，会转换为火山 `content[]`，默认 `role=reference_video`；如果只传一个视频，也请放在数组中 |
| `audios` | array[string] | 否 | 参考音频 URL 列表，会转换为火山 `content[]`，默认 `role=reference_audio`；如果只传一个音频，也请放在数组中 |
| `seed` | int | 否 | 随机种子 |

## 高级定价规则匹配说明

如果某个视频模型已经切换到“高级规则生效”，请求体建议遵循以下约定，否则可能出现“`advanced pricing did not match any active rule`”：

- 建议同时传：
  - `size`
  - `metadata.aspect_ratio` 或 `metadata.ratio`
  - `metadata.resolution`
  - `metadata.duration`
- 如果要命中“无输入视频”规则：
  - 建议显式传 `metadata.input_video=false`
- 如果要命中“输入含视频”规则：
  - 必须显式传 `metadata.input_video=true`
  - 并显式传 `metadata.input_video_duration`
- 当前高级定价匹配会优先读取 `metadata.aspect_ratio`，同时兼容 `metadata.ratio`；为减少歧义，仍建议统一使用 `metadata.aspect_ratio`

示例：

- 想命中“输入含视频 / 720p / 16:9 / 输出 5 秒 / 输入视频 2~15 秒”这类规则时，必须同时满足：
  - `videos` 已传
  - `input_video=true`
  - `input_video_duration` 落在规则区间内
  - `aspect_ratio=16:9`
  - `resolution=720p`
  - `duration=5`

- 想命中“无输入视频 / 720p / 16:9 / 输出 5 秒”这类规则时，建议同时满足：
  - `input_video=false`
  - `aspect_ratio=16:9`
  - `resolution=720p`
  - `duration=5`

## 图片引用与 role 规则

- 图生视频首帧、图生视频首尾帧、多模态参考视频，这 3 种场景按火山官方约束是互斥的，不建议混用
- 单图图生视频：
  - 传 `images`，长度为 1
  - 当前网关会保持图片项不带 `role`，交给上游按首帧场景处理
- 多模态参考视频：
  - 传 `images`、`videos`、`audios`
  - 当前网关会把图片默认转成 `role=reference_image`
- 首尾帧视频：
  - 传 `images`
  - 同时传 `metadata.image_roles`
  - 例如：`["first_frame", "last_frame"]`

## 首尾帧请求示例

```json
{
  "model": "doubao-seedance-1-5-pro-251215",
  "prompt": "图中女孩对着镜头说茄子，360度环绕运镜",
  "images": [
    "https://ark-project.tos-cn-beijing.volces.com/doc_image/seepro_first_frame.jpeg",
    "https://ark-project.tos-cn-beijing.volces.com/doc_image/seepro_last_frame.jpeg"
  ],
  "metadata": {
    "image_roles": [
      "first_frame",
      "last_frame"
    ],
    "generate_audio": true,
    "ratio": "adaptive",
    "duration": 5,
    "watermark": false
  }
}
```

## 多模态参考请求示例

```json
{
  "model": "doubao-seedance-2-0-260128",
  "prompt": "全程使用视频1的第一视角构图，全程使用音频1作为背景音乐。",
  "size": "1280x720",
  "images": [
    "https://ark-project.tos-cn-beijing.volces.com/doc_image/r2v_tea_pic1.jpg",
    "https://ark-project.tos-cn-beijing.volces.com/doc_image/r2v_tea_pic2.jpg"
  ],
  "metadata": {
    "videos": [
      "https://ark-project.tos-cn-beijing.volces.com/doc_video/r2v_tea_video1.mp4"
    ],
    "audios": [
      "https://ark-project.tos-cn-beijing.volces.com/doc_audio/r2v_tea_audio1.mp3"
    ],
    "generate_audio": true,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "duration": 11,
    "watermark": false
  }
}
```

## 命中“输入含视频”高级规则示例

下面这份请求体适合命中类似以下媒体任务规则：

- 输入含视频
- `720p`
- `16:9`
- 输出 `5` 秒
- 输入视频时长 `2~15` 秒

```json
{
  "model": "doubao-seedance-2-0-260128",
  "prompt": "基于输入视频延展生成一段镜头运动更流畅、光影更统一的短片，保持主体风格一致，电影感构图。",
  "size": "1280x720",
  "metadata": {
    "videos": [
      "https://ark-project.tos-cn-beijing.volces.com/doc_video/r2v_tea_video1.mp4"
    ],
    "input_video": true,
    "input_video_duration": 3,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "duration": 5,
    "watermark": false
  }
}
```

## 成功响应示例

```json
{
  "id": "task_01jsk2v8m4m4g9m2v7n3x8q1af"
}
```

## 响应字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 创建成功后返回的任务 ID，后续查询和删除使用该值 |

---

# 2. 查询单个视频生成任务

## 接口

```http
GET /v1/video/generations/{task_id}
```

## 请求示例

```bash
curl "{{base_url}}/v1/video/generations/{{task_id}}" \
  -H "Authorization: Bearer {{token}}"
```

## 路径参数说明

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `task_id` | string | 是 | 创建接口返回的任务 ID |

## 处理中响应示例

```json
{
  "id": "task_01jsk2v8m4m4g9m2v7n3x8q1af",
  "model": "doubao-seedance-2-0-260128",
  "status": "processing",
  "created_at": 1713088800,
  "updated_at": 1713088815
}
```

## 成功响应示例

```json
{
  "id": "task_01jsk2v8m4m4g9m2v7n3x8q1af",
  "model": "doubao-seedance-2-0-260128",
  "status": "succeeded",
  "content": {
    "video_url": "https://xxx.tos-cn-beijing.volces.com/xxx/output.mp4"
  },
  "duration": 5,
  "ratio": "16:9",
  "created_at": 1713088800,
  "updated_at": 1713088865
}
```

## 失败响应示例

```json
{
  "id": "task_01jsk2v8m4m4g9m2v7n3x8q1af",
  "model": "doubao-seedance-2-0-260128",
  "status": "failed",
  "created_at": 1713088800,
  "updated_at": 1713088820
}
```

## 已取消响应示例

```json
{
  "id": "task_01jsk2v8m4m4g9m2v7n3x8q1af",
  "model": "doubao-seedance-2-0-260128",
  "status": "cancelled",
  "created_at": 1713088800,
  "updated_at": 1713088820
}
```

## 响应字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 任务 ID |
| `model` | string | 模型名称 |
| `status` | string | 任务状态：`pending` / `processing` / `succeeded` / `cancelled` / `failed` |
| `content.video_url` | string | 视频结果地址，成功时通常有值 |
| `seed` | int | 随机种子，若上游返回则透出 |
| `resolution` | string | 分辨率 |
| `duration` | int | 视频时长 |
| `ratio` | string | 画面比例 |
| `framespersecond` | int | 帧率 |
| `service_tier` | string | 服务等级 |
| `usage.completion_tokens` | int | 完成 token 数 |
| `usage.total_tokens` | int | 总 token 数 |
| `created_at` | int64 | 创建时间，Unix 时间戳 |
| `updated_at` | int64 | 更新时间，Unix 时间戳 |

---

# 3. 查询视频生成任务列表

## 接口

```http
GET /v1/video/generations
```

## 请求示例

```bash
curl "{{base_url}}/v1/video/generations?page_num=1&page_size=10&filter.status=processing" \
  -H "Authorization: Bearer {{token}}"
```

## 查询参数说明

| 参数 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `page_num` | int | 否 | 页码，默认 `1` |
| `page_size` | int | 否 | 每页条数，默认 `10` |
| `filter.status` | string | 否 | 按状态过滤 |
| `filter.task_ids` | string / array[string] | 否 | 按任务 ID 过滤，可重复传或逗号分隔 |

## 状态过滤可选值

- `pending`
- `processing`
- `succeeded`
- `cancelled`
- `failed`

## 请求示例：按任务 ID 查询

```bash
curl "{{base_url}}/v1/video/generations?page_num=1&page_size=10&filter.task_ids={{task_id}}" \
  -H "Authorization: Bearer {{token}}"
```

## 成功响应示例

```json
{
  "total": 2,
  "items": [
    {
      "id": "task_01jsk2v8m4m4g9m2v7n3x8q1af",
      "model": "doubao-seedance-2-0-260128",
      "status": "processing",
      "created_at": 1713088800,
      "updated_at": 1713088815
    },
    {
      "id": "task_01jsk2w1h9k3q2c6v0b8n4d7pe",
      "model": "doubao-seedance-2-0-260128",
      "status": "succeeded",
      "content": {
        "video_url": "https://xxx.tos-cn-beijing.volces.com/xxx/output2.mp4"
      },
      "duration": 5,
      "ratio": "16:9",
      "created_at": 1713088700,
      "updated_at": 1713088760
    }
  ]
}
```

## 响应字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| `total` | int64 | 符合条件的任务总数 |
| `items` | array | 当前页任务列表 |
| `items[].id` | string | 任务 ID |
| `items[].model` | string | 模型名称 |
| `items[].status` | string | 任务状态 |
| `items[].content.video_url` | string | 视频结果地址，成功时可能返回 |
| `items[].duration` | int | 视频时长 |
| `items[].ratio` | string | 画面比例 |
| `items[].created_at` | int64 | 创建时间 |
| `items[].updated_at` | int64 | 更新时间 |

---

# 4. 删除或取消视频生成任务

## 接口

```http
DELETE /v1/video/generations/{task_id}
```

## 请求示例

```bash
curl -X DELETE "{{base_url}}/v1/video/generations/{{task_id}}" \
  -H "Authorization: Bearer {{token}}"
```

## 路径参数说明

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `task_id` | string | 是 | 创建接口返回的任务 ID |

## 成功响应示例

```json
{}
```

## 响应说明

- 当前删除成功返回空对象 `{}`
- 火山官方 `DELETE` 语义是：
  - `queued`：取消排队，后续任务状态会变成 `cancelled`
  - `running`：不支持删除，会上游返回 `409 Conflict`
  - `succeeded` / `failed` / `expired`：删除任务记录，后续不再支持查询

---

# 推荐测试顺序

1. 调用创建接口获取 `task_id`
2. 使用 `task_id` 调用单任务查询
3. 轮询查询直到 `status = succeeded`
4. 调用任务列表确认任务存在
5. 如需测试取消或删除，再调用删除接口

---

# 注意事项

- 当前创建接口使用的是网关封装后的请求格式
- `resolution`、`service_tier`、`generate_audio` 等字段是否可用，取决于具体模型和模式
- 如果上游返回参数错误，建议先保留以下基础参数；启用高级定价时不要省略 `resolution` 和 `input_video`：
  - `model`
  - `prompt`
  - `size`
  - `metadata.aspect_ratio`
  - `metadata.resolution`
  - `metadata.duration`
  - `metadata.input_video`
  - `metadata.watermark`
