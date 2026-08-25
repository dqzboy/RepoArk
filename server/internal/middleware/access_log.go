package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const slowRequestThreshold = time.Second

// AccessLogger 记录有排障价值的 HTTP 请求。
// 成功且快速的任务轮询与前端静态资源请求会被静默，避免备份期间每秒刷屏；
// 错误响应、慢请求和所有写操作始终保留。
func AccessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		latency := time.Since(startedAt)
		status := c.Writer.Status()
		if !shouldLogAccess(c.Request.Method, c.Request.URL.Path, c.FullPath(), status, latency) {
			return
		}

		log.Printf("[HTTP] %d | %s | %s | %s %q", status, latency, c.ClientIP(), c.Request.Method, c.Request.URL.Path)
	}
}

func shouldLogAccess(method, path, route string, status int, latency time.Duration) bool {
	if status >= http.StatusBadRequest || latency >= slowRequestThreshold {
		return true
	}
	if method == http.MethodOptions {
		return false
	}
	if method != http.MethodGet && method != http.MethodHead {
		return true
	}

	if route == "/api/jobs/:id" {
		return false
	}
	return path != "/" && path != "/favicon.ico" && !strings.HasPrefix(path, "/assets/")
}
