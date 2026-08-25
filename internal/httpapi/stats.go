package httpapi

import (
	"errors"
	"net/http"

	"github.com/virtuxa/tz_task_manager/internal/stats"
)

type statsHandler struct {
	service StatsService
}

type statusCountResponse struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type assigneeStatResponse struct {
	UserID           int64  `json:"user_id"`
	Name             string `json:"name"`
	ClosedTasksCount int64  `json:"closed_tasks_count"`
}

type statsResponse struct {
	TeamID                      int64                  `json:"team_id"`
	TasksByStatus               []statusCountResponse  `json:"tasks_by_status"`
	TopAssignees                []assigneeStatResponse `json:"top_assignees"`
	AverageCloseDurationSeconds float64                `json:"average_close_duration_seconds"`
	CommentsCount               int64                  `json:"comments_count"`
}

func newStatsHandler(service StatsService) *statsHandler {
	return &statsHandler{service: service}
}

func (handler *statsHandler) get(writer http.ResponseWriter, request *http.Request) {
	userID, teamID, ok := requestUserAndTeamID(writer, request)
	if !ok {
		return
	}

	result, err := handler.service.Get(request.Context(), userID, teamID)
	if err != nil {
		writeStatsError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, toStatsResponse(result))
}

func toStatsResponse(result stats.Stats) statsResponse {
	statuses := make([]statusCountResponse, 0, len(result.TasksByStatus))
	for _, item := range result.TasksByStatus {
		statuses = append(statuses, statusCountResponse{
			Status: item.Status,
			Count:  item.Count,
		})
	}

	assignees := make([]assigneeStatResponse, 0, len(result.TopAssignees))
	for _, item := range result.TopAssignees {
		assignees = append(assignees, assigneeStatResponse{
			UserID:           item.UserID,
			Name:             item.Name,
			ClosedTasksCount: item.ClosedTasksCount,
		})
	}

	return statsResponse{
		TeamID:                      result.TeamID,
		TasksByStatus:               statuses,
		TopAssignees:                assignees,
		AverageCloseDurationSeconds: result.AverageCloseDurationSeconds,
		CommentsCount:               result.CommentsCount,
	}
}

func writeStatsError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, stats.ErrInvalidInput):
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request")
	case errors.Is(err, stats.ErrForbidden):
		writeError(writer, http.StatusForbidden, "forbidden", "action is forbidden")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
