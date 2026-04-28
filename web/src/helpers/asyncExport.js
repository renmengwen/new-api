import {
  downloadBlobFile,
  EXCEL_BLOB_MIME_TYPE,
  extractDownloadFilename,
} from './exportExcel.js';

export const DEFAULT_ASYNC_EXPORT_POLL_INTERVAL_MS = 1000;
export const DEFAULT_ASYNC_EXPORT_MAX_ATTEMPTS = 120;

const JSON_CONTENT_TYPE_PATTERN = /(^|\/|\+)json\b/i;
const TERMINAL_ASYNC_EXPORT_STATUSES = new Set(['succeeded', 'failed', 'expired']);

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

const normalizeAsyncExportJob = (payload) => {
  if (payload?.success === false) {
    throw new Error(payload.message || 'Export failed');
  }

  if (payload?.success === true && payload.data) {
    return payload.data;
  }

  if (payload && typeof payload === 'object' && payload.status) {
    return payload;
  }

  throw new Error('Invalid async export job response');
};

const ensureAsyncExportJobUrls = (job) => {
  if (!job?.status_url || !job?.download_url) {
    throw new Error('Invalid async export job response');
  }
};

const defaultWait = (ms) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export const downloadAsyncExportFile = async ({
  job,
  apiClient,
  fallbackFileName,
  documentApi,
  urlApi,
  blobCtor,
}) => {
  const resolvedDocumentApi = documentApi ?? globalThis.document;
  const resolvedUrlApi = urlApi ?? globalThis.URL;
  const ResolvedBlobCtor = blobCtor ?? globalThis.Blob;
  const response = await apiClient.get(job.download_url, {
    responseType: 'blob',
  });

  if (isJsonContentType(response?.headers)) {
    const responseText = await readBlobText(response?.data);
    let message = 'Export failed';

    try {
      const payload = JSON.parse(responseText);
      message = payload?.message || payload?.data?.error_message || message;
    } catch {}

    throw new Error(message);
  }

  const fileName = extractDownloadFilename(
    response?.headers,
    job?.file_name || fallbackFileName,
  );
  const excelBlob = new ResolvedBlobCtor([response.data], {
    type: EXCEL_BLOB_MIME_TYPE,
  });

  downloadBlobFile(excelBlob, fileName, {
    documentApi: resolvedDocumentApi,
    urlApi: resolvedUrlApi,
  });

  return response;
};

export const pollAsyncExportJob = async ({
  job,
  apiClient,
  fallbackFileName = 'export.xlsx',
  pollIntervalMs = DEFAULT_ASYNC_EXPORT_POLL_INTERVAL_MS,
  maxAttempts = DEFAULT_ASYNC_EXPORT_MAX_ATTEMPTS,
  wait = defaultWait,
  onProgress,
  timeoutMessage = 'Export timed out. Please try again later.',
  documentApi,
  urlApi,
  blobCtor,
}) => {
  const client = apiClient ?? (await import('./api.js')).API;
  let currentJob = normalizeAsyncExportJob(job);
  let lastReportedStatus = '';

  ensureAsyncExportJobUrls(currentJob);

  const reportProgress = () => {
    if (typeof onProgress !== 'function') {
      return;
    }

    if (currentJob?.status === lastReportedStatus) {
      return;
    }

    lastReportedStatus = currentJob?.status;
    onProgress(currentJob);
  };

  for (let attempt = 0; attempt <= maxAttempts; attempt += 1) {
    if (currentJob?.status === 'succeeded') {
      const downloadResponse = await downloadAsyncExportFile({
        job: currentJob,
        apiClient: client,
        fallbackFileName,
        documentApi,
        urlApi,
        blobCtor,
      });

      reportProgress();

      return {
        job: currentJob,
        downloadResponse,
      };
    }

    reportProgress();

    if (TERMINAL_ASYNC_EXPORT_STATUSES.has(currentJob?.status)) {
      throw new Error(
        currentJob?.error_message ||
          (currentJob?.status === 'expired'
            ? 'Export file has expired.'
            : 'Export failed'),
      );
    }

    if (attempt === maxAttempts) {
      throw new Error(timeoutMessage);
    }

    await wait(pollIntervalMs);

    const response = await client.get(currentJob.status_url);
    currentJob = normalizeAsyncExportJob(response?.data);
    ensureAsyncExportJobUrls(currentJob);
  }

  throw new Error(timeoutMessage);
};
