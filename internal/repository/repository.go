package repository

import (
	"api/internal/repository/entity"
	"api/internal/repository/postgres/user"
	"api/internal/repository/postgres/workout"
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type User interface {
	Create(ctx context.Context, email string, username string, passwordHash string) (*entity.User, error)
	SetUserConfirmed(ctx context.Context, email string, token string) error
	UpdateUser(ctx context.Context, userID string, email string, username string, displayName string, avatarURL string, passwordHash string, isPrivate bool) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	GetByCredentialsWithEmail(ctx context.Context, email string, passwordHash string) (*entity.User, error)
	GetByCredentialsWithUsername(ctx context.Context, email string, passwordHash string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByUsername(ctx context.Context, username string) (*entity.User, error)
	GetByConfirmationToken(ctx context.Context, token string) (*entity.User, error)
	UpdatePasswordByEmail(ctx context.Context, email string, password string) error
	CreatePasswordResetRequest(ctx context.Context, token string, email string) (*entity.Request, error)
	GetRequestByToken(ctx context.Context, token string) (*entity.Request, error)
	GetRequestByEmail(ctx context.Context, email string) (*entity.Request, error)
	MarkRequestAsUsed(ctx context.Context, token string) error

	RemoveExpiredRecords(ctx context.Context) (n int64, err error)
}

type Workout interface {
	Create(ctx context.Context, userID string, date time.Time, duration int, kind string) (*entity.Workout, error)
	GetByID(ctx context.Context, id string) (*entity.Workout, error)
	GetAllUserWorkouts(ctx context.Context, userID string) ([]entity.Workout, error)
	GetUserWorkouts(ctx context.Context, userID string, bedginDate time.Time, endDate time.Time) ([]entity.Workout, error)
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
