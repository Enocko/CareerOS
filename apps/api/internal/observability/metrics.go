package observability

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/careeros/api/internal/ingestion"
	"github.com/careeros/api/internal/jobs"
	"github.com/careeros/api/internal/notifications"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "careeros_http_requests_total",
			Help: "Total HTTP requests by method, route pattern, and status class.",
		},
		[]string{"method", "route", "status_class"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "careeros_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route"},
	)
	httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "careeros_http_requests_in_flight",
			Help: "Number of HTTP requests currently being served.",
		},
	)

	backgroundJobsQueued = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_background_jobs_queued",
		Help: "Background jobs in queued status.",
	})
	backgroundJobsProcessing = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_background_jobs_processing",
		Help: "Background jobs in processing status.",
	})
	backgroundJobsRetryable = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_background_jobs_retryable",
		Help: "Background jobs awaiting retry.",
	})
	backgroundJobsFailed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_background_jobs_failed",
		Help: "Background jobs permanently failed.",
	})
	backgroundJobsCompleted = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_background_jobs_completed_total",
		Help: "Background jobs completed (cumulative count).",
	})
	backgroundJobRetries = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_background_job_retries_total",
		Help: "Cumulative retry attempts across all jobs.",
	})
	backgroundJobsOldestQueuedAge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_background_jobs_oldest_queued_age_seconds",
		Help: "Age in seconds of the oldest queued or retryable job.",
	})
	notificationsCreated = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_notifications_created_total",
		Help: "Total in-app notifications created.",
	})
	ingestionSuccessTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_ingestion_success_total",
		Help: "Count of enabled sources with a recorded successful ingestion.",
	})
	ingestionFailureTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "careeros_ingestion_failure_total",
		Help: "Count of enabled sources whose latest run failed.",
	})
	recommendationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "careeros_recommendation_duration_seconds",
		Help:    "Recommendation scoring duration in seconds (handler-measured).",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
	})

	registry = prometheus.NewRegistry()
	once     sync.Once
)

func initRegistry() {
	once.Do(func() {
		registry.MustRegister(
			httpRequestsTotal,
			httpRequestDuration,
			httpRequestsInFlight,
			backgroundJobsQueued,
			backgroundJobsProcessing,
			backgroundJobsRetryable,
			backgroundJobsFailed,
			backgroundJobsCompleted,
			backgroundJobRetries,
			backgroundJobsOldestQueuedAge,
			notificationsCreated,
			ingestionSuccessTotal,
			ingestionFailureTotal,
			recommendationDuration,
			prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
			prometheus.NewGoCollector(),
		)
	})
}

// RecordRecommendationDuration records recommendation handler latency.
func RecordRecommendationDuration(seconds float64) {
	initRegistry()
	recommendationDuration.Observe(seconds)
}

// MetricsHandler serves Prometheus metrics and refreshes domain gauges on scrape.
func MetricsHandler(pool *pgxpool.Pool) http.Handler {
	initRegistry()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshDomainMetrics(r.Context(), pool)
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})
}

func refreshDomainMetrics(ctx context.Context, pool *pgxpool.Pool) {
	now := time.Now().UTC()
	jobsRepo := jobs.NewRepository(pool)
	if m, err := jobsRepo.Metrics(ctx, now); err == nil {
		backgroundJobsQueued.Set(float64(m.Queued))
		backgroundJobsProcessing.Set(float64(m.Processing))
		backgroundJobsRetryable.Set(float64(m.Retryable))
		backgroundJobsFailed.Set(float64(m.Failed))
		backgroundJobsCompleted.Set(float64(m.Completed))
		backgroundJobRetries.Set(float64(m.TotalRetries))
		backgroundJobsOldestQueuedAge.Set(m.OldestQueuedAge.Seconds())
	}

	notifRepo := notifications.NewRepository(pool)
	if count, err := notifRepo.CountCreated(ctx); err == nil {
		notificationsCreated.Set(float64(count))
	}

	ingestRepo := ingestion.NewRepository(pool)
	if stats, err := ingestRepo.IngestionSourceHealth(ctx); err == nil {
		ingestionSuccessTotal.Set(float64(stats.SuccessSources))
		ingestionFailureTotal.Set(float64(stats.FailedSources))
	}
}

// HTTPMetricsMiddleware records request count, latency, and in-flight gauge.
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	initRegistry()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpRequestsInFlight.Inc()
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		route := routePattern(r)
		statusClass := strconv.Itoa(wrapped.status/100) + "xx"
		httpRequestsTotal.WithLabelValues(r.Method, route, statusClass).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		httpRequestsInFlight.Dec()
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := rc.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return r.URL.Path
}
