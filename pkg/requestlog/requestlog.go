package requestlog

import (
	"api/pkg/requestid"
	"log/slog"

	"github.com/gin-gonic/gin"
)

func Handled(c *gin.Context) {
	slog.Info("Request handled",
		slog.String("request_id", requestid.Get(c)),
		slog.String("method", c.Request.Method),
		slog.String("uri", c.Request.URL.Path),
		slog.String("client_ip", c.ClientIP()),
	)
}
