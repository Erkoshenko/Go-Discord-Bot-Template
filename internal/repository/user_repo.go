package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateTableIfNotExists(ctx context.Context) error
	// GetBalance возвращает баланс; для неизвестного пользователя — 0, а не ошибку.
	GetBalance(ctx context.Context, discordID string) (int64, error)
	// AddBalance меняет баланс на delta (можно отрицательное) и возвращает новый.
	// Пользователь создаётся автоматически, если его ещё нет.
	AddBalance(ctx context.Context, discordID string, delta int64) (int64, error)
}

type User struct {
	DiscordId string
	Money     int64
}

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(db *pgxpool.Pool) UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) CreateTableIfNotExists(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS users(
		discord_id TEXT PRIMARY KEY,
		money BIGINT NOT NULL DEFAULT 0
	)`
	if _, err := r.db.Exec(ctx, query); err != nil {
		return fmt.Errorf("не удалось создать таблицу пользователей: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) GetBalance(ctx context.Context, discordID string) (int64, error) {
	var money int64
	err := r.db.QueryRow(ctx, `SELECT money FROM users WHERE discord_id = $1`, discordID).Scan(&money)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("не удалось получить баланс: %w", err)
	}
	return money, nil
}

func (r *PostgresUserRepository) AddBalance(ctx context.Context, discordID string, delta int64) (int64, error) {
	var money int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (discord_id, money) VALUES ($1, $2)
		ON CONFLICT (discord_id) DO UPDATE SET money = users.money + EXCLUDED.money
		RETURNING money`,
		discordID, delta,
	).Scan(&money)
	if err != nil {
		return 0, fmt.Errorf("не удалось изменить баланс: %w", err)
	}
	return money, nil
}
