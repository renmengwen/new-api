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
    "metadata": {
      "ratio": "16:9",
      "duration": 5,
      "watermark": false
    }
  }'
```

## 请求字段说明

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `model` | string | 是 | 视频模型名称 |
| `prompt` | string | 是 | 文生视频提示词 |
| `image` | string | 否 | 单张图片 URL，用于图生视频 |
| `images` | array[string] | 否 | 多张图片 URL |
| `metadata` | object | 否 | Seedance 扩展参数 |

## `metadata` 常用字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `ratio` | string | 否 | 画面比例，如 `16:9` |
| `duration` | int | 否 | 视频时长，单位秒 |
| `watermark` | bool | 否 | 是否添加水印 |
| `generate_audio` | bool | 否 | 是否生成音频，模型相关 |
| `resolution` | string | 否 | 分辨率，模型相关，部分模型或模式不支持 |
| `service_tier` | string | 否 | 服务等级，模型相关 |
| `video_url` | string | 否 | 单个参考视频 URL |
| `audio_url` | string | 否 | 单个参考音频 URL |
| `videos` | array[string] | 否 | 多个参考视频 URL |
| `audios` | array[string] | 否 | 多个参考音频 URL |
| `seed` | int | 否 | 随机种子 |

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

## 响应字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 任务 ID |
| `model` | string | 模型名称 |
| `status` | string | 任务状态：`pending` / `processing` / `succeeded` / `failed` |
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
- 如果上游返回参数错误，建议先只保留以下最小参数：
  - `model`
  - `prompt`
  - `metadata.ratio`
  - `metadata.duration`
  - `metadata.watermark`
