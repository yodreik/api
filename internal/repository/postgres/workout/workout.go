package workout

import (
	repoerr "api/internal/repository/errors"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type Postgres struct {
	db *sqlx.DB
}

type Workout struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Date      time.Time `db:"date"`
	Duration  int       `db:"duration"`
	Kind      string    `db:"kind"`
	CreatedAt time.Time `db:"created_at"`
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) Create(ctx context.Context, userID string, date time.Time, duration int, kind string) (*Workout, error) {
	query := "INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *"
	row := p.db.QueryRowContext(ctx, query, userID, date, duration, kind)
	if row.Err() != nil {
		return nil, row.Err()
	}

	var workout Workout
	err := row.Scan(&workout.ID, &workout.UserID, &workout.Date, &workout.Duration, &workout.Kind, &workout.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &workout, err
}

func (p *Postgres) GetByID(ctx context.Context, id string) (*Workout, error) {
	query := "SELECT * FROM workouts WHERE id = $1"

	var workout Workout
	err := p.db.GetContext(ctx, &workout, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrWorkoutNotFound
	}
	if err != nil {
		return nil, err
	}

	return &workout, nil
}

func (p *Postgres) GetUserWorkouts(ctx context.Context, userID string, begin time.Time, end time.Time) ([]Workout, error) {
	query := "SELECT * FROM workouts WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date ASC"

	var workouts []Workout
	err := p.db.SelectContext(ctx, &workouts, query, userID, begin, end)
	if errors.Is(err, sql.ErrNoRows) {
		return workouts, nil
	}
	if err != nil {
		return nil, err
	}

	return workouts, nil
}
