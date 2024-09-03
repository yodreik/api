package errors

import "errors"

var (
	ErrUserNotFound                 = errors.New("repository.User: user not found")
	ErrUserAlreadyExists            = errors.New("repository.User: user already exists")
	ErrWorkoutNotFound              = errors.New("repository.Workout: workout not found")
	ErrPasswordResetRequestNotFound = errors.New("repository.Cache: password reset request not found")
)
