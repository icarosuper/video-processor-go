# Conventions — VidroProcessor

Rules when write/modify Go code. New pipeline step, MinIO/Redis call, or config knob — read first.

## Language and formatting

- **All logs, errors, comments, commits, identifiers: English.** No Portuguese mix.
- Standard `gofmt` / `go vet`. No custom linter.
- Package names: lower-case, short (`queue`, `minio`, `metrics`, `processor`).

## Logging

- Use `zerolog` via `github.com/rs/zerolog/log`. No stdlib `log` in worker code (exception: `config/config.go` startup).
- Prefer structured fields:

  ```go
  log.Info().Str("videoID", videoID).Int("workerID", id).Msg("Processing video")
  ```

- Log levels:
  - `Info` — lifecycle events, step boundaries.
  - `Warn` — recoverable problems (non-critical step fail, webhook fail, circuit breaker trip, orphan).
  - `Error` — job-level or health check failures needing operator.
  - `Fatal` — startup only. Never inside job.

## Error handling

- Wrap with `fmt.Errorf("context: %w", err)`. Not `%v`.
- Short imperative prefix: `"failed to download video: %w"`, `"failed to serialize job state: %w"`.
- Critical step errors bubble up, abort job. Non-critical: log, swallow.

## Critical vs non-critical pipeline steps

Most important rule in `internal/processor`:

- **Critical** (validate, transcode): return error → `ProcessVideo` aborts, job marked failed/retried.
- **Non-critical** (analyze, thumbnails, audio, preview, streaming): log `Warn`, return error — orchestrator (`runNonCriticalStepsSequential` / `runNonCriticalStepsParallel`) swallows. `ProcessingResult` path set only on success; upload code skips missing artifacts.
- New step: decide category upfront, wire into orchestrator. Never let non-critical step fail pipeline.

## Configuration

- All runtime config in `config/config.go` as `Config` struct, loaded via `caarlos0/env/v10`.
- **Required** vars: `notEmpty` tag — `env.Parse` refuses start without them.
- **Optional** vars: `envDefault:"..."`. Always sensible dev default.
- Mirror every new env var in `.env-example`.
- No `os.Getenv` outside `config/config.go`. Use `Config` struct.

## Metrics

- Declare new metrics in `metrics/metrics.go` with `promauto`.
- Durations: histograms (`video_processing_step_duration_seconds` pattern), not gauges.
- Labels: `status` (success/error), `step` (fixed set) ok. Never label by `videoID` or arbitrary user input.

## Tracing

- All pipeline steps via `runStep` — opens `step/<name>` span, records errors. No direct `telemetry.Tracer().Start` inside step.
- Root span `process_job` opened in `main.go`; children in `processor.ProcessVideo` and `runStep`. Pass `ctx` through.

## External calls

- All MinIO/Redis calls via circuit breakers (`circuitbreaker.MinIO.Execute`, `circuitbreaker.Redis.Execute`). New functions: wrap same as existing.
- No blocking external calls without context/timeout. Exception: `ConsumeMessage` (`BRPOPLPUSH` blocking timeout 0) — cancellation via shutdown context.

## Tests

- Unit tests: `*_test.go` same package.
- Integration tests: `test/integration/`, need docker-compose (Redis + MinIO). Slow; run `-timeout 10m`.
- Tests using `ffmpeg`/`ffprobe`: must skip when binaries missing — use `GenerateTestVideo` from `test_helpers.go`.
- No mocking MinIO/Redis in integration tests. Test real contract.

## File layout

- New pipeline steps: `internal/processor/processor-steps/<name>.go` + `<name>_test.go`. Register in `processor.go` (both orchestrators).
- New external-service clients: own top-level package (`queue`, `minio`, ...), not under `internal/`.
- Shared internal helpers (webhook, circuitbreaker, telemetry): `internal/`.

## Commits and branches

- Feature branches off `master` (`feature/<topic>`), merged via PR.
- `master` always deployable; tags (`vX.Y.Z`) trigger prod deploys.
- Changes touching shared contract with VidroApi (queue names, MinIO paths, webhook payload): coordinate joint tag + deploy with API repo.