package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/config"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/migration"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(db *sql.DB) (*DBStorage, error) {
	err := migration.Apply(db)
	if err != nil {
		return nil, err
	}
	return &DBStorage{
		db: db,
	}, nil
}

func NewStorage(cfg config.AppConfig) (Storage, func(ctx context.Context) error, error) {
	if cfg.DatabaseURI != "" {
		db, err := sql.Open("pgx", cfg.DatabaseURI)
		if err != nil {
			return nil, nil, err
		}

		store, err := NewDBStorage(db)
		if err != nil {
			db.Close()
			return nil, nil, err
		}
		closeDB := func(context.Context) error {
			return db.Close()
		}
		return store, closeDB, nil
	}
	return nil, nil, fmt.Errorf("no Database URI provided")
}

func (dbs *DBStorage) RegisterUser(ctx context.Context, login string, password string) error {
	var pgErr *pgconn.PgError
	_, err := dbs.db.ExecContext(ctx, "INSERT INTO users (login, password) VALUES ($1, $2)", login, password)
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return ErrLoginTaken
	}
	return err
}

func (dbs *DBStorage) GetPasswordHash(ctx context.Context, login string) (string, error) {
	var userHash string
	err := dbs.db.QueryRowContext(ctx, "SELECT password FROM users WHERE login = $1", login).Scan(&userHash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrUserNotFound
	}
	return userHash, err
}

func (dbs *DBStorage) CreateOrder(ctx context.Context, number string, login string) error {
	var owner string
	res, err := dbs.db.ExecContext(ctx, "INSERT INTO orders (number, login) VALUES ($1, $2) ON CONFLICT(number) DO NOTHING", number, login)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 1 {
		return nil
	}
	err = dbs.db.QueryRowContext(ctx, "SELECT login FROM orders WHERE number = $1", number).Scan(&owner)
	if err != nil {
		return err
	}
	if owner == login {
		return ErrOrderIsAlreadyExists
	}
	return ErrOrderCreatedByAnotherUser
}

func (dbs *DBStorage) GetOrdersByLogin(ctx context.Context, login string) ([]models.Order, error) {
	rows, err := dbs.db.QueryContext(ctx, "SELECT number, status, accrual, uploaded_at FROM orders WHERE login = $1 ORDER BY uploaded_at DESC", login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var accrual sql.NullFloat64
		if err = rows.Scan(&order.Number, &order.Status, &accrual, &order.UploadedAt); err != nil {
			return nil, err
		}
		if accrual.Valid {
			order.Accrual = accrual.Float64
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (dbs *DBStorage) GetBalance(ctx context.Context, login string) (models.Balance, error) {
	var balance models.Balance
	rows, err := dbs.db.QueryContext(ctx, "SELECT balance, withdrawn FROM users WHERE login = $1", login)
	if err != nil {
		return balance, err
	}
	defer rows.Close()
	for rows.Next() {
		if err = rows.Scan(&balance.Current, &balance.Withdrawn); err != nil {
			return balance, err
		}
	}
	if err = rows.Err(); err != nil {
		return balance, err
	}
	return balance, nil
}

func (dbs *DBStorage) Withdraw(ctx context.Context, login string, w models.Withdraw) error {
	tx, err := dbs.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.ExecContext(ctx, "UPDATE users SET balance = balance - $1, withdrawn = withdrawn + $1 WHERE login = $2 AND balance >= $1", w.Sum, login)
	if err != nil {
		return err
	}

	count, err := rows.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrInsufficientFunds
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO withdrawals(order_number, login, sum) VALUES ($1, $2, $3)", w.OrderID, login, w.Sum)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (dbs *DBStorage) GetWithdrawHistory(ctx context.Context, login string) ([]models.Withdraw, error) {
	rows, err := dbs.db.QueryContext(ctx, "SELECT order_number, sum, processed_at FROM withdrawals WHERE login = $1 ORDER BY processed_at DESC", login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ws []models.Withdraw
	for rows.Next() {
		var w models.Withdraw
		if err = rows.Scan(&w.OrderID, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, err
		}
		ws = append(ws, w)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ws, nil
}

func (dbs *DBStorage) UpdateOrderStatus(ctx context.Context, number string, status string, accrual float64) error {
	tx, err := dbs.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var login string
	err = tx.QueryRowContext(ctx,
		"UPDATE orders SET status = $1, accrual = $2 WHERE number = $3 AND status NOT IN ('PROCESSED', 'INVALID') RETURNING login",
		status, accrual, number).Scan(&login)

	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE users SET balance = balance + $1 WHERE login = $2", accrual, login)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (dbs *DBStorage) GetProcessingNumbers(ctx context.Context) ([]string, error) {
	rows, err := dbs.db.QueryContext(ctx, "SELECT number FROM orders WHERE status IN ('NEW', 'PROCESSING')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []string
	for rows.Next() {
		var number string
		if err = rows.Scan(&number); err != nil {
			return nil, err
		}
		orders = append(orders, number)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}
