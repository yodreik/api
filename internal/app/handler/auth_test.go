package handler

import (
	"api/internal/config"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/pkg/sha256"
	"encoding/json"
	"errors"
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

		r.POST("/api/auth/register", handler.Register)

		req, err := http.NewRequest(http.MethodPost, "/api/auth/register",
			strings.NewReader(`{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`))
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var body responsebody.User

		json.Unmarshal(w.Body.Bytes(), &body)

		expectedStatus := http.StatusCreated
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v", status, expectedStatus)
		}

		expectedBody := `{"id":"69","email":"john.doe@example.com","name":"John Doe"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})

	t.Run("Invalid request body", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.POST("/api/auth/register", handler.Register)

		req, err := http.NewRequest(http.MethodPost, "/api/auth/register",
			strings.NewReader(`{"some":"incorrect","fields":"for","request":"body"}`))
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

		expectedBody := `{"message":"invalid request body"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})

	t.Run("Invalid email", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.POST("/api/auth/register", handler.Register)

		req, err := http.NewRequest(http.MethodPost, "/api/auth/register",
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

	t.Run("User already exists", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO users (email, name, password_hash) values ($1, $2, $3) RETURNING *").
			WithArgs("john.doe@example.com", "John Doe", sha256.String("testword")).WillReturnError(repoerr.ErrUserAlreadyExists)

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.POST("/api/auth/register", handler.Register)

		req, err := http.NewRequest(http.MethodPost, "/api/auth/register",
			strings.NewReader(`{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`))
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var body responsebody.User

		json.Unmarshal(w.Body.Bytes(), &body)

		expectedStatus := http.StatusConflict
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v", status, expectedStatus)
		}

		expectedBody := `{"message":"user already exists"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})

	t.Run("Repository error", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO users (email, name, password_hash) values ($1, $2, $3) RETURNING *").
			WithArgs("john.doe@example.com", "John Doe", sha256.String("testword")).WillReturnError(errors.New("repo: Some repository error"))

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.POST("/api/auth/register", handler.Register)

		req, err := http.NewRequest(http.MethodPost, "/api/auth/register",
			strings.NewReader(`{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`))
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		var body responsebody.User

		json.Unmarshal(w.Body.Bytes(), &body)

		expectedStatus := http.StatusInternalServerError
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v", status, expectedStatus)
		}

		expectedBody := `{"message":"can't register"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})
}
