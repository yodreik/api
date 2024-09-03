package handler

import (
	"api/internal/config"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/internal/token"
	"api/pkg/sha256"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestMe(t *testing.T) {
	tokenSecret := "some-supa-secret-characters"
	tokenManager := token.New(tokenSecret)

	tokenWithID69, err := tokenManager.GenerateToken("69")
	if err != nil {
		t.Fatal("unexpected error while generating mock token")
	}

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo)

	tests := []table{
		{
			name: "ok",

			repo: &repoArgs{
				query: "SELECT * FROM users WHERE id = $1",
				args:  []driver.Value{"69"},
				rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
					AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), time.Now()),
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", tokenWithID69),
				},
			},

			expect: expect{
				status: http.StatusOK,
				body:   `{"id":"69","email":"john.doe@example.com","name":"John Doe"}`,
			},
		},
		{
			name: "user not found",

			repo: &repoArgs{
				query: "SELECT * FROM users WHERE id = $1",
				args:  []driver.Value{"69"},
				err:   repoerr.ErrUserNotFound,
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", tokenWithID69),
				},
			},

			expect: expect{
				status: http.StatusUnauthorized,
				body:   `{"message":"invalid authorization token"}`,
			},
		},
		{
			name: "repository error",

			repo: &repoArgs{
				query: "SELECT * FROM users WHERE id = $1",
				args:  []driver.Value{"69"},
				err:   errors.New("repo: Some repository error"),
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", tokenWithID69),
				},
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"can't get me"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodGet, "/api/me", handler.UserIdentity, handler.Me))
	}
}
