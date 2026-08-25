package team

import "errors"

var (
	ErrInvalidInput          = errors.New("invalid team input")
	ErrForbidden             = errors.New("team action is forbidden")
	ErrNotFound              = errors.New("team not found")
	ErrMemberNotFound        = errors.New("team member not found")
	ErrAlreadyMember         = errors.New("user is already a team member")
	ErrOpenAssignedTasks     = errors.New("member has open assigned tasks")
	ErrOwnerMembershipChange = errors.New("owner membership cannot be changed")
)
