# Video Processor Go - Roadmap e Implementação Pendente

## 🚨 Status Atual: ✅ FUNCIONAL

O projeto possui uma arquitetura sólida implementada e **todas as etapas do pipeline de processamento de vídeo foram implementadas com FFmpeg**. O sistema agora:

- ✅ Valida vídeos com ffprobe
- ✅ Transcodifica para MP4 (H.264 + AAC)
- ✅ Gera thumbnails em múltiplos timestamps
- ✅ Extrai áudio em MP3
- ✅ Cria preview de baixa qualidade
- ✅ Analisa metadados do vídeo
- ✅ Segmenta para streaming HLS

**Próximos Passos**: Melhorar observabilidade e resiliência do sistema.

---

## 📋 Matriz de Implementação

### ✅ PRIORIDADE CRÍTICA (Concluído)

~~Esses itens impediam o funcionamento básico do sistema e foram implementados.~~

#### ~~1. Correção de Bug na Fila de Sucesso~~ ✅
**Status**: CONCLUÍDO

#### ~~2. Implementação das Etapas do Pipeline com FFmpeg~~ ✅
**Status**: CONCLUÍDO

Todas as etapas em `internal/processor/processor-steps/` foram implementadas com FFmpeg.

##### 2.1 Validação de Vídeo (`validate.go`)
**Objetivo**: Verificar se o vídeo é válido e pode ser processado

**Implementação**:
```go
package processor_steps

import (
    "os/exec"
    "strings"
)

func ValidateVideo(videoPath string) error {
    // Usar ffprobe para validar
    cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)
    output, err := cmd.CombinedOutput()

    if err != nil {
        return fmt.Errorf("vídeo inválido: %w", err)
    }

    // Verificar duração mínima (ex: 1 segundo)
    duration := strings.TrimSpace(string(output))
    if duration == "" || duration == "N/A" {
        return fmt.Errorf("vídeo não possui duração válida")
    }

    return nil
}
```

**Bibliotecas Necessárias**:
- `os/exec`: Já está no Go stdlib
- FFmpeg/FFprobe: Já instalado no Dockerfile

**Testes**:
- Teste com vídeo válido
- Teste com arquivo corrompido
- Teste com arquivo não-vídeo

**Tempo Estimado**: 1-2 horas

---

##### 2.2 Transcodificação (`transcode.go`)
**Objetivo**: Converter vídeo para formatos padronizados (MP4, H.264, AAC)

**Implementação**:
```go
package processor_steps

import (
    "os/exec"
    "path/filepath"
)

func TranscodeVideo(inputPath, outputPath string) error {
    // Converter para MP4 com H.264 video e AAC audio
    cmd := exec.Command("ffmpeg",
        "-i", inputPath,
        "-c:v", "libx264",           // Codec de vídeo H.264
        "-preset", "medium",          // Balance entre velocidade e compressão
        "-crf", "23",                 // Qualidade (lower = better, 18-28 é bom)
        "-c:a", "aac",                // Codec de áudio
        "-b:a", "128k",               // Bitrate de áudio
        "-movflags", "+faststart",    // Otimização para streaming
        outputPath,
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("falha na transcodificação: %w, output: %s", err, string(output))
    }

    return nil
}
```

**Formatos Alvo**:
- Vídeo: H.264
- Áudio: AAC
- Container: MP4
- CRF: 23 (boa qualidade)

**Testes**:
- Transcodificação de MP4 para MP4
- Conversão de AVI/MOV para MP4
- Verificação de qualidade do output

**Tempo Estimado**: 2-3 horas

---

##### 2.3 Geração de Thumbnails (`thumbnail.go`)
**Objetivo**: Gerar imagens de pré-visualização em múltiplos timestamps

