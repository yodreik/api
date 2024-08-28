package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"api/pkg/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	responsebody "api/internal/app/handler/response/body"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func TestRegister(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	c := config.Config{}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))

	t.Run("OK", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
			AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), time.Now())

		mock.ExpectQuery("INSERT INTO users (email, name, password_hash) values ($1, $2, $3) RETURNING *").
			WithArgs("john.doe@example.com", "John Doe", sha256.String("testword")).WillReturnRows(rows)

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.GET("/api/auth/register", handler.Register)

		req, err := http.NewRequest(http.MethodGet, "/api/auth/register",
			strings.NewReader(`{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`))
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var body responsebody.User

		json.Unmarshal(w.Body.Bytes(), &body)

		if status := w.Code; status != http.StatusCreated {
			t.Fatalf("handler returned wrong status code: got %v, want %v", status, http.StatusOK)
		}

		expectedBody := `{"id":"69","email":"john.doe@example.com","name":"John Doe"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})

	t.Run("Invalid email", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.GET("/api/auth/register", handler.Register)

		req, err := http.NewRequest(http.MethodGet, "/api/auth/register",
			strings.NewReader(`{"email":"incorrect-email","name":"John Doe","password":"testword"}`))
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var body responsebody.User

		json.Unmarshal(w.Body.Bytes(), &body)

		expectedStatus := http.StatusBadRequest
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v", status, expectedStatus)
		}

		expectedBody := `{"message":"invalid email format"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})
}
