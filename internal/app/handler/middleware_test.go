package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserIdentity(t *testing.T) {
	c := config.Config{}
	repo := repository.Repository{}

	t.Run("Empty header", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, &repo)

		r.GET("/api/me", handler.UserIdentity, handler.Me)

		req, err := http.NewRequest(http.MethodGet, "/api/me", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Add("Authorization", fmt.Sprintf(""))

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		expectedStatus := http.StatusUnauthorized
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v\n", status, expectedStatus)
		}

		expectedBody := `{"message":"invalid authorization header"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})

	t.Run("Invalid token type", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, &repo)

		r.GET("/api/me", handler.UserIdentity, handler.Me)

		req, err := http.NewRequest(http.MethodGet, "/api/me", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bot <token>"))

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		expectedStatus := http.StatusUnauthorized
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v\n", status, expectedStatus)
		}

		expectedBody := `{"message":"invalid authorization token type"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})

	t.Run("Incorrect token format", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, &repo)

		r.GET("/api/me", handler.UserIdentity, handler.Me)

		req, err := http.NewRequest(http.MethodGet, "/api/me", nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", "some-incorrect-jwonwebtoken"))

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		expectedStatus := http.StatusUnauthorized
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v\n", status, expectedStatus)
		}

		expectedBody := `{"message":"invalid authorization token"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})
}
