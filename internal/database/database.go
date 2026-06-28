package database

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	config.MaxConns = 10
	config.MinConns = 1

	return pgxpool.NewWithConfig(ctx, config)
}