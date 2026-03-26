# Roadmap - Video Processor Go

## Objetivo do Projeto

Worker assíncrono que uma API chama para processar vídeos enviados por usuários (modelo YouTube): recebe um `videoID`, processa em pipeline de 7 etapas com FFmpeg, e entrega os artefatos no MinIO.

## Status Atual: ~30% pronto para produção

O pipeline FFmpeg funciona. A infraestrutura básica existe. Mas faltam as peças que tornam o sistema **confiável e integrável** com uma API real.

---

## ✅ Concluído

- Pipeline de 7 etapas com FFmpeg (validação, transcodificação, thumbnails, áudio, preview, HLS)
- Upload de todos os artefatos ao MinIO (thumbnails, áudio, preview, segmentos HLS)
- Workers concorrentes com graceful shutdown funcional
- Métricas Prometheus, health check, logging estruturado
- Testes unitários (88.2% no pipeline) e de integração (testcontainers)
- **B1**: Workers não travavam mais no shutdown — `ConsumeMessage(ctx)`
- **B2**: Artefatos das etapas 4–7 agora chegam ao MinIO
- **B3**: `docker-compose.yml` com todas as env vars obrigatórias
- **B4**: Senha MinIO não é mais impressa em log
- **B5**: Deploy sem arquivo `.env` funciona
- **C1**: Estado de job implementado (`queue/job.go`) — `pending → processing → done/failed` no Redis com TTL de 24h; artefatos gerados registrados em `done`
- **C2**: Consumo seguro com `BRPOPLPUSH` — job movido atomicamente para fila `{queue}:processing` ao consumir; `AcknowledgeMessage` remove ao concluir; `PublishJob` cria estado `pending` no produtor
- **C4**: Recovery de jobs órfãos — `StartRecovery` verifica a cada minuto jobs em `processing` com `updated_at` antigo e os recoloca na fila principal
- **P2**: Retry automático — `SetJobFailed` incrementa `RetryCount`; até 3 tentativas o job é recolocado na fila; acima disso vai para DLQ
- **P3**: Dead letter queue — `{queue}:dead` recebe jobs que esgotaram tentativas; state permanece `failed` com mensagem de erro auditável

---

## 🔴 Crítico — Bloqueadores para uso em produção

### C3. Falhas não chegam à API (parcialmente resolvido)

O estado do job é gravado como `failed` com a mensagem de erro e a API pode consultar via `GetJobState(videoID)`. Mas requer polling ativo — não há notificação push.

**Pendente**: implementar webhook ou callback para a API não precisar fazer polling.

---

## 🟠 Importante — Necessário para qualidade de produto

### P1. Múltiplas resoluções de saída

Hoje gera um único MP4 transcodificado. Para streaming adaptativo funcionar, precisa de múltiplas qualidades: 360p, 480p, 720p e 1080p (quando o original permitir). O HLS gerado hoje usa resolução única.

**Solução**: etapa de transcodificação gerar múltiplos outputs; playlist HLS master referenciando cada qualidade.

### P2. Retry com exponential backoff

Falha transitória (MinIO instável por 2s, FFmpeg com OOM) descarta o job permanentemente. Deveria tentar N vezes antes de mover para dead letter queue.

### P3. Dead Letter Queue

Jobs com falha permanente (arquivo corrompido, codec não suportado) precisam de um destino auditável, não simplesmente desaparecer.

### P4. Persistência de metadados do vídeo

A etapa de análise extrai duração, codec, resolução, bitrate — mas só loga. Esses dados deveriam ser gravados junto ao estado do job (C1) para a API devolver ao usuário.

### P5. Validação de entrada mais rigorosa

Hoje qualquer arquivo passa para o FFmpeg. Falta:
- Limite de tamanho de arquivo (ex: 5GB max)
- Whitelist de codecs/containers aceitos
- Verificação de que o arquivo não é malicioso antes de processar

---

## 🟡 Melhorias — Qualidade operacional

### Observabilidade
- `active_workers`: valor estático no startup, nunca atualizado
- `queue_size`: métrica existe mas nunca é populada
- `video_size_bytes`: métrica existe mas nunca é registrada
- Tracing distribuído por job (OpenTelemetry) para ver tempo por etapa em produção

### Resiliência
- **Circuit breaker**: proteger chamadas ao MinIO e Redis de falhas em cascata
- **Timeout por etapa**: hoje o timeout de 5 minutos é global; etapas críticas deveriam ter limites individuais

### Configuração
- **SSL MinIO**: `useSsl` hardcoded como `false`; expor via env var `MINIO_USE_SSL`
- **Porta HTTP**: `:8080` hardcoded em `main.go`
- **`go-redis/v8` → `v9`**: versão mais recente com melhor suporte a context

---

## 🔵 Longo Prazo — Escalabilidade e features avançadas

- **Webhooks**: notificar a API quando o processamento terminar, sem polling
- **Auto-scaling**: aumentar workers baseado no tamanho da fila
- **Escala horizontal**: múltiplas instâncias do worker em máquinas diferentes
- **Priorização de fila**: vídeos curtos na frente, vídeos longos em fila separada
- **Dashboard Grafana**: configuração pronta para uso

---

**Última Atualização**: 2026-03-26
