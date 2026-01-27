# Video Processor Go - Documentação do Projeto

## 📋 Visão Geral

O **Video Processor Go** é um sistema distribuído de processamento de vídeos construído em Go, utilizando uma arquitetura baseada em workers e filas de mensagens. O sistema foi projetado para processar vídeos de forma assíncrona e escalável, implementando um pipeline de múltiplas etapas que inclui validação, transcodificação, geração de thumbnails, extração de áudio, análise de conteúdo e preparação para streaming.

## 🎯 Objetivo Principal

O objetivo do sistema é receber vídeos brutos, processá-los através de um pipeline de transformações, e armazenar os resultados processados em um serviço de armazenamento de objetos (MinIO). A comunicação entre os componentes é desacoplada através do Redis, permitindo escalabilidade horizontal e resiliência.

## 🏗️ Arquitetura

### Arquitetura de Alto Nível

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

#### 1. **Workers Concurrentes**
- Sistema baseado em múltiplos workers que processam vídeos em paralelo
- Número configurável de workers (padrão: número de CPUs da máquina)
- Cada worker consome mensagens da fila de forma independente
- Shutdown gracioso para encerramento controlado

#### 2. **Pipeline de Processamento**
O pipeline consiste em 7 etapas sequenciais:

1. **Validação** (`validate.go`)
   - Verifica integridade do arquivo de vídeo
   - Valida formato e codecs suportados

2. **Transcodificação** (`transcode.go`)
   - Conversão para formatos padronizados
   - Ajuste de resoluções e bitrates

3. **Geração de Thumbnails** (`thumbnail.go`)
   - Criação de imagens de pré-visualização
   - Múltiplos timestamps do vídeo

4. **Extração de Áudio** (`audio.go`)
   - Separação do track de áudio
   - Conversão para formatos de áudio padronizados

5. **Geração de Preview** (`preview.go`)
   - Criação de versões de baixa resolução
   - Clips curtos para pré-visualização

6. **Análise de Conteúdo** (`analysis.go`)
   - Detecção de características do vídeo
   - Metadados e propriedades técnicas

7. **Segmentação para Streaming** (`streaming.go`)
   - Criação de segmentos HLS/DASH
   - Preparação para streaming adaptativo

#### 3. **Sistema de Filas (Redis)**
- **Fila de Requisições**: `PROCESSING_REQUEST_QUEUE`
  - Recebe IDs de vídeos para processamento
  - Consumo via BLPop (blocking pop)

- **Fila de Sucesso**: `PROCESSING_FINISHED_QUEUE`
  - Recebe IDs de vídeos processados com sucesso
  - Publicação via LPush

#### 4. **Armazenamento (MinIO)**
- **Buckets**: Organização por tipo de conteúdo
  - `videos/`: Bucket principal
  - Prefixos `raw/`: Vídeos originais
  - Prefixos `processed/`: Vídeos processados

## 🛠️ Stack Tecnológica

### Linguagem e Runtime
- **Go 1.24.5**: Linguagem principal
- **Context**: Uso de context para timeout e cancelamento

### Dependências Principais
```go
require (
    github.com/caarlos0/env/v10 v10.0.0          // Parsing de environment variables
    github.com/go-redis/redis/v8 v8.11.5         // Cliente Redis
    github.com/joho/godotenv v1.5.1              // Carregamento de .env
    github.com/minio/minio-go/v7 v7.0.95         // Cliente MinIO
)
```

### Infraestrutura
- **Redis 7**: Sistema de filas e mensageria
- **MinIO**: Armazenamento de objetos compatível com S3
- **FFmpeg**: Ferramenta de processamento de vídeo (instalada no container)
- **Docker & Docker Compose**: Orquestração de containers
- **Alpine Linux**: Imagem base leve para produção

### Padrões de Design
- **Pipeline Pattern**: Sequência ordenada de etapas de processamento
- **Worker Pool**: Múltiplos workers consumindo da mesma fila
- **Configuration Pattern**: Configurações centralizadas
- **Graceful Shutdown**: Encerramento controlado de recursos

## 📂 Estrutura do Projeto

