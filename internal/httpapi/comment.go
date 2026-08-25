package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/comment"
)

type commentHandler struct {
	service CommentService
}

type createCommentRequest struct {
	Content string `json:"content"`
}

type commentResponse struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func newCommentHandler(service CommentService) *commentHandler {
	return &commentHandler{service: service}
}

func (handler *commentHandler) create(writer http.ResponseWriter, request *http.Request) {
	userID, taskID, ok := requestUserAndTaskID(writer, request)
	if !ok {
		return
	}

	var input createCommentRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	createdComment, err := handler.service.Create(request.Context(), userID, taskID, input.Content)
	if err != nil {
		writeCommentError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, toCommentResponse(createdComment))
}

func (handler *commentHandler) list(writer http.ResponseWriter, request *http.Request) {
	userID, taskID, ok := requestUserAndTaskID(writer, request)
	if !ok {
		return
	}

	comments, err := handler.service.List(request.Context(), userID, taskID)
	if err != nil {
		writeCommentError(writer, err)
		return
	}

	response := make([]commentResponse, 0, len(comments))
	for _, item := range comments {
		response = append(response, toCommentResponse(item))
	}

	writeJSON(writer, http.StatusOK, response)
}

func toCommentResponse(item comment.Comment) commentResponse {
	return commentResponse{
		ID:        item.ID,
		TaskID:    item.TaskID,
		UserID:    item.UserID,
		Content:   item.Content,
		CreatedAt: item.CreatedAt,
	}
}

func writeCommentError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, comment.ErrInvalidInput):
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request")
	case errors.Is(err, comment.ErrNotFound):
		writeError(writer, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, comment.ErrForbidden):
		writeError(writer, http.StatusForbidden, "forbidden", "action is forbidden")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
