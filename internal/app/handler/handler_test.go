package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

type table struct {
	name    string
	repo    *repoArgs
	request request
	expect  expect
}

type repoArgs struct {
	query string
	args  []driver.Value
	err   error
	rows  *sqlmock.Rows
}

type request struct {
	body    string
	headers map[string]string
}

type expect struct {
	status     int
	body       string
	bodyFields []string
}

func TemplateTestHandler(tt table, mock sqlmock.Sqlmock, method string, path string, handlers ...gin.HandlerFunc) func(t *testing.T) {
	return func(t *testing.T) {
		if tt.repo != nil {
			if tt.repo.err != nil {
				mock.ExpectQuery(tt.repo.query).WithArgs(tt.repo.args...).WillReturnError(tt.repo.err)
			} else {
				mock.ExpectQuery(tt.repo.query).WithArgs(tt.repo.args...).WillReturnRows(tt.repo.rows)
			}
		}

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		r.Handle(method, path, handlers...)

		req, err := http.NewRequest(method, path, strings.NewReader(tt.request.body))
		if err != nil {
			t.Fatal(err)
		}

		for key, value := range tt.request.headers {
			req.Header.Add(key, value)
		}

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if status := w.Code; status != tt.expect.status {
			t.Fatalf("unexpected status code returned: got %v, want %v\n", status, tt.expect.status)
		}

		if tt.expect.body != "" && w.Body.String() != tt.expect.body {
			t.Fatalf("unexpected body returned: got %v, want %v\n", w.Body.String(), tt.expect.body)
		}

		var body map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &body)
		if err != nil {
			t.Fatalf("can't unmarshall response body: %v\n", err)
		}

		for _, field := range tt.expect.bodyFields {
			value, exists := body[field]
			if !exists {
				t.Fatalf("expected body field not found: %v\n", field)
			}

			if value == "" {
				t.Fatalf("expected body field is empty: %v\n", field)
			}
		}
	}
}

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
		t.Fatalf("handler returned wrong status code: got %v, want %v\n", status, http.StatusOK)
	}

	expected := `ok`
	if w.Body.String() != expected {
		t.Fatalf("handler returned unexpected body: got %v, want %v\n", w.Body.String(), expected)
	}
}
