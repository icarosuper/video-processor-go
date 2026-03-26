package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Payload é o corpo enviado ao callback da API quando um job termina.
type Payload struct {
	VideoID   string      `json:"video_id"`
	Status    string      `json:"status"`
	Error     string      `json:"error,omitempty"`
	Artifacts interface{} `json:"artifacts,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Notify envia o payload ao callbackURL com até 3 tentativas e backoff exponencial.
// Se secret não for vazio, assina o body com HMAC-SHA256 no header X-Webhook-Signature.
// Retorna erro apenas se todas as tentativas falharem.
func Notify(callbackURL, secret string, payload Payload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload do webhook: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := send(callbackURL, secret, body); err != nil {
			lastErr = err
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*attempt) * time.Second)
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("webhook falhou após 3 tentativas: %w", lastErr)
}

func send(url, secret string, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook retornou status inesperado: %d", resp.StatusCode)
	}
	return nil
}
