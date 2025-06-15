package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metric struct {
	RequestsTotal *prometheus.CounterVec
}

func NewMetric() *Metric {
	// init metrics
	requestsTotal := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_requests_total",
		Help: "The total number of requests",
	}, []string{"origin"})

	return &Metric{
		RequestsTotal: requestsTotal,
	}
}
