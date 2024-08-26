package repository

import (
	"api/internal/repository/postgres/user"
	"context"

	"github.com/jmoiron/sqlx"
)

type User interface {
	Create(ctx context.Context, email string, name string, passwordHash string) (*user.User, error)
	GetByID(ctx context.Context, id string) (*user.User, error)
	GetByCredentials(ctx context.Context, email string, passwordHash string) (*user.User, error)
}

type Repository struct {
	User User
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		User: user.New(db),
	}
}
