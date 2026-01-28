package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// VideosProcessedTotal conta o número total de vídeos processados
	VideosProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "videos_processed_total",
			Help: "Total number of videos processed",
		},
		[]string{"status"}, // success ou error
	)

	// ProcessingDuration mede o tempo de processamento de vídeos
	ProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "video_processing_duration_seconds",
			Help:    "Time taken to process videos in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// ProcessingStepDuration mede o tempo de cada etapa do pipeline
	ProcessingStepDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "video_processing_step_duration_seconds",
			Help:    "Time taken for each processing step in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
		},
		[]string{"step"}, // validate, transcode, thumbnail, etc.
	)

	// ActiveWorkers conta o número de workers ativos
	ActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_workers",
			Help: "Number of currently active workers",
		},
	)

	// QueueSize mede o tamanho da fila
	QueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "queue_size",
			Help: "Current size of the processing queue",
		},
	)

	// VideoSizeBytes mede o tamanho dos vídeos processados
	VideoSizeBytes = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "video_size_bytes",
			Help:    "Size of processed videos in bytes",
			Buckets: prometheus.ExponentialBuckets(1024*1024, 2, 15), // 1MB to ~16GB
		},
	)
)
