# Video Processor Go - Documentação do Projeto

## Visão Geral

O **Video Processor Go** é um sistema distribuído de processamento de vídeos construído em Go, utilizando uma arquitetura baseada em workers e filas de mensagens. O sistema processa vídeos de forma assíncrona e escalável através de um pipeline de 7 etapas com FFmpeg.

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
video-processor-go/
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

> **Atenção**: o serviço `worker` no `docker-compose.yml` atual está sem as variáveis obrigatórias `PROCESSING_REQUEST_QUEUE`, `PROCESSING_FINISHED_QUEUE` e `MINIO_BUCKET_NAME`. Ver [Problemas Conhecidos](#problemas-conhecidos).

## Fluxo de Processamento

```
1. Produtor publica VideoID → PROCESSING_REQUEST_QUEUE
2. Worker consome via BLPop
3. Worker baixa raw/{VideoID} do MinIO
4. Pipeline executa as 7 etapas
5. Worker faz upload processed/{VideoID}_processed para MinIO
6. Worker publica VideoID → PROCESSING_FINISHED_QUEUE
```

## Problemas Conhecidos

Estes são bugs identificados que ainda não foram corrigidos.

### 1. Workers travam no shutdown

`queue.ConsumeMessage()` usa `BLPop` com um `context.Background()` global, ignorando o contexto de cancelamento passado pelos workers. Ao receber SIGTERM, os workers ficam bloqueados no BLPop até a próxima mensagem chegar ou o timeout de 30s forçar o encerramento.

**Arquivo**: `queue/client.go:35`
**Impacto**: Shutdown não é verdadeiramente gracioso.

### 2. Artefatos das etapas 4–7 são descartados

Thumbnails, áudio extraído, preview e segmentos HLS são gerados em `tempDir`, mas o `defer os.RemoveAll(tempDir)` apaga tudo antes de qualquer upload. Nenhum desses artefatos chega ao MinIO.

**Arquivo**: `internal/processor/processor.go:26`
**Impacto**: As etapas 4, 5, 6 e 7 do pipeline não têm efeito real.

### 3. `docker-compose.yml` — worker falha na inicialização

O serviço `worker` não define `PROCESSING_REQUEST_QUEUE`, `PROCESSING_FINISHED_QUEUE` e `MINIO_BUCKET_NAME`, que são marcadas como `notEmpty` no config. O container falha ao subir.

**Arquivo**: `docker-compose.yml`
**Impacto**: `docker-compose up` não funciona end-to-end.

### 4. Senha MinIO exposta em log

`config.LoadConfig()` usa `fmt.Printf("Config loaded successfully: %+v\n", cfg)`, que imprime `MinioRootPassword` em texto puro no stdout.

**Arquivo**: `config/config.go:60`
**Impacto**: Vazamento de credenciais em logs.

### 5. `godotenv.Load()` com Fatal

Se não existir arquivo `.env` (produção com env vars injetadas), o processo morre com `unable to load .env file`. O erro deveria ser ignorado quando o arquivo não existir.

**Arquivo**: `config/config.go:49`
**Impacto**: Impossível rodar em Docker/Kubernetes sem um `.env` em disco.

## Considerações Operacionais

- Processamento de vídeo é CPU-intensivo; calibrar `WORKER_COUNT` conforme hardware
- Armazenamento temporário em `/tmp`; recomendado SSD para performance
- SSL/TLS no MinIO está hardcoded como `false`; deve ser configurável para produção
- Logs em formato ConsoleWriter (não JSON) por padrão — trocar para JSON em produção

---

**Versão**: 0.1.0
**Status**: Funcional (pipeline básico) — com bugs conhecidos documentados acima
**Última Atualização**: 2026-03-25
