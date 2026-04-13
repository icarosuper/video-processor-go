# Design decisions — VidroProcessor

Non-obvious choices shaping worker. Read before proposing arch changes — each entry explains trade-off so you can tell if constraint still applies.

## Redis `BRPOPLPUSH` instead of Streams / plain `BRPOP`

`queue/client.go` — `ConsumeMessage` uses `BRPOPLPUSH` to atomically move job from main queue into `:processing` sibling list.

- **Why**: need at-least-once delivery + crash recovery without message broker. `BRPOP` alone loses jobs if worker dies mid-processing; Redis Streams would work but add consumer-group state API must also understand.
- **Implication**: worker must `LREM` job from `:processing` when done (`AcknowledgeMessage`), whether succeeded, failed, or moved to DLQ. Background recovery loop re-queues anything stuck in `:processing` beyond 10 min.
- **Trade-off**: job stuck exactly at orphan window but still running can be picked up twice. Acceptable — processing idempotent per `videoID` (uploads overwrite).

## Retry in place, then dead-letter

`queue/job.go` — `MaxJobRetries = 3`; after that, `MoveToDLQ` on `<queue>:dead`.

- **Why**: transient failures (MinIO blip, FFmpeg deadlock) should self-heal, but looping forever on truly broken video wastes workers + hides bugs.
- **No automatic DLQ drain**: jobs in `:dead` need human attention. Auto-retry would mask underlying problem.
- **Retries counted in two places**: explicit `SetJobFailed` and implicit `recoverStuckJobs` (orphan recovery). Both increment `RetryCount` so repeatedly-crashing worker eventually gives up.

## Critical vs non-critical pipeline steps

`internal/processor/processor.go`.

- **Why**: user's uploaded video "done" as soon as playable MP4 exists. Thumbnails, preview, audio, HLS are nice-to-haves — failing whole job on thumbnail glitch degrades product worse than silently shipping without one.
- **Consequence**: success webhook may contain empty `thumbnailPaths`, `hlsPath`, `previewPath`, or `audioPath`. API must not treat missing optional artifacts as failure.
- **Only `validate` and `transcode` are critical.** Changing classification is product-level decision — discuss with VidroApi before touching.

## Whole-job timeout of 5 minutes + per-step timeouts

`main.go` (5-minute `processCtx`) plus per-step timeouts in `internal/processor/processor.go`.

- **Why**: defence in depth. Per-step timeout prevents one FFmpeg hang from monopolising worker. Whole-job timeout catches everything else (download stalls, upload stalls, recoverable bugs that never raise error).
- **Tuning**: step timeouts assume short/medium content. For longer videos, raise transcode + streaming budgets first; whole-job budget must always exceed sum of critical-path steps (validate + analyze + transcode) plus download + upload slack.

## HLS single-command with sequential fallback

`internal/processor/processor-steps/streaming.go`.

- **Why**: single FFmpeg invocation using `-filter_complex split` + `-var_stream_map` decodes source once, emits all variants in parallel. Significantly faster than re-decoding per variant (what sequential path does).
- **But**: single command is fragile — can fail on unusual inputs (missing audio, odd codecs, filter graph edge cases). Rather than pick one path at build time, run single-command first, fall back to sequential if errors, guarded by `HLS_SINGLE_COMMAND_FALLBACK`.
- **Variant selection**: filter `hlsVariants` to those `Height <= sourceHeight` (probed via `ffprobe`) so never upscale. If probing fails, emit at least 240p variant rather than nothing.

## NVENC resolved at startup, CPU fallback inside each step

`internal/processor/processor-steps/video_encoder.go`, `transcode.go`, `streaming.go`.

- **Why**: GPU availability is deploy-time fact. `ResolveVideoEncoder` probes `ffmpeg -encoders` once during `main()` so every job sees consistent choice. `VIDEO_ENCODER=auto` is default so same binary works on CPU-only hosts and GPU nodes.
- **Per-step fallback**: even after selecting NVENC, individual FFmpeg calls can fail on specific inputs (unusual colour spaces, CUDA driver hiccups). Both `transcode.go` and `streaming.go` catch those errors and retry on `libx264` to keep throughput up.
- **NVENC preset** normalized to `p1`–`p7`; invalid values silently become `p5`. Default `p5` balances quality + speed on 1080p content.

## Raw videos soft-archived, then deleted by lifecycle rule

`minio/client.go` — `ArchiveRawVideo` + `configureRawArchivedLifecycle`.

- **Why**: keeping raw after processing is insurance — can reprocess if pipeline buggy, transcode params change, or user reports bad video. But keeping forever burns storage.
- **How**: on success copy `raw/<id>` → `raw-archived/<id>` and delete original. MinIO lifecycle rule (`expire-raw-archived`, 30 days) deletes archived copy automatically.
- **Why soft-move instead of single prefix**: lifecycle rule directly on `raw/` would sweep up unprocessed jobs (API uploaded, worker hasn't picked up). Explicit move defers TTL clock until processing actually done.
- **Failure is non-fatal**: if archiving fails, log warning + keep raw in place. Better to leak storage than lose source.

## Circuit breakers with different thresholds for MinIO vs Redis

`internal/circuitbreaker/circuitbreaker.go`.

- **Why separate breakers**: MinIO outage shouldn't prevent Redis ops (and vice versa) — queue bookkeeping must keep running even if object storage down.
- **Different thresholds**: Redis failures cheaper to retry + more likely to self-heal, so trip faster (3 vs 5) and reset sooner (30s vs 60s). MinIO ops expensive + sometimes slow — tolerate more failures before opening to avoid thrashing.
- **`MaxRequests: 1` in half-open**: one probe request only while half-open; don't flood recovering service.

## Worker count defaults to `runtime.NumCPU()`

`main.go`.

- **Why**: FFmpeg is CPU-bound, so one worker per core is right starting point. Default avoids wasted threads waiting on I/O while still saturating encoder.
- **Override via `WORKER_COUNT`**: set explicitly for NVENC deployments (GPU is bottleneck, fewer workers better) or containers with CPU quotas (where `NumCPU` reports host count).

## Webhook contract uses camelCase to match the .NET API

`internal/webhook/webhook.go`.

- **Why**: VidroApi (producer) is .NET service whose JSON serialiser produces camelCase. Matching contract here avoids custom converter on API side.
- **HMAC signature is optional**: `WEBHOOK_SECRET` empty = no signing. Off in local dev, must be set in prod.
- **Delivery is background-only**: webhook failures logged but never fail job. API can always recover state from `ProcessingFinishedQueue` or by polling `job:<id>`.

## Single bucket, path-based namespacing

`minio/client.go`.

- **Why**: one bucket easier to provision, replicate, + secure than many. Lifecycle rules + IAM policies still scopeable by prefix (`raw-archived/`).
- **Path layout is shared contract** with VidroApi. Changing it is coordinated release (tag both repos together).

## Graceful shutdown with a hard 30-second ceiling

`main.go`.

- **Why**: on `SIGTERM` want workers to finish current job if possible to avoid leaking in-flight work to DLQ. But stuck job must not block Kubernetes pod from terminating — force-exit after 30s.
- **Tuning**: ceiling should match or undercut orchestrator's termination grace period. If you raise `processCtx`'s whole-job budget beyond 5 min, revisit whether 30s is still enough to drain clean-shutdown case.