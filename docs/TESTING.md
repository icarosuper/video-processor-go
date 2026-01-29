# 🧪 Testes - Video Processor Go

Este documento descreve a suíte de testes do projeto e como executá-los.

## 📊 Cobertura de Testes

### Atual
- **processor-steps**: 63.7% de cobertura
- **metrics**: 100% (métricas testadas)
- **integration**: Testes de integração com testcontainers

### Pacotes com Testes
- ✅ `internal/processor/processor-steps` - Etapas do pipeline (unitários)
- ✅ `metrics` - Métricas Prometheus (unitários)
- ✅ `test/integration` - Testes de integração (Redis, MinIO, Pipeline)

### Pacotes sem Testes (Futuro)
- ⏳ `main` - Lógica principal
- ⏳ `config` - Configurações

---

## 🚀 Executando os Testes

### Todos os Testes
```bash
go test ./...
```

### Testes com Saída Detalhada
```bash
go test -v ./...
```

### Testes com Cobertura
```bash
go test ./... -cover
```

### Relatório de Cobertura HTML
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Testes de um Pacote Específico
```bash
# Apenas as etapas do processador
go test -v ./internal/processor/processor-steps/...

# Apenas métricas
go test -v ./metrics/...

# Apenas testes de integração
go test -v ./test/integration/...
```

### Testes de Integração
```bash
# Executar testes de integração (requer Docker)
go test -v ./test/integration/... -timeout 10m

# Pular testes de integração
SKIP_INTEGRATION_TESTS=true go test ./...
```

**Nota**: Os testes de integração são **pulados automaticamente** se o Docker não estiver disponível.

---

## 📝 Testes Implementados

### 1. Validação de Vídeo (`validate_test.go`)

**Testes**:
- ✅ `TestValidateVideo_ValidVideo` - Valida vídeo correto
- ✅ `TestValidateVideo_InvalidVideo` - Falha com arquivo inválido
- ✅ `TestValidateVideo_NonExistentFile` - Falha com arquivo inexistente
- ✅ `TestValidateVideo_EmptyFile` - Falha com arquivo vazio

**Cobertura**: Casos de sucesso e erro

---

### 2. Transcodificação (`transcode_test.go`)

**Testes**:
- ✅ `TestTranscodeVideo_ValidVideo` - Transcodifica vídeo válido
- ✅ `TestTranscodeVideo_InvalidInput` - Falha com entrada inválida
- ✅ `TestTranscodeVideo_NonExistentInput` - Falha com arquivo inexistente

**Validações**:
- Arquivo de saída criado
- Arquivo de saída não está vazio
- Arquivo de saída é um vídeo válido

---

### 3. Geração de Thumbnails (`thumbnail_test.go`)

**Testes**:
- ✅ `TestGenerateThumbnails_ValidVideo` - Gera 5 thumbnails
- ✅ `TestGenerateThumbnails_InvalidVideo` - Falha com vídeo inválido
- ✅ `TestGenerateThumbnails_NonExistentVideo` - Falha com arquivo inexistente

**Validações**:
- Diretório de saída criado
- 5 thumbnails gerados (thumb_001.jpg até thumb_005.jpg)
- Cada thumbnail tem tamanho > 0

---

### 4. Análise de Conteúdo (`analysis_test.go`)

**Testes**:
- ✅ `TestAnalyzeContent_ValidVideo` - Analisa vídeo válido
- ✅ `TestAnalyzeContent_InvalidVideo` - Falha com vídeo inválido
- ✅ `TestAnalyzeContent_NonExistentVideo` - Falha com arquivo inexistente

**Validações**:
- Extração de metadados sem erro
- Logs estruturados com zerolog

---

### 5. Métricas Prometheus (`metrics_test.go`)

**Testes**:
- ✅ `TestVideosProcessedTotal_Increment` - Contador de vídeos
- ✅ `TestProcessingDuration_Observe` - Histograma de duração
- ✅ `TestProcessingStepDuration_MultipleSteps` - Duração por etapa
- ✅ `TestActiveWorkers_SetAndGet` - Gauge de workers
- ✅ `TestQueueSize_SetAndGet` - Gauge de fila

**Validações**:
- Contadores incrementam corretamente
- Histogramas registram observações
- Gauges atualizam valores

---

## 🔗 Testes de Integração (`test/integration/`)

### Requisitos
- **Docker** instalado e rodando
- **testcontainers-go** para gerenciar containers

### Arquivos de Teste

#### `setup_test.go`
Configuração dos containers de teste (Redis e MinIO).

**Funções**:
- `SetupContainers()` - Inicia Redis e MinIO containers
- `TeardownContainers()` - Para e remove containers
- `isDockerAvailable()` - Verifica se Docker está disponível

---

#### `queue_test.go` - Testes do Redis

**Testes**:
- ✅ `TestRedisQueue_PublishAndConsume` - Publica e consome mensagem
- ✅ `TestRedisQueue_MultipleMessages` - Múltiplas mensagens em FIFO
- ✅ `TestRedisQueue_EmptyQueue` - Timeout em fila vazia
- ✅ `TestRedisQueue_SuccessQueue` - Workflow completo de filas

