# Async Image Task Design

## Goal

Support `gpt-image-2` asynchronous image generation through the existing `/v1/images/generations` API, show the task in the existing task log page, and expose a downstream task-id query API.

## Behavior

- When an image generation request includes `enable_sync_mode: false`, the gateway treats the upstream response as an asynchronous task submission.
- The gateway returns a local `task_xxx` id to the client and stores the upstream prediction id in task private data.
- The existing task polling loop queries the upstream prediction result endpoint.
- Completed image results are stored on the task and shown in the existing task log page.
- Downstream clients can query the task with `GET /v1/images/generations/{task_id}`.

## Architecture

- Add `enable_sync_mode` to `dto.ImageRequest` as `*bool` to preserve explicit `false`.
- Add an image task platform and task adaptor for polling result status.
- Extend `relay.ImageHelper` to create `model.Task` entries for asynchronous image submissions.
- Add a controller route for `GET /v1/images/generations/{task_id}` that returns the local task state.
- Extend the task log table to identify image-generation tasks and preview image result URLs.

## Compatibility

- Synchronous image generation remains unchanged unless `enable_sync_mode` is explicitly `false`.
- The database change reuses the existing `tasks` table; no new migration is required.
- JSON operations use project wrappers where production Go code already follows that convention.
