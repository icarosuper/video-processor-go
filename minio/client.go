package minio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// VideoType representa o tipo de vídeo (raw, processed, etc.)
type VideoType string

const (
	VideoTypeRaw       VideoType = "raw"
	VideoTypeProcessed VideoType = "processed"
)

var (
	client     *minio.Client
	bucketName = getBucketName()
)

func getBucketName() string {
	name := os.Getenv("MINIO_BUCKET_NAME")
	if name == "" {
		return "videos"
	}
	return name
}

func getObjectPath(videoType VideoType, objectID string) string {
	// videoType: "raw", "processed", etc.
	return filepath.Join(string(videoType), objectID)
}

func getMinioClient() (*minio.Client, error) {
	if client != nil {
		return client, nil
	}
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKeyID := os.Getenv("MINIO_ROOT_USER")
	secretAccessKey := os.Getenv("MINIO_ROOT_PASSWORD")
	useSSL := false

	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("[minio] erro ao criar client: %w", err)
	}
	client = c
	fmt.Printf("[minio] Conectado ao MinIO em %s\n", endpoint)
	return client, nil
}

// DownloadVideo baixa um vídeo do MinIO dado um tipo (VideoType) e um ID.
func DownloadVideo(videoType VideoType, objectID, destPath string) error {
	cli, err := getMinioClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	objectPath := getObjectPath(videoType, objectID)
	object, err := cli.GetObject(ctx, bucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("[minio] erro ao obter objeto: %w", err)
	}
	defer object.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("[minio] erro ao criar arquivo destino: %w", err)
	}
	defer outFile.Close()

	if _, err := outFile.ReadFrom(object); err != nil {
		return fmt.Errorf("[minio] erro ao salvar arquivo: %w", err)
	}
	fmt.Printf("[minio] Download de %s para %s concluído!\n", objectPath, destPath)
	return nil
}

// UploadVideo faz upload de um vídeo processado para o MinIO em uma "pasta" (prefixo).
func UploadVideo(srcPath string, videoType VideoType, objectID string) error {
	cli, err := getMinioClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("[minio] erro ao abrir arquivo para upload: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("[minio] erro ao obter info do arquivo: %w", err)
	}

	objectPath := getObjectPath(videoType, objectID)
	_, err = cli.PutObject(ctx, bucketName, objectPath, file, fileInfo.Size(), minio.PutObjectOptions{ContentType: "video/mp4"})
	if err != nil {
		return fmt.Errorf("[minio] erro ao fazer upload: %w", err)
	}
	fmt.Printf("[minio] Upload de %s para bucket %s concluído!\n", objectPath, bucketName)
	return nil
}
