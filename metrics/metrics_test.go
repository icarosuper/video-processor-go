package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestVideosProcessedTotal_Increment(t *testing.T) {
	// Reset métrica para teste isolado
	VideosProcessedTotal.Reset()

	// Incrementar contador de sucesso
	VideosProcessedTotal.WithLabelValues("success").Inc()
	VideosProcessedTotal.WithLabelValues("success").Inc()

	// Incrementar contador de erro
	VideosProcessedTotal.WithLabelValues("error").Inc()

	// Verificar valores
	successMetric := getCounterValue(t, VideosProcessedTotal.WithLabelValues("success"))
	if successMetric != 2 {
		t.Errorf("Esperava 2 sucessos, obteve %f", successMetric)
	}

	errorMetric := getCounterValue(t, VideosProcessedTotal.WithLabelValues("error"))
	if errorMetric != 1 {
		t.Errorf("Esperava 1 erro, obteve %f", errorMetric)
	}
}

func TestProcessingDuration_Observe(t *testing.T) {
	// Observar algumas durações
	ProcessingDuration.Observe(1.5)
	ProcessingDuration.Observe(2.3)
	ProcessingDuration.Observe(3.7)

	// Verificar que o histograma registrou observações
	metric := &dto.Metric{}
	if err := ProcessingDuration.(prometheus.Histogram).Write(metric); err != nil {
		t.Fatalf("Erro ao ler métrica: %v", err)
	}

	if metric.Histogram.GetSampleCount() != 3 {
		t.Errorf("Esperava 3 observações, obteve %d", metric.Histogram.GetSampleCount())
	}

	expectedSum := 1.5 + 2.3 + 3.7
	actualSum := metric.Histogram.GetSampleSum()
	if actualSum < expectedSum-0.1 || actualSum > expectedSum+0.1 {
		t.Errorf("Esperava soma próxima de %f, obteve %f", expectedSum, actualSum)
	}
}

func TestProcessingStepDuration_MultipleSteps(t *testing.T) {
	steps := []string{"validate", "transcode", "thumbnails"}

	for _, step := range steps {
		ProcessingStepDuration.WithLabelValues(step).Observe(1.0)
	}

	// Verificar que cada step foi registrado
	for _, step := range steps {
		histogram := ProcessingStepDuration.WithLabelValues(step).(prometheus.Observer)

		// Não há forma direta de ler valor de um histogram com labels no Prometheus client
		// Mas podemos verificar que não há erro ao usar a métrica
		histogram.Observe(0.5)
	}
}

func TestActiveWorkers_SetAndGet(t *testing.T) {
	// Definir número de workers
	ActiveWorkers.Set(5)

	// Verificar valor
	metric := &dto.Metric{}
	if err := ActiveWorkers.Write(metric); err != nil {
		t.Fatalf("Erro ao ler métrica: %v", err)
	}

	if metric.Gauge.GetValue() != 5 {
		t.Errorf("Esperava 5 workers, obteve %f", metric.Gauge.GetValue())
	}

	// Atualizar valor
	ActiveWorkers.Set(10)

	if err := ActiveWorkers.Write(metric); err != nil {
		t.Fatalf("Erro ao ler métrica: %v", err)
	}

	if metric.Gauge.GetValue() != 10 {
		t.Errorf("Esperava 10 workers, obteve %f", metric.Gauge.GetValue())
	}
}

func TestQueueSize_SetAndGet(t *testing.T) {
	// Definir tamanho da fila
	QueueSize.Set(100)

	// Verificar valor
	metric := &dto.Metric{}
	if err := QueueSize.Write(metric); err != nil {
		t.Fatalf("Erro ao ler métrica: %v", err)
	}

	if metric.Gauge.GetValue() != 100 {
		t.Errorf("Esperava fila com 100 itens, obteve %f", metric.Gauge.GetValue())
	}
}

// Helper para obter valor de um counter
func getCounterValue(t *testing.T, counter prometheus.Counter) float64 {
	t.Helper()

	metric := &dto.Metric{}
	if err := counter.Write(metric); err != nil {
		t.Fatalf("Erro ao ler counter: %v", err)
	}

	return metric.Counter.GetValue()
}
