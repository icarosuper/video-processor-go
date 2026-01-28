# 📊 Observabilidade - Video Processor Go

Este documento descreve como monitorar e observar o sistema de processamento de vídeos.

## 🔍 Endpoints Disponíveis

### Health Check - `/health`
Verifica se o serviço está saudável e se as dependências estão disponíveis.

**URL**: `http://localhost:8080/health`

**Método**: `GET`

**Respostas**:
- `200 OK` - Todos os serviços estão funcionando
- `503 Service Unavailable` - Redis ou MinIO indisponível

**Exemplo**:
```bash
curl http://localhost:8080/health
# Resposta: OK
```

**Verificações realizadas**:
- ✅ Conectividade com Redis
- ✅ Conectividade com MinIO
- ✅ Existência do bucket configurado

---

### Métricas Prometheus - `/metrics`
Expõe métricas do sistema no formato Prometheus.

**URL**: `http://localhost:8080/metrics`

**Método**: `GET`

**Exemplo**:
```bash
curl http://localhost:8080/metrics
```

---

## 📈 Métricas Disponíveis

### 1. **videos_processed_total** (Counter)
Contador total de vídeos processados, com label de status.

**Labels**:
- `status`: `success` ou `error`

**Exemplo**:
```
videos_processed_total{status="success"} 42
videos_processed_total{status="error"} 3
```

---

### 2. **video_processing_duration_seconds** (Histogram)
Tempo total de processamento de vídeos (do início ao fim).

**Buckets**: Default do Prometheus (0.005s até 10s)

**Exemplo**:
```
video_processing_duration_seconds_bucket{le="5"} 10
video_processing_duration_seconds_bucket{le="10"} 25
video_processing_duration_seconds_sum 127.5
video_processing_duration_seconds_count 30
```

---

### 3. **video_processing_step_duration_seconds** (Histogram)
Tempo de processamento de cada etapa individual do pipeline.

**Labels**:
- `step`: `validate`, `analyze`, `transcode`, `thumbnails`, `audio`, `preview`, `streaming`

**Buckets**: 0.1s, 0.5s, 1s, 2s, 5s, 10s, 30s, 60s, 120s, 300s

**Exemplo**:
```
video_processing_step_duration_seconds_bucket{step="transcode",le="60"} 15
video_processing_step_duration_seconds_sum{step="transcode"} 450.2
video_processing_step_duration_seconds_count{step="transcode"} 20
```

**Uso**: Identificar gargalos no pipeline (qual etapa está mais lenta).

---

### 4. **active_workers** (Gauge)
Número de workers atualmente ativos processando vídeos.

**Exemplo**:
```
active_workers 4
```

---

### 5. **queue_size** (Gauge)
Tamanho atual da fila de processamento (vídeos aguardando).

**Exemplo**:
```
queue_size 15
```

**Uso**: Monitorar acúmulo de trabalho na fila.

---

### 6. **video_size_bytes** (Histogram)
Distribuição do tamanho dos vídeos processados em bytes.

**Buckets**: Exponencial de 1MB até ~16GB

**Exemplo**:
```
video_size_bytes_bucket{le="1.048576e+06"} 5
video_size_bytes_bucket{le="2.097152e+06"} 12
video_size_bytes_sum 2.5e+08
video_size_bytes_count 30
```

---

## 📊 Integração com Grafana

### Prometheus Configuration
Adicione ao seu `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'video-processor'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
```

### Grafana Dashboards

#### Dashboard Recomendado

**1. Taxa de Processamento**
```promql
rate(videos_processed_total[5m])
```

**2. Taxa de Erro**
```promql
rate(videos_processed_total{status="error"}[5m]) / rate(videos_processed_total[5m])
```

**3. Tempo Médio de Processamento**
```promql
rate(video_processing_duration_seconds_sum[5m]) / rate(video_processing_duration_seconds_count[5m])
```

**4. Tempo por Etapa (p95)**
```promql
histogram_quantile(0.95, rate(video_processing_step_duration_seconds_bucket[5m]))
```

**5. Workers Ativos**
```promql
active_workers
```

**6. Tamanho da Fila**
```promql
queue_size
```

---

## 🚨 Alertas Recomendados

### 1. Alta Taxa de Erro
```yaml
alert: HighVideoProcessingErrorRate
expr: rate(videos_processed_total{status="error"}[5m]) > 0.1
for: 5m
annotations:
  summary: "Alta taxa de erro no processamento de vídeos"
```

### 2. Fila Crescendo
```yaml
alert: VideoQueueGrowing
expr: delta(queue_size[10m]) > 50
for: 10m
annotations:
  summary: "Fila de vídeos crescendo rapidamente"
```

### 3. Processamento Lento
```yaml
alert: SlowVideoProcessing
expr: histogram_quantile(0.95, rate(video_processing_duration_seconds_bucket[5m])) > 300
for: 15m
annotations:
  summary: "Processamento de vídeos muito lento (p95 > 5min)"
```

### 4. Serviço Indisponível
```yaml
alert: VideoProcessorDown
expr: up{job="video-processor"} == 0
for: 1m
annotations:
  summary: "Video Processor está indisponível"
```

---

## 📝 Logs Estruturados (Zerolog)

Os logs são emitidos em formato JSON e podem ser coletados pelo **Grafana Loki**.

### Configuração Loki

```yaml
# promtail-config.yaml
server:
  http_listen_port: 9080

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: video-processor
    static_configs:
      - targets:
          - localhost
        labels:
          job: video-processor
          __path__: /var/log/video-processor/*.log
```

### Queries Loki Úteis

**1. Todos os erros**
```logql
{job="video-processor"} | json | level="error"
```

**2. Processamento de vídeo específico**
```logql
{job="video-processor"} | json | videoID="abc123"
```

**3. Erros de transcodificação**
```logql
{job="video-processor"} | json | message=~"transcodificação falhou"
```

---

## 🔧 Troubleshooting

### Health Check Falhando

**Redis Unavailable**:
```bash
# Verificar se Redis está rodando
docker ps | grep redis

# Testar conexão
redis-cli ping
```

**MinIO Unavailable**:
```bash
# Verificar se MinIO está rodando
docker ps | grep minio

# Testar acesso ao bucket
mc ls myminio/videos
```

### Métricas Não Aparecendo

1. Verificar se o servidor HTTP está rodando:
```bash
curl http://localhost:8080/metrics
```

2. Verificar logs do Prometheus:
```bash
docker logs prometheus
```

3. Verificar configuração de scrape do Prometheus

---

## 📚 Referências

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Grafana Loki Documentation](https://grafana.com/docs/loki/)
- [Zerolog Documentation](https://github.com/rs/zerolog)

---

**Última Atualização**: 2026-01-27
