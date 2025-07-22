package queue

import (
	"context"
	"fmt"
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
	// Testa a conexão
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Printf("[queue] Falha ao conectar ao Redis em %s: %v\n", redisAddr, err)
		rdb = nil
	} else {
		fmt.Printf("[queue] Conectado ao Redis em %s com sucesso!\n", redisAddr)
	}
}

func getRequestQueueName() string {
	name := os.Getenv("PROCESSING_REQUEST_QUEUE")
	if name == "" {
		return "video_queue"
	}
	return name
}

func getFinishedQueueName() string {
	name := os.Getenv("PROCESSING_FINISHED_QUEUE")
	if name == "" {
		return "video_success_queue"
	}
	return name
}

// ConsumeMessage consome uma mensagem da fila Redis (lista de requests).
func ConsumeMessage() (*Message, error) {
	if rdb == nil {
		InitQueue(os.Getenv("REDIS_HOST"))
	}
	if rdb == nil {
		return nil, fmt.Errorf("[queue] Não foi possível conectar ao Redis")
	}
	queueName := getRequestQueueName()
	result, err := rdb.BLPop(ctx, 0, queueName).Result()
	if err != nil {
		return nil, err
	}
	if len(result) < 2 {
		return nil, nil
	}
	return &Message{VideoID: result[1]}, nil
}

// PublishSuccessMessage publica uma mensagem de sucesso na fila de finished.
func PublishSuccessMessage(videoID string) error {
	if rdb == nil {
		InitQueue(os.Getenv("REDIS_HOST"))
	}
	if rdb == nil {
		return fmt.Errorf("[queue] Não foi possível conectar ao Redis")
	}
	queueName := getFinishedQueueName()
	return rdb.LPush(ctx, queueName, videoID).Err()
}
