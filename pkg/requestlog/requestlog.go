package requestlog

import (
	"api/pkg/requestid"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Completed initialize a middleware for logging every request
func Completed(c *gin.Context) {
	start := time.Now()

	c.Next()

	slog.Info("Request completed",
		slog.String("id", requestid.Get(c)),
		slog.String("method", c.Request.Method),
		slog.String("uri", c.Request.URL.Path),
		slog.String("client_ip", c.ClientIP()),
		slog.String("duration", fmt.Sprintf("%v", time.Since(start))),
		slog.String("host", c.Request.Host),
		slog.String("user_agent", c.Request.UserAgent()),
		slog.Int("status", c.Writer.Status()),
	)
}
