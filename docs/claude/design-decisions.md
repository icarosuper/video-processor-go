# Design decisions — VidroProcessor

Non-obvious choices that shape the worker. Read before proposing architectural changes — each entry explains the trade-off so you can tell whether the constraint still applies.

## Redis `BRPOPLPUSH` instead of Streams / plain `BRPOP`

`queue/client.go` — `ConsumeMessage` uses `BRPOPLPUSH` to atomically move the job from the main queue into a `:processing` sibling list.

- **Why**: we need at-least-once delivery and crash recovery without a message broker. `BRPOP` alone loses jobs if a worker dies mid-processing; Redis Streams would work but would add consumer-group state the API would also have to understand.
- **Implication**: the worker must `LREM` the job from `:processing` when it's done (`AcknowledgeMessage`), whether the job succeeded, failed, or moved to DLQ. A background recovery loop re-queues anything stuck in `:processing` beyond 10 minutes.
- **Trade-off**: a job that gets stuck for exactly the orphan window but is still running can be picked up twice. Acceptable because processing is idempotent per `videoID` (uploads overwrite).

## Retry in place, then dead-letter

`queue/job.go` — `MaxJobRetries = 3`; after that, `MoveToDLQ` on `<queue>:dead`.

- **Why**: transient failures (MinIO blip, FFmpeg deadlock) should self-heal, but looping forever on a truly broken video wastes workers and hides bugs.
- **No automatic DLQ drain**: jobs in `:dead` need human attention. If we auto-retried, we'd mask the underlying problem.
- **Retries are counted in two places**: explicit `SetJobFailed` and implicit `recoverStuckJobs` (orphan recovery). Both increment `RetryCount` so a repeatedly-crashing worker eventually gives up.

## Critical vs non-critical pipeline steps

`internal/processor/processor.go`.

- **Why**: a user's uploaded video is "done" as soon as we have a playable MP4. Thumbnails, preview, audio, HLS are nice-to-haves — failing the whole job because of a thumbnail glitch would degrade the product worse than silently shipping without one.
- **Consequence**: the success webhook may contain empty `thumbnailPaths`, `hlsPath`, `previewPath`, or `audioPath`. The API must not treat missing optional artifacts as a failure.
- **Only `validate` and `transcode` are critical.** Changing this classification is a product-level decision — discuss with VidroApi before touching it.

## Whole-job timeout of 5 minutes + per-step timeouts

`main.go` (5-minute `processCtx`) plus per-step timeouts in `internal/processor/processor.go`.

- **Why**: defence in depth. A per-step timeout prevents one FFmpeg hang from monopolising a worker. The whole-job timeout catches everything else (download stalls, upload stalls, recoverable bugs that never raise an error).
- **Tuning**: step timeouts assume short/medium user content. For longer videos, raise the transcode and streaming budgets first; the whole-job budget must always exceed the sum of critical-path steps (validate + analyze + transcode) plus download + upload slack.

## HLS single-command with sequential fallback

`internal/processor/processor-steps/streaming.go`.

- **Why**: a single FFmpeg invocation using `-filter_complex split` + `-var_stream_map` decodes the source exactly once and emits all variants in parallel. That's significantly faster than re-decoding per variant, which is what the sequential path does.
- **But**: the single command is fragile — it can fail on unusual inputs (missing audio, odd codecs, filter graph edge cases). Rather than pick one path at build time, we run single-command first and fall back to the sequential path if it errors, guarded by `HLS_SINGLE_COMMAND_FALLBACK`.
- **Variant selection**: we filter `hlsVariants` to those `Height <= sourceHeight` (probed via `ffprobe`) so we never upscale. If probing fails, we still emit at least the 240p variant rather than nothing.

## NVENC resolved at startup, CPU fallback inside each step

`internal/processor/processor-steps/video_encoder.go`, `transcode.go`, `streaming.go`.

