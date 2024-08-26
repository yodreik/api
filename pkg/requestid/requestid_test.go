package requestid

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(New)

	router.GET("/status", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get(headerRequestID), "X-Request-ID header should be set")

	rid := w.Header().Get(headerRequestID)
	_, err = uuid.Parse(rid)
	assert.NoError(t, err, "X-Request-ID header should be a valid UUID")
}
