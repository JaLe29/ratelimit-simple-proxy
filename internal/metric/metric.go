package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metric struct {
	RequestsTotal  *prometheus.CounterVec
	ResponseTime   *prometheus.HistogramVec
	ResponseStatus *prometheus.CounterVec
}

func NewMetric() *Metric {
	// init metrics
	requestsTotal := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_requests_total",
		Help: "The total number of requests",
	}, []string{"origin"})

	responseTime := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rlsp_response_time_seconds",
		Help:    "Response time in seconds",
		Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
	}, []string{"origin"})

	responseStatus := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_response_status_total",
		Help: "The total number of responses by HTTP status code",
	}, []string{"origin", "status"})

	return &Metric{
		RequestsTotal:  requestsTotal,
		ResponseTime:   responseTime,
		ResponseStatus: responseStatus,
	}
}
