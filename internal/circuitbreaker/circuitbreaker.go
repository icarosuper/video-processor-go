// Package circuitbreaker fornece circuit breakers pré-configurados para os
// serviços externos (MinIO e Redis) usados pelo worker.
//
// Um circuit breaker monitora falhas consecutivas em chamadas a um serviço.
// Quando o limiar é atingido, ele "abre" e rejeita chamadas imediatamente,
// evitando cascata de falhas. Após o timeout, permite uma chamada de teste
// ("half-open"); se bem-sucedida, fecha o circuito novamente.
package circuitbreaker

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sony/gobreaker"
)

// MinIO é o circuit breaker para chamadas ao MinIO (download/upload).
var MinIO *gobreaker.CircuitBreaker

// Redis é o circuit breaker para chamadas ao Redis (consumo de fila, estado de job).
var Redis *gobreaker.CircuitBreaker

func init() {
	MinIO = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "minio",
		MaxRequests: 1,                // 1 requisição de teste no estado half-open
		Interval:    30 * time.Second, // janela de contagem de falhas
		Timeout:     60 * time.Second, // tempo em open antes de tentar half-open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Warn().
				Str("service", name).
				Str("from", from.String()).
				Str("to", to.String()).
				Msg("Circuit breaker mudou de estado")
		},
	})

	Redis = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "redis",
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Warn().
				Str("service", name).
				Str("from", from.String()).
				Str("to", to.String()).
				Msg("Circuit breaker mudou de estado")
		},
	})
}
