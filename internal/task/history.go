package task

import (
	"encoding/json"
	"fmt"
)

type FieldChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

type taskSnapshot struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      Status `json:"status"`
	CreatedBy   int64  `json:"created_by"`
	AssigneeID  *int64 `json:"assignee_id"`
}

func marshalChanges(changes map[string]FieldChange) (json.RawMessage, error) {
	encodedChanges, err := json.Marshal(changes)
	if err != nil {
		return nil, fmt.Errorf("marshal task changes: %w", err)
	}

	return encodedChanges, nil
}

func snapshot(task Task) taskSnapshot {
	return taskSnapshot{
		TeamID:      task.TeamID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedBy:   task.CreatedBy,
		AssigneeID:  task.AssigneeID,
	}
}
