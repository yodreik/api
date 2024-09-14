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
	ID                string    `db:"id"`
	Email             string    `db:"email"`
	Username          string    `db:"username"`
	DisplayName       string    `db:"display_name"`
	PasswordHash      string    `db:"password_hash"`
	IsConfirmed       bool      `db:"is_confirmed"`
	ConfirmationToken string    `db:"confirmation_token"`
	CreatedAt         time.Time `db:"created_at"`
}

type Request struct {
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

func (p *Postgres) Create(ctx context.Context, email string, username string, passwordHash string) (*User, error) {
	query := "INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING *"
	row := p.db.QueryRowContext(ctx, query, email, username, passwordHash)
	if pqErr, ok := row.Err().(*pq.Error); ok && pqErr.Code == "23505" {
		return nil, repoerr.ErrUserAlreadyExists
	}
	if row.Err() != nil {
		return nil, row.Err()
	}

	var user User
	err := row.Scan(&user.ID, &user.Email, &user.Username, &user.DisplayName, &user.PasswordHash, &user.IsConfirmed, &user.ConfirmationToken, &user.CreatedAt)
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
	return err
}

func (p *Postgres) CreatePasswordResetRequest(ctx context.Context, token string, email string) (*Request, error) {
	query := "INSERT INTO requests (email, token, expires_at) VALUES ($1, $2, $3) RETURNING *"
	row := p.db.QueryRowContext(ctx, query, email, token, time.Now().Add(5*time.Minute).Truncate(time.Minute))
	if row.Err() != nil {
		return nil, row.Err()
	}

	var request Request
	err := row.Scan(&request.ID, &request.Email, &request.Token, &request.IsUsed, &request.ExpiresAt, &request.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (p *Postgres) GetRequestByToken(ctx context.Context, token string) (*Request, error) {
	query := "SELECT * FROM requests WHERE token = $1"

	var request Request
	err := p.db.GetContext(ctx, &request, query, token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrRequestNotFound
	}
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (p *Postgres) GetRequestByEmail(ctx context.Context, email string) (*Request, error) {
	query := "SELECT * FROM requests WHERE email = $1"

	var request Request
	err := p.db.GetContext(ctx, &request, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrRequestNotFound
	}
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (p *Postgres) MarkRequestAsUsed(ctx context.Context, token string) error {
	query := "UPDATE requests SET is_used = true WHERE token = $1"

	_, err := p.db.ExecContext(ctx, query, token)
	return err
}

func (p *Postgres) ConfirmEmail(ctx context.Context, email string, token string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := "UPDATE users SET is_email_confirmed=true WHERE email = $1"
	_, err = tx.ExecContext(ctx, query, email)
	if err != nil {
		return err
	}

	query = "UPDATE requests SET is_used=true WHERE token = $1"
	_, err = tx.ExecContext(ctx, query, token)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