```
video-processor-go/
├── config/
│   └── config.go                    # Gerenciamento de configurações
├── internal/
│   └── processor/
│       ├── processor.go             # Orquestrador do pipeline
│       └── processor-steps/        # Etapas do processamento
│           ├── analysis.go          # Análise de conteúdo
│           ├── audio.go             # Extração de áudio
│           ├── preview.go           # Geração de pré-visualização
│           ├── streaming.go         # Segmentação para streaming
│           ├── thumbnail.go         # Geração de thumbnails
│           ├── transcode.go         # Transcodificação
│           └── validate.go          # Validação de vídeo
├── minio/
│   └── client.go                    # Cliente MinIO
├── queue/
│   └── client.go                    # Cliente Redis
├── main.go                          # Ponto de entrada
├── docker-compose.yml               # Orquestração de serviços
├── Dockerfile                       # Configuração do container
├── go.mod                           # Dependências Go
├── go.sum                           # Lock de dependências
├── .env-example                     # Exemplo de variáveis de ambiente
└── .gitignore                       # Arquivos ignorados pelo git
```

## ⚙️ Configuração

### Variáveis de Ambiente

#### Redis (Obrigatório)
```bash
REDIS_HOST=localhost:6379                    # Host e porta do Redis
PROCESSING_REQUEST_QUEUE=video_queue         # Nome da fila de requisições
PROCESSING_FINISHED_QUEUE=video_success_queue # Nome da fila de sucesso
```

#### MinIO (Obrigatório)
```bash
MINIO_ENDPOINT=localhost:9000                 # Endpoint do MinIO
MINIO_ROOT_USER=minioadmin                    # Usuário administrador
MINIO_ROOT_PASSWORD=minioadmin                # Senha administrador
MINIO_BUCKET_NAME=videos                      # Nome do bucket
```

#### Workers (Opcional)
```bash
WORKER_COUNT=4                                # Número de workers (padrão: número de CPUs)
```

### Configurações com Defaults

- **SSL MinIO**: Desativado (`useSSL = false`)
- **Arquivos Temporários**: Armazenados em `/tmp`
- **Timeout de Context**: Sem limite definido

## 🚀 Como Executar

### Desenvolvimento

```bash
# 1. Clone o repositório
git clone <repository-url>
cd video-processor-go

# 2. Copie o arquivo de ambiente
cp .env-example .env

# 3. Edite as variáveis de ambiente conforme necessário
nano .env

# 4. Suba os serviços (Redis e MinIO)
docker-compose up -d

# 5. Instale as dependências
go mod download

# 6. Execute o projeto
go run main.go
```

### Produção (Docker)

```bash
# 1. Build da imagem
docker build -t video-processor:latest .

# 2. Execute o container
docker run -d \
  --name video-processor \
  --env-file .env \
  video-processor:latest
```

## 🔄 Fluxo de Processamento

### 1. Ingestão
```
Produtor → Redis (PROCESSING_REQUEST_QUEUE) → {VideoID}
```

### 2. Consumo
```
Worker → BLPop(PROCESSING_REQUEST_QUEUE) → VideoID
```

### 3. Download
```
Worker → MinIO → Download vídeo original (raw/)
```

### 4. Pipeline de Processamento
```
Worker → Validate → Transcode → Thumbnail → Audio → Preview → Analysis → Streaming
```

### 5. Upload
```
Worker → MinIO → Upload processado (processed/)
```

### 6. Notificação de Sucesso
```
Worker → Redis (PROCESSING_FINISHED_QUEUE) → {VideoID}
```

## 📊 Estado Atual do Projeto

### ✅ Funcionalidades Implementadas

- [x] Arquitetura de workers concorrentes
- [x] Integração com MinIO (download/upload)
- [x] Sistema de filas com Redis
- [x] Gerenciamento de configurações
- [x] Shutdown gracioso
- [x] Dockerização completa
- [x] Pipeline orquestrado
- [x] Tratamento de erros estruturado

### ⚠️ Funcionalidades Incompletas

**ATENÇÃO**: As etapas de processamento de vídeo são atualmente **placeholders**. O sistema baixa e faz upload dos vídeos, mas **não realiza processamento real**.

- [ ] Validação de vídeo (placeholder apenas)
- [ ] Transcodificação com FFmpeg
- [ ] Geração de thumbnails
- [ ] Extração de áudio
- [ ] Geração de preview
- [ ] Análise de conteúdo
- [ ] Segmentação para streaming (HLS/DASH)

