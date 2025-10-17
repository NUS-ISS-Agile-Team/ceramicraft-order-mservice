package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware 采集HTTP指标
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next() // 先执行请求

		duration := float64(time.Since(start).Milliseconds())
		method := c.Request.Method
		path := c.FullPath() // 获取注册路由路径模板（如 /api/users/:id）
		if path == "" {
			path = c.Request.URL.Path // 未命中路由时回退真实路径
		}
		status := strconv.Itoa(c.Writer.Status())

		HttpRequestsTotal.WithLabelValues(method, path).Inc()
		HttpRequestDuration.WithLabelValues(method, path).Observe(duration)

		if c.Writer.Status() >= 400 {
			HttpRequestsErrors.WithLabelValues(method, path, status).Inc()
		}
	}
}
