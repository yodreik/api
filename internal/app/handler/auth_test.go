package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"crypto/sha256"
	"encoding/hex"
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

	passwordHash := sha256.New()
	passwordHash.Write([]byte("testword"))

	rows := sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
		AddRow("69", "john.doe@example.com", "John Doe", "22", time.Now())

	mock.ExpectQuery("INSERT INTO users (email, name, password_hash) values ($1, $2, $3) RETURNING *").
		WithArgs("john.doe@example.com", "John Doe", hex.EncodeToString(passwordHash.Sum(nil))).WillReturnRows(rows)

	gin.SetMode(gin.TestMode)
	r := gin.Default()

	c := config.Config{}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))

	handler := New(&c, repo)

	r.GET("/api/auth/register", handler.Register)

	req, err := http.NewRequest(http.MethodGet, "/api/auth/register", strings.NewReader(`{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var body responsebody.User

	json.Unmarshal(w.Body.Bytes(), &body)

	t.Logf("Code: %d, body: %s", w.Code, w.Body.String())
}
