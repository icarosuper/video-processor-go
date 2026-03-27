# 🎥 VidaroProcessor

Sistema distribuído de processamento de vídeos construído em Go, utilizando arquitetura baseada em workers e filas de mensagens.

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen)](docs/TESTING.md)
[![Coverage](https://img.shields.io/badge/coverage-63.7%25-yellow)](docs/TESTING.md)
[![Status](https://img.shields.io/badge/status-bugs%20known-orange)](docs/documentation.md#problemas-conhecidos)

## ✨ Características

- ✅ **Pipeline Completo de Processamento** - 7 etapas com FFmpeg
- ✅ **Arquitetura Distribuída** - Workers concorrentes escaláveis
- ✅ **Logging Estruturado** - Zerolog com output JSON
- ✅ **Métricas Prometheus** - Observabilidade completa
- ✅ **Health Checks** - Kubernetes ready
- ✅ **Processamento Assíncrono** - Filas Redis
- ✅ **Armazenamento S3** - MinIO compatível

## 🚀 Quick Start

### Pré-requisitos

- Go 1.24+
- Docker & Docker Compose
- FFmpeg (para processamento local)

### Instalação

```bash
# Clone o repositório
git clone <repository-url>
cd VidaroProcessor

# Copie o arquivo de ambiente
cp .env-example .env

# Suba as dependências (Redis e MinIO)
docker-compose up -d

# Instale dependências Go
go mod download

# Compile o projeto
go build -o video-processor

# Execute
./video-processor
```

## 📋 Pipeline de Processamento

O sistema processa vídeos através de 7 etapas:

1. **Validação** - Verifica integridade com ffprobe
2. **Análise** - Extrai metadados (duração, resolução, codecs)
3. **Transcodificação** - Converte para MP4 (H.264 + AAC)
4. **Thumbnails** - Gera 5 imagens de preview (320x180)
5. **Áudio** - Extrai track de áudio em MP3
6. **Preview** - Cria versão de baixa qualidade (640px, 30s)
7. **Streaming** - Segmenta para HLS (6s por segmento)

## 🏗️ Arquitetura

```
┌─────────────┐      ┌──────────┐      ┌────────────┐
│  Produtor   │─────▶│  Redis   │─────▶│  Workers   │
└─────────────┘      │  Queue   │      │ (Múltiplos)│
                     └──────────┘      └────────────┘
                                              │
                                              ▼
                                       ┌──────────────┐
                                       │   Pipeline   │
                                       │ (7 etapas)   │
                                       └──────────────┘
                                              │
                                              ▼
                                       ┌──────────────┐
                                       │    MinIO     │
                                       │  (Storage)   │
                                       └──────────────┘
```

### Componentes

- **Workers**: Processam vídeos em paralelo (configurável)
- **Redis**: Fila de mensagens para coordenação
- **MinIO**: Armazenamento de objetos (compatível S3)
- **FFmpeg**: Engine de processamento de vídeo
- **Prometheus**: Coleta de métricas
- **Grafana**: Visualização (opcional)

## 📊 Observabilidade

### Endpoints HTTP (`:8080`)

- **`/health`** - Health check (Redis + MinIO)
- **`/metrics`** - Métricas Prometheus

### Métricas Disponíveis

- `videos_processed_total{status}` - Total de vídeos processados
- `video_processing_duration_seconds` - Tempo de processamento
- `video_processing_step_duration_seconds{step}` - Tempo por etapa
- `active_workers` - Workers ativos
- `queue_size` - Tamanho da fila
- `video_size_bytes` - Distribuição de tamanhos

Ver [OBSERVABILITY.md](docs/OBSERVABILITY.md) para detalhes completos.

## 🧪 Testes

```bash
# Executar todos os testes
go test ./...

# Com cobertura
go test ./... -cover

# Com saída detalhada
go test -v ./...
```

**Cobertura Atual**: 63.7% (processor-steps)

Ver [TESTING.md](docs/TESTING.md) para mais detalhes.

## ⚙️ Configuração

### Variáveis de Ambiente

```bash
# Redis
REDIS_HOST=localhost:6379
PROCESSING_REQUEST_QUEUE=video_queue
PROCESSING_FINISHED_QUEUE=video_success_queue

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_BUCKET_NAME=videos

# Workers (opcional)
WORKER_COUNT=4  # Padrão: número de CPUs
```

Ver [.env-example](./.env-example) para exemplo completo.

## 📦 Estrutura do Projeto

```
VidaroProcessor/
├── config/                 # Configurações
├── internal/
│   └── processor/
│       ├── processor.go           # Orquestrador do pipeline
│       └── processor-steps/       # Etapas do processamento
├── metrics/                # Métricas Prometheus
├── minio/                  # Cliente MinIO
├── queue/                  # Cliente Redis
├── docs/                   # Documentação
│   ├── documentation.md    # Documentação do projeto
│   └── roadmap.md          # Roadmap e melhorias
├── main.go                 # Ponto de entrada
├── OBSERVABILITY.md        # Guia de observabilidade
├── TESTING.md              # Guia de testes
└── docker-compose.yml      # Serviços
```

## 🐳 Docker

### Build

```bash
docker build -t video-processor:latest .
```

### Run

```bash
docker run -d \
  --name video-processor \
  --env-file .env \
  -p 8080:8080 \
  video-processor:latest
```

### Docker Compose

```bash
# Subir todos os serviços
docker-compose up -d

# Ver logs
docker-compose logs -f video-processor

# Parar
docker-compose down
```

## 📖 Documentação

- [📚 Documentação Completa](./docs/documentation.md) - Visão geral do projeto
- [🗺️ Roadmap](./docs/roadmap.md) - Funcionalidades e melhorias
- [📊 Observabilidade](docs/OBSERVABILITY.md) - Métricas e monitoramento
- [🧪 Testes](docs/TESTING.md) - Guia de testes

## 🛣️ Roadmap

### ✅ Implementado

- [x] Pipeline de processamento com FFmpeg (7 etapas)
- [x] Logging estruturado (Zerolog)
- [x] Métricas Prometheus
- [x] Health check endpoint
- [x] Testes unitários (63.7% cobertura)
- [x] Testes de integração (testcontainers)

### Bugs Conhecidos

- Workers travam no shutdown (BLPop sem context)
- Artefatos das etapas 4–7 (thumbnails, HLS, áudio) não chegam ao MinIO
- `docker-compose.yml` com env vars faltando no serviço worker
- Senha MinIO exposta em log no startup
- Fatal se não existir arquivo `.env`

### Próximo

- [ ] Corrigir bugs listados acima
- [ ] Retry com exponential backoff
- [ ] Dead Letter Queue
- [ ] Circuit breaker
- [ ] Dashboard Grafana

Ver [roadmap completo](./docs/roadmap.md).

## 🤝 Contribuindo

1. Fork o projeto
2. Crie uma branch (`git checkout -b feature/amazing-feature`)
3. Commit suas mudanças (`git commit -m 'Add amazing feature'`)
4. Push para a branch (`git push origin feature/amazing-feature`)
5. Abra um Pull Request

## 📄 Licença

Este projeto é fornecido como está, sem garantias.

## 🙏 Agradecimentos

- [FFmpeg](https://ffmpeg.org/) - Processamento de vídeo
- [Zerolog](https://github.com/rs/zerolog) - Logging estruturado
- [Prometheus](https://prometheus.io/) - Métricas
- [MinIO](https://min.io/) - Armazenamento de objetos

---

**Versão**: 0.1.0
**Status**: 🚀 Funcional
**Última Atualização**: 2026-01-27
