// Package apperr defines the client-safe error vocabulary shared by every
// boundary (HTTP and WebSocket). Domain code returns *Error for anything a
// client caused; any other error is treated as internal and never leaked.
package apperr

import "errors"

type Kind uint8

const (
	Invalid Kind = iota + 1
	Unauthorized
	Forbidden
	NotFound
	Conflict
	TooLarge
	Unsupported
	RateLimited
)

type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func New(kind Kind, msg string) *Error { return &Error{Kind: kind, Msg: msg} }

// As extracts a client-safe error, reporting false for internal errors.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
