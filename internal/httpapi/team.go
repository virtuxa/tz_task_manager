package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

type teamHandler struct {
	service TeamService
}

type createTeamRequest struct {
	Name string `json:"name"`
}

type inviteRequest struct {
	Email string    `json:"email"`
	Role  team.Role `json:"role"`
}

type changeRoleRequest struct {
	Role team.Role `json:"role"`
}

type teamResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	Role      team.Role `json:"role"`
}

func newTeamHandler(service TeamService) *teamHandler {
	return &teamHandler{service: service}
}

func (handler *teamHandler) create(writer http.ResponseWriter, request *http.Request) {
	userID, ok := userIDFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	var input createTeamRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	createdTeam, err := handler.service.Create(request.Context(), userID, input.Name)
	if err != nil {
		writeTeamError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, toTeamResponse(createdTeam))
}

func (handler *teamHandler) list(writer http.ResponseWriter, request *http.Request) {
	userID, ok := userIDFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	teams, err := handler.service.List(request.Context(), userID)
	if err != nil {
		writeTeamError(writer, err)
		return
	}

	response := make([]teamResponse, 0, len(teams))
	for _, item := range teams {
		response = append(response, toTeamResponse(item))
	}

	writeJSON(writer, http.StatusOK, response)
}

func (handler *teamHandler) invite(writer http.ResponseWriter, request *http.Request) {
	userID, teamID, ok := requestUserAndTeamID(writer, request)
	if !ok {
		return
	}

	var input inviteRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if err := handler.service.Invite(request.Context(), userID, teamID, input.Email, input.Role); err != nil {
		writeTeamError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (handler *teamHandler) changeRole(writer http.ResponseWriter, request *http.Request) {
	userID, teamID, ok := requestUserAndTeamID(writer, request)
	if !ok {
		return
	}

	memberID, err := pathID(request, "userID")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid user ID")
		return
	}

	var input changeRoleRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if err := handler.service.ChangeRole(request.Context(), userID, teamID, memberID, input.Role); err != nil {
		writeTeamError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (handler *teamHandler) removeMember(writer http.ResponseWriter, request *http.Request) {
	userID, teamID, ok := requestUserAndTeamID(writer, request)
	if !ok {
		return
	}

	memberID, err := pathID(request, "userID")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid user ID")
		return
	}

	if err := handler.service.RemoveMember(request.Context(), userID, teamID, memberID); err != nil {
		writeTeamError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (handler *teamHandler) delete(writer http.ResponseWriter, request *http.Request) {
	userID, teamID, ok := requestUserAndTeamID(writer, request)
	if !ok {
		return
	}

	if err := handler.service.Delete(request.Context(), userID, teamID); err != nil {
		writeTeamError(writer, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func requestUserAndTeamID(writer http.ResponseWriter, request *http.Request) (int64, int64, bool) {
	userID, ok := userIDFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
		return 0, 0, false
	}

	teamID, err := pathID(request, "teamID")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid team ID")
		return 0, 0, false
	}

	return userID, teamID, true
}

func pathID(request *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(request.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid path ID")
	}

	return id, nil
}

func toTeamResponse(item team.Team) teamResponse {
	return teamResponse{
		ID:        item.ID,
		Name:      item.Name,
		CreatedBy: item.CreatedBy,
		Role:      item.Role,
	}
}

func writeTeamError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, team.ErrInvalidInput):
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request")
	case errors.Is(err, team.ErrAlreadyMember):
		writeError(writer, http.StatusConflict, "already_member", "user is already a team member")
	case errors.Is(err, team.ErrOpenAssignedTasks):
		writeError(writer, http.StatusConflict, "open_assigned_tasks", "member has open assigned tasks")
	case errors.Is(err, team.ErrNotFound), errors.Is(err, team.ErrMemberNotFound):
		writeError(writer, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, team.ErrForbidden), errors.Is(err, team.ErrOwnerMembershipChange):
		writeError(writer, http.StatusForbidden, "forbidden", "action is forbidden")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