**Implementação**:
```go
package processor_steps

import (
    "os/exec"
    "path/filepath"
    "strconv"
)

type ThumbnailConfig struct    {
    Count      int     // Número de thumbnails (ex: 5)
    Width      int     // Largura em pixels (ex: 320)
    Height     int     // Altura em pixels (ex: 180)
}

func GenerateThumbnails(videoPath string, outputDir string, config ThumbnailConfig) ([]string, error) {
    var thumbnails []string

    // Obter duração do vídeo
    durationCmd := exec.Command("ffprobe",
        "-v", "error",
        "-show_entries", "format=duration",
        "-of", "default=noprint_wrappers=1:nokey=1",
        videoPath,
    )

    durationOutput, err := durationCmd.Output()
    if err != nil {
        return nil, fmt.Errorf("falha ao obter duração: %w", err)
    }

    duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationOutput)), 64)
    if err != nil {
        return nil, fmt.Errorf("duração inválida: %w", err)
    }

    // Gerar thumbnails em intervalos regulares
    interval := duration / float64(config.Count+1)

    for i := 1; i <= config.Count; i++ {
        timestamp := interval * float64(i)
        thumbnailPath := filepath.Join(outputDir, fmt.Sprintf("thumb_%03d.jpg", i))

        cmd := exec.Command("ffmpeg",
            "-ss", strconv.FormatFloat(timestamp, 'f', 2, 64), // Timestamp
            "-i", videoPath,
            "-vframes", "1",           // Um único frame
            "-vf", fmt.Sprintf("scale=%d:%d", config.Width, config.Height),
            "-y",                      // Sobrescrever
            thumbnailPath,
        )

        if err := cmd.Run(); err != nil {
            return nil, fmt.Errorf("falha ao gerar thumbnail %d: %w", i, err)
        }

        thumbnails = append(thumbnails, thumbnailPath)
    }

    return thumbnails, nil
}
```

**Configuração Sugerida**:
- 5 thumbnails por vídeo
- Resolução: 320x180 (16:9)
- Formato: JPEG (qualidade 85%)

**Testes**:
- Vídeo curto (menos de 10 segundos)
- Vídeo longo (mais de 1 hora)
- Vídeos com diferentes aspect ratios

**Tempo Estimado**: 2-3 horas

---

##### 2.4 Extração de Áudio (`audio.go`)
**Objetivo**: Extrair track de áudio do vídeo

**Implementação**:
```go
package processor_steps

import (
    "os/exec"
    "path/filepath"
)

func ExtractAudio(videoPath string, outputDir string) error {
    audioPath := filepath.Join(outputDir, "audio.mp3")

    cmd := exec.Command("ffmpeg",
        "-i", videoPath,
        "-vn",                     // No video
        "-acodec", "libmp3lame",   // Codec MP3
        "-ab", "192k",             // Bitrate de áudio
        "-y",
        audioPath,
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("falha na extração de áudio: %w, output: %s", err, string(output))
    }

    return nil
}
```

**Formatos Suportados**:
- MP3 (192kbps) - padrão
- AAC (128kbps) - alternativo

**Testes**:
- Vídeo com áudio estéreo
- Vídeo com múltiplos tracks de áudio
- Vídeo sem áudio (deve falhar graciosamente)

**Tempo Estimado**: 1-2 horas

---

##### 2.5 Geração de Preview (`preview.go`)
**Objetivo**: Criar versão de baixa qualidade para pré-visualização rápida

**Implementação**:
```go
package processor_steps

import (
    "os/exec"
    "path/filepath"
    "time"
)

func GeneratePreview(videoPath string, outputDir string, duration time.Duration) error {
    previewPath := filepath.Join(outputDir, "preview.mp4")

    // Criar preview de 30 segundos ou 10% do vídeo (o que for menor)
    previewDuration := duration.Seconds()
    if previewDuration > 30 {
        previewDuration = 30
    }

    cmd := exec.Command("ffmpeg",
        "-i", videoPath,
        "-t", strconv.FormatFloat(previewDuration, 'f', 0, 64), // Duração
        "-vf", "scale=640:-2",         // Escalar para largura 640
        "-b:v", "500k",                // Bitrate baixo
        "-c:a", "aac",                 // Áudio AAC
        "-b:a", "64k",                 // Bitrate de áudio baixo
        "-y",
        previewPath,
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("falha na geração de preview: %w, output: %s", err, string(output))
    }

    return nil
}
```

