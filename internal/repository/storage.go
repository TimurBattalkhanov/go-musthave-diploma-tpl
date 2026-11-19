package repository

import (
	"context"
	"errors"

	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/models"
)

var ErrLoginTaken = errors.New("login is already taken")
var ErrUserNotFound = errors.New("user not found")
var ErrOrderIsAlreadyExists = errors.New("order is already exists")
var ErrOrderCreatedByAnotherUser = errors.New("order is already created by another user")

type Storage interface {
	RegisterUser(ctx context.Context, login, password string) error
	GetPasswordHash(ctx context.Context, login string) (string, error)
	CreateOrder(ctx context.Context, number string, login string) error
	GetOrdersByLogin(ctx context.Context, login string) ([]models.Order, error)
}
