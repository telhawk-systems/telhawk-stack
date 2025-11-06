package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Event ingestion metrics
	EventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telhawk_ingest_events_total",
			Help: "Total number of events received",
		},
		[]string{"endpoint", "status"},
	)

	EventBytesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "telhawk_ingest_event_bytes_total",
			Help: "Total bytes of event data received",
		},
	)

	// Queue metrics
	QueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "telhawk_ingest_queue_depth",
			Help: "Current depth of the event queue",
		},
	)

	QueueCapacity = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "telhawk_ingest_queue_capacity",
			Help: "Maximum capacity of the event queue",
		},
	)

	// Normalization metrics
	NormalizationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "telhawk_ingest_normalization_duration_seconds",
			Help:    "Duration of event normalization in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	NormalizationErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "telhawk_ingest_normalization_errors_total",
			Help: "Total number of normalization errors",
		},
	)

	// Storage metrics
	StorageDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "telhawk_ingest_storage_duration_seconds",
			Help:    "Duration of storage operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	StorageErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "telhawk_ingest_storage_errors_total",
			Help: "Total number of storage errors",
		},
	)

	// Rate limiting metrics
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telhawk_ingest_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"token"},
	)

	// HEC ack metrics
	AcksPending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "telhawk_ingest_acks_pending",
			Help: "Number of pending acknowledgements",
		},
	)

	AcksCompleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "telhawk_ingest_acks_completed_total",
			Help: "Total number of completed acknowledgements",
		},
	)
)
