package queue

import (
	"context"
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

// HealthCheck verifica se o cliente Redis está saudável
func HealthCheck() error {
	return client.Ping(context.Background()).Err()
}
