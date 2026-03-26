package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker"
)

var errFalha = errors.New("falha simulada")

func TestMinIO_EstadoInicial_Fechado(t *testing.T) {
	chamadas := 0
	_, err := MinIO.Execute(func() (interface{}, error) {
		chamadas++
		return nil, nil
	})
	if err != nil {
		t.Fatalf("circuito MinIO deveria estar fechado, mas retornou erro: %v", err)
	}
	if chamadas != 1 {
		t.Fatalf("esperava 1 chamada, got %d", chamadas)
	}
}

func TestRedis_EstadoInicial_Fechado(t *testing.T) {
	chamadas := 0
	_, err := Redis.Execute(func() (interface{}, error) {
		chamadas++
		return nil, nil
	})
	if err != nil {
		t.Fatalf("circuito Redis deveria estar fechado, mas retornou erro: %v", err)
	}
	if chamadas != 1 {
		t.Fatalf("esperava 1 chamada, got %d", chamadas)
	}
}

func TestCircuitBreaker_AbreApos5FalhasConsecutivas(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "test-minio",
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})

	for i := 0; i < 5; i++ {
		cb.Execute(func() (interface{}, error) { //nolint:errcheck
			return nil, errFalha
		})
	}

	if cb.State() != gobreaker.StateOpen {
		t.Fatalf("circuito deveria estar aberto após 5 falhas, estado: %s", cb.State())
	}
}

func TestCircuitBreaker_AbreApos3FalhasConsecutivas_Redis(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "test-redis",
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) { //nolint:errcheck
			return nil, errFalha
		})
	}

	if cb.State() != gobreaker.StateOpen {
		t.Fatalf("circuito Redis deveria estar aberto após 3 falhas, estado: %s", cb.State())
	}
}

func TestCircuitBreaker_RejeitaChamadasQuandoAberto(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "test-open",
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) { //nolint:errcheck
			return nil, errFalha
		})
	}

	_, err := cb.Execute(func() (interface{}, error) {
		return nil, nil
	})

	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("esperava ErrOpenState, got: %v", err)
	}
}

func TestCircuitBreaker_NaoAbreComFalhasNaoConsecutivas(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "test-intermitente",
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})

	// 4 falhas intercaladas com 1 sucesso — não deve abrir
	for i := 0; i < 4; i++ {
		cb.Execute(func() (interface{}, error) { //nolint:errcheck
			return nil, errFalha
		})
		cb.Execute(func() (interface{}, error) { //nolint:errcheck
			return "ok", nil
		})
	}

	if cb.State() != gobreaker.StateClosed {
		t.Fatalf("circuito não deveria abrir com falhas não consecutivas, estado: %s", cb.State())
	}
}

func TestCircuitBreaker_RetornaResultadoQuandoFechado(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name: "test-resultado",
	})

	resultado, err := cb.Execute(func() (interface{}, error) {
		return "valor-esperado", nil
	})

	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}
	if resultado != "valor-esperado" {
		t.Fatalf("esperava 'valor-esperado', got: %v", resultado)
	}
}
