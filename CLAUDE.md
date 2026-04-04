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

## Branching and release strategy

- **Feature branches** — one branch per feature group (e.g. `feature/dlq-retry`, `feature/hls`), branching off `master` and merged back via PR.
- **`master`** — always deployable. CI/CD deploys `master` HEAD to staging automatically.
- **Releases** — marked with a git tag (`v1.0.0`, `v1.1.0`, etc.) on `master`. Production deploys from tags.
- **No staging branches** — no `staging/vX.Y.Z` branches. Rollback is done by redeploying a previous tag.
- **Coordination with VidroApi** — when a change affects the shared contract (MinIO paths, Redis queue name, webhook format), both repos must be tagged and deployed together.

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
