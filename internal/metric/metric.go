package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metric struct {
	RequestsTotal     *prometheus.CounterVec
	ResponseTime      *prometheus.HistogramVec
	ResponseStatus    *prometheus.CounterVec
	RateLimitHits     *prometheus.CounterVec
	ActiveConnections *prometheus.GaugeVec
}

func NewMetric() *Metric {
	// init metrics
	requestsTotal := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_requests_total",
		Help: "The total number of requests",
	}, []string{"origin"})

	// Optimized buckets for proxy response times
	responseTime := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rlsp_response_time_seconds",
		Help:    "Response time in seconds",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2, 5}, // Proxy-optimized buckets
	}, []string{"origin"})

	responseStatus := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_response_status_total",
		Help: "The total number of responses by HTTP status code",
	}, []string{"origin", "status"})

	rateLimitHits := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_rate_limit_hits_total",
		Help: "The total number of rate limit hits",
	}, []string{"origin", "ip"})

	activeConnections := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rlsp_active_connections",
		Help: "The number of active connections",
	}, []string{"origin"})

	return &Metric{
		RequestsTotal:     requestsTotal,
		ResponseTime:      responseTime,
		ResponseStatus:    responseStatus,
		RateLimitHits:     rateLimitHits,
		ActiveConnections: activeConnections,
	}
}
