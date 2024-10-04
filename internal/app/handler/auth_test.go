package handler

import (
	"api/internal/app/handler/test"
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	"api/internal/repository/entity"
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

func TestCreateAccount(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, false, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs("john.doe@example.com", "johndoe", sha256.String("testword")).WillReturnRows(rows)
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com","username":"johndoe","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusCreated,
				Body:   `{"id":"USER_ID","email":"john.doe@example.com","username":"johndoe","display_name":"","avatar_url":"","is_private":false,"is_confirmed":false}`,
			},
		},
		{
			Name: "invalid request body",

			Request: test.Request{
				Body: `{"some":"invalid","request":"structure"}`,
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid request body"}`,
			},
		},
		{
			Name: "invalid email format",

			Request: test.Request{
				Body: `{"email":"incorrect-email","username":"John Doe","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid email format"}`,
			},
		},
		{
			Name: "user already exists",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs("john.doe@example.com", "John Doe", sha256.String("testword")).WillReturnError(repoerr.ErrUserAlreadyExists)
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com","username":"John Doe","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusConflict,
				Body:   `{"message":"user already exists"}`,
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs("john.doe@example.com", "John Doe", sha256.String("testword")).WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com","username":"John Doe","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
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

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnRows(rows)
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			Expect: test.Expect{
				Status:     http.StatusOK,
				BodyFields: []string{"token"},
			},
		},
		{
			Name: "invalid request body",

			Request: test.Request{
				Body: `{"some":"invalid","body":"poo"}`,
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid request body"}`,
			},
		},
		{
			Name: "user not found",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnError(repoerr.ErrUserNotFound)
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusUnauthorized,
				Body:   `{"message":"user not found"}`,
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
		{
			Name: "user not confirmed",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, false, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnRows(rows)
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusForbidden,
				Body:   `{"message":"email confirmation needed"}`,
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/auth/session", "/api/auth/session", handler.CreateSession)
	}
}

func TestResetPassword(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").WithArgs("john.doe@example.com").WillReturnRows(rows)

				rows = sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(5*time.Minute).Truncate(time.Minute), time.Now())

				mock.ExpectQuery("INSERT INTO reset_password_requests (email, token, expires_at) VALUES ($1, $2, $3) RETURNING *").WithArgs("john.doe@example.com", "LONGTOKEN", time.Now().Add(5*time.Minute).Truncate(time.Minute)).WillReturnRows(rows)
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com"}`,
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "invalid request body",

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid request body"}`,
			},
		},
		{
			Name: "user not found",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").WithArgs("john.doe@example.com").WillReturnError(repoerr.ErrUserNotFound)
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com"}`,
			},

			Expect: test.Expect{
				Status: http.StatusNotFound,
				Body:   `{"message":"user not found"}`,
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").WithArgs("john.doe@example.com").WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com"}`,
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").WithArgs("john.doe@example.com").WillReturnRows(rows)

				mock.ExpectQuery("INSERT INTO reset_password_requests (email, token, expires_at) VALUES ($1, $2, $3) RETURNING *").WithArgs("john.doe@example.com", "LONGTOKEN", time.Now().Add(5*time.Minute).Truncate(time.Minute)).WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"email":"john.doe@example.com"}`,
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/auth/password/reset", "/api/auth/password/reset", handler.ResetPassword)
	}
}

func TestUpdatePassword(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	c := config.Config{}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET password_hash = $1 WHERE email = $2").
					WithArgs(sha256.String("testword"), "john.doe@example.com").WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec("UPDATE reset_password_requests SET is_used = true WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnResult(sqlmock.NewResult(1, 1))
			},

			Request: test.Request{
				Body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "invalid request body",

			Request: test.Request{
				Body: `{"some":"invalid","request":"body"}`,
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid request body"}`,
			},
		},
		{
			Name: "token doesn't exists",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnError(repoerr.ErrRequestNotFound)
			},

			Request: test.Request{
				Body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusNotFound,
				Body:   `{"message":"password reset request not found"}`,
			},
		},
		{
			Name: "repository error on getting token",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
		{
			Name: "reset password request expired",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(-5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)
			},

			Request: test.Request{
				Body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusForbidden,
				Body:   `{"message":"recovery token expired"}`,
			},
		},
		{
			Name: "reset password request already used",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", true, time.Now().Add(5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)
			},

			Request: test.Request{
				Body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusForbidden,
				Body:   `{"message":"this recovery token has been used"}`,
			},
		},
		{
			Name: "repository error on updating password",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET password_hash = $1 WHERE email = $2").
					WithArgs(sha256.String("testword"), "john.doe@example.com").
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
		{
			Name: "can't mark request as used",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET password_hash = $1 WHERE email = $2").
					WithArgs(sha256.String("testword"), "john.doe@example.com").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec("UPDATE reset_password_requests SET is_used = true WHERE token = $1").
					WithArgs("LONGTOKEN").
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPatch, "/api/auth/password", "/api/auth/password", handler.UpdatePassword)
	}
}

