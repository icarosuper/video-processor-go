# Roadmap - Video Processor Go

## Status Atual

O pipeline de processamento está implementado e funcional para o fluxo básico (download → transcodificação → upload). Existem bugs conhecidos que precisam ser corrigidos antes de avançar com novas features.

---

## Bugs a Corrigir (Prioridade Alta)

### B1. Workers travam no shutdown
`ConsumeMessage()` usa `context.Background()` global em vez do contexto passado pelo worker. Ao cancelar o contexto no shutdown, o BLPop não é interrompido.

**Solução**: `ConsumeMessage(ctx context.Context)` passando o contexto do worker.

### B2. Artefatos gerados não são enviados ao MinIO
Thumbnails, áudio, preview e HLS são gerados em `tempDir` mas apagados pelo `defer os.RemoveAll` sem upload. As etapas 4–7 do pipeline não têm efeito real.

**Solução**: Fazer upload dos artefatos para MinIO com prefixos adequados (ex: `thumbnails/`, `audio/`, `preview/`, `hls/`) antes da limpeza do `tempDir`.

### B3. `docker-compose.yml` com env vars faltando
O serviço `worker` não define `PROCESSING_REQUEST_QUEUE`, `PROCESSING_FINISHED_QUEUE` e `MINIO_BUCKET_NAME`.

**Solução**: Adicionar as variáveis faltantes no `docker-compose.yml`.

### B4. Senha exposta em log
`fmt.Printf("Config loaded: %+v", cfg)` imprime `MinioRootPassword` em plaintext.

**Solução**: Remover ou mascarar o print de configuração.

### B5. `godotenv.Load()` com Fatal sem `.env`
O processo falha se não houver arquivo `.env`, impossibilitando deploy sem ele.

**Solução**: Tratar o erro do `godotenv.Load()` como warning quando o arquivo não existir (`os.IsNotExist`).

---

## Melhorias de Médio Prazo

### Resiliência
- **Retry com exponential backoff**: reprocessar mensagens que falharam transitoriamente
- **Dead Letter Queue**: mover mensagens com falhas permanentes para uma fila separada
- **Circuit breaker**: proteger chamadas ao MinIO e Redis de falhas em cascata
- **Timeout por etapa**: hoje o timeout de 5 minutos é global; cada etapa deveria ter seu próprio limite

### Observabilidade
- **Métrica `active_workers` real**: hoje é um valor estático setado no startup; deveria refletir workers em processamento vs. ociosos
- **Métrica `queue_size` populada**: `QueueSize` existe mas nunca é atualizada
- **Dashboard Grafana**: configuração pronta para uso com as queries documentadas em `OBSERVABILITY.md`
- **`video_size_bytes`**: métrica existe mas não é registrada em nenhum ponto do código

### Configuração
- **SSL MinIO configurável**: `useSsl` hardcoded como `false`; expor via env var `MINIO_USE_SSL`
- **Porta HTTP configurável**: `:8080` hardcoded em `main.go`

### Código
- **`config.validate()`**: método definido mas nunca chamado — remover ou integrar ao `LoadConfig()`
- **`go-redis/v8` → `v9`**: versão mais recente com melhor suporte a context

---

## Features de Longo Prazo

- API REST para submissão de vídeos (hoje requer publicar diretamente no Redis)
- Webhooks para notificação de conclusão
- Suporte a múltiplos formatos de output configuráveis por job
- Auto-scaling baseado no tamanho da fila
- Tracing distribuído com OpenTelemetry

---

**Última Atualização**: 2026-03-25
