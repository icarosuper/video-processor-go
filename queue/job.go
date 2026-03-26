package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// VideoMetadata espelha os metadados extraídos pela etapa de análise do pipeline.
type VideoMetadata struct {
	Duration   float64 `json:"duration"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	VideoCodec string  `json:"video_codec"`
	AudioCodec string  `json:"audio_codec"`
	FPS        float64 `json:"fps"`
	Bitrate    int64   `json:"bitrate"`
	Size       int64   `json:"size"`
}

// JobStatus representa o estado de um job de processamento.
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusDone       JobStatus = "done"
	JobStatusFailed     JobStatus = "failed"

	// MaxJobRetries é o número máximo de retentativas após a tentativa inicial.
	MaxJobRetries = 3

	jobTTL = 24 * time.Hour
)

// JobArtifacts contém os paths dos artefatos gerados no MinIO.
type JobArtifacts struct {
	Video      string `json:"video,omitempty"`
	Thumbnails string `json:"thumbnails,omitempty"`
	Audio      string `json:"audio,omitempty"`
	Preview    string `json:"preview,omitempty"`
	HLS        string `json:"hls,omitempty"`
}

// JobState representa o estado completo de um job de processamento.
type JobState struct {
	Status      JobStatus      `json:"status"`
	Error       string         `json:"error,omitempty"`
	Artifacts   *JobArtifacts  `json:"artifacts,omitempty"`
	Metadata    *VideoMetadata `json:"metadata,omitempty"`
	RetryCount  int            `json:"retry_count"`
	CallbackURL string         `json:"callback_url,omitempty"`
	CreatedAt   int64          `json:"created_at"`
	UpdatedAt   int64          `json:"updated_at"`
}

func jobKey(videoID string) string {
	return "job:" + videoID
}

func setJobState(videoID string, state JobState) error {
	state.UpdatedAt = time.Now().Unix()
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("erro ao serializar estado do job: %w", err)
	}
	return client.Set(context.Background(), jobKey(videoID), string(data), jobTTL).Err()
}

// PublishJob publica um videoID na fila e registra o estado inicial como pending.
// callbackURL é opcional: se não vazio, o worker notificará essa URL ao concluir.
// Deve ser chamado pelo produtor (API) ao submeter um vídeo para processamento.
func PublishJob(videoID, callbackURL string) error {
	state := JobState{
		Status:      JobStatusPending,
		CallbackURL: callbackURL,
		CreatedAt:   time.Now().Unix(),
	}
	if err := setJobState(videoID, state); err != nil {
		return fmt.Errorf("erro ao criar estado do job: %w", err)
	}
	return client.LPush(context.Background(), cfg.ProcessingRequestQueue, videoID).Err()
}

// SetJobProcessing atualiza o estado do job para processing.
func SetJobProcessing(videoID string) error {
	existing, _ := GetJobState(videoID)
	if existing == nil {
		existing = &JobState{CreatedAt: time.Now().Unix()}
	}
	existing.Status = JobStatusProcessing
	return setJobState(videoID, *existing)
}

// SetJobDone atualiza o estado do job para done com os artefatos e metadados gerados.
func SetJobDone(videoID string, artifacts JobArtifacts, metadata *VideoMetadata) error {
	existing, _ := GetJobState(videoID)
	if existing == nil {
		existing = &JobState{CreatedAt: time.Now().Unix()}
	}
	existing.Status = JobStatusDone
	existing.Artifacts = &artifacts
	existing.Metadata = metadata
	existing.Error = ""
	return setJobState(videoID, *existing)
}

// SetJobFailed atualiza o estado do job para failed, incrementa o contador de tentativas
// e retorna o estado atualizado para que o chamador decida entre retry e DLQ.
func SetJobFailed(videoID string, jobErr error) (*JobState, error) {
	existing, _ := GetJobState(videoID)
	if existing == nil {
		existing = &JobState{CreatedAt: time.Now().Unix()}
	}
	existing.Status = JobStatusFailed
	existing.Error = jobErr.Error()
	existing.RetryCount++
	if err := setJobState(videoID, *existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// RequeueJob recoloca o job na fila principal para reprocessamento.
// AcknowledgeMessage ainda deve ser chamado para remover da fila de processamento.
func RequeueJob(videoID string) error {
	existing, _ := GetJobState(videoID)
	if existing != nil {
		existing.Status = JobStatusPending
		if err := setJobState(videoID, *existing); err != nil {
			return fmt.Errorf("erro ao atualizar estado para requeue: %w", err)
		}
	}
	return client.LPush(context.Background(), cfg.ProcessingRequestQueue, videoID).Err()
}

// MoveToDLQ move o job para a dead letter queue após esgotar as tentativas.
// AcknowledgeMessage ainda deve ser chamado para remover da fila de processamento.
func MoveToDLQ(videoID string) error {
	return client.LPush(context.Background(), deadLetterQueueName(), videoID).Err()
}

// GetJobState retorna o estado atual de um job. Retorna nil se o job não existir.
func GetJobState(videoID string) (*JobState, error) {
	data, err := client.Get(context.Background(), jobKey(videoID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("job não encontrado: %w", err)
	}
	var state JobState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("erro ao deserializar estado do job: %w", err)
	}
	return &state, nil
}
