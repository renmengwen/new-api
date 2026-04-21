const BASE_URL_PLACEHOLDER = '{{base_url}}';

const DEFAULT_AUTH = {
  type: 'bearer',
  location: 'header',
  example: 'Authorization: Bearer sk-xxxxxxxx',
};

export const AI_MODEL_DOC_DEFAULT_ID = 'audio-native-gemini';

export const AI_MODEL_DOC_GROUPS = [
  { key: 'audio', title: 'Audio' },
  { key: 'chat', title: 'Chat' },
  { key: 'completions', title: 'Completions' },
  { key: 'embeddings', title: 'Embeddings' },
  { key: 'images', title: 'Images' },
  { key: 'models', title: 'Models' },
  { key: 'moderations', title: 'Moderations' },
  { key: 'realtime', title: 'Realtime' },
  { key: 'rerank', title: 'Rerank' },
  { key: 'unimplemented', title: 'Unimplemented' },
  { key: 'videos', title: 'Videos' },
];

const AI_MODEL_DOC_GROUP_KEYS = new Set(AI_MODEL_DOC_GROUPS.map((group) => group.key));

const makeJsonRequestExample = (method, path, requestBody) => {
  const lines = [`curl -X ${method} '${BASE_URL_PLACEHOLDER}${path}'`, `  -H 'Authorization: Bearer sk-xxxxxxxx'`];

  if (requestBody !== undefined) {
    lines.push(`  -H 'Content-Type: application/json'`);
    lines.push(`  -d '${JSON.stringify(requestBody)}'`);
  }

  return lines.join(' \\\n');
};

const makeGetRequestExample = (path) =>
  [`curl '${BASE_URL_PLACEHOLDER}${path}'`, `  -H 'Authorization: Bearer sk-xxxxxxxx'`].join(' \\\n');

const makeMultipartRequestExample = (method, path, multipartFields) => {
  const lines = [`curl -X ${method} '${BASE_URL_PLACEHOLDER}${path}'`, `  -H 'Authorization: Bearer sk-xxxxxxxx'`];

  multipartFields.forEach((field) => {
    lines.push(`  -F '${field}'`);
  });

  return lines.join(' \\\n');
};

const makeResponseExample = (responseBody) => JSON.stringify(responseBody, null, 2);

const createDoc = ({
  id,
  groupKey,
  title,
  method,
  path,
  summary,
  description,
  transport = 'json',
  requestBody,
  multipartFields,
  requestExample,
  responseBody,
}) => {
  if (!AI_MODEL_DOC_GROUP_KEYS.has(groupKey)) {
    throw new Error(`Unknown AI model docs group key: ${groupKey}`);
  }

  const doc = {
    id,
    groupKey,
    title,
    method,
    path,
    summary,
    description,
    transport,
    auth: { ...DEFAULT_AUTH },
    responseExample: makeResponseExample(responseBody),
  };

  if (requestExample) {
    doc.requestExample = requestExample;
  } else if (transport === 'get') {
    doc.requestExample = makeGetRequestExample(path);
  } else if (transport === 'multipart') {
    doc.requestExample = makeMultipartRequestExample(method, path, multipartFields || []);
  } else {
    doc.requestExample = makeJsonRequestExample(method, path, requestBody);
  }

  return doc;
};

