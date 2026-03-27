# Roadmap - VidroProcessor

## Project Goal

Async worker that an API calls to process user-submitted videos (YouTube model): receives a `videoID`, processes it through a 7-step FFmpeg pipeline, and delivers the artifacts to MinIO.

## Current Status: ~95% production-ready

The FFmpeg pipeline works. The basic infrastructure exists. But the pieces that make the system **reliable and integrable** with a real API are complete.

---

## ✅ Completed

- 7-step FFmpeg pipeline (validation, transcoding, thumbnails, audio, preview, HLS)
- Upload of all artifacts to MinIO (thumbnails, audio, preview, HLS segments)
- Concurrent workers with working graceful shutdown
- Prometheus metrics, health check, structured logging
- Unit tests (pipeline, metrics, circuitbreaker, webhook, telemetry) and integration tests (testcontainers)
- **B1**: Workers no longer blocked on shutdown — `ConsumeMessage(ctx)`
- **B2**: Artifacts from steps 4–7 now reach MinIO
- **B3**: `docker-compose.yml` with all required env vars
- **B4**: MinIO password no longer printed in log
- **B5**: Deploy without `.env` file works
- **C1**: Job state implemented (`queue/job.go`) — `pending → processing → done/failed` in Redis with 24h TTL; generated artifacts recorded in `done`
- **C2**: Safe consumption with `BRPOPLPUSH` — job atomically moved to `{queue}:processing` on consume; `AcknowledgeMessage` removes it on completion; `PublishJob` creates `pending` state on producer
- **C4**: Orphan job recovery — `StartRecovery` checks every minute for jobs in `processing` with old `updated_at` and re-queues them
- **P2**: Automatic retry — `SetJobFailed` increments `RetryCount`; up to 3 attempts the job is re-queued; beyond that it goes to DLQ
- **P3**: Dead letter queue — `{queue}:dead` receives jobs that exhausted retries; state remains `failed` with auditable error message
- **P4**: Video metadata persisted — `AnalyzeContent` returns `*VideoMetadata`; stored in the `done` job state; available to the API via `GetJobState`
- **P5**: Stricter input validation — configurable size limit via `MAX_FILE_SIZE_MB` (default 5GB); checked before download with `StatObject`
- **Operational metrics**: `active_workers` Inc/Dec per job; `queue_size` updated every 30s; `video_size_bytes` recorded after download
- **C3**: Webhook/callback — on completion (success or permanent failure), the worker POSTs to the `callbackURL` registered on the job with the full payload; optional HMAC-SHA256 via `WEBHOOK_SECRET`
- **P1**: Multiple HLS resolutions — `SegmentForStreaming` generates 240p/360p/480p/720p/1080p (only ≤ original resolution) + `master.m3u8`; `UploadDirectory` is now recursive; HLS generated directly from the original input (no double transcoding)

---

## 🟡 Improvements — Operational quality

### Observability
- ✅ **OpenTelemetry**: per-job tracing with root span `process_job` and child spans per step (`step/validate`, `step/transcode`, etc.); OTLP exporter via `OTEL_ENDPOINT`; automatic no-op if not configured

### Resilience
- ✅ **Circuit breaker**: MinIO opens after 5 consecutive failures (timeout 60s); Redis opens after 3 (timeout 30s); state changes logged
- ✅ **Per-step timeout**: each pipeline step has its own `context.WithTimeout` (validate/analyze 30s, transcode 3min, thumbnails/audio/preview 1-2min, streaming 4min)

### Configuration
- ✅ **MinIO SSL**: configurable via `MINIO_USE_SSL` (default `false`)
- ✅ **HTTP port**: configurable via `HTTP_PORT` (default `8080`)
- ✅ **`go-redis/v8` → `v9`**: migrated to `github.com/redis/go-redis/v9`

---

## 🟡 Recent improvements

- ✅ **Grafana Dashboard**: `grafana/provisioning/dashboards/video-processor.json` with 9 panels (workers, queue, throughput, p50/p90/p99 per step, video sizes); Prometheus and Grafana in `docker-compose`

---

## 🔵 Long Term — Scalability and advanced features

- **Auto-scaling**: increase workers based on queue size
- **Horizontal scaling**: multiple worker instances on different machines
- **Queue prioritization**: short videos first, long videos in a separate queue

---

**Last Updated**: 2026-03-26
