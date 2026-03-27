# CLAUDE.md — VidroProcessor

## What this project is

Async worker in Go that an API calls to process user-submitted videos (YouTube model). Consumes video IDs from a Redis queue, downloads from MinIO, processes with FFmpeg in a 7-step pipeline, and uploads the artifacts back to MinIO.

## Essential structure

```
main.go                          # worker pool + graceful shutdown + HTTP server
config/config.go                 # env vars via caarlos0/env
queue/client.go                  # Redis BRPopLPush (atomic consumption) + orphan recovery
queue/job.go                     # job state (pending→processing→done/failed), retry, DLQ
internal/webhook/webhook.go      # POST notification to callbackURL with retry and optional HMAC
internal/circuitbreaker/circuitbreaker.go  # circuit breakers for MinIO and Redis
internal/telemetry/telemetry.go            # OpenTelemetry: init, tracer, shutdown
minio/client.go                  # download/upload of videos and artifacts
metrics/metrics.go               # Prometheus metrics (promauto)
internal/processor/processor.go  # orchestrator of the 7 steps, returns ProcessingResult
internal/processor/processor-steps/*.go  # each pipeline step
```

## Current state (~95% production-ready)

The FFmpeg pipeline works end-to-end with job reliability (retry, DLQ, orphan recovery), notification webhook, persisted metadata, real operational metrics, adaptive HLS with multiple resolutions, circuit breakers for MinIO/Redis, and individual timeouts per step. Remaining items: scalability (auto-scaling, multiple instances, queue prioritization) and Grafana Dashboard.

See `docs/roadmap.md` for the full plan.

## Project conventions

- Logs in English (`"Starting video-processor"`, `"Step 1/7: Validating video"`)
- Errors wrapped with `fmt.Errorf("context: %w", err)`
- Non-critical steps (thumbnails, audio, preview, HLS) use `log.Warn` and do not return error — the pipeline continues even if they fail; paths are only set in `ProcessingResult` when the step succeeds
- Critical steps (validation, transcoding) return error and abort the pipeline
- Required environment variables with `notEmpty` tag via caarlos0/env
- Processing tests automatically skip if FFmpeg is not available (`GenerateTestVideo` in `test_helpers.go`)

## Running locally

```bash
cp .env-example .env
docker-compose up -d redis minio
go run main.go
```

## Running tests

```bash
go test ./...                                      # unit tests
go test -v ./test/integration/... -timeout 10m    # integration (requires Docker)
```
