# VidaroProcessor - Documentação do Projeto

## Visão Geral

O **VidaroProcessor** é um sistema distribuído de processamento de vídeos construído em Go, utilizando uma arquitetura baseada em workers e filas de mensagens. O sistema processa vídeos de forma assíncrona e escalável através de um pipeline de 7 etapas com FFmpeg.

## Objetivo Principal

Receber vídeos brutos, processá-los através de um pipeline de transformações (validação, transcodificação, thumbnails, áudio, preview e HLS), e armazenar os resultados no MinIO. A comunicação é desacoplada via Redis, permitindo escalabilidade horizontal.

## Arquitetura

```
┌─────────────────┐      ┌──────────────┐      ┌──────────────┐
│   Produtor      │──────│   Redis      │──────│   Workers    │
│  de Vídeos      │      │    Filas     │      │  (Múltiplos) │
└─────────────────┘      └──────────────┘      └──────────────┘
                                                        │
                                                        ▼
                                                 ┌──────────────┐
                                                 │  Pipeline de  │
                                                 │ Processamento│
                                                 └──────────────┘
                                                        │
                                                        ▼
                                                 ┌──────────────┐
                                                 │    MinIO     │
                                                 │  (Resultado) │
                                                 └──────────────┘
```

### Componentes Principais

#### 1. Workers Concorrentes
- Múltiplos workers processando vídeos em paralelo
- Número configurável via `WORKER_COUNT` (padrão: número de CPUs)
- Graceful shutdown com timeout de 30 segundos

#### 2. Pipeline de Processamento

7 etapas sequenciais em `internal/processor/processor-steps/`:

| Etapa | Arquivo | Obrigatória | Saída |
|---|---|---|---|
| 1. Validação | `validate.go` | Sim | — |
| 2. Análise | `analysis.go` | Não (informativa) | metadados no log |
| 3. Transcodificação | `transcode.go` | Sim | `*_output.mp4` |
| 4. Thumbnails | `thumbnail.go` | Não | `thumbnails/thumb_00N.jpg` |
| 5. Extração de Áudio | `audio.go` | Não | `audio.mp3` |
| 6. Preview | `preview.go` | Não | `preview.mp4` |
| 7. Streaming HLS | `streaming.go` | Não | `streaming/*.ts` + `playlist.m3u8` |

> **Atenção**: atualmente apenas o vídeo transcodificado (etapa 3) é enviado ao MinIO. Os artefatos das etapas 4–7 são gerados em `tempDir` e descartados ao final. Ver [Problemas Conhecidos](#problemas-conhecidos).

#### 3. Sistema de Filas (Redis)

- **`PROCESSING_REQUEST_QUEUE`**: recebe IDs de vídeos para processar (BLPop)
- **`PROCESSING_FINISHED_QUEUE`**: recebe IDs de vídeos processados com sucesso (LPush)

#### 4. Armazenamento (MinIO)

- Prefixo `raw/`: vídeos originais
- Prefixo `processed/`: vídeos transcodificados

## Stack Tecnológica

| Componente | Tecnologia |
|---|---|
| Linguagem | Go 1.24 |
| Filas | Redis 7 (go-redis/v8) |
| Armazenamento | MinIO (minio-go/v7) |
| Processamento | FFmpeg / FFprobe |
| Métricas | Prometheus (promauto) |
| Logging | Zerolog |
| Config | caarlos0/env v10 + godotenv |
| Containers | Docker + Docker Compose |

## Estrutura do Projeto

```
VidaroProcessor/
├── config/
│   └── config.go                    # Gerenciamento de configurações
├── internal/
│   └── processor/
│       ├── processor.go             # Orquestrador do pipeline
│       └── processor-steps/        # Etapas do processamento
│           ├── analysis.go
│           ├── audio.go
│           ├── preview.go
│           ├── streaming.go
│           ├── thumbnail.go
│           ├── transcode.go
│           ├── validate.go
│           ├── test_helpers.go      # Helpers para testes
│           └── testdata/
├── metrics/
│   └── metrics.go                   # Métricas Prometheus
├── minio/
│   └── client.go                    # Cliente MinIO
├── queue/
│   └── client.go                    # Cliente Redis
├── test/
│   └── integration/                 # Testes de integração (testcontainers)
├── docs/
├── main.go
├── docker-compose.yml
└── Dockerfile
```

## Configuração

### Variáveis de Ambiente

| Variável | Obrigatória | Descrição |
|---|---|---|
| `REDIS_HOST` | Sim | Ex: `localhost:6379` |
| `PROCESSING_REQUEST_QUEUE` | Sim | Nome da fila de entrada |
| `PROCESSING_FINISHED_QUEUE` | Sim | Nome da fila de sucesso |
| `MINIO_ENDPOINT` | Sim | Ex: `localhost:9000` |
| `MINIO_ROOT_USER` | Sim | Usuário MinIO |
| `MINIO_ROOT_PASSWORD` | Sim | Senha MinIO |
| `MINIO_BUCKET_NAME` | Sim | Nome do bucket |
| `WORKER_COUNT` | Não | Padrão: `runtime.NumCPU()` |

### Observação sobre `.env`

O `config.LoadConfig()` chama `godotenv.Load()` com `log.Fatal` caso o arquivo `.env` não exista. Em ambientes Docker/Kubernetes onde as variáveis são injetadas diretamente, isso causa falha na inicialização. Ver [Problemas Conhecidos](#problemas-conhecidos).

## Como Executar

### Desenvolvimento

```bash
cp .env-example .env
# editar .env com suas configurações

docker-compose up -d redis minio

go mod download
go run main.go
```

### Produção (Docker Compose)

```bash
docker-compose up -d
```

## Fluxo de Processamento

```
1. Produtor publica VideoID → PROCESSING_REQUEST_QUEUE
2. Worker consome via BLPop
3. Worker baixa raw/{VideoID} do MinIO
4. Pipeline executa as 7 etapas
5. Worker faz upload de todos os artefatos ao MinIO:
   - processed/{VideoID}_processed       (vídeo transcodificado)
   - thumbnails/{VideoID}/thumb_00N.jpg  (5 frames)
   - audio/{VideoID}.mp3
   - preview/{VideoID}_preview.mp4
   - hls/{VideoID}/playlist.m3u8 + segment_*.ts
6. Worker publica VideoID → PROCESSING_FINISHED_QUEUE
```

## Limitações Atuais

O sistema está funcional para o fluxo básico, mas ainda não está pronto para produção. Os bloqueadores principais são:

- **Sem estado de job**: a API não sabe se um vídeo está processando, terminou ou falhou
- **BLPop destrutivo**: crash durante o processamento = job perdido
- **Resolução única**: gera um MP4 só; streaming adaptativo precisa de múltiplas qualidades
- **Falhas silenciosas**: jobs que falham não notificam a API

Ver o plano completo em [roadmap.md](./roadmap.md).

## Considerações Operacionais

- Processamento de vídeo é CPU-intensivo; calibrar `WORKER_COUNT` conforme hardware
- Armazenamento temporário em `/tmp`; recomendado SSD para performance
- SSL/TLS no MinIO está hardcoded como `false`; deve ser configurável para produção
- Logs em formato ConsoleWriter (não JSON) por padrão — trocar para JSON em produção

---

**Versão**: 0.1.0
**Status**: Pipeline funcional — bloqueadores de produção documentados no roadmap
**Última Atualização**: 2026-03-26
