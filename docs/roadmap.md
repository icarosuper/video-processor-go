# Roadmap - Video Processor Go

## Objetivo do Projeto

Worker assíncrono que uma API chama para processar vídeos enviados por usuários (modelo YouTube): recebe um `videoID`, processa em pipeline de 7 etapas com FFmpeg, e entrega os artefatos no MinIO.

## Status Atual: ~85% pronto para produção

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
- **P4**: Metadados do vídeo persistidos — `AnalyzeContent` retorna `*VideoMetadata`; gravados no estado `done` do job; disponíveis para a API via `GetJobState`
- **P5**: Validação de entrada mais rigorosa — limite de tamanho configurável via `MAX_FILE_SIZE_MB` (default 5GB); verificado antes do download com `StatObject`
- **Métricas operacionais**: `active_workers` Inc/Dec por job; `queue_size` atualizado a cada 30s; `video_size_bytes` registrado após download
- **C3**: Webhook/callback — ao concluir (sucesso ou falha permanente), o worker faz POST ao `callbackURL` registrado no job com o payload completo; HMAC-SHA256 opcional via `WEBHOOK_SECRET`
- **P1**: Múltiplas resoluções HLS — `SegmentForStreaming` gera 240p/360p/480p/720p/1080p (somente ≤ resolução original) + `master.m3u8`; `UploadDirectory` agora é recursivo; HLS gerado direto do input original (sem dupla transcodificação)

---

## 🟡 Melhorias — Qualidade operacional

### Observabilidade
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

- **Auto-scaling**: aumentar workers baseado no tamanho da fila
- **Escala horizontal**: múltiplas instâncias do worker em máquinas diferentes
- **Priorização de fila**: vídeos curtos na frente, vídeos longos em fila separada
- **Dashboard Grafana**: configuração pronta para uso

---

**Última Atualização**: 2026-03-26
