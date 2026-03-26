# Testes - Video Processor Go

## Cobertura Atual

| Pacote | Cobertura | Tipo |
|---|---|---|
| `internal/processor/processor-steps` | 63.7% | Unitários |
| `metrics` | ~100% | Unitários |
| `test/integration` | — | Integração |

**Pacotes sem testes**: `main`, `config`, `queue`, `minio`

---

## Executando os Testes

```bash
# Todos os testes
go test ./...

# Com cobertura
go test ./... -cover

# Saída detalhada
go test -v ./...

# Relatório HTML
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Por pacote

```bash
go test -v ./internal/processor/processor-steps/...
go test -v ./metrics/...
go test -v ./test/integration/... -timeout 10m
```

### Testes de integração

Requerem Docker. São pulados automaticamente se Docker não estiver disponível.

```bash
go test -v ./test/integration/... -timeout 10m
```

---

## Testes Unitários

### `validate_test.go`
- `TestValidateVideo_ValidVideo`
- `TestValidateVideo_InvalidVideo`
- `TestValidateVideo_NonExistentFile`
- `TestValidateVideo_EmptyFile`

### `transcode_test.go`
- `TestTranscodeVideo_ValidVideo` — verifica arquivo de saída criado e não vazio
- `TestTranscodeVideo_InvalidInput`
- `TestTranscodeVideo_NonExistentInput`

### `thumbnail_test.go`
- `TestGenerateThumbnails_ValidVideo` — verifica 5 arquivos `thumb_00N.jpg`
- `TestGenerateThumbnails_InvalidVideo`
- `TestGenerateThumbnails_NonExistentVideo`

### `analysis_test.go`
- `TestAnalyzeContent_ValidVideo`
- `TestAnalyzeContent_InvalidVideo`
- `TestAnalyzeContent_NonExistentVideo`

### `metrics_test.go`
- `TestVideosProcessedTotal_Increment`
- `TestProcessingDuration_Observe`
- `TestProcessingStepDuration_MultipleSteps`
- `TestActiveWorkers_SetAndGet`
- `TestQueueSize_SetAndGet`

---

## Testes de Integração (`test/integration/`)

Usam **testcontainers-go** para subir Redis e MinIO reais.

### `minio_test.go`
- `TestMinIO_BucketOperations`
- `TestMinIO_ObjectUploadDownload`
- `TestMinIO_VideoWorkflow` — fluxo raw → processed
- `TestMinIO_DownloadToFile`
- `TestMinIO_NonExistentObject`

### Testes do pipeline (`pipeline_test.go`)
- `TestPipeline_ValidateStep`
- `TestPipeline_TranscodeStep`
- `TestPipeline_FullWorkflow` — Redis → download → FFmpeg → upload → fila de sucesso
- `TestPipeline_ThumbnailGeneration`

---

## Helpers de Teste

### `GenerateTestVideo(t, duration int) string`

Gera um vídeo de teste via FFmpeg (640x480, H.264+AAC, sine wave 1000Hz).
Pula o teste automaticamente se FFmpeg não estiver disponível.

```go
videoPath := GenerateTestVideo(t, 5) // 5 segundos
```

### `CreateInvalidFile(t) string`

Cria um arquivo com conteúdo inválido para testes de erro.

---

## Requisitos

**FFmpeg** é obrigatório para os testes de processamento:

```bash
# Ubuntu/Debian
sudo apt-get install ffmpeg

# macOS
brew install ffmpeg
```

Testes que dependem de FFmpeg são pulados automaticamente com:
```
FFmpeg não está disponível - pulando teste
```

---

## O que Falta Testar

- `config.LoadConfig()` — incluindo o comportamento sem `.env`
- `queue.ConsumeMessage()` e `PublishSuccessMessage()`
- `minio.DownloadVideo()` e `UploadVideo()`
- `main.processNextMessage()` — lógica de orquestração dos workers
- Etapas sem testes: `audio.go`, `preview.go`, `streaming.go`
- Benchmark de transcodificação e throughput

---

## CI/CD

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: sudo apt-get install -y ffmpeg
      - run: go test -v ./... -cover -timeout 10m
```

---

**Última Atualização**: 2026-03-25
**Cobertura Atual**: 63.7% (processor-steps)
