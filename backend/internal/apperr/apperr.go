// internal/apperr/apperr.go
package apperr

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrExpired   = errors.New("expired")
)
