package requestid

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerRequestID = "X-Request-ID"

// New initializes the RequestID middleware.
func New(ctx *gin.Context) {
	rid := ctx.GetHeader(headerRequestID)
	if rid == "" {
		rid = uuid.NewString()
		ctx.Request.Header.Add(headerRequestID, rid)
	}

	ctx.Header(headerRequestID, rid)
	ctx.Next()
}

// Get returns the request identifier.
func Get(c *gin.Context) string {
	return c.Writer.Header().Get(headerRequestID)
}
