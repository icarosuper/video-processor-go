package config

import (
	"fmt"
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	"log"
	"reflect"
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

	// Workers
	WorkerCount int `env:"WORKER_COUNT" default:"0"`
}

func (c *Config) validate() error {
	val := reflect.ValueOf(c).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := typ.Field(i).Name

		// Verifica se o campo é uma string vazia
		if field.Kind() == reflect.String && field.String() == "" {
			return fmt.Errorf("campo %s não está definido", fieldName)
		}

		// Verifica se o campo é um int e está com valor 0 (exceto WorkerCount que tem default)
		if field.Kind() == reflect.Int && field.Int() == 0 && fieldName != "WorkerCount" {
			return fmt.Errorf("campo %s não está definido", fieldName)
		}
	}
	return nil
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("unable to load .env file: %e", err)
	}

	cfg := Config{}
	err = env.Parse(&cfg)
	if err != nil {
		log.Fatalf("unable to parse environment variables: %e", err)
	}

	fmt.Printf("Config loaded successfully: %+v\n", cfg)

	return &cfg
}
