package response

import (
	"api/internal/app/handler/response/responsebody"
	"net/http"

	"github.com/gin-gonic/gin"
)

func WithMessage(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, responsebody.Message{Message: message})
}

func InvalidRequestBody(c *gin.Context) {
	WithMessage(c, http.StatusBadRequest, "invalid request body")
}

func InternalServerError(c *gin.Context) {
	WithMessage(c, http.StatusInternalServerError, "internal server error")
}
