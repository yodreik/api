package repository

import (
	"api/internal/repository/postgres/user"

	"github.com/jmoiron/sqlx"
)

type User interface {
}

type Repository struct {
	User *user.Postgres
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		User: user.New(db),
	}
}
