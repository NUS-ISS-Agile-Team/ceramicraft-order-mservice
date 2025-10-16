package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// 请求总数（用于吞吐量、QPS）
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_service_http_requests_total",
			Help: "Total number of HTTP requests processed.(订单服务请求总量)",
		},
		[]string{"method", "path"},
	)

	// 请求耗时（用于P95、P99）
	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_service_http_request_duration_milliseconds",
			Help:    "Histogram of response latency (seconds) of HTTP requests.(订单服务响应耗时ms)",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000}, // 5ms~5s
		},
		[]string{"method", "path"},
	)

	// 错误总数
	HttpRequestsErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_service_errors_total",
			Help: "Total number of error responses (status >= 400).(订单服务错误总数)",
		},
		[]string{"method", "path", "status"},
	)
)

func RegisterMetrics() {
	prometheus.MustRegister(HttpRequestsTotal, HttpRequestDuration, HttpRequestsErrors)
}
