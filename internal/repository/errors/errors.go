package errors

import "errors"

var (
	ErrUserNotFound      = errors.New("repository.User: user not found")
	ErrUserAlreadyExists = errors.New("repository.User: user already exists")
)
