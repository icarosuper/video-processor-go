# CLAUDE.md — Video Processor Go

## O que é este projeto

Worker assíncrono em Go que uma API chama para processar vídeos enviados por usuários (modelo YouTube). Consome IDs de vídeos de uma fila Redis, baixa do MinIO, processa com FFmpeg em um pipeline de 7 etapas, e faz upload dos artefatos de volta ao MinIO.

## Estrutura essencial

```
main.go                          # worker pool + graceful shutdown + HTTP server
config/config.go                 # env vars via caarlos0/env
queue/client.go                  # Redis BRPopLPush (consumo atômico) + recovery de órfãos
queue/job.go                     # estado do job (pending→processing→done/failed), retry, DLQ
internal/webhook/webhook.go      # notificação POST ao callbackURL com retry e HMAC opcional
internal/circuitbreaker/circuitbreaker.go  # circuit breakers para MinIO e Redis
internal/telemetry/telemetry.go            # OpenTelemetry: init, tracer, shutdown
minio/client.go                  # download/upload de vídeos e artefatos
metrics/metrics.go               # métricas Prometheus (promauto)
internal/processor/processor.go  # orquestrador das 7 etapas, retorna ProcessingResult
internal/processor/processor-steps/*.go  # cada etapa do pipeline
```

## Estado atual (~95% pronto para produção)

O pipeline FFmpeg funciona end-to-end com confiabilidade de jobs (retry, DLQ, recovery de órfãos), webhook de notificação, metadados persistidos, métricas operacionais reais, HLS adaptativo com múltiplas resoluções, circuit breakers para MinIO/Redis e timeouts individuais por etapa. Itens restantes: escalabilidade (auto-scaling, múltiplas instâncias, priorização de fila) e Dashboard Grafana.

Ver `docs/roadmap.md` para o plano completo.

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
