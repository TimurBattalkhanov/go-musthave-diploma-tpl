package repository

import (
	"context"
	"errors"
)

var ErrLoginTaken = errors.New("login is already taken")
var ErrUserNotFound = errors.New("user not found")

type Storage interface {
	RegisterUser(ctx context.Context, login, password string) error
	GetPasswordHash(ctx context.Context, login string) (string, error)
}
