package handler

import (
	"api/internal/app/handler/test"
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

	layout := "02-01-2006"
	now := time.Now().UTC()

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	expectedDate, err := time.Parse(layout, "21-08-2024")
	if err != nil {
		t.Fatal("err no expected while parsing mock date")
	}

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "date", "duration", "kind", "created_at"}).
					AddRow("WORKOUT_ID", "USER_ID", today, 69, "Calisthenics", time.Now())

				mock.ExpectQuery("INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *").
					WithArgs("USER_ID", today, 69, "Calisthenics").WillReturnRows(rows)
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
				Body: fmt.Sprintf(`{"date":"%s","duration":69,"kind":"Calisthenics"}`, today.Format(layout)),
			},

			Expect: test.Expect{
				Status: http.StatusCreated,
				Body:   fmt.Sprintf(`{"id":"WORKOUT_ID","date":"%s","duration":69,"kind":"Calisthenics"}`, today.Format(layout)),
			},
		},
		{
			Name: "invalid request body",

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
				Body: `{"invalid":"body"}`,
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid request body"}`,
			},
		},
		{
			Name: "invalid date format",

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
				Body: `{"date":"69-11-2024","duration":69,"kind":"Calisthenics"}`,
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body:   `{"message":"invalid date format"}`,
			},
		},
		{
			Name: "unauthorized",

			Expect: test.Expect{
				Status: http.StatusUnauthorized,
				Body:   `{"message":"empty authorization header"}`,
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *").
					WithArgs("USER_ID", expectedDate, 69, "Calisthenics").WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
				Body: fmt.Sprintf(`{"date":"%s","duration":69,"kind":"Calisthenics"}`, today.Format(layout)),
			},

			Expect: test.Expect{
				Status: http.StatusInternalServerError,
				Body:   `{"message":"internal server error"}`,
			},
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/workout", "/api/workout", handler.UserIdentity, handler.CreateWorkout)
	}
}
