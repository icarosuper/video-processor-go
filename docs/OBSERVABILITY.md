# Observability - VidroProcessor

## HTTP Endpoints (`:8080`)

### `/health`

Checks connectivity with Redis and MinIO.

```bash
curl http://localhost:8080/health
# 200 OK → "OK"
# 503 Service Unavailable → "Redis unavailable" or "MinIO unavailable"
```

### `/metrics`

Exposes metrics in Prometheus format.

```bash
curl http://localhost:8080/metrics
```

---

## Available Metrics

### `videos_processed_total` (Counter)

Total videos processed by status.

**Labels**: `status` = `success` | `error`

```
videos_processed_total{status="success"} 42
videos_processed_total{status="error"} 3
```

### `video_processing_duration_seconds` (Histogram)

Total processing time per video (download → upload).

**Buckets**: Prometheus default (0.005s to 10s)

### `video_processing_step_duration_seconds` (Histogram)

Time for each pipeline step.

**Labels**: `step` = `validate` | `analyze` | `transcode` | `thumbnails` | `audio` | `preview` | `streaming`

**Buckets**: 0.1s, 0.5s, 1s, 2s, 5s, 10s, 30s, 60s, 120s, 300s

Useful for identifying bottlenecks: which step is slowest.

### `active_workers` (Gauge)

Number of workers with a job in progress at the moment. Incremented when each job starts, decremented when it finishes.

### `queue_size` (Gauge)

Current size of the input queue. Updated every 30 seconds via `LLEN` on Redis.

### `video_size_bytes` (Histogram)

Size of videos downloaded from MinIO, recorded after the download.

**Buckets**: exponential from 1MB to ~16GB

---

## Grafana Dashboard

The project includes a pre-configured dashboard at `grafana/provisioning/dashboards/video-processor.json`, loaded automatically when `docker-compose` starts.

**Available panels:**
- Active workers and queue size (with color thresholds)
- Total videos processed by status
- Success rate (gauge %)
- Throughput in videos/min
- Duration per step p50/p90/p99
- Total job duration p50/p90/p99
- Video size distribution

Access at `http://localhost:3000` (admin/admin) after `docker-compose up`.

---

## Prometheus Integration

`prometheus.yml` is already configured at `prometheus/prometheus.yml` and mounted in the container via `docker-compose`. To run the worker outside Docker, add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'video-processor'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
```

---

## Useful Grafana Queries

**Processing rate**
```promql
rate(videos_processed_total[5m])
```

**Error rate**
```promql
rate(videos_processed_total{status="error"}[5m])
/ rate(videos_processed_total[5m])
```

**Average processing time**
```promql
rate(video_processing_duration_seconds_sum[5m])
/ rate(video_processing_duration_seconds_count[5m])
```

**p95 time per step**
```promql
histogram_quantile(0.95,
  rate(video_processing_step_duration_seconds_bucket[5m])
)
```

---

## Recommended Alerts

```yaml
# High error rate
alert: HighVideoProcessingErrorRate
expr: rate(videos_processed_total{status="error"}[5m]) > 0.1
for: 5m

# Slow processing (p95 > 5min)
alert: SlowVideoProcessing
expr: histogram_quantile(0.95, rate(video_processing_duration_seconds_bucket[5m])) > 300
for: 15m

# Service unavailable
alert: VideoProcessorDown
expr: up{job="video-processor"} == 0
for: 1m
```

---

## Structured Logs

Logs use **Zerolog** in ConsoleWriter format (human-readable) by default. For production with centralized collection (Loki, ELK), replace in `main.go`:

```go
// Switch from ConsoleWriter to JSON output
log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
```

**Context fields in logs**:
- `workerID` — worker ID
- `videoID` — ID of the video being processed
- `duration_seconds` — processing time
- `object` — path of the object in MinIO

---

## Troubleshooting

**Health check failing — Redis**
```bash
docker ps | grep redis
redis-cli -h localhost ping
```

**Health check failing — MinIO**
```bash
docker ps | grep minio
# Access console: http://localhost:9001
```

**Metrics not appearing in Prometheus**
```bash
# Check if the endpoint responds
curl http://localhost:8080/metrics | head -20

# Check target in Prometheus
# http://localhost:9090/targets
```

---

**Last Updated**: 2026-03-26
