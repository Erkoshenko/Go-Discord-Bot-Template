package database

import (
	"bot/internal/config"
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect создаёт пул соединений с Postgres и проверяет связь.
// Пул принадлежит вызывающему: он создаёт его в main и закрывает через pool.Close().
// Глобальных переменных здесь нет — пул передаётся вниз как зависимость.
func Connect(ctx context.Context, cfg config.PGConfig) (*pgxpool.Pool, error) {
	// url.URL сам экранирует логин и пароль со спецсимволами.
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),
		Path:     "/" + cfg.DBName,
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}}.Encode(),
	}

	poolConfig, err := pgxpool.ParseConfig(dsn.String())
	if err != nil {
		return nil, fmt.Errorf("не удалось разобрать конфиг Postgres: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать пул Postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("не удалось подключиться к Postgres (%s): %w", poolConfig.ConnConfig.Host, err)
	}

	return pool, nil
}
