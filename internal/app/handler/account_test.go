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

func TestResetPassword(t *testing.T) {
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

	request := entity.Request{
		ID:        "REQUEST_ID",
		Email:     user.Email,
		Token:     "LONG_PASSWORD_RESET_REQUEST_TOKEN",
		IsUsed:    false,
		ExpiresAt: time.Now().Add(5 * time.Minute).Truncate(time.Minute),
		CreatedAt: time.Now(),
	}

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").
					WithArgs(user.Email).
					WillReturnRows(rows)

				rows = sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow(request.ID, request.Email, request.Token, request.IsUsed, request.ExpiresAt, request.CreatedAt)

				mock.ExpectQuery("INSERT INTO reset_password_requests (email, token, expires_at) VALUES ($1, $2, $3) RETURNING *").
					WithArgs(request.Email, request.Token, request.ExpiresAt).
					WillReturnRows(rows)
			},

			Request: test.Request{
				Body: requestbody.ResetPassword{
					Email: request.Email,
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
			},
		},
		{
			Name: "invalid request body",

			Expect: test.ResponseInvalidRequestBody,
		},
		{
			Name: "user not found",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").
					WithArgs(user.Email).
					WillReturnError(repoerr.ErrUserNotFound)
			},

			Request: test.Request{
				Body: requestbody.ResetPassword{
					Email: user.Email,
				},
			},

			Expect: test.Expect{
				Status: http.StatusNotFound,
				Body: responsebody.Message{
					Message: "user not found",
				},
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").
					WithArgs(user.Email).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: requestbody.ResetPassword{
					Email: user.Email,
				},
			},

			Expect: test.ResponseInternalServerError,
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE email = $1").
					WithArgs(user.Email).
					WillReturnRows(rows)

				mock.ExpectQuery("INSERT INTO reset_password_requests (email, token, expires_at) VALUES ($1, $2, $3) RETURNING *").
					WithArgs(request.Email, request.Token, request.ExpiresAt).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: requestbody.ResetPassword{
					Email: user.Email,
				},
			},

			Expect: test.ResponseInternalServerError,
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/account/reset-password/request", "/api/account/reset-password/request", handler.ResetPassword)
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

	request := entity.Request{
		ID:        "REQUEST_ID",
		Email:     user.Email,
		Token:     "LONG_PASSWORD_RESET_REQUEST_TOKEN",
		IsUsed:    false,
		ExpiresAt: time.Now().Add(5 * time.Minute).Truncate(time.Minute),
		CreatedAt: time.Now(),
	}

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow(request.ID, request.Email, request.Token, request.IsUsed, request.ExpiresAt, request.CreatedAt)

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs(request.Token).
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET password_hash = $1 WHERE email = $2").
					WithArgs(sha256.String("new-password"), user.Email).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec("UPDATE reset_password_requests SET is_used = true WHERE token = $1").
					WithArgs(request.Token).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},

			Request: test.Request{
				Body: requestbody.UpdatePassword{
					Token:    request.Token,
					Password: "new-password",
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
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
			Name: "token doesn't exists",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs(request.Token).
					WillReturnError(repoerr.ErrRequestNotFound)
			},

			Request: test.Request{
				Body: requestbody.UpdatePassword{
					Token:    request.Token,
					Password: "new-password",
				},
			},

			Expect: test.Expect{
				Status: http.StatusNotFound,
				Body: responsebody.Message{
					Message: "password reset request not found",
				},
			},
		},
		{
			Name: "repository error on getting token",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs(request.Token).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: requestbody.UpdatePassword{
					Token:    request.Token,
					Password: "new-password",
				},
			},

			Expect: test.ResponseInternalServerError,
		},
		{
			Name: "reset password request expired",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow(request.ID, request.Email, request.Token, request.IsUsed, time.Now().Add(-5*time.Minute), request.CreatedAt)

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs(request.Token).
					WillReturnRows(rows)
			},

			Request: test.Request{
				Body: requestbody.UpdatePassword{
					Token:    request.Token,
					Password: "new-password",
				},
			},

			Expect: test.Expect{
				Status: http.StatusForbidden,
				Body: responsebody.Message{
					Message: "recovery token expired",
				},
			},
		},
		{
			Name: "reset password request already used",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow(request.ID, request.Email, request.Token, true, request.ExpiresAt, request.CreatedAt)

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs(request.Token).
					WillReturnRows(rows)
			},

			Request: test.Request{
				Body: requestbody.UpdatePassword{
					Token:    request.Token,
					Password: "new-password",
				},
			},

			Expect: test.Expect{
				Status: http.StatusForbidden,
				Body: responsebody.Message{
					Message: "this recovery token has been used",
				},
			},
		},
		{
			Name: "repository error on updating password",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow(request.ID, request.Email, request.Token, request.IsUsed, request.ExpiresAt, request.CreatedAt)

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs(request.Token).
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET password_hash = $1 WHERE email = $2").
					WithArgs(user.PasswordHash, user.Email).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: requestbody.UpdatePassword{
					Token:    request.Token,
					Password: "new-password",
				},
			},

			Expect: test.ResponseInternalServerError,
		},
		{
			Name: "can't mark request as used",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "token", "is_used", "expires_at", "created_at"}).
					AddRow(request.ID, request.Email, request.Token, request.IsUsed, request.ExpiresAt, request.CreatedAt)

				mock.ExpectQuery("SELECT * FROM reset_password_requests WHERE token = $1").
					WithArgs(request.Token).
					WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET password_hash = $1 WHERE email = $2").
					WithArgs(sha256.String("new-password"), user.Email).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec("UPDATE reset_password_requests SET is_used = true WHERE token = $1").
					WithArgs(request.Token).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Body: requestbody.UpdatePassword{
					Token:    request.Token,
					Password: "new-password",
				},
			},

			Expect: test.ResponseInternalServerError,
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPatch, "/api/account/reset-password", "/api/account/reset-password", handler.UpdatePassword)
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
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/account/confirm", "/api/account/confirm", handler.ConfirmAccount)
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

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs("USER_ID").WillReturnRows(rows)
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
				Body:   fmt.Sprintf(`{"id":"USER_ID","email":"john.doe@example.com","username":"johndoe","display_name":"John Doe","avatar_url":"https://cdn.domain.com/avatar.jpeg","is_private":false,"is_confirmed":true,"created_at":"%s"}`, user.CreatedAt.Format(time.RFC3339)),
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

				mock.ExpectQuery("SELECT * FROM users WHERE username = $1").WithArgs("johndoe2").WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6, is_confirmed = $7, confirmation_token = $8 WHERE id = $9").
					WithArgs(user.Email, "johndoe2", user.DisplayName, user.AvatarURL, user.PasswordHash, false, user.IsConfirmed, user.ConfirmationToken, user.ID).
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

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6, is_confirmed = $7, confirmation_token = $8 WHERE id = $9").
					WithArgs(user.Email, user.Username, "John Doe Ver2", user.AvatarURL, user.PasswordHash, false, user.IsConfirmed, user.ConfirmationToken, user.ID).
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
			Name: "ok: password",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(user.ID, user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, user.IsPrivate, user.IsConfirmed, user.ConfirmationToken, user.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE id = $1").WithArgs(user.ID).WillReturnRows(rows)

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6, is_confirmed = $7, confirmation_token = $8 WHERE id = $9").
					WithArgs(user.Email, user.Username, user.DisplayName, user.AvatarURL, sha256.String("newpassword"), false, user.IsConfirmed, user.ConfirmationToken, user.ID).
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

				mock.ExpectExec("UPDATE users SET email = $1, username = $2, display_name = $3, avatar_url = $4, password_hash = $5, is_private = $6, is_confirmed = $7, confirmation_token = $8 WHERE id = $9").
					WithArgs(user.Email, user.Username, user.DisplayName, user.AvatarURL, user.PasswordHash, true, user.IsConfirmed, user.ConfirmationToken, user.ID).
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
		test.Endpoint(t, tc, mock, http.MethodPatch, "/api/account", "/api/account", handler.UserIdentity, handler.UpdateAccount)
	}
}
