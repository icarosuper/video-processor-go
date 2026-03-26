package minio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"video-processor/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
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
		log.Fatal().Err(err).Msg("Erro ao inicializar cliente MinIO")
	}

	exists, err := client.BucketExists(context.Background(), cfg.MinioBucketName)
	if err != nil {
		log.Fatal().Err(err).Str("bucket", cfg.MinioBucketName).Msg("Erro ao verificar se bucket existe")
	}

	log.Info().Str("bucket", cfg.MinioBucketName).Bool("exists", exists).Msg("Status do bucket MinIO")

	if !exists {
		err = client.MakeBucket(context.Background(), cfg.MinioBucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatal().Err(err).Str("bucket", cfg.MinioBucketName).Msg("Erro ao criar bucket")
		}
		log.Info().Str("bucket", cfg.MinioBucketName).Msg("Bucket criado com sucesso")
	}
}

func getObjectPath(videoType VideoType, objectID string) string {
	return string(videoType) + "/" + objectID
}

func DownloadVideo(videoType VideoType, objectID, destPath string) error {
	ctx := context.Background()
	objectPath := getObjectPath(videoType, objectID)

	// Verifica se o objeto existe e valida o tamanho antes de baixar
	info, err := client.StatObject(ctx, cfg.MinioBucketName, objectPath, minio.StatObjectOptions{})
	if err != nil {
		log.Error().Err(err).Str("object", objectPath).Msg("Objeto não encontrado")
		return err
	}
	if maxBytes := cfg.MaxFileSizeMB * 1024 * 1024; info.Size > maxBytes {
		return fmt.Errorf("vídeo muito grande: %.0fMB (máximo: %dMB)", float64(info.Size)/1024/1024, cfg.MaxFileSizeMB)
	}

	object, err := client.GetObject(ctx, cfg.MinioBucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		log.Error().Err(err).Str("object", objectPath).Msg("Erro ao obter objeto")
		return err
	}
	defer object.Close()
	log.Info().Str("object", objectPath).Msg("Download iniciado")

	outFile, err := os.Create(destPath)
	if err != nil {
		log.Error().Err(err).Str("destPath", destPath).Msg("Erro ao criar arquivo destino")
		return err
	}
	defer outFile.Close()

	if _, err := outFile.ReadFrom(object); err != nil {
		log.Error().Err(err).Str("object", objectPath).Msg("Erro ao ler objeto")
		return err
	}
	log.Info().Str("object", objectPath).Str("destPath", destPath).Msg("Download concluído")

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
		return fmt.Errorf("erro ao fazer upload: %w", err)
	}
	log.Info().Str("object", objectPath).Str("bucket", cfg.MinioBucketName).Int64("size", fileInfo.Size()).Msg("Upload concluído")
	return nil
}

// UploadFile faz upload de qualquer arquivo para um caminho específico no MinIO.
func UploadFile(srcPath, objectPath string) error {
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

	_, err = client.PutObject(ctx, cfg.MinioBucketName, objectPath, file, fileInfo.Size(),
		minio.PutObjectOptions{ContentType: contentTypeByExt(filepath.Ext(srcPath))})
	if err != nil {
		return fmt.Errorf("erro ao fazer upload de %s: %w", objectPath, err)
	}
	log.Info().Str("object", objectPath).Int64("size", fileInfo.Size()).Msg("Upload concluído")
	return nil
}

// UploadDirectory faz upload de todos os arquivos (não-recursivo) de srcDir
// para o MinIO com o prefixo objectPrefix.
func UploadDirectory(srcDir, objectPrefix string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("[minio] erro ao ler diretório %s: %w", srcDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		objectPath := objectPrefix + "/" + entry.Name()
		if err := UploadFile(srcPath, objectPath); err != nil {
			return err
		}
	}
	return nil
}

func contentTypeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".ts":
		return "video/MP2T"
	case ".m3u8":
		return "application/x-mpegURL"
	default:
		return "application/octet-stream"
	}
}

// HealthCheck verifica se o cliente MinIO está saudável
func HealthCheck() error {
	ctx := context.Background()
	_, err := client.BucketExists(ctx, cfg.MinioBucketName)
	return err
}
