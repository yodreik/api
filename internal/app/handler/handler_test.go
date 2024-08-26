package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthcheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	c := config.Config{}
	repo := repository.Repository{}

	h := New(&c, &repo)

	r.GET("/healthcheck", h.Healthcheck)

	req, err := http.NewRequest(http.MethodGet, "/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v, want %v", status, http.StatusOK)
	}

	expected := `OK`
	if w.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v, want %v", w.Body.String(), expected)
	}
}