---

#### `minio_test.go` - Testes do MinIO

**Testes**:
- ✅ `TestMinIO_BucketOperations` - Criação e listagem de buckets
- ✅ `TestMinIO_ObjectUploadDownload` - Upload e download de objetos
- ✅ `TestMinIO_VideoWorkflow` - Workflow raw → processed
- ✅ `TestMinIO_DownloadToFile` - Download para arquivo local
- ✅ `TestMinIO_NonExistentObject` - Erro em objeto inexistente

---

#### `pipeline_test.go` - Testes do Pipeline Completo

**Testes**:
- ✅ `TestPipeline_ValidateStep` - Validação com MinIO
- ✅ `TestPipeline_TranscodeStep` - Transcodificação end-to-end
- ✅ `TestPipeline_FullWorkflow` - Workflow completo (Redis → MinIO → FFmpeg → MinIO)
- ✅ `TestPipeline_ThumbnailGeneration` - Geração e upload de thumbnails

**`TestPipeline_FullWorkflow` simula**:
1. Adiciona video à fila de requisições
2. Consome da fila (como worker)
3. Baixa vídeo do MinIO
4. Valida, analisa e transcodifica
5. Faz upload do vídeo processado
6. Publica na fila de sucesso
7. Verifica resultado

---

## 🛠️ Helpers de Teste

### `test_helpers.go`

#### `GenerateTestVideo(t *testing.T, duration int) string`
Gera um vídeo de teste usando FFmpeg.

**Características**:
- Duração: 5 segundos
- Resolução: 640x480
- FPS: 30
- Codec vídeo: H.264
- Codec áudio: AAC
- Padrão visual: testsrc
- Áudio: sine wave (1000Hz)

**Uso**:
```go
func TestMyFunction(t *testing.T) {
    videoPath := GenerateTestVideo(t, 5)
    // videoPath é um vídeo válido de teste
}
```

**Nota**: Pula o teste se FFmpeg não estiver disponível.

---

#### `CreateInvalidFile(t *testing.T) string`
Cria um arquivo inválido (não é um vídeo) para testes de erro.

**Uso**:
```go
func TestInvalidInput(t *testing.T) {
    invalidPath := CreateInvalidFile(t)
    // invalidPath é um arquivo inválido
}
```

---

## ⚙️ Requisitos

### FFmpeg (Obrigatório)
Os testes de processamento de vídeo **requerem FFmpeg** instalado:

```bash
# Ubuntu/Debian
sudo apt-get install ffmpeg

# macOS
brew install ffmpeg

# Windows
# Baixe de https://ffmpeg.org/download.html
```

**Se FFmpeg não estiver disponível**, os testes serão **pulados automaticamente** com mensagem:
```
FFmpeg não está disponível - pulando teste
```

---

## 📈 Melhorias Futuras

### Testes a Adicionar

1. **Testes de Integração**
   - Pipeline completo end-to-end
   - Integração com Redis (testcontainers)
   - Integração com MinIO (testcontainers)

2. **Testes de Performance**
   - Benchmark de transcodificação
   - Benchmark de geração de thumbnails
   - Teste de carga com múltiplos vídeos

3. **Testes de Edge Cases**
   - Vídeos muito grandes (> 1GB)
   - Vídeos muito curtos (< 1 segundo)
   - Vídeos sem áudio
   - Vídeos corrompidos parcialmente

4. **Testes de Unidades Restantes**
   - `config.LoadConfig()`
   - `queue.ConsumeMessage()`
   - `minio.DownloadVideo()`
   - `minio.UploadVideo()`
   - `main.processNextMessage()`

---

## 🔍 Troubleshooting

### Testes Falhando

**Erro: "FFmpeg não está disponível"**
```bash
# Instale FFmpeg
sudo apt-get install ffmpeg

# Verifique instalação
ffmpeg -version
```

**Erro: "Timeout"**
```bash
# Aumente o timeout (padrão: 5 minutos)
go test ./... -timeout 10m
```

**Erro: "Permission denied"**
```bash
# Verifique permissões do diretório temporário
ls -la /tmp
```

---

## 📊 CI/CD

### GitHub Actions (Exemplo)

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Install FFmpeg
        run: sudo apt-get install -y ffmpeg

      - name: Run tests
        run: go test -v ./... -cover

      - name: Generate coverage report
        run: |
          go test ./... -coverprofile=coverage.out
          go tool cover -html=coverage.out -o coverage.html

      - name: Upload coverage
        uses: actions/upload-artifact@v3
        with:
          name: coverage
          path: coverage.html
```

---

## 📚 Referências

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Testify](https://github.com/stretchr/testify) - Framework de testes (opcional)
- [FFmpeg](https://ffmpeg.org/) - Ferramenta de processamento de vídeo

---

**Última Atualização**: 2026-01-27
**Cobertura Atual**: 63.7% (processor-steps)