**Configuração**:
- Duração: 30 segundos (ou 10% do vídeo)
- Resolução: 640px de largura
- Bitrate vídeo: 500kbps
- Bitrate áudio: 64kbps

**Testes**:
- Vídeo curto (menos de 30s)
- Vídeo longo

**Tempo Estimado**: 1-2 horas

---

##### 2.6 Análise de Conteúdo (`analysis.go`)
**Objetivo**: Extrair metadados e informações técnicas do vídeo

**Implementação**:
```go
package processor_steps

import (
    "encoding/json"
    "os/exec"
)

type VideoMetadata struct {
    Duration     float64 `json:"duration"`
    Width        int     `json:"width"`
    Height       int     `json:"height"`
    VideoCodec   string  `json:"video_codec"`
    AudioCodec   string  `json:"audio_codec"`
    FPS          float64 `json:"fps"`
    Bitrate      int64   `json:"bitrate"`
    Size         int64   `json:"size"`
}

func AnalyzeVideo(videoPath string) (*VideoMetadata, error) {
    cmd := exec.Command("ffprobe",
        "-v", "quiet",
        "-print_format", "json",
        "-show_format",
        "-show_streams",
        videoPath,
    )

    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("falha na análise: %w", err)
    }

    // Parse JSON output
    var probeData struct {
        Format struct {
            Duration string `json:"duration"`
            Size     string `json:"size"`
            BitRate string `json:"bit_rate"`
        } `json:"format"`
        Streams []struct {
            CodecType string `json:"codec_type"`
            CodecName string `json:"codec_name"`
            Width     int    `json:"width"`
            Height    int    `json:"height"`
            RFrameRate string `json:"r_frame_rate"`
        } `json:"streams"`
    }

    if err := json.Unmarshal(output, &probeData); err != nil {
        return nil, fmt.Errorf("falha ao parsear JSON: %w", err)
    }

    // Extrair metadados
    metadata := &VideoMetadata{}

    for _, stream := range probeData.Streams {
        if stream.CodecType == "video" {
            metadata.Width = stream.Width
            metadata.Height = stream.Height
            metadata.VideoCodec = stream.CodecName

            // Parse FPS (format: "30000/1001")
            if parts := strings.Split(stream.RFrameRate, "/"); len(parts) == 2 {
                numerator, _ := strconv.ParseFloat(parts[0], 64)
                denominator, _ := strconv.ParseFloat(parts[1], 64)
                metadata.FPS = numerator / denominator
            }
        } else if stream.CodecType == "audio" {
            metadata.AudioCodec = stream.CodecName
        }
    }

    // Parse duration, size, bitrate
    metadata.Duration, _ = strconv.ParseFloat(probeData.Format.Duration, 64)
    metadata.Size, _ = strconv.ParseInt(probeData.Format.Size, 10, 64)
    metadata.Bitrate, _ = strconv.ParseInt(probeData.Format.BitRate, 10, 64)

    return metadata, nil
}
```

**Metadados Coletados**:
- Duração
- Resolução (largura x altura)
- Codecs (vídeo e áudio)
- FPS (frames por segundo)
- Bitrate
- Tamanho do arquivo

**Testes**:
- Vídeo SD (480p)
- Vídeo HD (720p/1080p)
- Vídeo 4K

**Tempo Estimado**: 2-3 horas

---

##### 2.7 Segmentação para Streaming (`streaming.go`)
**Objetivo**: Criar segmentos HLS para streaming adaptativo

