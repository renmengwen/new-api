import test from 'node:test';
import assert from 'node:assert/strict';

test('pollAsyncExportJob polls queued exports until the artifact is ready and then downloads it', async () => {
  const { pollAsyncExportJob } = await import('./asyncExport.js');

  const getCalls = [];
  const waitCalls = [];
  const progressStatuses = [];
  const clickCalls = [];
  const revokeCalls = [];
  const mountedLinks = [];
  const statusResponses = [
    {
      data: {
        success: true,
        data: {
          id: 7,
          status: 'running',
          status_url: '/api/export-jobs/7',
          download_url: '/api/export-jobs/7/file',
          file_name: 'usage-logs-ready.xlsx',
        },
      },
    },
    {
      data: {
        success: true,
        data: {
          id: 7,
          status: 'succeeded',
          status_url: '/api/export-jobs/7',
          download_url: '/api/export-jobs/7/file',
          file_name: 'usage-logs-ready.xlsx',
        },
      },
    },
  ];
  const downloadResponse = {
    data: new Uint8Array([1, 2, 3, 4]),
    headers: {
      'content-disposition': `attachment; filename*=UTF-8''usage-logs-ready.xlsx`,
    },
  };
  const apiClient = {
    async get(url, config) {
      getCalls.push({ url, config });
      if (url === '/api/export-jobs/7/file') {
        return downloadResponse;
      }
      return statusResponses.shift();
    },
  };
  const documentApi = {
    body: {
      appendChild(node) {
        mountedLinks.push(node);
      },
      removeChild(node) {
        const index = mountedLinks.indexOf(node);
        if (index >= 0) {
          mountedLinks.splice(index, 1);
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
      return 'blob:usage-export';
    },
    revokeObjectURL(url) {
      revokeCalls.push(url);
    },
  };

  const result = await pollAsyncExportJob({
    job: {
      id: 7,
      status: 'queued',
      status_url: '/api/export-jobs/7',
      download_url: '/api/export-jobs/7/file',
      file_name: '',
    },
    apiClient,
    fallbackFileName: 'usage-logs.xlsx',
    pollIntervalMs: 25,
    wait: async (ms) => {
      waitCalls.push(ms);
    },
    onProgress: (job) => {
      progressStatuses.push(job.status);
    },
    documentApi,
    urlApi,
  });

  assert.equal(result.job.status, 'succeeded');
  assert.equal(result.downloadResponse, downloadResponse);
  assert.deepEqual(progressStatuses, ['queued', 'running', 'succeeded']);
  assert.deepEqual(waitCalls, [25, 25]);
  assert.deepEqual(getCalls, [
    { url: '/api/export-jobs/7', config: undefined },
    { url: '/api/export-jobs/7', config: undefined },
    {
      url: '/api/export-jobs/7/file',
      config: {
        responseType: 'blob',
      },
    },
  ]);
  assert.deepEqual(clickCalls, [
    {
      href: 'blob:usage-export',
      download: 'usage-logs-ready.xlsx',
    },
  ]);
  assert.deepEqual(revokeCalls, ['blob:usage-export']);
  assert.equal(mountedLinks.length, 0);
});

test('pollAsyncExportJob surfaces failed async export jobs without downloading a file', async () => {
  const { pollAsyncExportJob } = await import('./asyncExport.js');

  let downloadAttempted = false;
  const apiClient = {
    async get(url) {
      assert.equal(url, '/api/export-jobs/18');
      return {
        data: {
          success: true,
          data: {
            id: 18,
            status: 'failed',
            status_url: '/api/export-jobs/18',
            download_url: '/api/export-jobs/18/file',
            error_message: '导出任务失败',
          },
        },
      };
    },
  };

  await assert.rejects(
    () =>
      pollAsyncExportJob({
        job: {
          id: 18,
          status: 'running',
          status_url: '/api/export-jobs/18',
          download_url: '/api/export-jobs/18/file',
        },
        apiClient,
        wait: async () => {},
        pollIntervalMs: 10,
        documentApi: {
          body: {
            appendChild() {},
            removeChild() {},
          },
          createElement() {
            downloadAttempted = true;
            return {
              click() {},
            };
          },
        },
      }),
    (error) => {
      assert.equal(error.message, '导出任务失败');
      return true;
    },
  );

  assert.equal(downloadAttempted, false);
});

test('pollAsyncExportJob does not report succeeded before the final file download succeeds', async () => {
  const { pollAsyncExportJob } = await import('./asyncExport.js');

  const progressStatuses = [];
  const apiClient = {
    async get(url) {
      assert.equal(url, '/api/export-jobs/23/file');
      return {
        data: new Blob(
          [
            JSON.stringify({
              success: false,
              message: 'download failed',
            }),
          ],
          { type: 'application/json' },
        ),
        headers: {
          'content-type': 'application/json; charset=utf-8',
        },
      };
    },
  };

  await assert.rejects(
    () =>
      pollAsyncExportJob({
        job: {
          id: 23,
          status: 'succeeded',
          status_url: '/api/export-jobs/23',
          download_url: '/api/export-jobs/23/file',
          file_name: 'usage-logs.xlsx',
        },
        apiClient,
        wait: async () => {},
        pollIntervalMs: 10,
        onProgress: (job) => {
          progressStatuses.push(job.status);
        },
      }),
    (error) => {
      assert.equal(error.message, 'download failed');
      return true;
    },
  );

  assert.deepEqual(progressStatuses, []);
});
