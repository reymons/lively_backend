package core

import "errors"

var (
	ErrEntityNotFound     = errors.New("entity not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUsernameTaken      = errors.New("username taken")
	ErrInactiveStreamKey  = errors.New("inactive stream key")
)
