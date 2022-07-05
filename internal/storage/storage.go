package storage

import (
	"context"
	"database/sql"
	"gophermart/internal/domain"
)

type Storages struct {
	Users        Users
}

func NewStorages(db *sql.DB) *Storages {
	return &Storages{
		Users: NewUserStorage(db),
	}
}
//**********************************************************************************************************************
type Users interface {
	Create(ctx context.Context, user domain.User) error
	GetByCredentials(ctx context.Context, login, password string) (domain.User, error)
	SetSession(ctx context.Context, userID int, session domain.Session) error

	GetOrderByNumber(ctx context.Context, orderNumber string) (domain.Order, error)
	GetOrdersByUser(ctx context.Context, userID int) ([]domain.Order, error)
	GetOrdersByStatus(ctx context.Context, orderStatuses []domain.OrderStatus) ([]domain.Order, error)
	AddOrder(ctx context.Context, order domain.Order) error
	UpdateOrders(ctx context.Context, orders []domain.Order) error

	GetUserBalance(ctx context.Context, userID int) (domain.Balance, error)
	GetUserWithdrawals(ctx context.Context, userID int) ([]domain.Withdrawal, error)
	AddWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error
}