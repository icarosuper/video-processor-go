# Observabilidade - VidroProcessor

## Endpoints HTTP (`:8080`)

### `/health`

Verifica conectividade com Redis e MinIO.

```bash
curl http://localhost:8080/health
# 200 OK → "OK"
# 503 Service Unavailable → "Redis unavailable" ou "MinIO unavailable"
```

### `/metrics`

Expõe métricas no formato Prometheus.

```bash
curl http://localhost:8080/metrics
```

---

## Métricas Disponíveis

### `videos_processed_total` (Counter)

Total de vídeos processados por status.

**Labels**: `status` = `success` | `error`

```
videos_processed_total{status="success"} 42
videos_processed_total{status="error"} 3
```

### `video_processing_duration_seconds` (Histogram)

Tempo total de processamento por vídeo (download → upload).

**Buckets**: padrão Prometheus (0.005s a 10s)

### `video_processing_step_duration_seconds` (Histogram)

Tempo de cada etapa do pipeline.

**Labels**: `step` = `validate` | `analyze` | `transcode` | `thumbnails` | `audio` | `preview` | `streaming`

**Buckets**: 0.1s, 0.5s, 1s, 2s, 5s, 10s, 30s, 60s, 120s, 300s

Útil para identificar gargalos: qual etapa está mais lenta.

### `active_workers` (Gauge)

Número de workers com um job em andamento no momento. Incrementado ao iniciar cada job, decrementado ao concluir.

### `queue_size` (Gauge)

Tamanho atual da fila de entrada. Atualizado a cada 30 segundos via `LLEN` no Redis.

### `video_size_bytes` (Histogram)

Tamanho dos vídeos baixados do MinIO, registrado após o download.

**Buckets**: exponencial de 1MB a ~16GB

---

## Grafana Dashboard

O projeto inclui um dashboard pré-configurado em `grafana/provisioning/dashboards/video-processor.json`, carregado automaticamente ao subir o `docker-compose`.

**Painéis disponíveis:**
- Workers ativos e tamanho da fila (com thresholds de cor)
- Total de vídeos processados por status
- Taxa de sucesso (gauge %)
- Throughput em vídeos/min
- Duração por etapa p50/p90/p99
- Duração total do job p50/p90/p99
- Distribuição de tamanho dos vídeos

Acesse em `http://localhost:3000` (admin/admin) após `docker-compose up`.

---

## Integração com Prometheus

O `prometheus.yml` já está configurado em `prometheus/prometheus.yml` e montado no container via `docker-compose`. Para rodar o worker fora do Docker, adicione ao seu `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'video-processor'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
```

---

## Queries Grafana Úteis

**Taxa de processamento**
```promql
rate(videos_processed_total[5m])
```

**Taxa de erro**
```promql
rate(videos_processed_total{status="error"}[5m])
/ rate(videos_processed_total[5m])
```

**Tempo médio de processamento**
```promql
rate(video_processing_duration_seconds_sum[5m])
/ rate(video_processing_duration_seconds_count[5m])
```

**Tempo p95 por etapa**
```promql
histogram_quantile(0.95,
  rate(video_processing_step_duration_seconds_bucket[5m])
)
```

---

## Alertas Recomendados

```yaml
# Alta taxa de erro
alert: HighVideoProcessingErrorRate
expr: rate(videos_processed_total{status="error"}[5m]) > 0.1
for: 5m

# Processamento lento (p95 > 5min)
alert: SlowVideoProcessing
expr: histogram_quantile(0.95, rate(video_processing_duration_seconds_bucket[5m])) > 300
for: 15m

# Serviço indisponível
alert: VideoProcessorDown
expr: up{job="video-processor"} == 0
for: 1m
```

---

## Logs Estruturados

Os logs usam **Zerolog** em formato ConsoleWriter (legível) por padrão. Para produção com coleta centralizada (Loki, ELK), substituir em `main.go`:

```go
// Trocar ConsoleWriter por saída JSON
log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
```

**Campos de contexto nos logs**:
- `workerID` — ID do worker
- `videoID` — ID do vídeo em processamento
- `duration_seconds` — tempo de processamento
- `object` — path do objeto no MinIO

---

## Troubleshooting

**Health check falhando — Redis**
```bash
docker ps | grep redis
redis-cli -h localhost ping
```

**Health check falhando — MinIO**
```bash
docker ps | grep minio
# Acessar console: http://localhost:9001
```

**Métricas não aparecendo no Prometheus**
```bash
# Verificar se o endpoint responde
curl http://localhost:8080/metrics | head -20

# Verificar target no Prometheus
# http://localhost:9090/targets
```

---

**Última Atualização**: 2026-03-26
