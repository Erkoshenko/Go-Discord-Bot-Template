package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	DiscordId int64
	Money     int32
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateTableIfNotExists(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS users(
		discord_id TEXT PRIMARY KEY,
		money INTEGER DEFAULT 0
	)`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка при создания таблицы: %w", err)
	}
	return nil
}
