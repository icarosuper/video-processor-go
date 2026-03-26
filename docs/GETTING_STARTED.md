# Guia de Uso — Video Processor

Este worker não expõe uma API HTTP para submeter vídeos. A interface é a **fila Redis**: você publica um `videoID`, o worker processa e entrega os artefatos no MinIO.

---

## Pré-requisitos

- Go 1.21+
- Docker e Docker Compose
- FFmpeg instalado localmente (para rodar fora do Docker)

```bash
# Verificar se FFmpeg está disponível
ffmpeg -version
ffprobe -version
```

---

## 1. Configuração

```bash
cp .env-example .env
```

O `.env-example` já vem com os valores padrão para desenvolvimento local. Edite se necessário.

---

## 2. Subir a infraestrutura

```bash
docker-compose up -d redis minio
```

Isso sobe:
- **Redis** em `localhost:6379`
- **MinIO** em `localhost:9000` (API) e `localhost:9001` (console web)

---

## 3. Subir o worker

```bash
go run main.go
```

O worker vai:
- Conectar ao Redis e ao MinIO
- Criar o bucket `videos` se não existir
- Aguardar jobs na fila `video_queue`
- Expor `/health` e `/metrics` em `http://localhost:8080`

---

## 4. Fazer upload do vídeo

Antes de publicar um job, o vídeo precisa estar no MinIO no caminho `raw/{videoID}`.

### Via console web (mais fácil)

1. Acesse `http://localhost:9001`
2. Login: `minioadmin` / `minioadmin`
3. Crie o bucket `videos` (se não existir)
4. Faça upload do vídeo em `raw/meu-video`

### Via MinIO CLI (`mc`)

```bash
# Instalar mc (se não tiver)
# https://min.io/docs/minio/linux/reference/minio-mc.html

mc alias set local http://localhost:9000 minioadmin minioadmin
mc cp meu-video.mp4 local/videos/raw/meu-video
```

### Via curl

```bash
curl -X PUT "http://localhost:9000/videos/raw/meu-video" \
  -u minioadmin:minioadmin \
  --upload-file meu-video.mp4
```

---

## 5. Publicar o job

```bash
redis-cli LPUSH video_queue "meu-video"
```

O worker vai detectar o job imediatamente e começar o processamento.

---

## 6. Acompanhar o processamento

### Logs do worker

Os logs aparecem no terminal onde você rodou `go run main.go`:

```
Etapa 1/7: Validando vídeo
Etapa 2/7: Analisando conteúdo
Etapa 3/7: Transcodificando vídeo
...
Vídeo processado com sucesso
```

### Estado do job no Redis

```bash
# Ver estado atual (pending / processing / done / failed)
redis-cli GET job:meu-video
```

Exemplo de resposta quando concluído:
```json
{
  "status": "done",
  "artifacts": {
    "video": "processed/meu-video_processed",
    "thumbnails": "thumbnails/meu-video",
    "audio": "audio/meu-video.mp3",
    "preview": "preview/meu-video_preview.mp4",
    "hls": "hls/meu-video"
  },
  "metadata": {
    "duration": 120.5,
    "width": 1920,
    "height": 1080,
    "video_codec": "h264",
    "audio_codec": "aac",
    "fps": 30,
    "bitrate": 5000000,
    "size": 75000000
  },
  "retry_count": 0,
  "created_at": 1748304000,
  "updated_at": 1748304120
}
```

### Artefatos gerados no MinIO

Acesse `http://localhost:9001` e navegue pelo bucket `videos`:

| Caminho | Conteúdo |
|---|---|
| `processed/meu-video_processed` | MP4 transcodificado (H.264/AAC) |
| `thumbnails/meu-video/` | 5 thumbnails JPG |
| `audio/meu-video.mp3` | Faixa de áudio extraída |
| `preview/meu-video_preview.mp4` | Preview dos primeiros 30s |
| `hls/meu-video/master.m3u8` | Playlist HLS master |
| `hls/meu-video/240p/` | Segmentos HLS 240p |
| `hls/meu-video/360p/` | Segmentos HLS 360p |
| `hls/meu-video/720p/` | Segmentos HLS 720p (se resolução original permitir) |

### Fila de sucesso

Quando o job conclui com sucesso, o ID do vídeo processado é publicado em `video_success_queue`:

```bash
redis-cli BRPOP video_success_queue 10
# Retorna: "meu-video_processed"
```

### Métricas Prometheus

```bash
curl http://localhost:8080/metrics
```

### Health check

```bash
curl http://localhost:8080/health
# Retorna: OK
```

---

## 7. Rodar tudo via Docker Compose

Para rodar o worker também em container (sem precisar de Go local):

```bash
docker-compose up --build
```

> O worker dentro do Docker usa as env vars definidas no `docker-compose.yml`.

---

## Variáveis de ambiente

| Variável | Padrão | Descrição |
|---|---|---|
| `REDIS_HOST` | `localhost:6379` | Endereço do Redis |
| `PROCESSING_REQUEST_QUEUE` | `video_queue` | Fila de entrada de jobs |
| `PROCESSING_FINISHED_QUEUE` | `video_success_queue` | Fila de jobs concluídos |
| `MINIO_ENDPOINT` | `localhost:9000` | Endereço do MinIO |
| `MINIO_ROOT_USER` | `minioadmin` | Usuário MinIO |
| `MINIO_ROOT_PASSWORD` | `minioadmin` | Senha MinIO |
| `MINIO_BUCKET_NAME` | `videos` | Nome do bucket |
| `MINIO_USE_SSL` | `false` | Habilitar SSL no MinIO |
| `HTTP_PORT` | `8080` | Porta do servidor HTTP |
| `WORKER_COUNT` | núcleos da CPU | Número de workers paralelos |
| `MAX_FILE_SIZE_MB` | `5120` (5GB) | Tamanho máximo de arquivo |
| `WEBHOOK_SECRET` | — | Segredo HMAC para assinar webhooks |
| `OTEL_ENDPOINT` | — | Endpoint OTLP para tracing (ex: `localhost:4318`) |
| `OTEL_SERVICE_NAME` | `video-processor` | Nome do serviço nos traces |

---

## Simulando falhas e retries

Para testar o mecanismo de retry, publique um job com um ID que não existe no MinIO:

```bash
redis-cli LPUSH video_queue "video-inexistente"
```

O worker vai tentar 3 vezes e então mover o job para a Dead Letter Queue:

```bash
# Ver jobs na DLQ
redis-cli LRANGE video_queue:dead 0 -1

# Ver estado de erro
redis-cli GET job:video-inexistente
```
