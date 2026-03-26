package queue

import (
	"context"
	"time"
	"video-processor/config"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

var (
	client *redis.Client
	cfg    *config.Config
)

type Message struct {
	VideoID string
}

func InitRedisClient(configs *config.Config) {
	cfg = configs

	client = redis.NewClient(&redis.Options{
		Addr: cfg.RedisHost,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatal().Err(err).Msg("Erro ao conectar cliente Redis")
	}
	log.Info().Str("host", cfg.RedisHost).Msg("Cliente Redis conectado com sucesso")
}

func processingQueueName() string {
	return cfg.ProcessingRequestQueue + ":processing"
}

func deadLetterQueueName() string {
	return cfg.ProcessingRequestQueue + ":dead"
}

// ConsumeMessage bloqueia até receber uma mensagem da fila ou o ctx ser cancelado.
// Usa BRPOPLPUSH para mover o job atomicamente para a fila de processamento,
// garantindo que o job não seja perdido em caso de crash do worker.
func ConsumeMessage(ctx context.Context) (*Message, error) {
	videoID, err := client.BRPopLPush(ctx, cfg.ProcessingRequestQueue, processingQueueName(), 0).Result()
	if err != nil {
		return nil, err
	}
	return &Message{VideoID: videoID}, nil
}

// AcknowledgeMessage remove o job da fila de processamento após conclusão (sucesso ou falha).
// Deve sempre ser chamado ao fim do processamento para não deixar jobs órfãos.
func AcknowledgeMessage(videoID string) error {
	return client.LRem(context.Background(), processingQueueName(), 1, videoID).Err()
}

func PublishSuccessMessage(videoID string) error {
	return client.LPush(context.Background(), cfg.ProcessingFinishedQueue, videoID).Err()
}

// StartRecovery inicia uma goroutine que periodicamente verifica a fila de
// processamento e recoloca na fila principal jobs que ficaram presos (crash do worker).
// stuckTimeout define quanto tempo em processing antes de ser considerado órfão.
func StartRecovery(ctx context.Context, stuckTimeout time.Duration) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	log.Info().Dur("stuck_timeout", stuckTimeout).Msg("Recovery de jobs órfãos iniciado")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			recoverStuckJobs(stuckTimeout)
		}
	}
}

func recoverStuckJobs(stuckTimeout time.Duration) {
	videoIDs, err := client.LRange(context.Background(), processingQueueName(), 0, -1).Result()
	if err != nil {
		log.Warn().Err(err).Msg("Falha ao verificar fila de processamento para recovery")
		return
	}

	threshold := time.Now().Add(-stuckTimeout).Unix()
	for _, videoID := range videoIDs {
		state, err := GetJobState(videoID)
		if err != nil || state == nil {
			continue
		}
		if state.Status != JobStatusProcessing || state.UpdatedAt >= threshold {
			continue
		}

		log.Warn().Str("videoID", videoID).Int("retry_count", state.RetryCount).Msg("Job órfão detectado, recolocando na fila")

		state.RetryCount++
		state.Status = JobStatusPending
		if err := setJobState(videoID, *state); err != nil {
			log.Warn().Err(err).Str("videoID", videoID).Msg("Falha ao atualizar estado durante recovery")
			continue
		}
		client.LRem(context.Background(), processingQueueName(), 1, videoID)
		client.LPush(context.Background(), cfg.ProcessingRequestQueue, videoID)
	}
}

// GetQueueSize retorna o número de jobs aguardando na fila de requisições.
func GetQueueSize() (int64, error) {
	return client.LLen(context.Background(), cfg.ProcessingRequestQueue).Result()
}

// HealthCheck verifica se o cliente Redis está saudável
func HealthCheck() error {
	return client.Ping(context.Background()).Err()
}
