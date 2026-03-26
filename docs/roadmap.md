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

---

## 🔴 Crítico — Bloqueadores para uso em produção

Sem esses itens a API não consegue integrar de forma confiável.

### C1. Estado de job (maior gap)

Hoje a API publica um `videoID` no Redis e nunca mais sabe o que aconteceu. Não existe estado PENDING → PROCESSING → DONE/FAILED.

**Impacto**: a API não pode responder ao usuário "seu vídeo está processando" nem "falhou por este motivo".

**Solução**: ao consumir um job, gravar estado no Redis Hash (`job:{videoID}` com campos `status`, `error`, `artifacts`). Atualizar ao longo do pipeline. A API consulta esse hash para polling ou para disparar webhook.

### C2. Jobs perdidos em caso de crash (BLPop destrutivo)

`BLPop` remove a mensagem da fila imediatamente ao consumir. Se o worker travar durante o processamento, o job some — não há como reprocessar.

**Solução**: usar `BRPOPLPUSH` para mover o job para uma fila de "em progresso" ao consumir, e só remover após confirmação de conclusão. Jobs que ficam presos nessa fila por mais de X minutos são recolocados na fila principal.

### C3. Falhas não chegam à API

Quando o processamento falha, nada é publicado na fila de sucesso. A API não sabe que o job falhou, nunca notifica o usuário.

**Solução**: publicar na fila de resultado independente de sucesso ou falha, com campo `status` e `error`. Ou usar o estado de job do C1.

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