**Implementação**:
```go
package processor_steps

import (
    "os/exec"
    "path/filepath"
)

func CreateStreamingSegments(videoPath string, outputDir string) error {
    // Criar diretório para segmentos
    segmentsDir := filepath.Join(outputDir, "streaming")
    if err := os.MkdirAll(segmentsDir, 0755); err != nil {
        return fmt.Errorf("falha ao criar diretório: %w", err)
    }

    playlistPath := filepath.Join(segmentsDir, "playlist.m3u8")

    cmd := exec.Command("ffmpeg",
        "-i", videoPath,
        "-c:v", "libx264",
        "-c:a", "aac",
        "-f", "hls",                   // Formato HLS
        "-hls_time", "6",              // 6 segundos por segmento
        "-hls_list_size", "0",         // Manter todos os segmentos na playlist
        "-hls_segment_filename", filepath.Join(segmentsDir, "segment_%03d.ts"),
        playlistPath,
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("falha na segmentação: %w, output: %s", err, string(output))
    }

    return nil
}
```

**Configuração HLS**:
- Duração do segmento: 6 segundos
- Todos os segmentos na playlist
- Formato: MPEG-TS (.ts)
- Playlist: M3U8

**Melhorias Futuras**:
- Suporte a múltiplas resoluções (480p, 720p, 1080p)
- DASH além de HLS
- DRM para proteção de conteúdo

**Testes**:
- Reprodução em player HLS
- Verificação de todos os segmentos
- Teste de latência

**Tempo Estimado**: 3-4 horas

---

##### 2.8 Integração das Etapas no Processor

**Arquivo**: `internal/processor/processor.go`

**Modificações Necessárias**:

```go
func (p *Processor) processVideo(ctx context.Context, videoID string) error {
    // 1. Download do vídeo
    tempVideoPath, err := p.minioClient.DownloadVideo(ctx, videoID)
    if err != nil {
        return fmt.Errorf("erro ao fazer download: %w", err)
    }
    defer os.Remove(tempVideoPath) // Limpeza

    // 2. Criar diretório de output
    outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("video_%s", videoID))
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return fmt.Errorf("erro ao criar diretório: %w", err)
    }
    defer os.RemoveAll(outputDir) // Limpeza

    // 3. Validar
    if err := processor_steps.ValidateVideo(tempVideoPath); err != nil {
        return fmt.Errorf("vídeo inválido: %w", err)
    }

    // 4. Analisar
    metadata, err := processor_steps.AnalyzeVideo(tempVideoPath)
    if err != nil {
        return fmt.Errorf("falha na análise: %w", err)
    }
    log.Printf("Metadados: %+v", metadata)

    // 5. Transcodificar
    transcodedPath := filepath.Join(outputDir, "transcoded.mp4")
    if err := processor_steps.TranscodeVideo(tempVideoPath, transcodedPath); err != nil {
        return fmt.Errorf("falha na transcodificação: %w", err)
    }

    // 6. Gerar thumbnails
    thumbnails, err := processor_steps.GenerateThumbnails(
        transcodedPath,
        filepath.Join(outputDir, "thumbnails"),
        processor_steps.ThumbnailConfig{Count: 5, Width: 320, Height: 180},
    )
    if err != nil {
        return fmt.Errorf("falha nos thumbnails: %w", err)
    }
    log.Printf("Gerados %d thumbnails", len(thumbnails))

    // 7. Extrair áudio
    if err := processor_steps.ExtractAudio(transcodedPath, outputDir); err != nil {
        log.Printf("Aviso: falha na extração de áudio: %v", err)
    }

    // 8. Gerar preview
    previewDuration := time.Duration(metadata.Duration) * time.Second
    if previewDuration > 30*time.Second {
        previewDuration = 30 * time.Second
    }
    if err := processor_steps.GeneratePreview(transcodedPath, outputDir, previewDuration); err != nil {
        log.Printf("Aviso: falha no preview: %v", err)
    }

    // 9. Criar segmentos para streaming
    if err := processor_steps.CreateStreamingSegments(transcodedPath, outputDir); err != nil {
        log.Printf("Aviso: falha na segmentação: %v", err)
    }

    // 10. Upload de todos os arquivos gerados
    if err := p.minioClient.UploadProcessedVideo(ctx, videoID, outputDir); err != nil {
        return fmt.Errorf("erro ao fazer upload: %w", err)
    }

    return nil
}
```

