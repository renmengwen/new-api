import test from 'node:test';
import assert from 'node:assert/strict';

test('extractDownloadFilename reads content-disposition from header objects', async () => {
  const { extractDownloadFilename } = await import('./exportExcel.js');
  const response = {
    headers: {
      'content-disposition': `attachment; filename="audit.xlsx"; filename*=UTF-8''%E5%AE%A1%E8%AE%A1%E6%97%A5%E5%BF%97.xlsx`,
    },
  };
  const headersLikeObject = {
    get(name) {
      return name.toLowerCase() === 'content-disposition'
        ? 'attachment; filename="quota-ledger.xlsx"'
        : null;
    },
  };

  assert.equal(
    extractDownloadFilename(response.headers, 'fallback.xlsx'),
    '审计日志.xlsx',
  );
  assert.equal(
    extractDownloadFilename(headersLikeObject, 'fallback.xlsx'),
    'quota-ledger.xlsx',
  );
  assert.equal(extractDownloadFilename({}, 'fallback.xlsx'), 'fallback.xlsx');
});

test('downloadExcelBlob posts payload as blob request and downloads an excel blob', async () => {
  const { downloadExcelBlob } = await import('./exportExcel.js');

  const response = {
    data: new Uint8Array([1, 2, 3, 4]),
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
      assert.equal(blob instanceof Blob, true);
      assert.equal(blob.type, 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet');
      return 'blob:download-url';
    },
    revokeObjectURL(url) {
      revokeCalls.push(url);
    },
  };

  const result = await downloadExcelBlob({
    url: '/api/admin/audit-logs/export',
    payload: {
      action_module: 'quota',
      limit: 2000,
    },
    fallbackFileName: 'fallback.xlsx',
    apiClient,
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
