package handler

import (
	"api/internal/app/handler/response/responsebody"
	"api/internal/config"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/pkg/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
)

func TestMe(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))

	t.Run("OK", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
			AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), time.Now())

		mock.ExpectQuery("SELECT * FROM users WHERE id = $1").
			WithArgs("69").WillReturnRows(rows)

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.GET("/api/me", handler.Me)

		req, err := http.NewRequest(http.MethodGet, "/api/me", nil)
		if err != nil {
			t.Fatal(err)
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iat": time.Now().Unix(),
			"id":  "69",
		})

		tokenString, err := token.SignedString([]byte(c.Token.Secret))
		if err != nil {
			t.Fatalf("err not expected while signing jsonwebtoken: %v\n", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenString))

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		expectedStatus := http.StatusOK
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v\n", status, expectedStatus)
		}

		var body responsebody.User
		err = json.Unmarshal(w.Body.Bytes(), &body)
		if err != nil {
			t.Fatalf("can't unmarshall response body: %v\n", err)
		}

		if body.ID != "69" {
			t.Fatalf("unexpected user id: got %v, want %v\n", body.ID, "69")
		}
		if body.Email != "john.doe@example.com" {
			t.Fatalf("unexpected user email: got %v, want %v\n", body.ID, "john.dor@example.com")
		}
		if body.Name != "John Doe" {
			t.Fatalf("unexpected user name: got %v, want %v\n", body.ID, "John Doe")
		}
	})

	t.Run("Empty header", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.GET("/api/me", handler.Me)

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

		handler := New(&c, repo)

		r.GET("/api/me", handler.Me)

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

	t.Run("Incorrect JWT signing method", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.GET("/api/me", handler.Me)

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

	t.Run("User not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT * FROM users WHERE id = $1").
			WithArgs("69").WillReturnError(repoerr.ErrUserNotFound)

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.GET("/api/me", handler.Me)

		req, err := http.NewRequest(http.MethodGet, "/api/me", nil)
		if err != nil {
			t.Fatal(err)
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iat": time.Now().Unix(),
			"id":  "69",
		})

		tokenString, err := token.SignedString([]byte(c.Token.Secret))
		if err != nil {
			t.Fatalf("err not expected while signing jsonwebtoken: %v\n", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenString))

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

	t.Run("Repository error", func(t *testing.T) {
		mock.ExpectQuery("SELECT * FROM users WHERE id = $1").
			WithArgs("69").WillReturnError(errors.New("repo: Some repository error"))

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		handler := New(&c, repo)

		r.GET("/api/me", handler.Me)

		req, err := http.NewRequest(http.MethodGet, "/api/me", nil)
		if err != nil {
			t.Fatal(err)
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iat": time.Now().Unix(),
			"id":  "69",
		})

		tokenString, err := token.SignedString([]byte(c.Token.Secret))
		if err != nil {
			t.Fatalf("err not expected while signing jsonwebtoken: %v\n", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenString))

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		expectedStatus := http.StatusInternalServerError
		if status := w.Code; status != expectedStatus {
			t.Fatalf("handler returned wrong status code: got %v, want %v\n", status, expectedStatus)
		}

		expectedBody := `{"message":"can't get me"}`
		if w.Body.String() != expectedBody {
			t.Fatalf("handler returned unexpected body: got %v, want %v", w.Body.String(), expectedBody)
		}
	})
}
