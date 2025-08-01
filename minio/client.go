package minio

import (
	"context"
	"fmt"
	"log"
	"os"
	"video-processor/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type VideoType string

const (
	VideoTypeRaw       VideoType = "raw"
	VideoTypeProcessed VideoType = "processed"
)

var (
	client *minio.Client
	cfg    *config.Config
)

const (
	useSsl = false // TODO: Ver se precisa usar SSL
	token  = ""    // TODO: Ver se precisa adicionar esse token
)

func InitMinioClient(config *config.Config) {
	cfg = config

	var err error
	client, err = minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioRootUser, cfg.MinioRootPassword, token),
		Secure: useSsl,
	})

	if err != nil {
		log.Fatalf("error initializing minio client: %v", err)
	}

	exists, err := client.BucketExists(context.Background(), cfg.MinioBucketName)
	if err != nil {
		log.Fatalf("erro ao verificar se o bucket existe: %v", err)
	} else {
		log.Printf("Bucket %s existe: %t\n", cfg.MinioBucketName, exists)
	}

	if !exists {
		err = client.MakeBucket(context.Background(), cfg.MinioBucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("erro ao criar bucket: %v", err)
		} else {
			fmt.Printf("Bucket %s criado com sucesso!\n", cfg.MinioBucketName)
		}
	}
}

func getObjectPath(videoType VideoType, objectID string) string {
	return string(videoType) + "/" + objectID
}

func DownloadVideo(videoType VideoType, objectID, destPath string) error {
	ctx := context.Background()
	objectPath := getObjectPath(videoType, objectID)

	// Verifica se o objeto existe antes de tentar baixar
	_, err := client.StatObject(ctx, cfg.MinioBucketName, objectPath, minio.StatObjectOptions{})
	if err != nil {
		log.Printf("[minio] objeto %s não encontrado: %v", objectPath, err)
		return err
	}

	object, err := client.GetObject(ctx, cfg.MinioBucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		log.Printf("[minio] erro ao obter objeto %s: %v", objectPath, err)
		return err
	}
	defer object.Close()
	fmt.Printf("[minio] Download de %s iniciado...\n", objectPath)

	outFile, err := os.Create(destPath)
	if err != nil {
		log.Printf("[minio] erro ao criar arquivo destino %s: %v", destPath, err)
		return err
	}
	defer outFile.Close()
	fmt.Printf("[minio] Salvando arquivo em %s...\n", destPath)

	if _, err := outFile.ReadFrom(object); err != nil {
		log.Printf("[minio] erro ao ler objeto %s: %v", objectPath, err)
		return err
	}
	fmt.Printf("[minio] Download de %s para %s concluído!\n", objectPath, destPath)

	return nil
}

func UploadVideo(srcPath string, videoType VideoType, objectID string) error {
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
	_, err = client.PutObject(ctx, cfg.MinioBucketName, objectPath, file, fileInfo.Size(), minio.PutObjectOptions{ContentType: "video/mp4"})
	if err != nil {
		return fmt.Errorf("[minio] erro ao fazer upload: %w", err)
	}
	fmt.Printf("[minio] Upload de %s para bucket %s concluído!\n", objectPath, cfg.MinioBucketName)
	return nil
}
