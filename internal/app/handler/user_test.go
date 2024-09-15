package handler

import (
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/internal/token"
	mocktoken "api/internal/token/mock"
	"api/pkg/sha256"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestGetCurrentAccount(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	tokenManager := token.New(c.Token)
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	accessToken, err := tokenManager.GenerateJWT("USER_ID")
	if err != nil {
		t.Fatal("unexpected error while generating mock token")
	}

	tests := []table{
		{
			name: "ok",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "display_name", "email", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "johndoe", "John Doe", "john.doe@example.com", sha256.String("testword"), false, true, "CONFIRmATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs("USER_ID").WillReturnRows(rows)
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusOK,
				body:   `{"id":"USER_ID","email":"john.doe@example.com","username":"johndoe","display_name":"John Doe","is_private":false,"is_confirmed":true}`,
			},
		},
		{
			name: "user not found",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs("USER_ID").WillReturnError(repoerr.ErrUserNotFound)
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusUnauthorized,
				body:   `{"message":"invalid authorization token"}`,
			},
		},
		{
			name: "repository error",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs("USER_ID").WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodGet, "/api/account", handler.UserIdentity, handler.GetCurrentAccount))
	}
}