**Tempo Estimado**: 2-3 horas

---

### 🟡 PRIORIDADE ALTA (Importante para Operação)

> **📍 PRÓXIMOS PASSOS RECOMENDADOS**
> Com o pipeline de processamento implementado, os próximos 3 itens essenciais são:
> - **Item 4**: Logging Estruturado (observabilidade)
> - **Item 5**: Métricas com Prometheus (monitoramento)
> - **Item 6**: Health Check Endpoint (disponibilidade)

---

#### 3. Tratamento Adequado de Arquivos Temporários
**Problema**: Comentário `// todo: Handle these` indica que arquivos temporários não estão sendo limpos adequadamente.

**Como Fazer**:
1. Garantir que `defer os.Remove()` e `defer os.RemoveAll()` estão sendo usados
2. Implementar cleanup mesmo em caso de erro
3. Adicionar logging para debug de arquivos não removidos

**Tempo Estimado**: 1 hora

---

#### 4. Logging Estruturado
**Objetivo**: Melhorar observabilidade do sistema

**Biblioteca Escolhida**: `rs/zerolog` ✅
- Zero alocações de memória
- Melhor compatibilidade com Grafana Loki
- Output JSON nativo
- Altíssima performance

**Implementação**:
```go
// Usar zerolog para logging estruturado
import "github.com/rs/zerolog/log"

// Configurar no main.go
zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

// Em vez de:
log.Printf("Processando vídeo %s", videoID)

// Usar:
log.Info().
    Str("videoID", videoID).
    Int("workerID", workerID).
    Msg("Processando vídeo")
```

**Benefícios**:
- Logs estruturados em JSON
- Pronto para Grafana Loki
- Performance otimizada (zero-allocation)
- Níveis de log configuráveis

**Tempo Estimado**: 3-4 horas

---

#### 5. Métricas Básicas
**Objetivo**: Monitorar health do sistema

**Métricas a Implementar**:
- Número de vídeos processados
- Tempo médio de processamento
- Taxa de sucesso/falha
- Tamanho médio dos vídeos
- Uso de recursos (CPU, memória)

**Implementação com Prometheus**:
```go
import "github.com/prometheus/client_golang/prometheus"

var (
    videosProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "videos_processed_total",
            Help: "Total number of videos processed",
        },
        []string{"status"}, // success ou error
    )

    processingDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "video_processing_duration_seconds",
            Help:    "Time taken to process videos",
            Buckets: prometheus.DefBuckets,
        },
    )
)
```

**Tempo Estimado**: 4-6 horas

---

#### 6. Health Check Endpoint
**Objetivo**: Verificar se o serviço está saudável

**Implementação**:
```go
package main

import (
    "net/http"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Verificar Redis
    if err := redisClient.Ping(ctx).Err(); err != nil {
        http.Error(w, "Redis unavailable", http.StatusServiceUnavailable)
        return
    }

    // Verificar MinIO
    if _, err := minioClient.BucketExists(ctx, bucketName); err != nil {
        http.Error(w, "MinIO unavailable", http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func main() {
    http.HandleFunc("/health", healthHandler)
    go http.ListenAndServe(":8080", nil)

    // ... restante do código
}
```

**Tempo Estimado**: 2-3 horas

---

#### 7. Ativar SSL em Produção
**Problema**: SSL está desativado (`useSSL = false`)

