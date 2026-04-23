import test from 'node:test';
import assert from 'node:assert/strict';

test('runSmartExport downloads excel responses directly when /export-auto returns a file blob', async () => {
  const { runSmartExport } = await import('./smartExport.js');

  const postCalls = [];
  const clickCalls = [];
  const revokeCalls = [];
  const mountedLinks = [];
  const apiClient = {
    async post(url, data, config) {
      postCalls.push({ url, data, config });
      return {
        data: new Uint8Array([9, 8, 7, 6]),
        headers: {
          'content-disposition': `attachment; filename*=UTF-8''quota-ledger.xlsx`,
          'content-type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        },
      };
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
      return 'blob:quota-ledger-export';
    },
    revokeObjectURL(url) {
      revokeCalls.push(url);
    },
  };

  const result = await runSmartExport({
    url: '/api/admin/quota/ledger/export-auto',
    payload: {
      user_id: 12,
      limit: 2000,
    },
    fallbackFileName: 'quota-ledger.xlsx',
    apiClient,
    documentApi,
    urlApi,
  });

  assert.equal(result.mode, 'sync');
  assert.deepEqual(postCalls, [
    {
      url: '/api/admin/quota/ledger/export-auto',
      data: {
        user_id: 12,
        limit: 2000,
      },
      config: {
        responseType: 'blob',
      },
    },
  ]);
  assert.deepEqual(clickCalls, [
    {
      href: 'blob:quota-ledger-export',
      download: 'quota-ledger.xlsx',
    },
  ]);
  assert.deepEqual(revokeCalls, ['blob:quota-ledger-export']);
  assert.equal(mountedLinks.length, 0);
});

test('runSmartExport parses async export job payloads and delegates polling to asyncExport', async () => {
  const { runSmartExport } = await import('./smartExport.js');

  const asyncStarts = [];
  const progressStatuses = [];
  const apiClient = {
    async post(url, data, config) {
      assert.equal(url, '/api/log/export-auto');
      assert.deepEqual(data, { type: 2, limit: 2000 });
      assert.deepEqual(config, { responseType: 'blob' });
      return {
        data: new Blob(
          [
            JSON.stringify({
              success: true,
              data: {
                mode: 'async',
                decision: 'forced_async',
                job: {
                  id: 51,
                  status: 'queued',
                  status_url: '/api/export-jobs/51',
                  download_url: '/api/export-jobs/51/file',
                  file_name: 'usage-logs.xlsx',
                },
              },
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
  const pollJobCalls = [];
  const pollJob = async (options) => {
    pollJobCalls.push(options);
    options.onProgress?.({ status: 'running' });
    return {
      job: {
        ...options.job,
        status: 'succeeded',
      },
      downloadResponse: {
        data: new Uint8Array([1, 2, 3]),
      },
    };
  };

  const result = await runSmartExport({
    url: '/api/log/export-auto',
    payload: { type: 2, limit: 2000 },
    fallbackFileName: 'usage-logs.xlsx',
    apiClient,
    pollJob,
    onAsyncStart: ({ decision, job }) => {
      asyncStarts.push({ decision, job });
    },
    onAsyncProgress: (job) => {
      progressStatuses.push(job.status);
    },
  });

  assert.equal(result.mode, 'async');
  assert.deepEqual(asyncStarts, [
    {
      decision: 'forced_async',
      job: {
        id: 51,
        status: 'queued',
        status_url: '/api/export-jobs/51',
        download_url: '/api/export-jobs/51/file',
        file_name: 'usage-logs.xlsx',
      },
    },
  ]);
  assert.equal(pollJobCalls.length, 1);
  assert.equal(pollJobCalls[0].job.status_url, '/api/export-jobs/51');
  assert.equal(pollJobCalls[0].fallbackFileName, 'usage-logs.xlsx');
  assert.deepEqual(progressStatuses, ['running']);
});

test('runSmartExport surfaces initial /export-auto JSON errors before polling', async () => {
  const { runSmartExport } = await import('./smartExport.js');

  const apiClient = {
    async post() {
      return {
        data: new Blob(
          [
            JSON.stringify({
              success: false,
              message: 'export request failed',
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
      runSmartExport({
        url: '/api/log/export-auto',
        payload: { type: 2, limit: 4200 },
        apiClient,
      }),
    (error) => {
      assert.equal(error.message, 'export request failed');
      return true;
    },
  );
});

test('runSmartExport rejects malformed initial /export-auto JSON payloads', async () => {
  const { runSmartExport } = await import('./smartExport.js');

  const apiClient = {
    async post() {
      return {
        data: new Blob(['{"success": true'], {
          type: 'application/json',
        }),
        headers: {
          'content-type': 'application/json; charset=utf-8',
        },
      };
    },
  };

  await assert.rejects(
    () =>
      runSmartExport({
        url: '/api/log/export-auto',
        payload: { type: 2, limit: 4200 },
        apiClient,
      }),
    (error) => {
      assert.equal(error.message, 'Invalid export response');
      return true;
    },
  );
});

test('createSmartExportStatusNotifier emits queue running and ready messages once per status', async () => {
  const { createSmartExportStatusNotifier } = await import('./smartExport.js');

  const infoMessages = [];
  const successMessages = [];
  const notify = createSmartExportStatusNotifier({
    t: (value) => value,
    showInfo: (message) => {
      infoMessages.push(message);
    },
    showSuccess: (message) => {
      successMessages.push(message);
    },
  });

  notify({ status: 'queued' });
  notify({ status: 'queued' });
  notify({ status: 'running' });
  notify({ status: 'running' });
  notify({ status: 'succeeded' });

  assert.deepEqual(infoMessages, [
    '导出任务已创建，正在后台生成文件，请稍候',
    '导出任务处理中，请稍候',
  ]);
  assert.deepEqual(successMessages, ['导出文件已准备完成，开始下载']);
});
