package repository

import (
	"api/internal/repository/postgres/user"
	"api/internal/repository/postgres/workout"
	"api/internal/repository/redis/cache"
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type User interface {
	Create(ctx context.Context, email string, name string, passwordHash string) (*user.User, error)
	GetByID(ctx context.Context, id string) (*user.User, error)
	GetByCredentials(ctx context.Context, email string, passwordHash string) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	UpdatePasswordByEmail(ctx context.Context, email string, password string) error
}

type Workout interface {
	Create(ctx context.Context, userID string, date time.Time, duration int, kind string) (*workout.Workout, error)
	GetAllByUserID(ctx context.Context, userID string) ([]workout.Workout, error)
	GetByID(ctx context.Context, id string) (*workout.Workout, error)
}

type Cache interface {
	SetPasswordResetRequest(ctx context.Context, email string, token string) error
	GetPasswordResetEmailByToken(ctx context.Context, token string) (string, error)
}

type Repository struct {
	User    User
	Workout Workout
	Cache   Cache
}

func New(pdb *sqlx.DB, rdb *redis.Client) *Repository {
	return &Repository{
		User:    user.New(pdb),
		Workout: workout.New(pdb),
		Cache:   cache.New(rdb),
	}
}
