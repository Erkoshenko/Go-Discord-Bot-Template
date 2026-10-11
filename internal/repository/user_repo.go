package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateTableIfNotExists(ctx context.Context) error
}

type User struct {
	DiscordId string
	Money     int32
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
		money INTEGER DEFAULT 0
	)`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("не удалось создать таблицу пользователей: %w", err)
	}
	return nil
}
