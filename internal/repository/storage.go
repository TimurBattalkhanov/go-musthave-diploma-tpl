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
var ErrInsufficientFunds = errors.New("insufficient funds")

type Storage interface {
	RegisterUser(ctx context.Context, login, password string) error
	GetPasswordHash(ctx context.Context, login string) (string, error)
	CreateOrder(ctx context.Context, number string, login string) error
	GetOrdersByLogin(ctx context.Context, login string) ([]models.Order, error)
	GetBalance(ctx context.Context, login string) (models.Balance, error)
	Withdraw(ctx context.Context, login string, amount models.Withdraw) error
	GetWithdrawHistory(ctx context.Context, login string) ([]models.Withdraw, error)
	UpdateOrderStatus(ctx context.Context, number string, status string, accrual float64) error
	GetProcessingNumbers(ctx context.Context) ([]string, error)
}
