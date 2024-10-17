package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response/responsebody"
	"api/internal/app/handler/test"
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	"api/internal/repository/entity"
	repoerr "api/internal/repository/errors"
	mocktoken "api/internal/token/mock"
	"api/pkg/sha256"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestCreateAccount(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	user := entity.User{
		ID:                "USER_ID",
		Email:             "john.doe@example.com",
		Username:          "johndoe",
		DisplayName:       "",
		AvatarURL:         "",
		PasswordHash:      sha256.String("testword"),
		IsPrivate:         false,
		IsConfirmed:       false,
		ConfirmationToken: "CONFIRMATION_TOKEN",
		CreatedAt:         time.Now(),
	}

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs(user.Email, user.Username, user.PasswordHash).WillReturnRows(rows)
			},

			Request: test.Request{
				Body: requestbody.CreateAccount{
					Email:    user.Email,
					Username: user.Username,
					Password: "testword",
				},
			},

			Expect: test.Expect{
				Status: http.StatusCreated,
				Body: responsebody.Account{
					ID:          user.ID,
					Email:       user.Email,
					Username:    user.Username,
					DisplayName: user.DisplayName,
					AvatarURL:   user.AvatarURL,
					IsPrivate:   user.IsPrivate,
					IsConfirmed: user.IsConfirmed,
					CreatedAt:   user.CreatedAt.Format(time.RFC3339),
				},
			},
		},
		{
			Name: "invalid request body",

			Request: test.Request{
				Body: map[string]string{
					"some":    "invalid",
					"request": "body",
				},
			},

			Expect: test.ResponseInvalidRequestBody,
		},
		{
			Name: "invalid email format",

			Request: test.Request{
				Body: requestbody.CreateAccount{
					Email:    "incorrect email",
					Username: user.Username,
					Password: "testword",
				},
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body: responsebody.Message{
					Message: "invalid email format",
				},
			},
		},
		{
			Name: "user already exists",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs(user.Email, user.Username, user.PasswordHash).WillReturnError(repoerr.ErrUserAlreadyExists)
			},

			Request: test.Request{
				Body: requestbody.CreateAccount{
					Email:    user.Email,
					Username: user.Username,
					Password: "testword",
				},
			},

			Expect: test.Expect{
				Status: http.StatusConflict,
				Body: responsebody.Message{
					Message: "user already exists",
				},
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs(user.Email, user.Username, user.PasswordHash).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: requestbody.CreateAccount{
					Email:    user.Email,
					Username: user.Username,
					Password: "testword",
				},
			},

			Expect: test.ResponseInternalServerError,
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/auth/account", "/api/auth/account", handler.CreateAccount)
	}
}

func TestCreateSession(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	user := entity.User{
		ID:                "USER_ID",
		Email:             "john.doe@example.com",
		Username:          "johndoe",
		DisplayName:       "John Doe",
		AvatarURL:         "https://cdn.content.com/avatar.jpeg",
		PasswordHash:      sha256.String("testword"),
		IsPrivate:         false,
		IsConfirmed:       true,
		ConfirmationToken: "CONFIRMATION_TOKEN",
		CreatedAt:         time.Now(),
	}

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs(user.Email, user.PasswordHash).WillReturnRows(rows)
			},

			Request: test.Request{
				Body: requestbody.CreateSession{
					Login:    user.Email,
					Password: "testword",
				},
			},

			Expect: test.Expect{
				Status:     http.StatusOK,
				BodyFields: []string{"token"},
			},
		},
		{
			Name: "invalid request body",

			Request: test.Request{
				Body: map[string]string{
					"some":    "invalid",
					"request": "body",
				},
			},

			Expect: test.ResponseInvalidRequestBody,
		},
		{
			Name: "user not found",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnError(repoerr.ErrUserNotFound)
			},

			Request: test.Request{
				Body: requestbody.CreateSession{
					Login:    user.Email,
					Password: "testword",
				},
			},

			Expect: test.Expect{
				Status: http.StatusUnauthorized,
				Body: responsebody.Message{
					Message: "user not found",
				},
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs(user.Email, user.PasswordHash).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: requestbody.CreateSession{
					Login:    user.Email,
					Password: "testword",
				},
			},

			Expect: test.ResponseInternalServerError,
		},
		{
			Name: "user not confirmed",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, false, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs(user.Email, user.PasswordHash).
					WillReturnRows(rows)
			},

			Request: test.Request{
				Body: requestbody.CreateSession{
					Login:    user.Email,
					Password: "testword",
				},
			},

			Expect: test.Expect{
				Status: http.StatusForbidden,
				Body: responsebody.Message{
					Message: "email confirmation needed",
				},
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/auth/session", "/api/auth/session", handler.CreateSession)
	}
}
