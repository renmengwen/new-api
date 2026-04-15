export const MAX_EXCEL_EXPORT_ROWS = 2000;

const FILENAME_STAR_PATTERN = /filename\*\s*=\s*(?:UTF-8''|utf-8'')?([^;]+)/;
const FILENAME_PATTERN = /filename\s*=\s*("?)([^";]+)\1/;

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

export const extractDownloadFilename = (
  contentDisposition,
  fallbackFileName = 'export.xlsx',
) => {
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
  apiClient,
  url,
  data,
  fallbackFileName = 'export.xlsx',
  documentApi = document,
  urlApi = URL,
}) => {
  const response = await apiClient.post(url, data, {
    responseType: 'blob',
  });
  const contentDisposition =
    response?.headers?.['content-disposition'] ||
    response?.headers?.['Content-Disposition'] ||
    '';
  const fileName = extractDownloadFilename(contentDisposition, fallbackFileName);

  downloadBlobFile(response.data, fileName, {
    documentApi,
    urlApi,
  });

  return response;
};

export const resolveExcelFilename = extractDownloadFilename;
export const postExcelBlob = downloadExcelBlob;
