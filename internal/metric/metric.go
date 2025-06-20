package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metric struct {
	RequestsTotal *prometheus.CounterVec
	LoginSuccess  *prometheus.CounterVec
	LoginFailure  *prometheus.CounterVec
}

func NewMetric() *Metric {
	// init metrics
	requestsTotal := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_requests_total",
		Help: "The total number of requests",
	}, []string{"origin"})

	loginSuccess := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_login_success_total",
		Help: "The total number of successful login attempts",
	}, []string{"email", "domain"})

	loginFailure := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rlsp_login_failure_total",
		Help: "The total number of failed login attempts",
	}, []string{"email", "domain", "reason"})

	return &Metric{
		RequestsTotal: requestsTotal,
		LoginSuccess:  loginSuccess,
		LoginFailure:  loginFailure,
	}
}
