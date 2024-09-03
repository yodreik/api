package user

import (
	repoerr "api/internal/repository/errors"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

type ResetPasswordRequest struct {
	ID        string    `db:"id"`
	Email     string    `db:"email"`
	Token     string    `db:"token"`
	IsUsed    bool      `db:"is_used"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) Create(ctx context.Context, email string, name string, passwordHash string) (*User, error) {
	query := "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *"
	row := p.db.QueryRowContext(ctx, query, email, name, passwordHash)
	if pqErr, ok := row.Err().(*pq.Error); ok && pqErr.Code == "23505" {
		return nil, repoerr.ErrUserAlreadyExists
	}
	if row.Err() != nil {
		return nil, row.Err()
	}

	var user User
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
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *Postgres) GetByCredentials(ctx context.Context, email string, passwordHash string) (*User, error) {
	query := "SELECT * FROM users WHERE email = $1 AND password_hash = $2"

	var user User
	err := p.db.GetContext(ctx, &user, query, email, passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *Postgres) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := "SELECT * FROM users WHERE email = $1"

	var user User
	err := p.db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *Postgres) UpdatePasswordByEmail(ctx context.Context, email string, passwordHash string) error {
	query := "UPDATE users SET password_hash = $1 WHERE email = $2"

	_, err := p.db.ExecContext(ctx, query, passwordHash, email)
	if err != nil {
		return err
	}

	return nil
}

func (p *Postgres) CreatePasswordResetRequest(ctx context.Context, token string, email string) error {
	query := "INSERT INTO reset_password_requests (email, token, expires_at) VALUES ($1, $2, $3)"

	row := p.db.QueryRowContext(ctx, query, email, token, time.Now().Add(15*time.Minute))
	return row.Err()
}

func (p *Postgres) GetPasswordResetRequestByToken(ctx context.Context, token string) (*ResetPasswordRequest, error) {
	query := "SELECT * FROM reset_password_requests WHERE token = $1"

	var request ResetPasswordRequest
	err := p.db.GetContext(ctx, &request, query, token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrPasswordResetRequestNotFound
	}
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (p *Postgres) MarkResetPasswordTokenAsUsed(ctx context.Context, token string) error {
	query := "UPDATE reset_password_requests SET is_used = true WHERE token = $1"

	_, err := p.db.ExecContext(ctx, query, token)
	return err
}
