package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/config"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/migration"
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