func TestConfirmAccount(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	c := config.Config{}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE confirmation_token = $1").
					WithArgs("CONFIRMATION_TOKEN").
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET is_confirmed = true WHERE email = $1 AND confirmation_token = $2").
					WithArgs("john.doe@example.com", "CONFIRMATION_TOKEN").
					WillReturnResult(driver.RowsAffected(1))
			},

			Request: test.Request{
				Body: `{"token":"CONFIRMATION_TOKEN"}`,
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "invalid request body",

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid request body"}`,
			},
		},
		{
			Name: "request not found",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE confirmation_token = $1").
					WithArgs("CONFIRMATION_TOKEN").
					WillReturnError(repoerr.ErrUserNotFound)
			},

			Request: test.Request{
				Body: `{"token":"CONFIRMATION_TOKEN"}`,
			},

			Expect: test.Expect{
				Status: http.StatusNotFound,
				Body:   `{"message":"user not found"}`,
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE confirmation_token = $1").
					WithArgs("CONFIRMATION_TOKEN").
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"token":"CONFIRMATION_TOKEN"}`,
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
		{
			Name: "repository error on confirming",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE confirmation_token = $1").
					WithArgs("CONFIRMATION_TOKEN").
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET is_confirmed = true WHERE email = $1 AND confirmation_token = $2").
					WithArgs("john.doe@example.com", "CONFIRMATION_TOKEN").
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"token":"CONFIRMATION_TOKEN"}`,
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/auth/account/confirm", "/api/auth/account/confirm", handler.ConfirmAccount)
	}
}

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

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs("USER_ID").WillReturnRows(rows)
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
				Body:   `{"id":"USER_ID","email":"john.doe@example.com","username":"johndoe","display_name":"John Doe","avatar_url":"https://cdn.domain.com/avatar.jpeg","is_private":false,"is_confirmed":true}`,
			},
		},
		{
			Name: "user not found",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs("USER_ID").WillReturnError(repoerr.ErrUserNotFound)
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusUnauthorized,
				Body:   `{"message":"invalid authorization token"}`,
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs("USER_ID").WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodGet, "/api/account", "/api/account", handler.UserIdentity, handler.GetCurrentAccount)
	}
}

func TestUpdateAccount(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	tokenManager := token.New(c.Token)
	handler := New(&c, repo, mockmailer.New(), mocktoken.New(c.Token))

	user := entity.User{
		ID:                "USER_ID",
		Email:             "john.doe@example.com",
		Username:          "johndoe",
		DisplayName:       "John Doe",
		AvatarURL:         "https://cdn.domain.com/avatar.jpeg",
		PasswordHash:      sha256.String("testword"),
		IsPrivate:         false,
		IsConfirmed:       true,
		ConfirmationToken: "CONFIRMATION_TOKEN",
		CreatedAt:         time.Now(),
	}

	accessToken, err := tokenManager.GenerateJWT(user.ID)
	if err != nil {
		t.Fatal("unexpected error while generating mock token")
	}

	tests := []test.Case{
		{
			Name: "ok: username",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, "johndoe2", user.DisplayName, user.AvatarURL, user.PasswordHash, false, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			Request: test.Request{
				Body: `{"username":"johndoe2"}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "ok: display_name",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, "John Doe Ver2", user.AvatarURL, user.PasswordHash, false, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			Request: test.Request{
				Body: `{"display_name":"John Doe Ver2"}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "ok: avatar_url",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, user.DisplayName, "https://cdn.avatar.com/picture.image", user.PasswordHash, false, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			Request: test.Request{
				Body: `{"avatar_url":"https://cdn.avatar.com/picture.image"}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "invalid avatar_url",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)
			},

			Request: test.Request{
				Body: `{"avatar_url":"invalid link"}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"avatar_url should be a valid link"}`,
			},
		},
		{
			Name: "ok: password",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, user.DisplayName, user.AvatarURL, sha256.String("newpassword"), false, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			Request: test.Request{
				Body: `{"password":"newpassword"}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "ok: is_private",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, true, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			Request: test.Request{
				Body: `{"is_private":true}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "invalid request body",

			Request: test.Request{
				Body: `{"invalid":"request, "body}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid request body"}`,
			},
		},
		{
			Name: "user not found",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnError(repoerr.ErrUserNotFound)
			},

			Request: test.Request{
				Body: `{"is_private":true}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusNotFound,
				Body:   `{"message":"user not found"}`,
			},
		},
		{
			Name: "get: repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"is_private":true}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
		{
			Name: "update: repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, true, user.ID).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: `{"is_private":true}`,
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPatch, "/api/auth/account", "/api/auth/account", handler.UserIdentity, handler.UpdateAccount)
	}
}
