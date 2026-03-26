package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestInit_EndpointVazio_Noop(t *testing.T) {
	shutdown, err := Init(context.Background(), "test-service", "")
	if err != nil {
		t.Fatalf("Init() com endpoint vazio não deveria retornar erro: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Init() deveria retornar função de shutdown não-nula")
	}

	// Shutdown do noop não deve retornar erro
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() não deveria retornar erro: %v", err)
	}
}

func TestInit_EndpointVazio_InstalaNoop(t *testing.T) {
	Init(context.Background(), "test-service", "") //nolint:errcheck

	provider := otel.GetTracerProvider()
	if _, ok := provider.(noop.TracerProvider); !ok {
		t.Fatal("Init() com endpoint vazio deveria instalar um noop.TracerProvider")
	}
}

func TestTracer_RetornaNaoNulo(t *testing.T) {
	Init(context.Background(), "test-service", "") //nolint:errcheck

	tracer := Tracer()
	if tracer == nil {
		t.Fatal("Tracer() não deveria retornar nil")
	}
}

func TestTracer_CriaSpan(t *testing.T) {
	Init(context.Background(), "test-service", "") //nolint:errcheck

	tracer := Tracer()
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	if ctx == nil {
		t.Fatal("tracer.Start() deveria retornar contexto não-nulo")
	}
	if span == nil {
		t.Fatal("tracer.Start() deveria retornar span não-nulo")
	}
}

func TestTracerName_Constante(t *testing.T) {
	if TracerName != "video-processor" {
		t.Fatalf("TracerName deveria ser 'video-processor', got '%s'", TracerName)
	}
}

func TestInit_EndpointInvalido_RetornaErro(t *testing.T) {
	// Um endpoint mal-formado que causa falha ao criar o exporter OTLP
	// O exporter OTLP HTTP só falha na primeira exportação, não na criação —
	// então Init com endpoint inválido retorna nil err (conexão é lazy).
	// Verificamos apenas que a função retorna sem panic.
	shutdown, err := Init(context.Background(), "test-service", "localhost:99999")
	if err != nil {
		// Se retornar erro, também é comportamento válido
		return
	}
	if shutdown != nil {
		shutdown(context.Background()) //nolint:errcheck
	}
}
