package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/task"
)

type taskHandler struct {
	service TaskService
}

type createTaskRequest struct {
	TeamID      int64         `json:"team_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      task.Status   `json:"status"`
	AssigneeID  optionalInt64 `json:"assignee_id"`
}

type updateTaskRequest struct {
	Version     int64         `json:"version"`
	Title       *string       `json:"title"`
	Description *string       `json:"description"`
	Status      *task.Status  `json:"status"`
	AssigneeID  optionalInt64 `json:"assignee_id"`
}

type taskResponse struct {
	ID          int64       `json:"id"`
	TeamID      int64       `json:"team_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      task.Status `json:"status"`
	CreatedBy   int64       `json:"created_by"`
	AssigneeID  *int64      `json:"assignee_id"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	ClosedAt    *time.Time  `json:"closed_at"`
	Version     int64       `json:"version"`
}

type taskHistoryResponse struct {
	ID        int64           `json:"id"`
	TaskID    int64           `json:"task_id"`
	ChangedBy int64           `json:"changed_by"`
	Changes   json.RawMessage `json:"changes"`
	CreatedAt time.Time       `json:"created_at"`
}

type optionalInt64 struct {
	value *int64
	set   bool
}

func (value *optionalInt64) UnmarshalJSON(data []byte) error {
	value.set = true
	if string(data) == "null" {
		value.value = nil
		return nil
	}

	var decoded int64
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	value.value = &decoded
	return nil
}

func newTaskHandler(service TaskService) *taskHandler {
	return &taskHandler{service: service}
}

func (handler *taskHandler) create(writer http.ResponseWriter, request *http.Request) {
	userID, ok := userIDFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	var input createTaskRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	createdTask, err := handler.service.Create(request.Context(), userID, task.CreateInput{
		TeamID:      input.TeamID,
		Title:       input.Title,
		Description: input.Description,
		Status:      input.Status,
		AssigneeID:  input.AssigneeID.value,
	})
	if err != nil {
		writeTaskError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, toTaskResponse(createdTask))
}

func (handler *taskHandler) list(writer http.ResponseWriter, request *http.Request) {
	userID, ok := userIDFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	filter, err := taskFilterFromQuery(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid task filters")
		return
	}

	tasks, err := handler.service.List(request.Context(), userID, filter)
	if err != nil {
		writeTaskError(writer, err)
		return
	}

	response := make([]taskResponse, 0, len(tasks))
	for _, item := range tasks {
		response = append(response, toTaskResponse(item))
	}

	writeJSON(writer, http.StatusOK, response)
}

func (handler *taskHandler) update(writer http.ResponseWriter, request *http.Request) {
	userID, taskID, ok := requestUserAndTaskID(writer, request)
	if !ok {
		return
	}

	var input updateTaskRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	updatedTask, err := handler.service.Update(request.Context(), userID, taskID, task.UpdateInput{
		Version:         input.Version,
		Title:           input.Title,
		Description:     input.Description,
		Status:          input.Status,
		AssigneeID:      input.AssigneeID.value,
		AssigneeIDIsSet: input.AssigneeID.set,
	})
	if err != nil {
		writeTaskError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, toTaskResponse(updatedTask))
}

func (handler *taskHandler) delete(writer http.ResponseWriter, request *http.Request) {
	userID, taskID, ok := requestUserAndTaskID(writer, request)
	if !ok {
		return
	}

	if err := handler.service.Delete(request.Context(), userID, taskID); err != nil {
		writeTaskError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (handler *taskHandler) history(writer http.ResponseWriter, request *http.Request) {
	userID, taskID, ok := requestUserAndTaskID(writer, request)
	if !ok {
		return
	}

	history, err := handler.service.History(request.Context(), userID, taskID)
	if err != nil {
		writeTaskError(writer, err)
		return
	}

	response := make([]taskHistoryResponse, 0, len(history))
	for _, item := range history {
		response = append(response, taskHistoryResponse{
			ID:        item.ID,
			TaskID:    item.TaskID,
			ChangedBy: item.ChangedBy,
			Changes:   item.Changes,
			CreatedAt: item.CreatedAt,
		})
	}

	writeJSON(writer, http.StatusOK, response)
}

func requestUserAndTaskID(writer http.ResponseWriter, request *http.Request) (int64, int64, bool) {
	userID, ok := userIDFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
		return 0, 0, false
	}

	taskID, err := pathID(request, "taskID")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid task ID")
		return 0, 0, false
	}

	return userID, taskID, true
}

func taskFilterFromQuery(request *http.Request) (task.Filter, error) {
	query := request.URL.Query()
	teamID, err := parseQueryID(query.Get("team_id"))
	if err != nil {
		return task.Filter{}, err
	}

	filter := task.Filter{TeamID: teamID}
	if rawStatus := query.Get("status"); rawStatus != "" {
		status := task.Status(rawStatus)
		filter.Status = &status
	}
	if rawAssigneeID := query.Get("assignee_id"); rawAssigneeID != "" {
		assigneeID, err := parseQueryID(rawAssigneeID)
		if err != nil {
			return task.Filter{}, err
		}
		filter.AssigneeID = &assigneeID
	}
	if rawLimit := query.Get("limit"); rawLimit != "" {
		filter.Limit, err = strconv.Atoi(rawLimit)
		if err != nil {
			return task.Filter{}, err
		}
	}
	if rawOffset := query.Get("offset"); rawOffset != "" {
		filter.Offset, err = strconv.Atoi(rawOffset)
		if err != nil {
			return task.Filter{}, err
		}
	}

	return filter, nil
}

func parseQueryID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid query ID")
	}

	return id, nil
}

func toTaskResponse(item task.Task) taskResponse {
	return taskResponse{
		ID:          item.ID,
		TeamID:      item.TeamID,
		Title:       item.Title,
		Description: item.Description,
		Status:      item.Status,
		CreatedBy:   item.CreatedBy,
		AssigneeID:  item.AssigneeID,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		ClosedAt:    item.ClosedAt,
		Version:     item.Version,
	}
}

func writeTaskError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, task.ErrInvalidInput):
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request")
	case errors.Is(err, task.ErrVersionConflict):
		writeError(writer, http.StatusConflict, "version_conflict", "task version is outdated")
	case errors.Is(err, task.ErrNotFound):
		writeError(writer, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, task.ErrForbidden):
		writeError(writer, http.StatusForbidden, "forbidden", "action is forbidden")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
