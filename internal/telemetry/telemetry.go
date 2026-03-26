// Package telemetry inicializa o tracing distribuído com OpenTelemetry.
//
// Se OTEL_ENDPOINT não estiver configurado, um tracer no-op é usado
// e nenhuma dependência de rede é criada — zero overhead em produção
// sem Jaeger/OTLP configurado.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const TracerName = "video-processor"

// Init inicializa o provider de tracing e retorna uma função de shutdown.
// Se endpoint estiver vazio, instala um tracer no-op e retorna imediatamente.
func Init(ctx context.Context, serviceName, endpoint string) (shutdown func(context.Context) error, err error) {
	if endpoint == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar exporter OTLP: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar resource OTel: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(shutdownCtx)
	}, nil
}

// Tracer retorna o tracer do pacote para criação de spans.
func Tracer() trace.Tracer {
	return otel.Tracer(TracerName)
}
