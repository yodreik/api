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
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	UpdatePasswordByEmail(ctx context.Context, email string, password string) error
	CreatePasswordResetRequest(ctx context.Context, token string, email string) error
	GetPasswordResetRequestByToken(ctx context.Context, token string) (*user.ResetPasswordRequest, error)
	MarkResetPasswordTokenAsUsed(ctx context.Context, token string) error
}

type Workout interface {
	Create(ctx context.Context, userID string, date time.Time, duration int, kind string) (*workout.Workout, error)
	GetAllByUserID(ctx context.Context, userID string) ([]workout.Workout, error)
	GetByID(ctx context.Context, id string) (*workout.Workout, error)
}

type Repository struct {
	User    User
	Workout Workout
}

func New(pdb *sqlx.DB) *Repository {
	return &Repository{
		User:    user.New(pdb),
		Workout: workout.New(pdb),
	}
}
