const IMAGE_MODEL_SUBSTRINGS = [
  'dall-e-2',
  'dall-e-3',
  'gpt-image-1',
  'gpt-image-2',
  'flux-',
  'flux.1-',
];

export function isImageGenerationModel(model) {
  const normalized = String(model || '').toLowerCase();
  if (!normalized) {
    return false;
  }

  if (normalized.startsWith('imagen-')) {
    return true;
  }

  return IMAGE_MODEL_SUBSTRINGS.some((pattern) =>
    normalized.includes(pattern),
  );
}

export function buildImageGenerationPayload(prompt, inputs = {}) {
  const payload = {
    model: inputs.model,
    prompt: String(prompt || '').trim(),
  };

  if (inputs.group) {
    payload.group = inputs.group;
  }

  return payload;
}

function getImageUrl(item) {
  if (!item || typeof item !== 'object') {
    return '';
  }
  if (item.url) {
    return item.url;
  }
  if (item.b64_json) {
    return `data:image/png;base64,${item.b64_json}`;
  }
  return '';
}

export function buildImageResponseContent(data) {
  const content = [];

  for (const item of Array.isArray(data) ? data : []) {
    const url = getImageUrl(item);
    if (url) {
      content.push({
        type: 'image_url',
        image_url: { url },
      });
    }

    if (item?.revised_prompt) {
      content.push({
        type: 'text',
        text: item.revised_prompt,
      });
    }
  }

  return content;
}
