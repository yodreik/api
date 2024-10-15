package handler

import (
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

	var minutesSpent int
	var longestActivity int
	for _, workout := range workouts {
		minutesSpent += workout.Duration

		if workout.Duration > longestActivity {
			longestActivity = workout.Duration
		}
	}

	accessToken, err := tokenManager.GenerateJWT(user.ID)
	if err != nil {
		t.Fatal("unexpected error while generating mock token")
	}

	headerAuthorization := fmt.Sprintf("Bearer %s", accessToken)

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
					"Authorization": headerAuthorization,
				},
			},

			Expect: test.Expect{
				Status: http.StatusOK,
				Body: responsebody.Statistics{
					UserID:          user.ID,
					MinutesSpent:    minutesSpent,
					LongestActivity: longestActivity,
				},
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
					"Authorization": headerAuthorization,
				},
			},

			Expect: test.ResponseInternalServerError,
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodGet, "/api/statistics", "/api/statistics", handler.UserIdentity, handler.GetStatistics)
	}
}

func TestGetByUsername(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("err not expected: %v\n", err)
	}

	c := config.Empty()
	repo := repository.New(sqlx.NewDb(db, "sqlmock"))
	handler := New(c, repo, mockmailer.New(), mocktoken.New(c.Token))

	publicUser := entity.User{
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

	privateUser := publicUser
	privateUser.IsPrivate = true

	tests := []test.Case{
		{
			Name: "private: ok",

			Repo: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "username", "display_name", "avatar_url", "password_hash", "is_private", "is_confirmed", "confirmation_token", "created_at"}).
					AddRow(privateUser.ID, privateUser.Email, privateUser.Username, privateUser.DisplayName, privateUser.AvatarURL, privateUser.PasswordHash, privateUser.IsPrivate, privateUser.IsConfirmed, privateUser.ConfirmationToken, privateUser.CreatedAt)

				mock.ExpectQuery("SELECT * FROM users WHERE username = $1").
					WithArgs(privateUser.Username).
					WillReturnRows(rows)
			},

			Expect: test.Expect{
				Status: http.StatusOK,
				Body: responsebody.Profile{
					ID:        privateUser.ID,
					Username:  privateUser.Username,
					IsPrivate: privateUser.IsPrivate,
				},
			},
		},
		{
			Name: "private: user not found",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE username = $1").
					WithArgs(privateUser.Username).
					WillReturnError(repoerr.ErrUserNotFound)
			},

			Expect: test.Expect{
				Status: http.StatusNotFound,
				Body: responsebody.Message{
					Message: "user not found",
				},
			},
		},
		{
			Name: "private: repository error",

			Repo: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT * FROM users WHERE username = $1").
					WithArgs(privateUser.Username).
					WillReturnError(errors.New("repo: Some repository error"))
			},

			Expect: test.ResponseInternalServerError,
		},
	}

	for _, tc := range tests {
		test.Endpoint(t, tc, mock, http.MethodGet, "/api/user/:username", fmt.Sprintf("/api/user/%s", publicUser.Username), handler.GetUserByUsername)
	}
}
