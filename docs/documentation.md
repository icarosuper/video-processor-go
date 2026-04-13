# VidroProcessor - Project Documentation

## Overview

**VidroProcessor** = distributed video proc system, Go, worker-based + message queues. Process videos async/scalable via 7-step FFmpeg pipeline.

## Main Goal

Recv raw videos → transform pipeline (validate, transcode, thumbnails, audio, preview, HLS) → store MinIO. Redis decouples comms, enables horiz scale.

## Architecture

```
┌─────────────────┐      ┌──────────────┐      ┌──────────────┐
│   Producer      │──────│   Redis      │──────│   Workers    │
│   (Videos)      │      │   Queues     │      │  (Multiple)  │
└─────────────────┘      └──────────────┘      └──────────────┘
                                                        │
                                                        ▼
                                                 ┌──────────────┐
                                                 │  Processing  │
                                                 │  Pipeline    │
                                                 └──────────────┘
                                                        │
                                                        ▼
                                                 ┌──────────────┐
                                                 │    MinIO     │
                                                 │  (Results)   │
                                                 └──────────────┘
```

### Main Components

#### 1. Concurrent Workers
- Multi workers, parallel proc
- Configurable via `WORKER_COUNT` (default: num CPUs)
- Graceful shutdown, 30s timeout

#### 2. Processing Pipeline

7 sequential steps in `internal/processor/processor-steps/`:

| Step | File | Required | Output |
|---|---|---|---|
| 1. Validation | `validate.go` | Yes | — |
| 2. Analysis | `analysis.go` | No (informational) | metadata in log |
| 3. Transcoding | `transcode.go` | Yes | `*_output.mp4` |
| 4. Thumbnails | `thumbnail.go` | No | `thumbnails/thumb_00N.jpg` |
| 5. Audio Extraction | `audio.go` | No | `audio.mp3` |
| 6. Preview | `preview.go` | No | `preview.mp4` |
| 7. HLS Streaming | `streaming.go` | No | `streaming/*.ts` + `playlist.m3u8` |

> **Note**: only transcoded video (step 3) sent to MinIO. Steps 4–7 artifacts gen in `tempDir`, discarded after. See [Known Issues](#known-issues).

#### 3. Queue System (Redis)

- **`PROCESSING_REQUEST_QUEUE`**: recv video IDs to proc (BLPop)
- **`PROCESSING_FINISHED_QUEUE`**: recv success video IDs (LPush)

#### 4. Storage (MinIO)

- Prefix `raw/`: orig videos
- Prefix `processed/`: transcoded videos

## Technology Stack

| Component | Technology |
|---|---|
| Language | Go 1.24 |
| Queues | Redis 7 (go-redis/v8) |
| Storage | MinIO (minio-go/v7) |
| Processing | FFmpeg / FFprobe |
| Metrics | Prometheus (promauto) |
| Logging | Zerolog |
| Config | caarlos0/env v10 + godotenv |
| Containers | Docker + Docker Compose |

## Project Structure

```
VidroProcessor/
├── config/
│   └── config.go                    # Configuration management
├── internal/
│   └── processor/
│       ├── processor.go             # Pipeline orchestrator
│       └── processor-steps/        # Processing steps
│           ├── analysis.go
│           ├── audio.go
│           ├── preview.go
│           ├── streaming.go
│           ├── thumbnail.go
│           ├── transcode.go
│           ├── validate.go
│           ├── test_helpers.go      # Test helpers
│           └── testdata/
├── metrics/
│   └── metrics.go                   # Prometheus metrics
├── minio/
│   └── client.go                    # MinIO client
├── queue/
│   └── client.go                    # Redis client
├── test/
│   └── integration/                 # Integration tests (testcontainers)
├── docs/
├── main.go
├── docker-compose.yml
└── Dockerfile
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `REDIS_HOST` | Yes | e.g. `localhost:6379` |
| `PROCESSING_REQUEST_QUEUE` | Yes | Input queue name |
| `PROCESSING_FINISHED_QUEUE` | Yes | Success queue name |
| `MINIO_ENDPOINT` | Yes | e.g. `localhost:9000` |
| `MINIO_ROOT_USER` | Yes | MinIO user |
| `MINIO_ROOT_PASSWORD` | Yes | MinIO password |
| `MINIO_BUCKET_NAME` | Yes | Bucket name |
| `WORKER_COUNT` | No | Default: `runtime.NumCPU()` |

### Note on `.env`

`config.LoadConfig()` calls `godotenv.Load()` with `log.Fatal` if `.env` missing. Docker/K8s envs w/ direct var injection → startup fail. See [Known Issues](#known-issues).

## How to Run

### Development

```bash
cp .env-example .env
# edit .env with your configuration

docker-compose up -d redis minio

go mod download
go run main.go
```

### Production (Docker Compose)

```bash
docker-compose up -d
```

## Processing Flow

```
1. Producer publishes VideoID → PROCESSING_REQUEST_QUEUE
2. Worker consumes via BLPop
3. Worker downloads raw/{VideoID} from MinIO
4. Pipeline executes the 7 steps
5. Worker uploads all artifacts to MinIO:
   - processed/{VideoID}_processed       (transcoded video)
   - thumbnails/{VideoID}/thumb_00N.jpg  (5 frames)
   - audio/{VideoID}.mp3
   - preview/{VideoID}_preview.mp4
   - hls/{VideoID}/playlist.m3u8 + segment_*.ts
6. Worker publishes VideoID → PROCESSING_FINISHED_QUEUE
```

## Known Issues

System functional for basic flow, not prod-ready. Main blockers:

- **No job state**: API blind to proc/done/fail status
- **Destructive BLPop**: crash during proc = lost job
- **Single resolution**: one MP4 only; adaptive streaming needs multi quality
- **Silent failures**: failed jobs no notify API

Full plan: [roadmap.md](./roadmap.md).

## Operational Considerations

- Proc is CPU-heavy; calibrate `WORKER_COUNT` per hardware
- Temp storage in `/tmp`; SSD rec for perf
- SSL/TLS in MinIO hardcoded `false`; must be configurable for prod
- Logs in ConsoleWriter format (not JSON) by default — switch to JSON in prod

---

**Version**: 0.1.0
**Status**: Pipeline functional — prod blockers in roadmap
**Last Updated**: 2026-03-26