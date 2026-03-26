# CLAUDE.md — Video Processor Go

## O que é este projeto

Worker em Go que consome IDs de vídeos de uma fila Redis, baixa do MinIO, processa com FFmpeg em um pipeline de 7 etapas, e faz upload do resultado de volta ao MinIO.

## Estrutura essencial

```
main.go                          # worker pool + graceful shutdown + HTTP server
config/config.go                 # env vars via caarlos0/env
queue/client.go                  # Redis BLPop/LPush
minio/client.go                  # download/upload de vídeos
metrics/metrics.go               # métricas Prometheus (promauto)
internal/processor/processor.go  # orquestrador das 7 etapas
internal/processor/processor-steps/*.go  # cada etapa do pipeline
```

## Bugs conhecidos (não corrigidos)

Antes de sugerir features, verificar se os bugs abaixo foram resolvidos:

1. **Workers travam no shutdown** — `queue/client.go:35`: `BLPop` usa `context.Background()` global, ignorando o contexto cancelável do worker. Solução: `ConsumeMessage(ctx context.Context)`.

2. **Artefatos não chegam ao MinIO** — `internal/processor/processor.go:26`: `defer os.RemoveAll(tempDir)` apaga thumbnails, áudio, preview e HLS antes de qualquer upload. Apenas o vídeo transcodificado (etapa 3) é enviado.

3. **`docker-compose.yml` quebrado** — serviço `worker` não tem `PROCESSING_REQUEST_QUEUE`, `PROCESSING_FINISHED_QUEUE`, `MINIO_BUCKET_NAME`.

4. **Senha em log** — `config/config.go:60`: `fmt.Printf` printa `MinioRootPassword` em plaintext.

5. **Fatal sem `.env`** — `config/config.go:49`: `godotenv.Load()` faz `log.Fatal` se o arquivo não existir, impedindo deploy sem `.env` em disco.

## Métricas com limitações

- `active_workers`: valor estático definido no startup, nunca atualizado
- `queue_size`: nunca é populada
- `video_size_bytes`: nunca é registrada

## Convenções do projeto

- Logs em português (`"Iniciando video-processor"`, `"Etapa 1/7: Validando vídeo"`)
- Erros wrappados com `fmt.Errorf("contexto: %w", err)`
- Etapas não-críticas (thumbnails, áudio, preview, HLS) usam `log.Warn` e não retornam erro — o pipeline continua mesmo se falharem
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
