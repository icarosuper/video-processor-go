# Architecture — VidroProcessor

High-level shape of the worker, the queue protocol it shares with VidroApi, and the pipeline orchestration model. For per-file mapping, see `features-index.md`. For *why* things are the way they are, see `design-decisions.md`.

## System position

```
┌───────────┐   LPush raw/id  ┌─────────┐   BRPOPLPUSH   ┌──────────────────┐
│  VidroApi │ ───────────────▶│  Redis  │ ──────────────▶│ VidroProcessor N │
│  (.NET)   │                 │  queue  │                │    workers       │
└─────┬─────┘                 └─────────┘                └────────┬─────────┘
      │                            ▲                              │
      │                            │                              ▼
      │                            │                        ┌──────────┐
      │                            │       ArchiveRaw       │  MinIO   │
      │                            │ ◀──────────────────────┤  bucket  │
      │                            │                        └──────────┘
      │        POST webhook        │
      └────────────────────────────┘
```

The worker is stateless; all durable state (queue, in-flight jobs, job metadata, artifacts) lives in Redis + MinIO. You can scale horizontally by starting more instances against the same Redis.

## Worker lifecycle

1. `main.go` loads config, probes the video encoder (NVENC vs CPU), initializes OTel, MinIO, Redis, and the HTTP server (`/health`, `/metrics`).
2. Spawns `WORKER_COUNT` goroutines (defaults to `runtime.NumCPU()`). Each loops on `processNextMessage`.
3. A background goroutine (`queue.StartRecovery`) scans the `:processing` queue every minute and re-queues jobs that have been in flight for more than 10 minutes (crash recovery).
4. Another goroutine publishes `queue_size` into Prometheus every 30s.
5. On `SIGINT`/`SIGTERM`, the root context is cancelled; workers finish their current job or are killed after a 30s grace period.

## Queue protocol

Shared contract with VidroApi. Do not change queue names or job layout without tagging both repos together.

- **Main queue**: `ProcessingRequestQueue` (LPush by API, BRPopLPush by worker).
- **In-flight queue**: `<ProcessingRequestQueue>:processing`. Populated atomically by `BRPOPLPUSH`, acting as a visibility/lease list. Workers `LREM` on completion (`AcknowledgeMessage`).
- **Dead letter queue**: `<ProcessingRequestQueue>:dead`. Jobs land here after `MaxJobRetries = 3` failed attempts.
- **Success queue**: `ProcessingFinishedQueue`. Consumed by the API to react to completed jobs (in addition to the webhook).

Each job is a bare `videoID` string in the queue. The full job state lives under the Redis key `job:<videoID>` (`JobState` JSON, 24h TTL), containing status, retry count, callback URL, artifacts, and extracted metadata.

### Job state machine

```
         PublishJob                    processNextMessage
pending ───────────▶ pending(queued) ──────────────────▶ processing
                                                             │
                          ┌──────────────────────────────────┼──────────────────┐
                          ▼                                  ▼                  ▼
              done (with artifacts)              failed (retry < max)    failed (retry ≥ max)
                          │                                  │                  │
                     webhook success              RequeueJob → pending     MoveToDLQ + webhook failure
```

Retries are counted both on explicit `SetJobFailed` and implicitly by `recoverStuckJobs` (increments on orphan recovery).

## Per-job execution flow

Every job runs inside `processNextMessage` under:
- A root OTel span `process_job` (tagged with `video.id`).
- A 5-minute hard timeout context for the whole job.

Order of operations:

1. `queue.SetJobProcessing(videoID)`
2. `minio.DownloadVideo(raw, videoID, tmpInput)` — enforces `MAX_FILE_SIZE_MB`.
3. `processor.ProcessVideo(...)` — runs the 7-step pipeline (see below), returning a `ProcessingResult` with artifact paths inside a temp dir.
4. `minio.UploadVideo(tmpOutput, processed, "<id>_processed")` — primary MP4.
5. `minio.ArchiveRawVideo(videoID)` — soft delete: copy `raw/id` → `raw-archived/id`, remove original. Non-fatal.
6. Optional artifacts (thumbnails dir, audio, preview, HLS dir) uploaded if the corresponding step succeeded.
7. `queue.PublishSuccessMessage(processedID)` — notifies API via the finished queue.
8. `queue.SetJobDone` with artifacts + metadata.
9. `notifyWebhook` — fires only if `callbackURL` is set on the job state.
10. `defer`: local temp files removed; job acknowledged (`LREM` from `:processing`).

On any error, `defer` increments the retry count, requeues or moves to DLQ, and still acknowledges (to prevent double-processing). Metrics counter `videos_processed_total{status=error}` is bumped at the failure site.

## Processing pipeline

`internal/processor/processor.go` orchestrates 7 steps with individual timeouts and OTel spans (`runStep`):

```
┌───────────┐   ┌──────────┐   ┌───────────┐   ┌──────────────────────────────────────────┐
│ validate  │──▶│ analyze  │──▶│ transcode │──▶│ thumbnails │ audio │ preview │ streaming │
│ 30s       │   │ 30s      │   │ 3m        │   │  run in parallel (bounded), each with   │
│ critical  │   │ soft     │   │ critical  │   │  its own timeout; failures log & skip   │
└───────────┘   └──────────┘   └───────────┘   └──────────────────────────────────────────┘
```

- **Critical steps** (validate, transcode) return errors and abort the pipeline.
- **Soft steps** log a warning and leave the artifact path empty on failure; the job still completes successfully.
- **Analyze** is semi-critical: errors are logged, but downstream metadata will be missing in the webhook.
- Parallelism is toggled by `PARALLEL_NON_CRITICAL_STEPS` and bounded by `MAX_PARALLEL_POST_TRANSCODE_STEPS` (clamped to `[1, 4]`).
- **HLS**: single FFmpeg command with `-var_stream_map` by default; falls back to sequential per-variant encoding on failure if `HLS_SINGLE_COMMAND_FALLBACK=true`. Variants are filtered to those `<=` source height.
- **NVENC**: resolved once at startup via `ResolveVideoEncoder`. `auto` probes `ffmpeg -encoders` for `h264_nvenc`; transcode and HLS fall back to `libx264` on NVENC failure.

## Object storage layout

All under the single MinIO bucket `MINIO_BUCKET_NAME`:

```
raw/<id>                             ← API upload
raw-archived/<id>                    ← after success; auto-deleted after 30 days (lifecycle rule)
processed/<id>_processed             ← MP4
thumbnails/<id>/thumb_001..005.jpg
audio/<id>.mp3
preview/<id>_preview.mp4
hls/<id>/master.m3u8
hls/<id>/<variant>/playlist.m3u8
hls/<id>/<variant>/seg_NNN.ts
```

The `raw-archived/` lifecycle rule is installed by `configureRawArchivedLifecycle` at startup — it's idempotent.

## Resilience layers

- **Circuit breakers** (`internal/circuitbreaker`) wrap every Redis and MinIO call. MinIO trips on 5 consecutive failures (60s open); Redis on 3 consecutive failures (30s open). State changes are logged.
- **Orphan recovery** re-queues jobs stuck in `:processing` beyond 10 minutes — covers worker crashes mid-job.
- **Retry + DLQ** — automatic retry with state persistence, DLQ after exhaustion. Failed jobs in DLQ are not auto-retried; investigate and requeue manually.
- **Per-step timeouts** prevent a single bad video from holding a worker forever.
- **Whole-job timeout** of 5 minutes is the final backstop (`processCtx`).

## Observability

- Prometheus metrics exposed on `/metrics` (see `metrics/metrics.go`).
- OpenTelemetry traces exported via OTLP when `OTEL_ENDPOINT` is set; otherwise no-op.
- Structured `zerolog` logs (English only — see `conventions.md`).
- Grafana dashboards + Loki/Prometheus datasources pre-provisioned under `grafana/` for local stack.