**Como Fazer**:
1. Adicionar variável de ambiente `MINIO_USE_SSL=true`
2. Modificar `config/config.go`:
```go
type Config struct {
    // ... outros campos
    MinioUseSSL bool `env:"MINIO_USE_SSL" envDefault:"false"`
}

// No client.go:
minioClient, err := minio.New(config.MinioEndpoint, &minio.Options{
    Creds:  credentials.NewStaticV4(config.MinioRootUser, config.MinioRootPassword, ""),
    Secure: config.MinioUseSSL, // Usar da config
})
```

**Tempo Estimado**: 30 minutos

---

### 🟢 PRIORIDADE MÉDIA (Melhorias de Resiliência)

#### 8. Mecanismo de Retry com Exponential Backoff
**Objetivo**: Lidar com falhas temporárias gracefully

**Implementação**:
```go
package utils

import (
    "time"
    "math"
)

func RetryWithBackoff(fn func() error, maxRetries int) error {
    var err error
    for attempt := 0; attempt < maxRetries; attempt++ {
        if err = fn(); err == nil {
            return nil
        }

        // Exponential backoff: 2^attempt segundos
        waitTime := time.Duration(math.Pow(2, float64(attempt))) * time.Second
        time.Sleep(waitTime)
    }
    return fmt.Errorf("falha após %d tentativas: %w", maxRetries, err)
}

// Uso:
err := utils.RetryWithBackoff(func() error {
    return minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
}, 3)
```

**Tempo Estimado**: 3-4 horas

---

#### 9. Dead Letter Queue (DLQ)
**Objetivo**: Armazenar mensagens que falharam repetidamente

**Implementação**:
```go
const DeadLetterQueue = "video_dlq"

func (q *QueueClient) PublishToDLQ(ctx context.Context, videoID string, errorMsg string) error {
    message := map[string]interface{}{
        "videoID": videoID,
        "error":   errorMsg,
        "timestamp": time.Now().Unix(),
    }

    messageJSON, _ := json.Marshal(message)
    return q.client.LPush(ctx, DeadLetterQueue, messageJSON)
}
```

**Tempo Estimado**: 2-3 horas

---

#### 10. Timeout por Etapa do Pipeline
**Objetivo**: Evitar que uma etapa travada bloqueie todo o processamento

**Implementação**:
```go
func (p *Processor) processVideoWithTimeout(ctx context.Context, videoID string) error {
    // Timeout para cada etapa
    timeoutDuration := 30 * time.Minute

    for _, step := range p.steps {
        stepCtx, cancel := context.WithTimeout(ctx, timeoutDuration)
        err := step.Execute(stepCtx, videoID)
        cancel()

        if err != nil {
            return fmt.Errorf("erro na etapa %s: %w", step.Name(), err)
        }
    }

    return nil
}
```

**Tempo Estimado**: 2-3 horas

---

#### 11. Validação de Vídeos Maliciosos
**Objetivo**: Prevenir upload de arquivos maliciosos

**Implementação**:
```go
import (
    "mime"
    "os/exec"
)

func ValidateFileIntegrity(filePath string) error {
    // 1. Verificar MIME type
    mimeType, err := mimetype.DetectFile(filePath)
    if err != nil {
        return err
    }

    allowedTypes := []string{
        "video/mp4",
        "video/avi",
        "video/quicktime",
        "video/x-msvideo",
    }

    if !contains(allowedTypes, mimeType) {
        return fmt.Errorf("tipo de arquivo não permitido: %s", mimeType)
    }

    // 2. Verificar magic bytes
    // 3. Verificar se é um arquivo de vídeo válido com ffprobe
    // 4. Limitar tamanho do arquivo (ex: 5GB)

    return nil
}
```

**Tempo Estimado**: 3-4 horas

---

### 🔵 PRIORIDADE BAIXA (Melhorias de Longo Prazo)

#### 12. Circuit Breaker para Chamadas Externas
**Objetivo**: Prevenir chamadas a serviços que estão falhando

