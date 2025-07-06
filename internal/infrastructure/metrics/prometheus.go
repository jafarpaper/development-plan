package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	ActivityLogCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "activity_log_created_total",
			Help: "Total number of activity logs created",
		},
		[]string{"company_id", "activity_name", "status"},
	)

	ActivityLogProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "activity_log_processing_duration_seconds",
			Help:    "Duration of activity log processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "status"},
	)

	NATSMessageProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_message_processed_total",
			Help: "Total number of NATS messages processed",
		},
		[]string{"subject", "status"},
	)

	ArangoDBOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "arango_db_operation_duration_seconds",
			Help:    "Duration of ArangoDB operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "status"},
	)

	JSONFileOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "json_file_operation_duration_seconds",
			Help:    "Duration of JSON file operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "status"},
	)

	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	GRPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)
)

func StartMetricsServer(port int, logger *logrus.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}

	logger.WithField("port", port).Info("Starting metrics server")

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("Metrics server failed")
		}
	}()
}

func RecordActivityLogCreated(companyID, activityName, status string) {
	ActivityLogCreatedTotal.WithLabelValues(companyID, activityName, status).Inc()
}

func RecordActivityLogProcessingDuration(operation, status string, duration time.Duration) {
	ActivityLogProcessingDuration.WithLabelValues(operation, status).Observe(duration.Seconds())
}

func RecordNATSMessageProcessed(subject, status string) {
	NATSMessageProcessedTotal.WithLabelValues(subject, status).Inc()
}

func RecordArangoDBOperationDuration(operation, status string, duration time.Duration) {
	ArangoDBOperationDuration.WithLabelValues(operation, status).Observe(duration.Seconds())
}

func RecordJSONFileOperationDuration(operation, status string, duration time.Duration) {
	JSONFileOperationDuration.WithLabelValues(operation, status).Observe(duration.Seconds())
}

func RecordGRPCRequest(method, status string, duration time.Duration) {
	GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	GRPCRequestDuration.WithLabelValues(method, status).Observe(duration.Seconds())
}
