package user

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type Postgres struct {
	db *sqlx.DB
}

type User struct {
	ID           string    `db:"id"`
	Email        string    `db:"email"`
	Name         string    `db:"name"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) Create(ctx context.Context, email string, name string, passwordHash string) (*User, error) {
	query := "INSERT INTO users (email, name, password_hash) values ($1, $2, $3) RETURNING *"
	row := p.db.QueryRowContext(ctx, query, email, name, passwordHash)
	if row.Err() != nil {
		return nil, row.Err()
	}

	var user User
	// TODO: try to parse directly into user struct
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *Postgres) GetByID(ctx context.Context, id string) (*User, error) {
	query := "SELECT * FROM users WHERE id = $1"

	var user User
	err := p.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *Postgres) GetByCredentials(ctx context.Context, email string, passwordHash string) (*User, error) {
	query := "SELECT * FROM users WHERE email = $1 AND password_hash = $2"

	var user User
	err := p.db.GetContext(ctx, &user, query, email, passwordHash)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
