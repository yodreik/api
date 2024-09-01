package handler

import (
	"api/internal/config"
	"api/internal/repository"
	"api/internal/token"
	"database/sql/driver"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestCreateWorkout(t *testing.T) {
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
	handler := New(&c, repo)

	expectedDate, err := time.Parse("02.01.2006", "11.11.2024")
	if err != nil {
		t.Fatal("err no expected while parsing mock date")
	}

	tt := []table{
		{
			name: "ok",

			repo: &repoArgs{
				query: "INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *",
				args:  []driver.Value{"69", expectedDate, 71, "Calisthenics"},
				rows: sqlmock.NewRows([]string{"id", "user_id", "date", "duration", "kind", "created_at"}).
					AddRow("96", "69", expectedDate, 71, "Calisthenics", time.Now()),
			},

			request: request{
				headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", tokenWithID69),
				},
				body: `{"date":"11.11.2024","duration":71,"kind":"Calisthenics"}`,
			},

			expect: expect{
				status: http.StatusCreated,
				body:   `{"id":"96","date":"11.11.2024","duration":71,"kind":"Calisthenics"}`,
			},
		},
	}

	for _, tt := range tt {
		t.Run(tt.name, TemplateTestHandler(tt, mock, http.MethodPost, "/api/workout", handler.UserIdentity, handler.CreateWorkout))
	}
}
