package handler

import (
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/internal/token"
	mocktoken "api/internal/token/mock"
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
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	tokenManager := token.New(c.Token)
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	tokenWithID69, err := tokenManager.GenerateJWT("69")
	if err != nil {
		t.Fatal("unexpected error while generating mock token")
	}

	tests := []table{
		{
			name: "ok",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM users WHERE id = $1",
						args:  []driver.Value{"69"},
						rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
							AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), time.Now()),
					},
				},
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
				queries: []queryArgs{
					{
						query: "SELECT * FROM users WHERE id = $1",
						args:  []driver.Value{"69"},
						err:   repoerr.ErrUserNotFound,
					},
				},
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
				queries: []queryArgs{
					{
						query: "SELECT * FROM users WHERE id = $1",
						args:  []driver.Value{"69"},
						err:   errors.New("repo: Some repository error"),
					},
				},
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", tokenWithID69),
				},
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodGet, "/api/me", handler.UserIdentity, handler.Me))
	}
}
