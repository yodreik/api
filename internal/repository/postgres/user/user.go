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

type RequestKind string

const (
	RequestKindResetPassword     RequestKind = "reset_password"
	RequestKindEmailConfirmation RequestKind = "email_confirmation"
)

type Postgres struct {
	db *sqlx.DB
}

type User struct {
	ID               string    `db:"id"`
	Email            string    `db:"email"`
	Name             string    `db:"name"`
	PasswordHash     string    `db:"password_hash"`
	IsEmailConfirmed bool      `db:"is_email_confirmed"`
	CreatedAt        time.Time `db:"created_at"`
}

type Request struct {
	ID        string      `db:"id"`
	Kind      RequestKind `db:"kind"`
	Email     string      `db:"email"`
	Token     string      `db:"token"`
	IsUsed    bool        `db:"is_used"`
	ExpiresAt time.Time   `db:"expires_at"`
	CreatedAt time.Time   `db:"created_at"`
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
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.IsEmailConfirmed, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *Postgres) CreateWithEmailConfirmationRequest(ctx context.Context, email string, name string, passwordHash string, token string) (*User, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := "INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *"
	row := tx.QueryRowContext(ctx, query, email, name, passwordHash)

	if pqErr, ok := row.Err().(*pq.Error); ok && pqErr.Code == "23505" {
		return nil, repoerr.ErrUserAlreadyExists
	}
	if row.Err() != nil {
		return nil, row.Err()
	}

	var user User
	err = row.Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.IsEmailConfirmed, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	query = "INSERT INTO requests (kind, email, token, expires_at) VALUES ($1, $2, $3, $4)"
	_, err = tx.ExecContext(ctx, query, RequestKindEmailConfirmation, email, token, time.Now().Add(48*time.Hour))
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
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

func (p *Postgres) CreatePasswordResetRequest(ctx context.Context, token string, email string) (*Request, error) {
	return p.CreateRequest(ctx, RequestKindResetPassword, email, token, time.Now().Add(15*time.Minute))
}

func (p *Postgres) CreateEmailConfirmationRequest(ctx context.Context, token string, email string) (*Request, error) {
	return p.CreateRequest(ctx, RequestKindEmailConfirmation, email, token, time.Now().Add(48*time.Hour))
}

func (p *Postgres) CreateRequest(ctx context.Context, kind RequestKind, email string, token string, expiresAt time.Time) (*Request, error) {
	query := "INSERT INTO requests (kind, email, token, expires_at) VALUES ($1, $2, $3, $4) RETURNING *"
	row := p.db.QueryRowContext(ctx, query, kind, email, token, expiresAt)
	if row.Err() != nil {
		return nil, row.Err()
	}

	var request Request
	err := row.Scan(&request.ID, &request.Kind, &request.Email, &request.Token, &request.IsUsed, &request.ExpiresAt, &request.CreatedAt)
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