export const AI_MODEL_DOC_ITEMS = [
  createDoc({
    id: 'audio-native-gemini',
    groupKey: 'audio',
    title: '原生 Gemini 格式',
    method: 'POST',
    path: '/v1beta/models/{model}:generateContent',
    summary: '使用 Gemini 原生多模态请求格式处理音频输入。',
    description: '适合需要上传语音、音频片段或混合内容的场景，展示 Gemini 原生调用方式。',
    requestBody: {
      model: 'gemini-2.0-flash',
      contents: [
        {
          role: 'user',
          parts: [
            { text: '请转写这段音频。' },
            { inline_data: { mime_type: 'audio/wav', data: 'BASE64_AUDIO' } },
          ],
        },
      ],
    },
    responseBody: {
      candidates: [
        {
          content: { parts: [{ text: '音频转写结果。' }] },
        },
      ],
    },
  }),
  createDoc({
    id: 'audio-native-openai',
    groupKey: 'audio',
    title: '?? OpenAI ??',
    method: 'POST',
    transport: 'multipart',
    path: '/v1/audio/transcriptions',
    summary: '?? OpenAI ???????????????',
    description: '?? OpenAI ????????????????????????',
    multipartFields: ['model=whisper-1', 'file=@audio.wav', 'response_format=json'],
    responseBody: {
      text: '???????',
    },
  }),
  createDoc({
    id: 'chat-native-claude',
    groupKey: 'chat',
    title: '原生 Claude 格式',
    method: 'POST',
    path: '/v1/messages',
    summary: '使用 Claude 消息协议发送对话请求。',
    description: '适合 Anthropic/Claude 兼容客户端，展示消息数组和最大输出长度设置。',
    requestBody: {
      model: 'claude-3-5-sonnet',
      max_tokens: 1024,
      messages: [{ role: 'user', content: '你好。' }],
    },
    responseBody: {
      content: [{ type: 'text', text: '你好，我是 Claude。' }],
    },
  }),
  createDoc({
    id: 'chat-gemini-media-recognition',
    groupKey: 'chat',
    title: 'Gemini 媒体识别',
    method: 'POST',
    path: '/v1beta/models/{model}:generateContent',
    summary: '基于 Gemini 的多模态输入识别图片、音频或视频片段。',
    description: '用于演示媒体识别类对话请求，强调 parts 结构中混合媒体与文本。',
    requestBody: {
      model: 'gemini-2.0-flash',
      contents: [
        {
          role: 'user',
          parts: [{ text: '请描述这张图片里的主要内容。' }],
        },
      ],
    },
    responseBody: {
      candidates: [
        {
          content: { parts: [{ text: '图片里是一只坐着的猫。' }] },
        },
      ],
    },
  }),
  createDoc({
    id: 'chat-gemini-text-chat',
    groupKey: 'chat',
    title: 'Gemini 文本聊天',
    method: 'POST',
    path: '/v1beta/models/{model}:generateContent',
    summary: '使用 Gemini 原生文本聊天格式发起普通对话。',
    description: '适合纯文本问答和多轮聊天场景，保留 Gemini 的 contents 提交方式。',
    requestBody: {
      model: 'gemini-2.0-flash',
      contents: [{ role: 'user', parts: [{ text: '你好，请介绍一下这个接口。' }] }],
    },
    responseBody: {
      candidates: [
        {
          content: { parts: [{ text: '这是一个 Gemini 文本聊天示例。' }] },
        },
      ],
    },
  }),
  createDoc({
    id: 'chat-openai-chat-completions',
    groupKey: 'chat',
    title: 'ChatCompletions 格式',
    method: 'POST',
    path: '/v1/chat/completions',
    summary: '使用 OpenAI ChatCompletions 兼容格式发起对话。',
    description: '这是最常见的兼容入口之一，适合现有 SDK 直接切换到本服务。',
    requestBody: {
      model: 'gpt-4o-mini',
      messages: [{ role: 'user', content: 'Hello!' }],
      temperature: 0.7,
    },
    responseBody: {
      choices: [
        {
          index: 0,
          message: { role: 'assistant', content: 'Hello!' },
        },
      ],
    },
  }),
  createDoc({
    id: 'chat-openai-responses',
    groupKey: 'chat',
    title: 'Responses 格式',
    method: 'POST',
    path: '/v1/responses',
    summary: '使用 OpenAI Responses API 进行统一推理调用。',
    description: '适合需要响应对象式输出的客户端，展示新的 input/output 结构。',
    requestBody: {
      model: 'gpt-4.1-mini',
      input: 'Hello!',
    },
    responseBody: {
      output: [
        {
          type: 'message',
          role: 'assistant',
          content: [{ type: 'output_text', text: 'Hello!' }],
        },
      ],
    },
  }),
  createDoc({
    id: 'completions-native-openai',
    groupKey: 'completions',
    title: '原生 OpenAI 格式',
    method: 'POST',
    path: '/v1/completions',
    summary: '使用 OpenAI 经典补全接口生成续写内容。',
    description: '适合仍然依赖 prompt/completion 模型的旧客户端。',
    requestBody: {
      model: 'gpt-3.5-turbo-instruct',
      prompt: 'Complete this sentence:',
    },
    responseBody: {
      choices: [{ text: ' ... and it continues.' }],
    },
  }),
  createDoc({
    id: 'embeddings-native-openai',
    groupKey: 'embeddings',
    title: '原生 OpenAI 格式',
    method: 'POST',
    path: '/v1/embeddings',
    summary: '使用 OpenAI Embeddings 接口生成向量表示。',
    description: '适合文本检索、语义匹配和向量数据库写入流程。',
    requestBody: {
      model: 'text-embedding-3-small',
      input: ['hello', 'world'],
    },
    responseBody: {
      data: [{ index: 0, embedding: [0.01, 0.02, 0.03] }],
    },
  }),
  createDoc({
    id: 'embeddings-native-gemini',
    groupKey: 'embeddings',
    title: '原生 Gemini 格式',
    method: 'POST',
    path: '/v1beta/models/{model}:embedContent',
    summary: '使用 Gemini 原生接口生成向量嵌入。',
    description: '适合需要 Gemini 原生 embedding 调用的场景，保持与其内容结构一致。',
    requestBody: {
      model: 'text-embedding-004',
      content: 'hello world',
    },
    responseBody: {
      embedding: { values: [0.01, 0.02, 0.03] },
    },
  }),
  createDoc({
    id: 'images-gemini-native',
    groupKey: 'images',
    title: 'Gemini 原生格式',
    method: 'POST',
    path: '/v1beta/models/{model}:generateImages',
    summary: '使用 Gemini 原生接口生成图片。',
    description: '用于展示 Gemini 图像生成链路，保留原生请求体语义。',
    requestBody: {
      model: 'gemini-2.0-flash-exp-image-generation',
      prompt: '一只戴墨镜的猫。',
    },
    responseBody: {
      images: [{ mime_type: 'image/png', data: 'BASE64_IMAGE' }],
    },
  }),
  createDoc({
    id: 'images-gemini-openai-chat',
    groupKey: 'images',
    title: 'OpenAI 聊天格式',
    method: 'POST',
    path: '/v1/chat/completions',
    summary: '使用聊天格式驱动 Gemini 图片生成能力。',
    description: '面向以 ChatCompletions 方式调用图像能力的兼容场景。',
    requestBody: {
      model: 'gemini-2.0-flash',
      messages: [{ role: 'user', content: '生成一张海报。' }],
    },
    responseBody: {
      choices: [
        {
          message: { role: 'assistant', content: '已生成图片结果。' },
        },
      ],
    },
  }),
  createDoc({
    id: 'images-openai-edit',
    groupKey: 'images',
    title: '????',
    method: 'POST',
    transport: 'multipart',
    path: '/v1/images/edits',
    summary: '?? OpenAI ????????????????',
    description: '????????????????????????',
    multipartFields: ['model=gpt-image-1', 'image=@input.png', 'prompt=????????'],
    responseBody: {
      data: [{ url: 'https://example.com/edited.png' }],
    },
  }),
  createDoc({
    id: 'images-openai-generate',
    groupKey: 'images',
    title: '生成图像',
    method: 'POST',
    path: '/v1/images/generations',
    summary: '使用 OpenAI 风格的图片生成接口创建新图像。',
    description: '适合从提示词直接生成图片的标准兼容路径。',
    requestBody: {
      model: 'gpt-image-1',
      prompt: '一座未来城市。',
    },
    responseBody: {
      data: [{ url: 'https://example.com/generated.png' }],
    },
  }),
  createDoc({
    id: 'images-qwen-generate',
    groupKey: 'images',
    title: '生成图像',
    method: 'POST',
    path: '/v1/images/generations',
    summary: '使用 Qwen 风格参数生成图像。',
    description: '保留 Qwen 图像生成的兼容入口，便于前端统一调试。',
    requestBody: {
      model: 'qwen-image',
      prompt: '国风插画。',
    },
    responseBody: {
      data: [{ url: 'https://example.com/qwen-generated.png' }],
    },
  }),
  createDoc({
    id: 'images-qwen-edit',
    groupKey: 'images',
    title: '????',
    method: 'POST',
    transport: 'multipart',
    path: '/v1/images/edits',
    summary: '?? Qwen ???????????',
    description: '?? Qwen ??????????????',
    multipartFields: ['model=qwen-image', 'image=@input.png', 'prompt=???????????'],
    responseBody: {
      data: [{ url: 'https://example.com/qwen-edited.png' }],
    },
  }),
  createDoc({
    id: 'models-native-openai',
    groupKey: 'models',
    title: '?? OpenAI ??',
    method: 'GET',
    transport: 'get',
    path: '/v1/models',
    summary: '?? OpenAI ???????',
    description: '?????????????????',
    responseBody: {
      data: [{ id: 'gpt-4o-mini', object: 'model' }],
    },
  }),
  createDoc({
    id: 'models-native-gemini',
    groupKey: 'models',
    title: '?? Gemini ??',
    method: 'GET',
    transport: 'get',
    path: '/v1beta/models',
    summary: '?? Gemini ???????',
    description: '???? Gemini ????????????',
    responseBody: {
      models: [{ name: 'models/gemini-2.0-flash', displayName: 'Gemini 2.0 Flash' }],
    },
  }),
  createDoc({
    id: 'moderations-native-openai',
    groupKey: 'moderations',
    title: '原生 OpenAI 格式',
    method: 'POST',
    path: '/v1/moderations',
    summary: '使用 OpenAI 兼容内容审核接口检查输入文本。',
    description: '适合在请求进入主模型之前先做安全审查。',
    requestBody: {
      model: 'omni-moderation-latest',
      input: '违规内容示例',
    },
    responseBody: {
      results: [{ flagged: false, categories: {} }],
    },
  }),
  createDoc({
    id: 'realtime-native-openai',
    groupKey: 'realtime',
    title: '?? OpenAI ??',
    method: 'GET',
    transport: 'get',
    path: '/v1/realtime',
    summary: '?? OpenAI Realtime ????????',
    description: '??????????????????????',
    responseBody: {
      model: 'gpt-realtime',
      status: 'ready',
    },
  }),
  createDoc({
    id: 'rerank-document',
    groupKey: 'rerank',
    title: '文档重排序',
    method: 'POST',
    path: '/v1/rerank',
    summary: '根据查询词对候选文档进行相关性排序。',
    description: '适合搜索增强和检索后排序的场景。',
    requestBody: {
      model: 'rerank-v3',
      query: 'AI 模型接口',
      documents: ['文档 1', '文档 2'],
    },
    responseBody: {
      results: [{ index: 0, relevance_score: 0.98 }],
    },
  }),
  createDoc({
    id: 'unimplemented-files',
    groupKey: 'unimplemented',
    title: '文件',
    method: 'POST',
    path: '/v1/files',
    summary: '文件管理接口占位，后续补全。',
    description: '当前仅保留目录结构和占位文档，便于后续接入后端实现。',
    requestBody: {
      file_name: 'example.pdf',
    },
    responseBody: {
      message: '待后续实现',
    },
  }),
  createDoc({
    id: 'unimplemented-fine-tuning',
    groupKey: 'unimplemented',
    title: '微调',
    method: 'POST',
    path: '/v1/fine_tuning/jobs',
    summary: '微调任务接口占位，后续补全。',
    description: '先保留菜单和详情页入口，后续再挂接真实任务创建流程。',
    requestBody: {
      training_file: 'file-123',
    },
    responseBody: {
      message: '待后续实现',
    },
  }),
  createDoc({
    id: 'videos-create-task',
    groupKey: 'videos',
    title: '创建视频生成任务',
    method: 'POST',
    path: '/v1/videos/tasks',
    summary: '提交视频生成请求并创建异步任务。',
    description: '适合需要先提交任务、再轮询状态的视频生成工作流。',
    requestBody: {
      model: 'sora',
      prompt: '一只狗在公园散步。',
    },
    responseBody: {
      task_id: 'vt_123',
      status: 'queued',
    },
  }),
  createDoc({
    id: 'videos-get-task',
    groupKey: 'videos',
    title: '??????????',
    method: 'GET',
    transport: 'get',
    path: '/v1/videos/tasks/{task_id}',
    summary: '????????????????',
    description: '????????????????',
    responseBody: {
      task_id: 'vt_123',
      status: 'processing',
      progress: 50,
    },
  }),
  createDoc({
    id: 'videos-jimeng',
    groupKey: 'videos',
    title: '即梦格式',
    method: 'POST',
    path: '/v1/videos/generate',
    summary: '即梦视频接口占位文档。',
    description: '先保留即梦兼容入口，后续再补充完整请求参数和返回结构。',
    requestBody: {
      prompt: '国风视频。',
    },
    responseBody: {
      message: '待后续实现',
    },
  }),
  createDoc({
    id: 'videos-kling',
    groupKey: 'videos',
    title: '可灵格式',
    method: 'POST',
    path: '/v1/videos/generate',
    summary: '可灵视频接口占位文档。',
    description: '保留可灵兼容入口，便于后续接入真实任务创建逻辑。',
    requestBody: {
      prompt: '商品宣传视频。',
    },
    responseBody: {
      message: '待后续实现',
    },
  }),
  createDoc({
    id: 'videos-sora',
    groupKey: 'videos',
    title: 'Sora 格式',
    method: 'POST',
    path: '/v1/videos/generate',
    summary: 'Sora 视频接口占位文档。',
    description: '先保留 Sora 入口和页面结构，后续替换成真实后端实现。',
    requestBody: {
      prompt: '城市航拍。',
    },
    responseBody: {
      message: '待后续实现',
    },
  }),
];

const AI_MODEL_DOC_BY_ID = new Map(AI_MODEL_DOC_ITEMS.map((item) => [item.id, item]));

export function resolveAiModelDocId(docId) {
  return AI_MODEL_DOC_BY_ID.has(docId) ? docId : AI_MODEL_DOC_DEFAULT_ID;
}

export function getAiModelDocById(docId) {
  return AI_MODEL_DOC_BY_ID.get(resolveAiModelDocId(docId));
}

export function buildAiModelDocRoute(docId) {
  return `/console/docs/ai-model/${resolveAiModelDocId(docId)}`;
}

export function buildAiModelDocTree() {
  const itemsByGroup = new Map(AI_MODEL_DOC_GROUPS.map((group) => [group.key, []]));

  AI_MODEL_DOC_ITEMS.forEach((item) => {
    const groupItems = itemsByGroup.get(item.groupKey);
    if (!groupItems) {
      throw new Error(`Unknown AI model doc group key: ${item.groupKey}`);
    }
    groupItems.push(item);
  });

  return AI_MODEL_DOC_GROUPS.map((group) => ({
    ...group,
    items: itemsByGroup.get(group.key) || [],
  })).filter((group) => group.items.length > 0);
}
