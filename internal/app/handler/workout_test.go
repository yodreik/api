package handler

import (
	"api/internal/app/handler/request/requestbody"
	"api/internal/app/handler/response/responsebody"
	"api/internal/app/handler/test"
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	"api/internal/repository/entity"
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

	headerAuthorization := fmt.Sprintf("Bearer %s", accessToken)

	layout := "02-01-2006"
	now := time.Now().UTC()

	workout := entity.Workout{
		ID:        "WORKOUT_ID",
		UserID:    "USER_ID",
		Date:      time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
		Duration:  69,
		Kind:      "Calisthenics",
		CreatedAt: time.Now(),
	}

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "date", "duration", "kind", "created_at"}).
					AddRow(workout.ID, workout.UserID, workout.Date, workout.Duration, workout.Kind, workout.CreatedAt)

				mock.ExpectQuery("INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *").
					WithArgs(workout.UserID, workout.Date, workout.Duration, workout.Kind).
					WillReturnRows(rows)
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": headerAuthorization,
				},
				Body: requestbody.CreateWorkout{
					Date:     workout.Date.Format(layout),
					Duration: workout.Duration,
					Kind:     workout.Kind,
				},
			},

			Expect: test.Expect{
				Status: http.StatusCreated,
				Body: responsebody.Workout{
					ID:       workout.ID,
					Date:     workout.Date.Format(layout),
					Duration: workout.Duration,
					Kind:     workout.Kind,
				},
			},
		},
		{
			Name: "invalid request body",

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": headerAuthorization,
				},
				Body: map[string]string{
					"invalid": "body",
				},
			},

			Expect: test.ResponseInvalidRequestBody,
		},
		{
			Name: "invalid date format",

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": headerAuthorization,
				},
				Body: requestbody.CreateWorkout{
					Date:     "69--01-2024",
					Duration: workout.Duration,
					Kind:     workout.Kind,
				},
			},

			Expect: test.Expect{
				Status: http.StatusBadRequest,
				Body: responsebody.Message{
					Message: "invalid date format",
				},
			},
		},
		{
			Name: "unauthorized",

			Expect: test.Expect{
				Status: http.StatusUnauthorized,
				Body: responsebody.Message{
					Message: "empty authorization header",
				},
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *").
					WithArgs(workout.UserID, workout.Date, workout.Duration, workout.Kind).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": headerAuthorization,
				},
				Body: requestbody.CreateWorkout{
					Date:     workout.Date.Format(layout),
					Duration: workout.Duration,
					Kind:     workout.Kind,
				},
			},

			Expect: test.ResponseInternalServerError,
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodPost, "/api/workout", "/api/workout", handler.UserIdentity, handler.CreateWorkout)
	}
}
