package task

import "errors"

var (
	ErrInvalidInput    = errors.New("invalid task input")
	ErrForbidden       = errors.New("task action is forbidden")
	ErrNotFound        = errors.New("task not found")
	ErrVersionConflict = errors.New("task version conflict")
)
