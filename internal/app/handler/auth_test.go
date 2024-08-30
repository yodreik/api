package handler

import (
	"api/internal/config"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/pkg/sha256"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
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

func TestRegister(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	c := config.Config{}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))

	tt := []table{
		{
			name: "ok",

			repo: &repoArgs{
				query: "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *",
				args:  []driver.Value{"john.doe@example.com", "John Doe", sha256.String("testword")},
				rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
					AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), time.Now()),
			},

			request: request{
				body: `{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusCreated,
				body:   `{"id":"69","email":"john.doe@example.com","name":"John Doe"}`,
			},
		},
		{
			name: "invalid request body",

			request: request{
				body: `{"some":"invalid","request":"structure"}`,
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid request body"}`,
			},
		},
		{
			name: "invalid email format",

			request: request{
				body: `{"email":"incorrect-email","name":"John Doe","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid email format"}`,
			},
		},
		{
			name: "name is too long",

			request: request{
				body: `{"email":"john.doe@example.com","name":"very-looooooooooooooooooooooooooooooooooooooooooong-name","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"name is too long"}`,
			},
		},
		{
			name: "user already exists",

			repo: &repoArgs{
				query: "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *",
				args:  []driver.Value{"john.doe@example.com", "John Doe", sha256.String("testword")},
				err:   repoerr.ErrUserAlreadyExists,
			},

			request: request{
				body: `{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusConflict,
				body:   `{"message":"user already exists"}`,
			},
		},
		{
			name: "repository error",

			repo: &repoArgs{
				query: "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *",
				args:  []driver.Value{"john.doe@example.com", "John Doe", sha256.String("testword")},
				err:   errors.New("repo: Something goes wrong"),
			},

			request: request{
				body: `{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"can't register"}`,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.repo != nil {
				if tc.repo.err != nil {
					mock.ExpectQuery(tc.repo.query).WithArgs(tc.repo.args...).WillReturnError(tc.repo.err)
				} else {
					mock.ExpectQuery(tc.repo.query).WithArgs(tc.repo.args...).WillReturnRows(tc.repo.rows)
				}
			}

			gin.SetMode(gin.TestMode)
			r := gin.Default()

			handler := New(&c, repo)

			r.POST("/api/auth/register", handler.Register)

			req, err := http.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(tc.request.body))
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if status := w.Code; status != tc.expect.status {
				t.Fatalf("unexpected status code returned: got %v, want %v\n", status, tc.expect.status)
			}

			if w.Body.String() != tc.expect.body {
				t.Fatalf("unexpected body returned: got %v, want %v\n", w.Body.String(), tc.expect.body)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))

	tt := []table{
		{
			name: "ok",

			repo: &repoArgs{
				query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
				args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
				rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
					AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), time.Now()),
			},

			request: request{
				body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			expect: expect{
				status:     http.StatusOK,
				bodyFields: []string{"token"},
			},
		},
		{
			name: "invalid request body",

			request: request{
				body: `{"some":"invalid","body":"poo"}`,
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid request body"}`,
			},
		},
		{
			name: "user not found",

			repo: &repoArgs{
				query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
				args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
				err:   repoerr.ErrUserNotFound,
			},

			request: request{
				body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusNotFound,
				body:   `{"message":"user not found"}`,
			},
		},
		{
			name: "repository error",

			repo: &repoArgs{
				query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
				args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
				err:   errors.New("repo: Some repository error"),
			},

			request: request{
				body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"can't login"}`,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.repo != nil {
				if tc.repo.err != nil {
					mock.ExpectQuery(tc.repo.query).WithArgs(tc.repo.args...).WillReturnError(tc.repo.err)
				} else {
					mock.ExpectQuery(tc.repo.query).WithArgs(tc.repo.args...).WillReturnRows(tc.repo.rows)
				}
			}

			gin.SetMode(gin.TestMode)
			r := gin.Default()

			handler := New(&c, repo)

			r.POST("/api/auth/login", handler.Login)

			req, err := http.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(tc.request.body))
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if status := w.Code; status != tc.expect.status {
				t.Fatalf("unexpected status code returned: got %v, want %v\n", status, tc.expect.status)
			}

			var body map[string]string
			err = json.Unmarshal(w.Body.Bytes(), &body)
			if err != nil {
				t.Fatalf("can't unmarshall response body: %v\n", err)
			}

			for _, field := range tc.expect.bodyFields {
				value, exists := body[field]
				if !exists {
					t.Fatalf("expected body field not found: %v\n", field)
				}

				if value == "" {
					t.Fatalf("expected body field is empty: %v\n", field)
				}
			}

			if tc.expect.body != `` && w.Body.String() != tc.expect.body {
				t.Fatalf("unexpected body returned: got %v, want %v\n", w.Body.String(), tc.expect.body)
			}
		})
	}
}
