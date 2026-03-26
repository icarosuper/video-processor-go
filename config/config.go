package config

import (
	"errors"
	"log"
	"os"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	// Redis
	RedisHost               string `env:"REDIS_HOST,notEmpty"`
	ProcessingRequestQueue  string `env:"PROCESSING_REQUEST_QUEUE,notEmpty"`
	ProcessingFinishedQueue string `env:"PROCESSING_FINISHED_QUEUE,notEmpty"`

	// MinIO
	MinioEndpoint     string `env:"MINIO_ENDPOINT,notEmpty"`
	MinioRootUser     string `env:"MINIO_ROOT_USER,notEmpty"`
	MinioRootPassword string `env:"MINIO_ROOT_PASSWORD,notEmpty"`
	MinioBucketName   string `env:"MINIO_BUCKET_NAME,notEmpty"`
	MinioUseSSL       bool   `env:"MINIO_USE_SSL" envDefault:"false"`

	// HTTP
	HTTPPort string `env:"HTTP_PORT" envDefault:"8080"`

	// Observabilidade
	OTelEndpoint    string `env:"OTEL_ENDPOINT"`                              // opcional: endpoint OTLP (ex: jaeger:4318); vazio = no-op
	OTelServiceName string `env:"OTEL_SERVICE_NAME" envDefault:"video-processor"` // nome do serviço nos traces

	// Webhook
	WebhookSecret string `env:"WEBHOOK_SECRET"` // opcional: assina requisições com HMAC-SHA256

	// Workers
	WorkerCount int `env:"WORKER_COUNT" envDefault:"0"`

	// Processamento
	MaxFileSizeMB int64 `env:"MAX_FILE_SIZE_MB" envDefault:"5120"` // 5 GB
}

func LoadConfig() *Config {
	// Ignora erro se o arquivo não existir (ex: deploy via Docker com env vars injetadas)
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("erro ao carregar .env: %v", err)
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("erro ao ler variáveis de ambiente: %v", err)
	}

	log.Printf("Configuração carregada: redis=%s minio=%s bucket=%s workers=%d",
		cfg.RedisHost, cfg.MinioEndpoint, cfg.MinioBucketName, cfg.WorkerCount)

	return &cfg
}
