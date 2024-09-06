package handler

import (
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/internal/repository/postgres/user"
	"api/pkg/random"
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

func TestRegister(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	c := config.Config{}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New())

	tests := []table{
		// {
		// 	name: "ok",

		// 	repo: &repoArgs{
		// 		queries: []queryArgs{
		// 			{
		// 				query: "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *",
		// 				args:  []driver.Value{"john.doe@example.com", "John Doe", sha256.String("testword")},
		// 				rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
		// 					AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), time.Now()),
		// 			},
		// 		},
		// 	},

		// 	request: request{
		// 		body: `{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`,
		// 	},

		// 	expect: expect{
		// 		status: http.StatusCreated,
		// 		body:   `{"id":"69","email":"john.doe@example.com","name":"John Doe"}`,
		// 	},
		// },
		// {
		// 	name: "invalid request body",

		// 	request: request{
		// 		body: `{"some":"invalid","request":"structure"}`,
		// 	},

		// 	expect: expect{
		// 		status: http.StatusBadRequest,
		// 		body:   `{"message":"invalid request body"}`,
		// 	},
		// },
		// {
		// 	name: "invalid email format",

		// 	request: request{
		// 		body: `{"email":"incorrect-email","name":"John Doe","password":"testword"}`,
		// 	},

		// 	expect: expect{
		// 		status: http.StatusBadRequest,
		// 		body:   `{"message":"invalid email format"}`,
		// 	},
		// },
		// {
		// 	name: "name is too long",

		// 	request: request{
		// 		body: `{"email":"john.doe@example.com","name":"very-looooooooooooooooooooooooooooooooooooooooooong-name","password":"testword"}`,
		// 	},

		// 	expect: expect{
		// 		status: http.StatusBadRequest,
		// 		body:   `{"message":"name is too long"}`,
		// 	},
		// },
		// {
		// 	name: "user already exists",

		// 	repo: &repoArgs{
		// 		query: "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *",
		// 		args:  []driver.Value{"john.doe@example.com", "John Doe", sha256.String("testword")},
		// 		err:   repoerr.ErrUserAlreadyExists,
		// 	},

		// 	request: request{
		// 		body: `{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`,
		// 	},

		// 	expect: expect{
		// 		status: http.StatusConflict,
		// 		body:   `{"message":"user already exists"}`,
		// 	},
		// },
		// {
		// 	name: "repository error",

		// 	repo: &repoArgs{
		// 		query: "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *",
		// 		args:  []driver.Value{"john.doe@example.com", "John Doe", sha256.String("testword")},
		// 		err:   errors.New("repo: Something goes wrong"),
		// 	},

		// 	request: request{
		// 		body: `{"email":"john.doe@example.com","name":"John Doe","password":"testword"}`,
		// 	},

		// 	expect: expect{
		// 		status: http.StatusInternalServerError,
		// 		body:   `{"message":"internal server error"}`,
		// 	},
		// },
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
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New())

	tests := []table{
		{
			name: "ok",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
						args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
						rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "is_email_confirmed", "created_at"}).
							AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), true, time.Now()),
					},
				},
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
				queries: []queryArgs{
					{
						query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
						args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
						err:   repoerr.ErrUserNotFound,
					},
				},
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
				queries: []queryArgs{
					{
						query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
						args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
						err:   errors.New("repo: Some repository error"),
					},
				},
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

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
						args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
						rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "is_email_confirmed", "created_at"}).
							AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), false, time.Now()),
					},
					{
						query: "SELECT * FROM requests WHERE email = $1",
						args:  []driver.Value{"john.doe@example.com"},
						rows: sqlmock.NewRows([]string{"id", "kind", "email", "token", "is_used", "expires_at", "created_at"}).
							AddRow("69", user.RequestKindEmailConfirmation, "john.doe@example.com", random.String(64), false, time.Now().Add(48*time.Hour), time.Now()),
					},
				},
			},

			request: request{
				body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusForbidden,
				body:   `{"message":"email confirmation needed"}`,
			},
		},
		// {
		// 	name: "user not confirmed + request not found", // TODO: How to mock tokenizer

		// 	repo: &repoArgs{
		// 		queries: []queryArgs{
		// 			{
		// 				query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
		// 				args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
		// 				rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "is_email_confirmed", "created_at"}).
		// 					AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), false, time.Now()),
		// 			},
		// 			{
		// 				query: "SELECT * FROM requests WHERE email = $1",
		// 				args:  []driver.Value{"john.doe@example.com"},
		// 				err:   repoerr.ErrRequestNotFound,
		// 			},
		// 			{
		// 				query: "INSERT INTO requests (kind, email, token, expires_at) VALUES ($1, $2, $3, $4) RETURNING *",
		// 				args:  []driver.Value{user.RequestKindEmailConfirmation, "john.doe@example.com", random.String(64), time.Now().Add(48 * time.Hour)},
		// 				err:   errors.New("repo: Some repository error"),
		// 			},
		// 		},
		// 	},

		// 	request: request{
		// 		body: `{"email":"john.doe@example.com","password":"testword"}`,
		// 	},

		// 	expect: expect{
		// 		status: http.StatusInternalServerError,
		// 		body:   `{"message":"internal server error"}`,
		// 	},
		// },
		{
			name: "user not confirmed + repo error",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM users WHERE email = $1 AND password_hash = $2",
						args:  []driver.Value{"john.doe@example.com", sha256.String("testword")},
						rows: sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "is_email_confirmed", "created_at"}).
							AddRow("69", "john.doe@example.com", "John Doe", sha256.String("testword"), false, time.Now()),
					},
					{
						query: "SELECT * FROM requests WHERE email = $1",
						args:  []driver.Value{"john.doe@example.com"},
						err:   repoerr.ErrRequestNotFound,
					},
					{
						query: "INSERT INTO requests (kind, email, token, expires_at) VALUES ($1, $2, $3, $4) RETURNING *",
						args:  []driver.Value{user.RequestKindEmailConfirmation, "john.doe@example.com", random.String(64), time.Now().Add(48 * time.Hour)},
						err:   errors.New("repo: Some repository error"),
					},
				},
			},

			request: request{
				body: `{"email":"john.doe@example.com","password":"testword"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPost, "/api/auth/login", handler.Login))
	}
}

func TestUpdatePassword(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	tokenSecret := "some-supa-secret-characters"
	c := config.Config{Token: config.Token{Secret: tokenSecret}}
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(&c, repo, mockmailer.New())

	tok := random.String(64)

	tests := []table{
		{
			name: "ok",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM requests WHERE token = $1",
						args:  []driver.Value{tok},
						rows: sqlmock.NewRows([]string{"id", "kind", "email", "token", "is_used", "expires_at", "created_at"}).
							AddRow("69", "password_reset", "john.doe@example.com", tok, false, time.Now().Add(15*time.Minute), time.Now()),
					},
					{
						exec:   "UPDATE users SET password_hash = $1 WHERE email = $2",
						args:   []driver.Value{sha256.String("testword"), "john.doe@example.com"},
						result: sqlmock.NewResult(1, 1),
					},
					{
						exec:   "UPDATE requests SET is_used = true WHERE token = $1",
						args:   []driver.Value{tok},
						result: sqlmock.NewResult(1, 1),
					},
				},
			},

			request: request{
				body: fmt.Sprintf(`{"token":"%s","password":"testword"}`, tok),
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

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM requests WHERE token = $1",
						args:  []driver.Value{tok},
						err:   repoerr.ErrRequestNotFound,
					},
				},
			},

			request: request{
				body: fmt.Sprintf(`{"token":"%s","password":"testword"}`, tok),
			},

			expect: expect{
				status: http.StatusNotFound,
				body:   `{"message":"password reset request not found"}`,
			},
		},
		{
			name: "repository error on getting token",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM requests WHERE token = $1",
						args:  []driver.Value{tok},
						err:   errors.New("repo: Some repository error"),
					},
				},
			},

			request: request{
				body: fmt.Sprintf(`{"token":"%s","password":"testword"}`, tok),
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
		{
			name: "reset password request expired",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM requests WHERE token = $1",
						args:  []driver.Value{tok},
						rows: sqlmock.NewRows([]string{"id", "kind", "email", "token", "is_used", "expires_at", "created_at"}).
							AddRow("69", "password_reset", "john.doe@example.com", tok, false, time.Now().Add(-15*time.Minute), time.Now()),
					},
				},
			},

			request: request{
				body: fmt.Sprintf(`{"token":"%s","password":"testword"}`, tok),
			},

			expect: expect{
				status: http.StatusForbidden,
				body:   `{"message":"recovery token expired"}`,
			},
		},
		{
			name: "reset password request already used",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM requests WHERE token = $1",
						args:  []driver.Value{tok},
						rows: sqlmock.NewRows([]string{"id", "kind", "email", "token", "is_used", "expires_at", "created_at"}).
							AddRow("69", "password_reset", "john.doe@example.com", tok, true, time.Now().Add(15*time.Minute), time.Now()),
					},
				},
			},

			request: request{
				body: fmt.Sprintf(`{"token":"%s","password":"testword"}`, tok),
			},

			expect: expect{
				status: http.StatusForbidden,
				body:   `{"message":"this recovery token has been used"}`,
			},
		},
		{
			name: "repository error on updating password",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM requests WHERE token = $1",
						args:  []driver.Value{tok},
						rows: sqlmock.NewRows([]string{"id", "kind", "email", "token", "is_used", "expires_at", "created_at"}).
							AddRow("69", "password_reset", "john.doe@example.com", tok, false, time.Now().Add(15*time.Minute), time.Now()),
					},
					{
						exec: "UPDATE users SET password_hash = $1 WHERE email = $2",
						args: []driver.Value{sha256.String("testword"), "john.doe@example.com"},
						err:  errors.New("repo: Some repository error"),
					},
				},
			},

			request: request{
				body: fmt.Sprintf(`{"token":"%s","password":"testword"}`, tok),
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
		{
			name: "can't mark request as used",

			repo: &repoArgs{
				queries: []queryArgs{
					{
						query: "SELECT * FROM requests WHERE token = $1",
						args:  []driver.Value{tok},
						rows: sqlmock.NewRows([]string{"id", "kind", "email", "token", "is_used", "expires_at", "created_at"}).
							AddRow("69", "password_reset", "john.doe@example.com", tok, false, time.Now().Add(15*time.Minute), time.Now()),
					},
					{
						exec:   "UPDATE users SET password_hash = $1 WHERE email = $2",
						args:   []driver.Value{sha256.String("testword"), "john.doe@example.com"},
						result: sqlmock.NewResult(1, 1),
					},
					{
						exec: "UPDATE requests SET is_used = true WHERE token = $1",
						args: []driver.Value{tok},
						err:  errors.New("repo: Some repository error"),
					},
				},
			},

			request: request{
				body: fmt.Sprintf(`{"token":"%s","password":"testword"}`, tok),
			},

			expect: expect{
				status: http.StatusOK,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPatch, "/api/auth/password/update", handler.UpdatePassword))
	}
}
