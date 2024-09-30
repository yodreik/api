package handler

import (
	"api/internal/app/handler/test"
	"api/internal/config"
	mockmailer "api/internal/mailer/mock"
	"api/internal/repository"
	"api/internal/repository/entity"
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

func TestGetStatistics(t *testing.T) {
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

	workouts := []entity.Workout{
		{
			ID:        "WORKOUT_ID_1",
			UserID:    "USER_ID",
			Date:      time.Now(),
			Duration:  69,
			Kind:      "GYM",
			CreatedAt: time.Now(),
		},
		{
			ID:        "WORKOUT_ID_2",
			UserID:    "USER_ID",
			Date:      time.Now(),
			Duration:  121,
			Kind:      "Pool",
			CreatedAt: time.Now(),
		},
		{
			ID:        "WORKOUT_ID_3",
			UserID:    "USER_ID",
			Date:      time.Now(),
			Duration:  21,
			Kind:      "Calisthenics",
			CreatedAt: time.Now(),
		},
	}

	accessToken, err := tokenManager.GenerateJWT(user.ID)
	if err != nil {
		t.Fatal("unexpected error while generating mock token")
	}

	tests := []test.Case{
		{
			Name: "ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "date", "duration", "kind", "created_at"})

				for _, workout := range workouts {
					rows.AddRow(workout.ID, workout.UserID, workout.Date, workout.Duration, workout.Kind, workout.CreatedAt)
				}

				mock.ExpectQuery("SELECT * FROM workouts WHERE user_id = $1 ORDER BY date ASC").
					WithArgs(user.ID).
					WillReturnRows(rows)
			},

			Request: test.Request{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
				Body:   `{"user_id":"USER_ID","minutes_spent":211,"longest_activity":121}`,
			},
		},
		{
			Name: "repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM workouts WHERE user_id = $1 ORDER BY date ASC").
					WithArgs(user.ID).
					WillReturnError(errors.New("repo: Some repository error"))
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
		test.Endpoint(t, tc, mock, http.MethodGet, "/api/statistics", handler.UserIdentity, handler.GetStatistics)
	}
}
