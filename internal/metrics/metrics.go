package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Requests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whoiscache_requests_total",
			Help: "Total WHOIS requests",
		},
		[]string{"kind", "result"},
	)
	Errors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "whoiscache_errors_total",
			Help: "Total errors by stage",
		},
		[]string{"stage"},
	)
	Duration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "whoiscache_request_duration_seconds",
			Help:    "WHOIS request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"kind"},
	)
)

func Handler() http.Handler {
	return promhttp.Handler()
}
