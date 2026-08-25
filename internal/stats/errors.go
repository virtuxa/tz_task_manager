package stats

import "errors"

var (
	ErrInvalidInput = errors.New("invalid stats input")
	ErrForbidden    = errors.New("stats action is forbidden")
)
