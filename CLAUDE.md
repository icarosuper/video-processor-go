# CLAUDE.md — VidroProcessor

Async Go worker consumed by VidroApi. Pulls video IDs from a Redis queue, downloads the source from MinIO, runs a 7-step FFmpeg pipeline, and uploads the artifacts back. Stateless — scale by running more instances against the same Redis.

## Always-on rules

- **Language**: all code, logs, errors, comments, commit messages, and docs are in English. No Portuguese in source.
- **Error wrapping**: `fmt.Errorf("context: %w", err)` — never `%v`.
- **Required config**: required env vars use `notEmpty` (caarlos0/env); optional vars use `envDefault`. Mirror every new var in `.env-example`.
- **Shared contract with VidroApi**: queue names, MinIO paths, webhook payload. Any change here needs a coordinated tag + deploy in both repos.
- **Branching**: `feature/<topic>` off `master`, merged via PR. `master` is always deployable; production deploys from `vX.Y.Z` tags.
- **Only create files when necessary.** Don't add *.md or README docs unless explicitly asked.

## Running locally

```bash
cp .env-example .env
docker-compose up -d redis minio
go run main.go
```

## Running tests

```bash
go test ./...                                     # unit
go test -v ./test/integration/... -timeout 10m    # integration (needs Docker)
```

Tests that shell out to `ffmpeg`/`ffprobe` auto-skip when the binaries are missing — use `GenerateTestVideo` from `processor-steps/test_helpers.go`.

## Docs pointers — read these when the task calls for it

- **`docs/claude/features-index.md`** — read first when locating code. Maps every feature (worker lifecycle, queue, each pipeline step, MinIO operations, webhook, metrics) to its file, plus the canonical MinIO object layout and queue names. Update it whenever you add a module, a pipeline step, or a new external contract.

- **`docs/claude/architecture.md`** — read before making structural changes, adding a pipeline step, or debugging end-to-end flow. Covers the worker lifecycle, queue protocol (main / `:processing` / `:dead` / finished), job state machine, the 7-step pipeline orchestration, and resilience layers (circuit breakers, orphan recovery, retries).

- **`docs/claude/conventions.md`** — read before writing or modifying Go code. Rules for logging (zerolog, structured fields), error wrapping, critical vs non-critical step classification, config loading, metrics/tracing, circuit-breaker wrapping, and test layout.

- **`docs/claude/design-decisions.md`** — read before proposing an architectural change or questioning *why* something is done a certain way. Explains `BRPOPLPUSH` over Streams, retry→DLQ policy, HLS single-command with fallback, NVENC auto-probe + CPU fallback, soft-archived raws with lifecycle rule, separate MinIO/Redis circuit breakers, and the webhook contract shape.

- **`docs/roadmap.md`** — read when the user asks about project status, remaining work, or what to build next. The project is ~95% production-ready; remaining work centres on scalability and Grafana dashboards.

- **`docs/GETTING_STARTED.md`**, **`docs/OBSERVABILITY.md`**, **`docs/TESTING.md`** — user-facing guides for setup, the observability stack, and running the full integration suite. Read when the user's question is about operating the worker, not about the code.

## Keeping this index healthy

- Adding a feature → update `features-index.md`.
- Changing architecture or a cross-cutting flow → update `architecture.md` (and `design-decisions.md` if the *why* changes).
- Changing a coding rule → update `conventions.md`.
- This file should only change when the project structure itself changes.
