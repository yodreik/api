package handler

import (
	"api/internal/config"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/pkg/sha256"
	"database/sql/driver"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestRegister(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	c := config.Config{}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"), nil)
	handler := New(&c, repo)

	tests := []table{
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

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPost, "/api/auth/register", handler.Register))
	}
}

func TestLogin(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"), nil)
	handler := New(&c, repo)

	tests := []table{
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

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPost, "/api/auth/login", handler.Login))
	}
}
