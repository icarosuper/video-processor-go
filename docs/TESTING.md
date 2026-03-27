# Testing - VidroProcessor

## Current Coverage

| Package | Coverage | Type |
|---|---|---|
| `internal/processor/processor-steps` | 63.7% | Unit |
| `metrics` | ~100% | Unit |
| `internal/circuitbreaker` | ~100% | Unit |
| `internal/webhook` | ~100% | Unit |
| `internal/telemetry` | ~100% | Unit |
| `test/integration` | — | Integration |

**Packages without tests**: `main`, `config`, `queue`, `minio`

---

## Running Tests

```bash
# All tests
go test ./...

# With coverage
go test ./... -cover

# Verbose output
go test -v ./...

# HTML report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### By package

```bash
go test -v ./internal/processor/processor-steps/...
go test -v ./metrics/...
go test -v ./test/integration/... -timeout 10m
```

### Integration tests

Require Docker. Automatically skipped if Docker is not available.

```bash
go test -v ./test/integration/... -timeout 10m
```

---

## Unit Tests

### `validate_test.go`
- `TestValidateVideo_ValidVideo`
- `TestValidateVideo_InvalidVideo`
- `TestValidateVideo_NonExistentFile`
- `TestValidateVideo_EmptyFile`

### `transcode_test.go`
- `TestTranscodeVideo_ValidVideo` — checks output file was created and is non-empty
- `TestTranscodeVideo_InvalidInput`
- `TestTranscodeVideo_NonExistentInput`

### `thumbnail_test.go`
- `TestGenerateThumbnails_ValidVideo` — checks 5 `thumb_00N.jpg` files
- `TestGenerateThumbnails_InvalidVideo`
- `TestGenerateThumbnails_NonExistentVideo`

### `analysis_test.go`
- `TestAnalyzeContent_ValidVideo`
- `TestAnalyzeContent_InvalidVideo`
- `TestAnalyzeContent_NonExistentVideo`

### `metrics_test.go`
- `TestVideosProcessedTotal_Increment`
- `TestProcessingDuration_Observe`
- `TestProcessingStepDuration_MultipleSteps`
- `TestActiveWorkers_SetAndGet`
- `TestQueueSize_SetAndGet`

### `internal/circuitbreaker/circuitbreaker_test.go`
- `TestMinIO_InitialState_Closed`
- `TestRedis_InitialState_Closed`
- `TestCircuitBreaker_OpensAfter5ConsecutiveFailures`
- `TestCircuitBreaker_OpensAfter3ConsecutiveFailures_Redis`
- `TestCircuitBreaker_RejectsCallsWhenOpen`
- `TestCircuitBreaker_DoesNotOpenWithNonConsecutiveFailures`
- `TestCircuitBreaker_ReturnsResultWhenClosed`

### `internal/webhook/webhook_test.go`
- `TestNotify_Success`
- `TestNotify_ContentTypeJSON`
- `TestNotify_WithHMAC_CorrectSignature`
- `TestNotify_NoSecret_NoHeader`
- `TestNotify_RetryOnFailure`
- `TestNotify_ErrorAfter3Attempts`
- `TestNotify_InvalidURL`
- `TestNotify_ServerUnavailable`
- `TestPayload_JSONSerialization`

### `internal/telemetry/telemetry_test.go`
- `TestInit_EmptyEndpoint_Noop`
- `TestInit_EmptyEndpoint_InstallsNoop`
- `TestTracer_ReturnsNonNil`
- `TestTracer_CreatesSpan`
- `TestTracerName_Constant`
- `TestInit_InvalidEndpoint_ReturnsError`

---

## Integration Tests (`test/integration/`)

Use **testcontainers-go** to spin up real Redis and MinIO instances.

### `minio_test.go`
- `TestMinIO_BucketOperations`
- `TestMinIO_ObjectUploadDownload`
- `TestMinIO_VideoWorkflow` — raw → processed flow
- `TestMinIO_DownloadToFile`
- `TestMinIO_NonExistentObject`

### Pipeline tests (`pipeline_test.go`)
- `TestPipeline_ValidateStep`
- `TestPipeline_TranscodeStep`
- `TestPipeline_FullWorkflow` — Redis → download → FFmpeg → upload → success queue
- `TestPipeline_ThumbnailGeneration`

---

## Test Helpers

### `GenerateTestVideo(t, duration int) string`

Generates a test video via FFmpeg (640x480, H.264+AAC, 1000Hz sine wave).
Automatically skips the test if FFmpeg is not available.

```go
videoPath := GenerateTestVideo(t, 5) // 5 seconds
```

### `CreateInvalidFile(t) string`

Creates a file with invalid content for error testing.

---

## Requirements

**FFmpeg** is required for processing tests:

```bash
# Ubuntu/Debian
sudo apt-get install ffmpeg

# macOS
brew install ffmpeg
```

Tests that depend on FFmpeg are automatically skipped with:
```
FFmpeg is not available - skipping test
```

---

## What's Missing

- `config.LoadConfig()` — including behavior without `.env`
- `queue.ConsumeMessage()` and `PublishSuccessMessage()`
- `minio.DownloadVideo()` and `UploadVideo()`
- `main.processNextMessage()` — worker orchestration logic
- Steps without tests: `audio.go`, `preview.go`, `streaming.go`
- Transcoding and throughput benchmarks

---

## CI/CD

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: sudo apt-get install -y ffmpeg
      - run: go test -v ./... -cover -timeout 10m
```

---

**Last Updated**: 2026-03-26
**Current Coverage**: 63.7% (processor-steps), ~100% (metrics, circuitbreaker, webhook, telemetry)
