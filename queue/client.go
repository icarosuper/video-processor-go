package queue

import (
	"context"
	"os"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var rdb *redis.Client

// Message representa uma mensagem da fila.
type Message struct {
	VideoID string
}

// InitQueue inicializa a conexão com o Redis.
func InitQueue(redisAddr string) {
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
}

// ConsumeMessage consome uma mensagem da fila Redis (lista 'video_queue').
func ConsumeMessage() (*Message, error) {
	if rdb == nil {
		InitQueue(os.Getenv("REDIS_HOST"))
	}
	result, err := rdb.BLPop(ctx, 0, "video_queue").Result()
	if err != nil {
		return nil, err
	}
	if len(result) < 2 {
		return nil, nil
	}
	return &Message{VideoID: result[1]}, nil
}

// PublishSuccessMessage publica uma mensagem de sucesso na fila 'video_success_queue'.
func PublishSuccessMessage(videoID string) error {
	if rdb == nil {
		InitQueue(os.Getenv("REDIS_HOST"))
	}
	return rdb.LPush(ctx, "video_success_queue", videoID).Err()
}
