package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func init() {
	sleepFn = func(time.Duration) {} // noop: evita backoff real nos testes
}

func TestNotify_Sucesso(t *testing.T) {
	var recebido Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&recebido) //nolint:errcheck
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := Payload{VideoID: "video-123", Status: "done"}
	if err := Notify(srv.URL, "", payload); err != nil {
		t.Fatalf("Notify() não deveria retornar erro: %v", err)
	}
	if recebido.VideoID != "video-123" {
		t.Fatalf("esperava VideoID 'video-123', got '%s'", recebido.VideoID)
	}
}

func TestNotify_ContentTypeJSON(t *testing.T) {
	var contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	Notify(srv.URL, "", Payload{VideoID: "v1", Status: "done"}) //nolint:errcheck

	if contentType != "application/json" {
		t.Fatalf("esperava Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestNotify_ComHMAC_AssinaturaCorreta(t *testing.T) {
	secret := "meu-segredo"
	var assinatura string
	var bodyRecebido []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assinatura = r.Header.Get("X-Webhook-Signature")
		bodyRecebido = make([]byte, r.ContentLength)
		r.Body.Read(bodyRecebido) //nolint:errcheck
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := Payload{VideoID: "video-123", Status: "done"}
	if err := Notify(srv.URL, secret, payload); err != nil {
		t.Fatalf("Notify() não deveria retornar erro: %v", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(bodyRecebido)
	esperado := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if assinatura != esperado {
		t.Fatalf("assinatura HMAC incorreta\nesperado:  %s\nrecebido: %s", esperado, assinatura)
	}
}

func TestNotify_SemSecret_SemHeader(t *testing.T) {
	var assinatura string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assinatura = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	Notify(srv.URL, "", Payload{VideoID: "v1", Status: "done"}) //nolint:errcheck

	if assinatura != "" {
		t.Fatalf("não deveria enviar X-Webhook-Signature sem secret, got: '%s'", assinatura)
	}
}

func TestNotify_RetryEmFalha(t *testing.T) {
	tentativas := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tentativas++
		if tentativas < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := Notify(srv.URL, "", Payload{VideoID: "v1", Status: "done"}); err != nil {
		t.Fatalf("Notify() deveria ter sucesso na 3ª tentativa, got: %v", err)
	}
	if tentativas != 3 {
		t.Fatalf("esperava 3 tentativas, got %d", tentativas)
	}
}

func TestNotify_ErroApos3Tentativas(t *testing.T) {
	tentativas := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tentativas++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	err := Notify(srv.URL, "", Payload{VideoID: "v1", Status: "done"})
	if err == nil {
		t.Fatal("Notify() deveria retornar erro após 3 tentativas fracassadas")
	}
	if tentativas != 3 {
		t.Fatalf("esperava 3 tentativas, got %d", tentativas)
	}
	if !strings.Contains(err.Error(), "3 tentativas") {
		t.Fatalf("mensagem de erro deveria mencionar '3 tentativas', got: %s", err.Error())
	}
}

func TestNotify_URLInvalida(t *testing.T) {
	err := Notify("://url-invalida", "", Payload{VideoID: "v1", Status: "done"})
	if err == nil {
		t.Fatal("Notify() deveria retornar erro com URL inválida")
	}
}

func TestNotify_ServidorIndisponivel(t *testing.T) {
	// Porta que não tem ninguém ouvindo
	err := Notify("http://localhost:19999", "", Payload{VideoID: "v1", Status: "done"})
	if err == nil {
		t.Fatal("Notify() deveria retornar erro quando servidor está indisponível")
	}
}

func TestPayload_SerializacaoJSON(t *testing.T) {
	p := Payload{
		VideoID:   "abc",
		Status:    "done",
		Error:     "",
		Artifacts: map[string]string{"video": "processed/abc"},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("falha ao serializar payload: %v", err)
	}

	// campo Error omitempty — não deve aparecer quando vazio
	if strings.Contains(string(data), `"error"`) {
		t.Fatalf("campo 'error' não deveria aparecer quando vazio, got: %s", data)
	}
	if !strings.Contains(string(data), `"video_id":"abc"`) {
		t.Fatalf("campo 'video_id' deveria aparecer, got: %s", data)
	}
}
