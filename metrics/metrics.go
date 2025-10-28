// mettics add measuaring functionality
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// RequestTotal stores number of all request
	RequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_total",
			Help: "Total number of requests",
		},
		[]string{"path"},
	)

	// RequestDuration stores time for work with 1 request
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Duration of request",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path"},
	)

	ErrorRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_request_total",
			Help: "Total number of error request",
		},
		[]string{"path"},
	)
)

// InitMetrics() initialize metrics
func InitMetrics() {
	sync.OnceFunc(func() {
		prometheus.MustRegister(RequestDuration, RequestTotal)
	})()
}
