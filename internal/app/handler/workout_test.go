package handler

import (
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	"api/internal/token"
	mocktoken "api/internal/token/mock"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestCreateWorkout(t *testing.T) {
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

	expectedDate, err := time.Parse("02-01-2006", "21-08-2024")
	if err != nil {
		t.Fatal("err no expected while parsing mock date")
	}

	tests := []table{
		{
			name: "ok",

			repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "date", "duration", "kind", "created_at"}).
					AddRow("WORKOUT_ID", "USER_ID", expectedDate, 69, "Calisthenics", time.Now())

				mock.ExpectQuery("INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *").
					WithArgs("USER_ID", expectedDate, 69, "Calisthenics").WillReturnRows(rows)
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
				body: `{"date":"21-08-2024","duration":69,"kind":"Calisthenics"}`,
			},

			expect: expect{
				status: http.StatusCreated,
				body:   `{"id":"WORKOUT_ID","date":"21-08-2024","duration":69,"kind":"Calisthenics"}`,
			},
		},
		{
			name: "invalid request body",

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
				body: `{"invalid":"body"}`,
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid request body"}`,
			},
		},
		{
			name: "invalid date format",

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
				body: `{"date":"69-11-2024","duration":69,"kind":"Calisthenics"}`,
			},

			expect: expect{
				status: http.StatusBadRequest,
				body:   `{"message":"invalid date format"}`,
			},
		},
		{
			name: "unauthorized",

			expect: expect{
				status: http.StatusUnauthorized,
				body:   `{"message":"empty authorization header"}`,
			},
		},
		{
			name: "repository error",

			repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *").
					WithArgs("USER_ID", expectedDate, 69, "Calisthenics").WillReturnError(errors.New("repo: Some repository error"))
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
				body: `{"date":"21-08-2024","duration":69,"kind":"Calisthenics"}`,
			},

			expect: expect{
				status: http.StatusInternalServerError,
				body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, TemplateTestHandler(tc, mock, http.MethodPost, "/api/workout", handler.UserIdentity, handler.CreateWorkout))
	}
}
