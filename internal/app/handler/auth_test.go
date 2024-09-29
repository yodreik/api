package handler

import (
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

	tests := []table{
		{
			name: "ok",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, false, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs("john.doe@example.com", "johndoe", sha256.String("testword")).WillReturnRows(rows)
			},

			request: request{
				body: `{"email":"john.doe@example.com","username":"johndoe","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusCreated,
				body:   `{"id":"USER_ID","email":"john.doe@example.com","username":"johndoe","display_name":"","avatar_url":"","is_private":false,"is_confirmed":false}`,
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
				body: `{"email":"incorrect-email","username":"John Doe","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid email format"}`,
			},
		},
		{
			name: "user already exists",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs("john.doe@example.com", "John Doe", sha256.String("testword")).WillReturnError(repoerr.ErrUserAlreadyExists)
			},

			request: request{
				body: `{"email":"john.doe@example.com","username":"John Doe","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusConflict,
				body:   `{"message":"user already exists"}`,
			},
		},
		{
			name: "repository error",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *").
					WithArgs("john.doe@example.com", "John Doe", sha256.String("testword")).WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"email":"john.doe@example.com","username":"John Doe","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPost, "/api/auth/account", handler.CreateAccount))
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

	tests := []table{
		{
			name: "ok",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnRows(rows)
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

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnError(repoerr.ErrUserNotFound)
			},

			request: request{
				body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusUnauthorized,
				body:   `{"message":"user not found"}`,
			},
		},
		{
			name: "repository error",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").
					WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
		{
			name: "user not confirmed",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, false, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1 AND password_hash = $2").WithArgs("john.doe@example.com", sha256.String("testword")).WillReturnRows(rows)
			},

			request: request{
				body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusForbidden,
				body:   `{"message":"email confirmation needed"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPost, "/api/auth/session", handler.CreateSession))
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

	tests := []table{
		{
			name: "ok",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").WithArgs("john.doe@example.com").WillReturnRows(rows)

				rows = sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(5*time.Minute).Truncate(time.Minute), time.Now())

				mock.ExpectQuery("INSERT INTO reset_password_requests (email, token, expires_at) VALUES ($1, $2, $3) RETURNING *").WithArgs("john.doe@example.com", "LONGTOKEN", time.Now().Add(5*time.Minute).Truncate(time.Minute)).WillReturnRows(rows)
			},

			request: request{
				body: `{"email":"john.doe@example.com"}`,
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
		{
			name: "invalid request body",

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid request body"}`,
			},
		},
		{
			name: "user not found",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").WithArgs("john.doe@example.com").WillReturnError(repoerr.ErrUserNotFound)
			},

			request: request{
				body: `{"email":"john.doe@example.com"}`,
			},

			expect: expect{
				status: http.StatusNotFound,
				body:   `{"message":"user not found"}`,
			},
		},
		{
			name: "repository error",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").WithArgs("john.doe@example.com").WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"email":"john.doe@example.com"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
		{
			name: "repository error",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").WithArgs("john.doe@example.com").WillReturnRows(rows)

				mock.ExpectQuery("INSERT INTO reset_password_requests (email, token, expires_at) VALUES ($1, $2, $3) RETURNING *").WithArgs("john.doe@example.com", "LONGTOKEN", time.Now().Add(5*time.Minute).Truncate(time.Minute)).WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"email":"john.doe@example.com"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPost, "/api/auth/password/reset", handler.ResetPassword))
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

	tests := []table{
		{
			name: "ok",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET password_hash = $1 WHERE email = $2").
					WithArgs(sha256.String("testword"), "john.doe@example.com").WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec("UPDATE reset_password_requests SET is_used = true WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnResult(sqlmock.NewResult(1, 1))
			},

			request: request{
				body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
		{
			name: "invalid request body",

			request: request{
				body: `{"some":"invalid","request":"body"}`,
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid request body"}`,
			},
		},
		{
			name: "token doesn't exists",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnError(repoerr.ErrRequestNotFound)
			},

			request: request{
				body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusNotFound,
				body:   `{"message":"password reset request not found"}`,
			},
		},
		{
			name: "repository error on getting token",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
		{
			name: "reset password request expired",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(-5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)
			},

			request: request{
				body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusForbidden,
				body:   `{"message":"recovery token expired"}`,
			},
		},
		{
			name: "reset password request already used",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", true, time.Now().Add(5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)
			},

			request: request{
				body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusForbidden,
				body:   `{"message":"this recovery token has been used"}`,
			},
		},
		{
			name: "repository error on updating password",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "LONGTOKEN", false, time.Now().Add(5*time.Minute), time.Now())

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs("LONGTOKEN").WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET password_hash = $1 WHERE email = $2").
					WithArgs(sha256.String("testword"), "john.doe@example.com").
					WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
		{
			name: "can't mark request as used",

			repo: func(mock sqlmock.Sqlmock) {
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

			request: request{
				body: `{"token":"LONGTOKEN","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPatch, "/api/auth/password", handler.UpdatePassword))
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

	tests := []table{
		{
			name: "ok",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE confirmation_token = $1").
					WithArgs("CONFIRMATION_TOKEN").
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET is_confirmed = true WHERE email = $1 AND confirmation_token = $2").
					WithArgs("john.doe@example.com", "CONFIRMATION_TOKEN").
					WillReturnResult(driver.RowsAffected(1))
			},

			request: request{
				body: `{"token":"CONFIRMATION_TOKEN"}`,
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
		{
			name: "invalid request body",

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid request body"}`,
			},
		},
		{
			name: "request not found",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE confirmation_token = $1").
					WithArgs("CONFIRMATION_TOKEN").
					WillReturnError(repoerr.ErrUserNotFound)
			},

			request: request{
				body: `{"token":"CONFIRMATION_TOKEN"}`,
			},

			expect: expect{
				status: http.StatusNotFound,
				body:   `{"message":"user not found"}`,
			},
		},
		{
			name: "repository error",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE confirmation_token = $1").
					WithArgs("CONFIRMATION_TOKEN").
					WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"token":"CONFIRMATION_TOKEN"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
		{
			name: "repository error on confirming",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE confirmation_token = $1").
					WithArgs("CONFIRMATION_TOKEN").
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET is_confirmed = true WHERE email = $1 AND confirmation_token = $2").
					WithArgs("john.doe@example.com", "CONFIRMATION_TOKEN").
					WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"token":"CONFIRMATION_TOKEN"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPost, "/api/auth/account/confirm", handler.ConfirmAccount))
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

	tests := []table{
		{
			name: "ok",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow("USER_ID", "john.doe@example.com", "johndoe", "John Doe", "https://cdn.domain.com/avatar.jpeg", sha256.String("testword"), false, true, "CONFIRMATION_TOKEN", time.Now())

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs("USER_ID").WillReturnRows(rows)
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusOK,
				body:   `{"id":"USER_ID","email":"john.doe@example.com","username":"johndoe","display_name":"John Doe","avatar_url":"https://cdn.domain.com/avatar.jpeg","is_private":false,"is_confirmed":true}`,
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

	tests := []table{
		{
			name: "ok: username",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, "johndoe2", user.DisplayName, user.AvatarURL, user.PasswordHash, false, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			request: request{
				body: `{"username":"johndoe2"}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
		{
			name: "ok: display_name",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, "John Doe Ver2", user.AvatarURL, user.PasswordHash, false, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			request: request{
				body: `{"display_name":"John Doe Ver2"}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
		{
			name: "ok: avatar_url",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, user.DisplayName, "https://cdn.avatar.com/picture.image", user.PasswordHash, false, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			request: request{
				body: `{"avatar_url":"https://cdn.avatar.com/picture.image"}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
		{
			name: "invalid avatar_url",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)
			},

			request: request{
				body: `{"avatar_url":"invalid link"}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"avatar_url should be a valid link"}`,
			},
		},
		{
			name: "ok: password",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, user.DisplayName, user.AvatarURL, sha256.String("newpassword"), false, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			request: request{
				body: `{"password":"newpassword"}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
		{
			name: "ok: is_private",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, true, user.ID).
					WillReturnResult(driver.RowsAffected(1))
			},

			request: request{
				body: `{"is_private":true}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
		{
			name: "invalid request body",

			request: request{
				body: `{"invalid":"request, "body}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid request body"}`,
			},
		},
		{
			name: "user not found",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnError(repoerr.ErrUserNotFound)
			},

			request: request{
				body: `{"is_private":true}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusNotFound,
				body:   `{"message":"user not found"}`,
			},
		},
		{
			name: "get: repository error",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"is_private":true}`,
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
		{
			name: "update: repository error",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6 WHERE id = $7").
					WithArgs(user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, true, user.ID).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				body: `{"is_private":true}`,
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
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPatch, "/api/auth/account", handler.UserIdentity, handler.UpdateAccount))
	}
}
