# CLAUDE.md — Video Processor Go

## O que é este projeto

Worker assíncrono em Go que uma API chama para processar vídeos enviados por usuários (modelo YouTube). Consome IDs de vídeos de uma fila Redis, baixa do MinIO, processa com FFmpeg em um pipeline de 7 etapas, e faz upload dos artefatos de volta ao MinIO.

## Estrutura essencial

```
main.go                          # worker pool + graceful shutdown + HTTP server
config/config.go                 # env vars via caarlos0/env
queue/client.go                  # Redis BLPop/LPush
minio/client.go                  # download/upload de vídeos e artefatos
metrics/metrics.go               # métricas Prometheus (promauto)
internal/processor/processor.go  # orquestrador das 7 etapas, retorna ProcessingResult
internal/processor/processor-steps/*.go  # cada etapa do pipeline
```

## Estado atual (~30% pronto para produção)

O pipeline FFmpeg funciona end-to-end. Os bloqueadores críticos para uso real são:

1. **Jobs órfãos** — se o worker travar antes de `AcknowledgeMessage`, o job fica preso em `{queue}:processing`. Falta goroutine de recuperação.

2. **Sem notificação push** — a API precisa de polling em `queue.GetJobState(videoID)`. Falta webhook/callback.

3. **Falhas silenciosas** — quando o pipeline falha, nada é publicado. A API nunca fica sabendo.

4. **Resolução única** — gera um MP4 só. Streaming adaptativo real precisa de 360p/480p/720p/1080p.

Ver `docs/roadmap.md` para o plano completo.

## Métricas com limitações conhecidas

- `active_workers`: valor estático definido no startup, nunca atualizado
- `queue_size`: nunca é populada
- `video_size_bytes`: nunca é registrada

## Convenções do projeto

- Logs em português (`"Iniciando video-processor"`, `"Etapa 1/7: Validando vídeo"`)
- Erros wrappados com `fmt.Errorf("contexto: %w", err)`
- Etapas não-críticas (thumbnails, áudio, preview, HLS) usam `log.Warn` e não retornam erro — o pipeline continua mesmo se falharem; os caminhos só são setados em `ProcessingResult` quando a etapa tem sucesso
- Etapas críticas (validação, transcodificação) retornam erro e abortam o pipeline
- Variáveis de ambiente obrigatórias com tag `notEmpty` via caarlos0/env
- Testes de processamento pulam automaticamente se FFmpeg não estiver disponível (`GenerateTestVideo` em `test_helpers.go`)

## Rodando localmente

```bash
cp .env-example .env
docker-compose up -d redis minio
go run main.go
```

## Rodando testes

```bash
go test ./...                                      # unitários
go test -v ./test/integration/... -timeout 10m    # integração (requer Docker)
```
