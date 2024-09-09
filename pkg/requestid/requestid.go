package requestid

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerRequestID = "X-Request-ID"

// New initializes the RequestID middleware
func New(c *gin.Context) {
	rid := c.GetHeader(headerRequestID)
	if rid == "" {
		rid = uuid.NewString()
		c.Request.Header.Add(headerRequestID, rid)
	}

	c.Next()
}

// Get returns the request identifier
func Get(c *gin.Context) string {
	return c.Request.Header.Get(headerRequestID)
}
