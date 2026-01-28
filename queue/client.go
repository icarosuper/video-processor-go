package queue

import (
	"context"
	"video-processor/config"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

var (
	ctx    = context.Background()
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

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("Erro ao conectar cliente Redis")
	}
	log.Info().Str("host", cfg.RedisHost).Msg("Cliente Redis conectado com sucesso")
}

func ConsumeMessage() (*Message, error) {
	result, err := client.BLPop(ctx, 0, cfg.ProcessingRequestQueue).Result()

	if err != nil {
		return nil, err
	}
	if len(result) < 2 {
		return nil, nil
	}

	return &Message{VideoID: result[1]}, nil
}

func PublishSuccessMessage(videoID string) error {
	return client.LPush(ctx, cfg.ProcessingFinishedQueue, videoID).Err()
}

// HealthCheck verifica se o cliente Redis está saudável
func HealthCheck() error {
	return client.Ping(ctx).Err()
}