**Implementação** com `github.com/sony/gobreaker`:
```go
import "github.com/sony/gobreaker"

cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:    "MinIO",
    Timeout: 30 * time.Second,
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures > 5
    },
})

output, err := cb.Execute(func() (interface{}, error) {
    return minioClient.GetObject(...)
})
```

**Tempo Estimado**: 4-6 horas

---

#### 13. Pool de Conexões Redis
**Objetivo**: Otimizar performance de conexões

**Já está implícito no `go-redis/redis`**, mas pode ser configurado:
```go
client := redis.NewClient(&redis.Options{
    Addr:         config.RedisHost,
    PoolSize:     10,     // Número máximo de conexões
    MinIdleConns: 5,      // Conexões mínimas ociosas
    MaxRetries:   3,      // Tentativas de retry
})
```

**Tempo Estimado**: 1 hora

---

#### 14. Compressão para Uploads MinIO
**Objetivo**: Reduzir bandwidth e armazenamento

**Implementação**:
```go
import "compress/gzip"

func (c *MinioClient) UploadCompressed(ctx context.Context, bucket, object string, data []byte) error {
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    if _, err := gz.Write(data); err != nil {
        return err
    }
    if err := gz.Close(); err != nil {
        return err
    }

    _, err := c.client.PutObject(ctx, bucket, object, &buf, int64(buf.Len()), minio.PutObjectOptions{
        ContentEncoding: "gzip",
    })
    return err
}
```

**Tempo Estimado**: 2-3 horas

---

#### 15. Dashboard de Monitoramento
**Objetivo**: Visualização em tempo real do sistema

**Tecnologias**:
- Grafana + Prometheus (já sugerido nas métricas)
- Dashboards pré-configurados para:
  - Taxa de processamento
  - Erros por tipo
  - Latência
  - Recursos do sistema

**Tempo Estimado**: 8-10 horas

---

#### 16. Suporte a Múltiplos Formatos de Vídeo
**Objetivo**: Aceitar mais formatos além de MP4

**Formatos a Adicionar**:
- MKV (Matroska)
- WebM
- AVI
- MOV (QuickTime)
- FLV (Flash Video)

**Tempo Estimado**: 4-6 horas

---

#### 17. Cache de Metadados
**Objetivo**: Evitar reprocessamento de vídeos já analisados

**Implementação com Redis**:
```go
func (c *Cache) GetMetadata(videoID string) (*VideoMetadata, bool) {
    val, err := c.client.Get(ctx, fmt.Sprintf("metadata:%s", videoID)).Result()
    if err != nil {
        return nil, false
    }

    var metadata VideoMetadata
    json.Unmarshal([]byte(val), &metadata)
    return &metadata, true
}

func (c *Cache) SetMetadata(videoID string, metadata *VideoMetadata) error {
    data, _ := json.Marshal(metadata)
    return c.client.Set(ctx, fmt.Sprintf("metadata:%s", videoID), data, 24*time.Hour).Err()
}
```

**Tempo Estimado**: 3-4 horas

---

#### 18. API REST para Gerenciamento
**Objetivo**: Permitir controle externo do processador

**Endpoints Sugeridos**:
- `GET /videos/{id}` - Status do processamento
- `POST /videos/{id}/reprocess` - Reprocessar vídeo
- `GET /health` - Health check
- `GET /metrics` - Métricas Prometheus

**Tempo Estimado**: 8-10 horas

---

#### 19. Webhooks para Notificações
**Objetivo**: Notificar sistemas externos quando processamento terminar

**Implementação**:
```go
type WebhookNotifier struct {
    endpoints []string
    httpClient *http.Client
}

func (w *WebhookNotifier) NotifyProcessingComplete(videoID string, status string) {
    payload := map[string]interface{}{
        "videoID": videoID,
        "status":  status,
        "timestamp": time.Now().Unix(),
    }

    for _, endpoint := range w.endpoints {
        go func(url string) {
            jsonPayload, _ := json.Marshal(payload)
            http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
        }(endpoint)
    }
}
```

