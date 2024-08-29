package repository

import (
	"api/internal/repository/postgres/user"
	"api/internal/repository/postgres/workout"
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type User interface {
	Create(ctx context.Context, email string, name string, passwordHash string) (*user.User, error)
	GetByID(ctx context.Context, id string) (*user.User, error)
	GetByCredentials(ctx context.Context, email string, passwordHash string) (*user.User, error)
}

type Workout interface {
	Create(ctx context.Context, userID string, date time.Time, duration int, kind string) (*workout.Workout, error)
	GetManyByUserID(ctx context.Context, userID string) ([]*workout.Workout, error)
	GetByID(ctx context.Context, id string) (*workout.Workout, error)
}

type Repository struct {
	User User
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		User: user.New(db),
	}
}
