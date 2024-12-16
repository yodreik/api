package workout

import (
	"api/internal/repository/entity"
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

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db: db}
}

func (p *Postgres) Create(ctx context.Context, userID string, date time.Time, duration int, kind string) (*entity.Workout, error) {
	query := "INSERT INTO workouts (user_id, date, duration, kind) VALUES ($1, $2, $3, $4) RETURNING *"
	row := p.db.QueryRowContext(ctx, query, userID, date, duration, kind)
	if row.Err() != nil {
		return nil, row.Err()
	}

	var workout entity.Workout
	err := row.Scan(&workout.ID, &workout.UserID, &workout.Date, &workout.Duration, &workout.Kind, &workout.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &workout, err
}

func (p *Postgres) Delete(ctx context.Context, workoutID string) error {
	query := "DELETE FROM workouts WHERE id = $1"

	var err error
	// TODO: Return repoerr.ErrWorkoutNotFound if nothing to delete
	_, err = p.db.ExecContext(ctx, query, workoutID)
	if err != nil {
		return nil
	}

	return err
}

func (p *Postgres) GetByID(ctx context.Context, id string) (*entity.Workout, error) {
	query := "SELECT * FROM workouts WHERE id = $1"

	var workout entity.Workout
	err := p.db.GetContext(ctx, &workout, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrWorkoutNotFound
	}
	if err != nil {
		return nil, err
	}

	return &workout, nil
}

func (p *Postgres) GetUserWorkouts(ctx context.Context, userID string, begin time.Time, end time.Time) ([]entity.Workout, error) {
	query := "SELECT * FROM workouts WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date ASC"

	var workouts []entity.Workout
	err := p.db.SelectContext(ctx, &workouts, query, userID, begin, end)
	if errors.Is(err, sql.ErrNoRows) {
		return workouts, nil
	}
	if err != nil {
		return nil, err
	}

	return workouts, nil
}

func (p *Postgres) GetAllUserWorkouts(ctx context.Context, userID string) ([]entity.Workout, error) {
	query := "SELECT * FROM workouts WHERE user_id = $1 ORDER BY date ASC"

	var workouts []entity.Workout
	err := p.db.SelectContext(ctx, &workouts, query, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return workouts, nil
	}
	if err != nil {
		return nil, err
	}

	return workouts, nil
}