**Tempo Estimado**: 4-6 horas

---

#### 20. Tracing Distribuído (OpenTelemetry)
**Objetivo**: Rastrear requisições através do sistema

**Implementação**:
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (p *Processor) processVideo(ctx context.Context, videoID string) error {
    ctx, span := otel.Tracer("processor").Start(ctx, "processVideo")
    defer span.End()

    span.SetAttributes(attribute.String("videoID", videoID))

    // ... processamento
}
```

**Tempo Estimado**: 6-8 horas

---

## 📅 Cronograma Sugerido de Implementação

### ~~Semana 1: Funcionalidade Básica~~ ✅ CONCLUÍDA
- ✅ **Dia 1-2**: Corrigir bug da fila + implementar Validação e Transcodificação
- ✅ **Dia 3-4**: Implementar Thumbnails e Extração de Áudio
- ✅ **Dia 5**: Testar pipeline completo e corrigir bugs

### Semana 2: Observabilidade (EM ANDAMENTO)
- **Dia 1-2**: Logging estruturado + Métricas básicas
- **Dia 3**: Health Check endpoint
- **Dia 4**: Tratamento de arquivos temporários + SSL
- **Dia 5**: Preview e Análise de Conteúdo avançada

### Semana 3: Resiliência
- **Dia 1-2**: Retry mechanism + Dead Letter Queue
- **Dia 3**: Timeouts por etapa
- **Dia 4**: Validação de segurança
- **Dia 5**: Testes de carga e estresse

### Semana 4: Melhorias
- **Dia 1-2**: Circuit Breaker + Otimizações
- **Dia 3-4**: Dashboard + Monitoramento
- **Dia 5**: Documentação e limpeza técnica

---

## 🧪 Testes Necessários

### Testes Unitários
- [ ] Cada etapa do pipeline individualmente
- [ ] Clientes de MinIO e Redis
- [ ] Parser de configurações

### Testes de Integração
- [ ] Pipeline completo com vídeo real
- [ ] Comunicação Redis ↔ Workers
- [ ] Upload/Download MinIO

### Testes de Carga
- [ ] 100 vídeos simultâneos
- [ ] Vídeos de 1GB+
- [ ] Filas com milhares de mensagens

### Testes de Resiliência
- [ ] Matar Redis durante processamento
- [ ] Matar MinIO durante upload
- [ ] Falha de disco no meio do processamento

---

## 📊 Métricas de Sucesso

### Funcionalidade ✅ ATINGIDO
- ✅ Pipeline processando vídeos end-to-end
- ✅ Todas as 7 etapas implementadas com FFmpeg
- ✅ Arquivos corretos gerados (transcodificação, thumbnails, áudio, preview, HLS)

### Performance (A MEDIR)
- ⏳ Processar vídeo de 1GB em menos de 5 minutos
- ⏳ Suportar 10 workers simultâneos
- ⏳ Latência de fila < 1 segundo

### Confiabilidade (A IMPLEMENTAR)
- ⏳ 99% de taxa de sucesso
- ⏳ Zero vazamento de memória
- ⏳ Recuperação automática de falhas

---

## 🎯 Próximos Passos Imediatos

1. ✅ **~~CONCLUÍDO~~**: Bug da fila de sucesso corrigido
2. ✅ **~~CONCLUÍDO~~**: Pipeline de processamento implementado com FFmpeg
3. **PRÓXIMO**: Implementar Logging Estruturado (3-4 horas)
4. **PRÓXIMO**: Adicionar Métricas com Prometheus (4-6 horas)
5. **PRÓXIMO**: Criar Health Check Endpoint (2-3 horas)
6. **FUTURO**: Melhorias de resiliência e escalabilidade

---

**Última Atualização**: 2026-01-27
**Prioridade Atual**: 🟡 ALTA - Observabilidade e Monitoramento