- **Why**: GPU availability is a deploy-time fact. `ResolveVideoEncoder` probes `ffmpeg -encoders` once during `main()` so every job sees a consistent choice. `VIDEO_ENCODER=auto` is the default so the same binary works on CPU-only hosts and on GPU nodes.
- **Per-step fallback**: even after selecting NVENC, individual FFmpeg calls can fail on specific inputs (unusual colour spaces, CUDA driver hiccups). Both `transcode.go` and `streaming.go` catch those errors and retry on `libx264` to keep throughput up.
- **NVENC preset** is normalized to `p1`–`p7`; invalid values silently become `p5`. The default `p5` balances quality and speed on 1080p content.

## Raw videos soft-archived, then deleted by lifecycle rule

`minio/client.go` — `ArchiveRawVideo` + `configureRawArchivedLifecycle`.

- **Why**: keeping the raw after processing is insurance — we can reprocess if the pipeline is buggy, the transcoding parameters change, or a user reports their video looks wrong. But keeping it forever burns storage.
- **How**: on success we copy `raw/<id>` → `raw-archived/<id>` and delete the original. A MinIO lifecycle rule (`expire-raw-archived`, 30 days) deletes the archived copy automatically.
- **Why soft-move instead of relying on a single prefix**: putting the lifecycle rule directly on `raw/` would also sweep up jobs that haven't been processed yet (API just uploaded, worker hasn't picked up). The explicit move defers the TTL clock until processing is actually done.
- **Failure is non-fatal**: if archiving fails, we log a warning and keep the raw where it is. Better to leak storage than to lose the source.

## Circuit breakers with different thresholds for MinIO vs Redis

`internal/circuitbreaker/circuitbreaker.go`.

- **Why separate breakers**: a MinIO outage shouldn't prevent Redis operations (and vice versa) — we need queue bookkeeping to keep running even if object storage is down.
- **Different thresholds**: Redis failures are cheaper to retry and more likely to self-heal, so we trip faster (3 vs 5) and reset sooner (30s vs 60s). MinIO operations are expensive and sometimes slow — we tolerate a couple more failures before opening to avoid thrashing.
- **`MaxRequests: 1` in half-open**: only one probe request allowed while half-open, so we don't flood a recovering service.

## Worker count defaults to `runtime.NumCPU()`

`main.go`.

- **Why**: FFmpeg is CPU-bound, so one worker per core is the right starting point. The default avoids wasted threads waiting on I/O while still saturating the encoder.
- **Override via `WORKER_COUNT`**: set this explicitly for NVENC deployments (GPU is the bottleneck, so fewer workers is better) or for containers with CPU quotas (where `NumCPU` reports the host count).

## Webhook contract uses camelCase to match the .NET API

`internal/webhook/webhook.go`.

- **Why**: VidroApi (the producer) is a .NET service whose JSON serialiser produces camelCase. Matching the contract here avoids a custom converter on the API side.
- **HMAC signature is optional**: `WEBHOOK_SECRET` empty = no signing. Turned off in local dev, must be set in production.
- **Delivery is background-only**: webhook failures are logged but never fail the job. The API can always recover state from the `ProcessingFinishedQueue` or by polling `job:<id>`.

## Single bucket, path-based namespacing

`minio/client.go`.

- **Why**: one bucket is easier to provision, replicate, and secure than many. Lifecycle rules and IAM policies can still be scoped by prefix (`raw-archived/`).
- **Path layout is a shared contract** with VidroApi. Changing it is a coordinated release (tag both repos together).

## Graceful shutdown with a hard 30-second ceiling

`main.go`.

- **Why**: on `SIGTERM` we want workers to finish their current job if possible so we don't leak in-flight work to DLQ. But a stuck job must not block a Kubernetes pod from terminating, so we force-exit after 30s.
- **Tuning**: the ceiling should match or undercut the orchestrator's termination grace period. If you raise `processCtx`'s whole-job budget beyond 5 minutes, revisit whether 30 seconds is still enough to drain a clean-shutdown case.
