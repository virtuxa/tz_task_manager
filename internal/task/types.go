package task

import (
	"encoding/json"
	"time"
)

type Task struct {
	ID          int64
	TeamID      int64
	Title       string
	Description string
	Status      Status
	CreatedBy   int64
	AssigneeID  *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClosedAt    *time.Time
	Version     int64
}

type History struct {
	ID        int64
	TaskID    int64
	ChangedBy int64
	Changes   json.RawMessage
	CreatedAt time.Time
}

type CreateInput struct {
	TeamID      int64
	Title       string
	Description string
	Status      Status
	AssigneeID  *int64
}

type UpdateInput struct {
	Version         int64
	Title           *string
	Description     *string
	Status          *Status
	AssigneeID      *int64
	AssigneeIDIsSet bool
}

type Filter struct {
	TeamID     int64
	Status     *Status
	AssigneeID *int64
	Limit      int
	Offset     int
}
