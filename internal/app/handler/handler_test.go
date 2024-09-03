package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TemplateTestHandler(tc table, mock sqlmock.Sqlmock, method string, path string, handlers ...gin.HandlerFunc) func(t *testing.T) {
	return func(t *testing.T) {
		if tc.repo != nil {
			if tc.repo.err != nil {
				mock.ExpectQuery(tc.repo.query).WithArgs(tc.repo.args...).WillReturnError(tc.repo.err)
			} else {
				mock.ExpectQuery(tc.repo.query).WithArgs(tc.repo.args...).WillReturnRows(tc.repo.rows)
			}
		}

		gin.SetMode(gin.TestMode)
		r := gin.Default()

		r.Handle(method, path, handlers...)

		req, err := http.NewRequest(method, path, strings.NewReader(tc.request.body))
		if err != nil {
			t.Fatal(err)
		}

		for key, value := range tc.request.headers {
			req.Header.Add(key, value)
		}

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if status := w.Code; status != tc.expect.status {
			t.Fatalf("unexpected status code returned: got %v, want %v\n", status, tc.expect.status)
		}

		if tc.expect.body != "" && w.Body.String() != tc.expect.body {
			t.Fatalf("unexpected body returned: got %v, want %v\n", w.Body.String(), tc.expect.body)
		}

		var body map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &body)
		if err != nil {
			t.Fatalf("can't unmarshall response body: %v\n", err)
		}

		for _, field := range tc.expect.bodyFields {
			value, exists := body[field]
			if !exists {
				t.Fatalf("expected body field not found: %v\n", field)
			}

			v := reflect.ValueOf(value)
			if !v.IsValid() || v.IsZero() {
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
