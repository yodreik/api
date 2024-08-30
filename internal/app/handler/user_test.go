package handler

import (
	"api/internal/config"
	"api/internal/repository"
	repoerr "api/internal/repository/errors"
	"api/internal/token"
	"api/pkg/sha256"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
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

	tt := []table{
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

			r.GET("/api/me", handler.UserIdentity, handler.Me)

			req, err := http.NewRequest(http.MethodGet, "/api/me", strings.NewReader(tc.request.body))
			if err != nil {
				t.Fatal(err)
			}

			for key, value := range tc.request.headers {
				req.Header.Add(key, value)
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
