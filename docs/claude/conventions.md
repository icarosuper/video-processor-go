# Conventions — VidroProcessor

Rules to follow when writing or modifying Go code in this project. If you're adding a new pipeline step, a new MinIO/Redis call, or a new configuration knob, read this first.

## Language and formatting

- **All logs, error messages, comments, commit messages, and identifiers are English.** The entire source and docs are English (`"Starting video-processor"`, `"Step 1/7: Validating video"`). Do not mix Portuguese in.
- Standard `gofmt` / `go vet`. No custom linter rules.
- Package names stay lower-case and short (`queue`, `minio`, `metrics`, `processor`).

## Logging

- Use `zerolog` everywhere via `github.com/rs/zerolog/log`. Do not use the standard library `log` package in worker code (the only exception is `config/config.go` during startup).
- Prefer structured fields over formatted strings:

  ```go
  log.Info().Str("videoID", videoID).Int("workerID", id).Msg("Processing video")
  ```

- Log levels:
  - `Info` — normal lifecycle events and step boundaries (`"Step 3/7: Transcoding video"`).
  - `Warn` — recoverable problems (non-critical step failed, webhook delivery failed, circuit breaker tripped, orphan detected).
  - `Error` — job-level failures or health check failures that should page an operator.
  - `Fatal` — only during startup (`log.Fatal` kills the process). Never inside a job.

## Error handling

- Wrap errors with `fmt.Errorf("context: %w", err)` so `errors.Is` / `errors.As` works up the stack. Do not use `%v` for wrapping.
- Include a short imperative prefix describing what failed (`"failed to download video: %w"`, `"failed to serialize job state: %w"`).
- At the pipeline boundary, critical-step errors bubble up and abort the job; non-critical errors are logged and swallowed.

## Critical vs non-critical pipeline steps

This is the most important rule when touching `internal/processor`:

- **Critical** (validate, transcode): return the error from the step function. `ProcessVideo` aborts the pipeline and the job is marked failed/retried.
- **Non-critical** (analyze, thumbnails, audio, preview, streaming): log with `log.Warn` and return the error from the step function — but the orchestrator (`runNonCriticalStepsSequential` / `runNonCriticalStepsParallel`) swallows it. The corresponding `ProcessingResult` path is only set when the step succeeds, so downstream upload code can skip missing artifacts.
- If you add a new step, decide up front which category it belongs to and wire it into the orchestrator accordingly. Never let a non-critical step fail the whole pipeline.

## Configuration

- All runtime config lives in `config/config.go` as a single `Config` struct, loaded with `caarlos0/env/v10`.
- **Required** variables get the `notEmpty` tag — `env.Parse` will refuse to start without them. Use this for anything the worker can't function without.
- **Optional** variables use `envDefault:"..."`. Always provide a sensible default that works in dev.
- Mirror every new env var in `.env-example`.
- Never read environment variables with `os.Getenv` outside of `config/config.go`. Accept values through the `Config` struct.

## Metrics

- Declare new metrics in `metrics/metrics.go` with `promauto` so they register themselves.
- Observe durations with histograms (`video_processing_step_duration_seconds` pattern), not gauges.
- Use label cardinality carefully — `status` (success/error) and `step` (fixed set) are fine; never label by `videoID` or arbitrary user input.

## Tracing

- Wrap every pipeline step through `runStep`, which opens a `step/<name>` span and records errors. Do not call `telemetry.Tracer().Start` directly inside a step — add the step via `runStep` instead.
- The root per-job span `process_job` is opened in `main.go`; child spans are created inside `processor.ProcessVideo` and `runStep`. Everything else is a descendant by virtue of `ctx` propagation, so pass `ctx` through.

## External calls

- All MinIO and Redis calls go through the circuit breakers (`circuitbreaker.MinIO.Execute`, `circuitbreaker.Redis.Execute`). When adding a new MinIO/Redis function, wrap the inner call the same way the existing functions do.
- Don't block on external calls without a context/timeout. `ConsumeMessage` is the exception (`BRPOPLPUSH` with blocking timeout 0); cancellation comes from the shutdown context.

## Tests

- Unit tests live next to the code (`*_test.go` in the same package).
- Integration tests live in `test/integration/` and require docker-compose (Redis + MinIO). They're slow; run with `-timeout 10m`.
- Tests that shell out to `ffmpeg`/`ffprobe` **must** skip when the binaries are missing — use `GenerateTestVideo` from `test_helpers.go`, which handles the skip for you.
- Do not mock MinIO or Redis in integration tests. The point of those tests is to validate the real contract.

## File layout

- New pipeline steps go in `internal/processor/processor-steps/<name>.go` with a matching `<name>_test.go`. Register them in `processor.go` (both sequential and parallel orchestrators).
- New external-service clients get their own top-level package (`queue`, `minio`, ...), not a subfolder under `internal/`, to match the existing shape.
- Shared internal helpers (webhook, circuitbreaker, telemetry) live under `internal/`.

## Commits and branches

- Feature branches off `master` (`feature/<topic>`), merged via PR.
- `master` is always deployable; tags (`vX.Y.Z`) trigger production deploys.
- When a change touches the shared contract with VidroApi (queue names, MinIO paths, webhook payload), coordinate a joint tag + deploy with the API repo.
