package team

import "time"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type Team struct {
	ID        int64
	Name      string
	CreatedBy int64
	CreatedAt time.Time
	Role      Role
}

type Member struct {
	UserID int64
	Role   Role
}