## 🔮 Possíveis Evoluções e Melhorias

### Curto Prazo (Essencial para Funcionalidade Básica)

1. **Implementação das Etapas do Pipeline**
   - Integração real com FFmpeg
   - Validação de formatos suportados
   - Geração de thumbnails com múltiplos timestamps
   - Extração de áudio em diferentes formatos

2. **Correções Críticas**
   - Bug no publish da fila de sucesso (main.go:102)
   - Tratamento adequado de arquivos temporários
   - Ativação de SSL em produção

3. **Observabilidade**
   - Logging estruturado (JSON)
   - Métricas básicas (tempo de processamento, taxa de sucesso/falha)
   - Health checks endpoint

### Médio Prazo (Melhorias de Arquitetura)

1. **Resiliência**
   - Mecanismo de retry com exponential backoff
   - Circuit breaker para chamadas externas
   - Dead letter queue para falhas
   - Timeout por etapa do pipeline

2. **Monitoramento**
   - Integração com Prometheus
   - Dashboard com Grafana
   - Alertas para falhas críticas
   - Tracing distribuído (OpenTelemetry)

3. **Performance**
   - Pool de conexões Redis
   - Upload/download com multipart para MinIO
   - Compressão de arquivos processados
   - Cache de metadados

4. **Segurança**
   - Validação rigorosa de inputs
   - Rate limiting
   - Auditoria de logs
   - Secrets management (Vault/Kubernetes Secrets)

### Longo Prazo (Escalabilidade e Features Avançadas)

1. **Multi-tenant**
   - Suporte a múltiplos buckets
   - Isolamento por tenant
   - Configurações específicas por cliente

2. **Processamento Distribuído**
   - Suporte a processamento em múltiplas máquinas
   - Coordenação via etcd ou similar
   - Auto-scaling baseado em fila

3. **Features Avançadas**
   - Suporte a vídeos 360°
   - Processamento de vídeos ao vivo (real-time)
   - Reconhecimento de objetos/cenas (IA)
   - Transcrição automática (speech-to-text)
   - Detecção de conteúdo impróprio

4. **API e Integrações**
   - API REST para gerenciamento
   - Webhooks para notificações
   - Integração com CDNs
   - SDK para múltiplas linguagens

5. **Arquitetura de Eventos**
   - Event-driven architecture completa
   - Suporte a múltiplos event brokers (Kafka, RabbitMQ)
   - Event sourcing para auditoria
   - CQRS para leituras otimizadas

## 🔗 Pontos de Integração

### APIs Externas
- **MinIO S3 API**: Compatível com AWS S3
- **Redis**: Protocolo Redis padrão

### Serviços Dependentes
- **Redis**: Obrigatório para filas
- **MinIO**: Obrigatório para armazenamento
- **FFmpeg**: Obrigatório para processamento de vídeo

## 📝 Notas Importantes

### Performance Considerations
- Número de workers deve ser calibrado based em CPU e I/O
- Processamento de vídeo é intensivo em recursos
- Considere usar máquinas com GPU para transcodificação
- Armazenamento temporário deve ser rápido (SSD recomendado)

### Security Considerations
- SSL/TLS deve ser ativado em produção
- Credenciais não devem estar em código
- Validação de arquivos é essencial para evitar malware
- Rate limiting para evitar DoS

### Operational Considerations
- Logs devem ser centralizados (ELK, Loki, etc)
- Métricas são essenciais para operação
- Backup do Redis e MinIO são necessários
- Documentação de runbooks é recomendada

## 📚 Recursos Adicionais

### Documentação Relacionada
- [ROADMAP.md](./ROADMAP.md): Detalhes do que falta implementar
- [Go Documentation](https://golang.org/doc/)
- [FFmpeg Documentation](https://ffmpeg.org/documentation.html)
- [MinIO Documentation](https://min.io/docs/minio/linux/index.html)
- [Redis Documentation](https://redis.io/docs/)

### Comunidade e Suporte
- Repositório do projeto
- Issues e Pull Requests
- Documentação de API (quando disponível)

---

**Última Atualização**: 2026-01-27
**Versão**: 0.1.0 (Development)
**Status**: 🚧 Em Desenvolvimento - Pipeline Incompleto
