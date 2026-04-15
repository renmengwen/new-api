import test from 'node:test';
import assert from 'node:assert/strict';

test('extractDownloadFilename prefers filename* and falls back to filename', async () => {
  const { extractDownloadFilename } = await import('./exportExcel.js');

  assert.equal(
    extractDownloadFilename(
      `attachment; filename="audit.xlsx"; filename*=UTF-8''%E5%AE%A1%E8%AE%A1%E6%97%A5%E5%BF%97.xlsx`,
      'fallback.xlsx',
    ),
    '审计日志.xlsx',
  );
  assert.equal(
    extractDownloadFilename('attachment; filename="quota-ledger.xlsx"', 'fallback.xlsx'),
    'quota-ledger.xlsx',
  );
  assert.equal(extractDownloadFilename('', 'fallback.xlsx'), 'fallback.xlsx');
});

test('downloadExcelBlob posts with blob response type and downloads the returned file', async () => {
  const { downloadExcelBlob } = await import('./exportExcel.js');

  const response = {
    data: { blob: true },
    headers: {
      'content-disposition': `attachment; filename*=UTF-8''audit-export.xlsx`,
    },
  };
  const postCalls = [];
  const clickCalls = [];
  const revokeCalls = [];
  const createdLinks = [];
  const apiClient = {
    async post(url, data, config) {
      postCalls.push({ url, data, config });
      return response;
    },
  };
  const documentApi = {
    body: {
      appendChild(node) {
        createdLinks.push(node);
      },
      removeChild(node) {
        const index = createdLinks.indexOf(node);
        if (index >= 0) {
          createdLinks.splice(index, 1);
        }
      },
    },
    createElement(tagName) {
      assert.equal(tagName, 'a');
      return {
        href: '',
        download: '',
        click() {
          clickCalls.push({ href: this.href, download: this.download });
        },
      };
    },
  };
  const urlApi = {
    createObjectURL(blob) {
      assert.equal(blob, response.data);
      return 'blob:download-url';
    },
    revokeObjectURL(url) {
      revokeCalls.push(url);
    },
  };

  const result = await downloadExcelBlob({
    apiClient,
    url: '/api/admin/audit-logs/export',
    data: {
      action_module: 'quota',
      limit: 2000,
    },
    fallbackFileName: 'fallback.xlsx',
    documentApi,
    urlApi,
  });

  assert.equal(result, response);
  assert.deepEqual(postCalls, [
    {
      url: '/api/admin/audit-logs/export',
      data: {
        action_module: 'quota',
        limit: 2000,
      },
      config: {
        responseType: 'blob',
      },
    },
  ]);
  assert.deepEqual(clickCalls, [
    {
      href: 'blob:download-url',
      download: 'audit-export.xlsx',
    },
  ]);
  assert.deepEqual(revokeCalls, ['blob:download-url']);
  assert.equal(createdLinks.length, 0);
});
