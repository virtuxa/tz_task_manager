package comment

import "errors"

var (
	ErrInvalidInput = errors.New("invalid comment input")
	ErrForbidden    = errors.New("comment action is forbidden")
	ErrNotFound     = errors.New("comment task not found")
)
