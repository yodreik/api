package errors

import "errors"

var (
	ErrUserNotFound      = errors.New("repository.User: user not found")
	ErrUserAlreadyExists = errors.New("repository.User: user already exists")
	ErrRequestNotFound   = errors.New("repository.User: request not found")
	ErrWorkoutNotFound   = errors.New("repository.Workout: workout not found")
)
