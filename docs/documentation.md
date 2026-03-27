# VidroProcessor - Project Documentation

## Overview

**VidroProcessor** is a distributed video processing system built in Go, using a worker-based architecture with message queues. The system processes videos asynchronously and scalably through a 7-step FFmpeg pipeline.

## Main Goal

Receive raw videos, process them through a transformation pipeline (validation, transcoding, thumbnails, audio, preview, and HLS), and store the results in MinIO. Communication is decoupled via Redis, enabling horizontal scalability.

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
- Multiple workers processing videos in parallel
- Configurable count via `WORKER_COUNT` (default: number of CPUs)
- Graceful shutdown with 30-second timeout

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

> **Note**: currently only the transcoded video (step 3) is sent to MinIO. Artifacts from steps 4–7 are generated in `tempDir` and discarded at the end. See [Known Issues](#known-issues).

#### 3. Queue System (Redis)

- **`PROCESSING_REQUEST_QUEUE`**: receives video IDs to process (BLPop)
- **`PROCESSING_FINISHED_QUEUE`**: receives successfully processed video IDs (LPush)

#### 4. Storage (MinIO)

- Prefix `raw/`: original videos
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

`config.LoadConfig()` calls `godotenv.Load()` with `log.Fatal` if the `.env` file doesn't exist. In Docker/Kubernetes environments where variables are injected directly, this causes a startup failure. See [Known Issues](#known-issues).

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

The system is functional for the basic flow, but not yet production-ready. The main blockers are:

- **No job state**: the API doesn't know if a video is processing, finished, or failed
- **Destructive BLPop**: crash during processing = lost job
- **Single resolution**: generates one MP4; adaptive streaming needs multiple qualities
- **Silent failures**: failed jobs don't notify the API

See the full plan in [roadmap.md](./roadmap.md).

## Operational Considerations

- Video processing is CPU-intensive; calibrate `WORKER_COUNT` according to hardware
- Temporary storage in `/tmp`; SSD recommended for performance
- SSL/TLS in MinIO is hardcoded as `false`; should be configurable for production
- Logs in ConsoleWriter format (not JSON) by default — switch to JSON in production

---

**Version**: 0.1.0
**Status**: Pipeline functional — production blockers documented in roadmap
**Last Updated**: 2026-03-26
