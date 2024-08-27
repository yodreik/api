package errors

import "errors"

var (
	ErrUserNotFound = errors.New("repository.User: user not found")
)
