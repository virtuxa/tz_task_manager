package user

import "context"

type Repository interface {
	Create(context.Context, User) (User, error)
	FindByEmail(context.Context, string) (User, error)
}

type PasswordManager interface {
	Hash(string) (string, error)
	Compare(string, string) error
}

type TokenIssuer interface {
	Issue(int64) (string, error)
}
