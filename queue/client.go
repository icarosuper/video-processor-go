package queue

import (
	"context"
	"time"
	"video-processor/config"
	"video-processor/internal/circuitbreaker"

	"github.com/redis/go-redis/v9"
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
// Usa BRPOPLPUSH para mover o job atomicamente para a fila de processamento.
func ConsumeMessage(ctx context.Context) (*Message, error) {
	result, err := circuitbreaker.Redis.Execute(func() (interface{}, error) {
		videoID, err := client.BRPopLPush(ctx, cfg.ProcessingRequestQueue, processingQueueName(), 0).Result()
		if err != nil {
			return nil, err
		}
		return &Message{VideoID: videoID}, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*Message), nil
}

// AcknowledgeMessage remove o job da fila de processamento após conclusão (sucesso ou falha).
func AcknowledgeMessage(videoID string) error {
	_, err := circuitbreaker.Redis.Execute(func() (interface{}, error) {
		return nil, client.LRem(context.Background(), processingQueueName(), 1, videoID).Err()
	})
	return err
}

func PublishSuccessMessage(videoID string) error {
	_, err := circuitbreaker.Redis.Execute(func() (interface{}, error) {
		return nil, client.LPush(context.Background(), cfg.ProcessingFinishedQueue, videoID).Err()
	})
	return err
}

// StartRecovery inicia uma goroutine que periodicamente verifica a fila de
// processamento e recoloca na fila principal jobs que ficaram presos (crash do worker).
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
