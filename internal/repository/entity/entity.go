package entity

import "time"

type User struct {
	ID                string    `db:"id"`
	Email             string    `db:"email"`
	Username          string    `db:"username"`
	DisplayName       string    `db:"display_name"`
	AvatarURL         string    `db:"avatar_url"`
	PasswordHash      string    `db:"password_hash"`
	IsPrivate         bool      `db:"is_private"`
	IsConfirmed       bool      `db:"is_confirmed"`
	ConfirmationToken string    `db:"confirmation_token"`
	CreatedAt         time.Time `db:"created_at"`
}

type Workout struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Date      time.Time `db:"date"`
	Duration  int       `db:"duration"`
	Kind      string    `db:"kind"`
	CreatedAt time.Time `db:"created_at"`
}

type Request struct {
	ID        string    `db:"id"`
	Email     string    `db:"email"`
	Token     string    `db:"token"`
	IsUsed    bool      `db:"is_used"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}
