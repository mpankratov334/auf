package repo

import (
	"TemplatestPGSQL/internal/config"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type Repository interface {
	CreateUser(ctx context.Context, username, passwordHash string) error
	GetUser(ctx context.Context, username string) (*User, error)
	InitTables(ctx context.Context) error
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(ctx context.Context, cfg config.Memory) (Repository, error) {
	connString := fmt.Sprintf(
		`user=%s password=%s host=%s port=%d dbname=%s sslmode=%s 
        pool_max_conns=%d pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s`,
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
		cfg.PoolMaxConns,
		cfg.PoolMaxConnLifetime.String(),
		cfg.PoolMaxConnIdleTime.String(),
	)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse DB config")
	}

	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DB connection pool")
	}

	return &repository{pool}, nil
}

func (r *repository) InitTables(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, InitQuery)
	if err != nil {
		return errors.Wrap(err, "failed to initialise tables")
	}
	return nil
}

func (r *repository) CreateUser(ctx context.Context, username, passwordHash string) error {
	_, err := r.pool.Exec(
		ctx,
		InsertQuery,
		username,
		passwordHash,
	)

	if err != nil {
		return fmt.Errorf("db user insert failed: %w", err)
	}

	return nil
}

func (r *repository) GetUser(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, SelectQuery, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
	)

	if err != nil {
		return nil, fmt.Errorf("db query failed: %w", err)
	}

	return &user, nil
}
