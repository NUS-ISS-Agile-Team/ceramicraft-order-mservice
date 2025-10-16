package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// 请求总数（用于吞吐量、QPS）
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"method", "path"},
	)

	// 请求耗时（用于P95、P99）
	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response latency (seconds) of HTTP requests.",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000}, // 5ms~5s
		},
		[]string{"method", "path"},
	)

	// 错误总数
	HttpRequestsErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_errors_total",
			Help: "Total number of error responses (status >= 400).",
		},
		[]string{"method", "path", "status"},
	)
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal, HttpRequestDuration, HttpRequestsErrors)
}
