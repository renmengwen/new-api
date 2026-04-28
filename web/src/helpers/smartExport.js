import {
  downloadBlobFile,
  EXCEL_BLOB_MIME_TYPE,
  extractDownloadFilename,
} from './exportExcel.js';
import { pollAsyncExportJob } from './asyncExport.js';

const JSON_CONTENT_TYPE_PATTERN = /(^|\/|\+)json\b/i;

const getHeaderValue = (headers, headerName) => {
  if (!headers) {
    return '';
  }

  if (typeof headers === 'string') {
    return headers;
  }

  if (typeof headers.get === 'function') {
    return headers.get(headerName) || headers.get(headerName.toLowerCase()) || '';
  }

  const normalizedHeaderName = headerName.toLowerCase();
  const matchedHeaderKey = Object.keys(headers).find(
    (key) => key.toLowerCase() === normalizedHeaderName,
  );

  return matchedHeaderKey ? headers[matchedHeaderKey] : '';
};

const isJsonContentType = (headers) =>
  JSON_CONTENT_TYPE_PATTERN.test(getHeaderValue(headers, 'content-type'));

const readBlobText = async (blobData) => {
  if (blobData === null || blobData === undefined) {
    return '';
  }

  if (typeof blobData === 'string') {
    return blobData;
  }

  if (typeof blobData.text === 'function') {
    return await blobData.text();
  }

  if (blobData instanceof ArrayBuffer) {
    return new TextDecoder().decode(new Uint8Array(blobData));
  }

  if (ArrayBuffer.isView(blobData)) {
    return new TextDecoder().decode(blobData);
  }

  return String(blobData);
};

const downloadSmartExportBlob = ({
  response,
  fallbackFileName,
  documentApi,
  urlApi,
  blobCtor,
}) => {
  const resolvedDocumentApi = documentApi ?? globalThis.document;
  const resolvedUrlApi = urlApi ?? globalThis.URL;
  const ResolvedBlobCtor = blobCtor ?? globalThis.Blob;
  const fileName = extractDownloadFilename(response?.headers, fallbackFileName);
  const excelBlob = new ResolvedBlobCtor([response.data], {
    type: EXCEL_BLOB_MIME_TYPE,
  });

  downloadBlobFile(excelBlob, fileName, {
    documentApi: resolvedDocumentApi,
    urlApi: resolvedUrlApi,
  });
};

const parseSmartExportJsonPayload = async (response) => {
  const responseText = await readBlobText(response?.data);

  try {
    const payload = JSON.parse(responseText);
    if (payload?.success === false) {
      throw new Error(payload.message || 'Export failed');
    }
    return payload;
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error('Invalid export response');
    }

    if (error instanceof Error) {
      throw error;
    }
    throw new Error('Invalid export response');
  }
};

export const createSmartExportStatusNotifier = ({
  t = (value) => value,
  showInfo,
  showSuccess,
} = {}) => {
  let lastStatus = '';

  return (job) => {
    const status = String(job?.status || '').trim().toLowerCase();
    if (!status || status === lastStatus) {
      return;
    }

    lastStatus = status;

    if (status === 'queued') {
      showInfo?.(t('导出任务已创建，正在后台生成文件，请稍候'));
      return;
    }

    if (status === 'running') {
      showInfo?.(t('导出任务处理中，请稍候'));
      return;
    }

    if (status === 'succeeded') {
      showSuccess?.(t('导出文件已准备完成，开始下载'));
    }
  };
};

export const createExportCenterStartNotifier = ({
  t = (value) => value,
  showInfo,
} = {}) => () => {
  showInfo?.(t('导出任务已创建，请到导出中心查看进度'));
};

export const runSmartExport = async ({
  url,
  payload,
  fallbackFileName = 'export.xlsx',
  apiClient,
  pollJob = pollAsyncExportJob,
  onAsyncStart,
  onAsyncProgress,
  documentApi,
  urlApi,
  blobCtor,
  pollIntervalMs,
  maxAttempts,
  wait,
  timeoutMessage,
  autoDownloadAsync = true,
}) => {
  const client = apiClient ?? (await import('./api.js')).API;
  const response = await client.post(url, payload, {
    responseType: 'blob',
  });

  if (!isJsonContentType(response?.headers)) {
    downloadSmartExportBlob({
      response,
      fallbackFileName,
      documentApi,
      urlApi,
      blobCtor,
    });

    return {
      mode: 'sync',
      response,
    };
  }

  const jsonPayload = await parseSmartExportJsonPayload(response);
  const asyncPayload = jsonPayload?.data || {};
  const job = asyncPayload?.job;

  if (asyncPayload?.mode !== 'async' || !job) {
    throw new Error(jsonPayload?.message || 'Invalid async export response');
  }

  onAsyncStart?.({
    decision: asyncPayload?.decision,
    job,
  });

  if (!autoDownloadAsync) {
    return {
      mode: 'async',
      response,
      decision: asyncPayload?.decision,
      job,
    };
  }

  const asyncResult = await pollJob({
    job,
    apiClient: client,
    fallbackFileName: job?.file_name || fallbackFileName,
    pollIntervalMs,
    maxAttempts,
    wait,
    timeoutMessage,
    onProgress: onAsyncProgress,
    documentApi,
    urlApi,
    blobCtor,
  });

  return {
    mode: 'async',
    response,
    ...asyncResult,
  };
};
