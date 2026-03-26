package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// JobStatus representa o estado de um job de processamento.
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusDone       JobStatus = "done"
	JobStatusFailed     JobStatus = "failed"

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
	Status    JobStatus     `json:"status"`
	Error     string        `json:"error,omitempty"`
	Artifacts *JobArtifacts `json:"artifacts,omitempty"`
	CreatedAt int64         `json:"created_at"`
	UpdatedAt int64         `json:"updated_at"`
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
// Deve ser chamado pelo produtor (API) ao submeter um vídeo para processamento.
func PublishJob(videoID string) error {
	state := JobState{
		Status:    JobStatusPending,
		CreatedAt: time.Now().Unix(),
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

// SetJobDone atualiza o estado do job para done com os artefatos gerados.
func SetJobDone(videoID string, artifacts JobArtifacts) error {
	existing, _ := GetJobState(videoID)
	if existing == nil {
		existing = &JobState{CreatedAt: time.Now().Unix()}
	}
	existing.Status = JobStatusDone
	existing.Artifacts = &artifacts
	existing.Error = ""
	return setJobState(videoID, *existing)
}

// SetJobFailed atualiza o estado do job para failed com a mensagem de erro.
func SetJobFailed(videoID string, jobErr error) error {
	existing, _ := GetJobState(videoID)
	if existing == nil {
		existing = &JobState{CreatedAt: time.Now().Unix()}
	}
	existing.Status = JobStatusFailed
	existing.Error = jobErr.Error()
	return setJobState(videoID, *existing)
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
