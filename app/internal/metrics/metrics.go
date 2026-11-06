// Package metrics defines the Prometheus metrics exposed by the service.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	reg           *prometheus.Registry
	CheckTotal    *prometheus.CounterVec
	CheckDuration *prometheus.HistogramVec
	TargetUp      *prometheus.GaugeVec
	HTTPRequests  *prometheus.CounterVec
	HTTPDuration  *prometheus.HistogramVec
}

func New() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	f := promauto.With(reg)
	return &Metrics{
		reg: reg,
		CheckTotal: f.NewCounterVec(prometheus.CounterOpts{
			Name: "sentinel_checks_total",
			Help: "Health checks performed, by target and result.",
		}, []string{"target", "result"}),
		CheckDuration: f.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "sentinel_check_duration_seconds",
			Help:    "Latency of target health checks.",
			Buckets: []float64{.025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}, []string{"target"}),
		TargetUp: f.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sentinel_target_up",
			Help: "1 if the last check of the target succeeded, else 0.",
		}, []string{"target"}),
		HTTPRequests: f.NewCounterVec(prometheus.CounterOpts{
			Name: "sentinel_http_requests_total",
			Help: "HTTP requests served by the API.",
		}, []string{"handler", "method", "status"}),
		HTTPDuration: f.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "sentinel_http_request_duration_seconds",
			Help:    "API request latency.",
			Buckets: prometheus.DefBuckets,
		}, []string{"handler", "method"}),
	}
}

// Handler serves /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}
