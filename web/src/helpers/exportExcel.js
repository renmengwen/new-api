export const MAX_EXCEL_EXPORT_ROWS = 2000;
export const EXCEL_BLOB_MIME_TYPE =
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';

const FILENAME_STAR_PATTERN = /filename\*\s*=\s*(?:UTF-8''|utf-8'')?([^;]+)/;
const FILENAME_PATTERN = /filename\s*=\s*("?)([^";]+)\1/;

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

const normalizeFilename = (value, fallbackFileName) => {
  if (!value) {
    return fallbackFileName;
  }

  const trimmedValue = value.trim().replace(/^["']|["']$/g, '');
  if (!trimmedValue) {
    return fallbackFileName;
  }

  try {
    return decodeURIComponent(trimmedValue);
  } catch {
    return trimmedValue;
  }
};

const isJsonContentType = (headers) =>
  /(^|\/|\+)json\b/i.test(getHeaderValue(headers, 'content-type'));

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

const throwJsonExportError = async (response) => {
  if (!isJsonContentType(response?.headers)) {
    return;
  }

  const responseText = await readBlobText(response?.data);
  let message = 'Export failed';

  try {
    const payload = JSON.parse(responseText);
    if (payload?.message) {
      message = payload.message;
    }
  } catch {}

  throw new Error(message);
};

export const extractDownloadFilename = (
  headers,
  fallbackFileName = 'export.xlsx',
) => {
  const contentDisposition = getHeaderValue(headers, 'content-disposition');
  if (!contentDisposition) {
    return fallbackFileName;
  }

  const filenameStarMatch = contentDisposition.match(FILENAME_STAR_PATTERN);
  if (filenameStarMatch?.[1]) {
    return normalizeFilename(filenameStarMatch[1], fallbackFileName);
  }

  const filenameMatch = contentDisposition.match(FILENAME_PATTERN);
  if (filenameMatch?.[2]) {
    return normalizeFilename(filenameMatch[2], fallbackFileName);
  }

  return fallbackFileName;
};

export const downloadBlobFile = (
  blob,
  fileName,
  {
    documentApi = document,
    urlApi = URL,
  } = {},
) => {
  const objectUrl = urlApi.createObjectURL(blob);
  const link = documentApi.createElement('a');
  link.href = objectUrl;
  link.download = fileName;

  if (documentApi.body?.appendChild) {
    documentApi.body.appendChild(link);
  }

  link.click();

  if (documentApi.body?.removeChild) {
    documentApi.body.removeChild(link);
  }

  if (urlApi.revokeObjectURL) {
    urlApi.revokeObjectURL(objectUrl);
  }
};

export const downloadExcelBlob = async ({
  url,
  payload,
  fallbackFileName = 'export.xlsx',
  apiClient,
  documentApi = document,
  urlApi = URL,
  blobCtor = Blob,
}) => {
  const client = apiClient ?? (await import('./api.js')).API;
  const response = await client.post(url, payload, {
    responseType: 'blob',
  });
  await throwJsonExportError(response);
  const fileName = extractDownloadFilename(response?.headers, fallbackFileName);
  const excelBlob = new blobCtor([response.data], {
    type: EXCEL_BLOB_MIME_TYPE,
  });

  downloadBlobFile(excelBlob, fileName, {
    documentApi,
    urlApi,
  });

  return response;
};

export const resolveExcelFilename = extractDownloadFilename;
export const postExcelBlob = ({ data, ...options }) =>
  downloadExcelBlob({
    ...options,
    payload: options.payload ?? data,
  });
