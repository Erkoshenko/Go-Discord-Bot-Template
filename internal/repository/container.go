package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repositories — ОБЩИЕ репозитории: данные, которые нужны нескольким модулям
// (пользователи, баланс). Данные одного модуля лежат в самом модуле,
// например internal/moderation/repository.go.
type Repositories struct {
	Users UserRepository
}

func NewRepositories(db *pgxpool.Pool) *Repositories {
	return &Repositories{
		Users: NewPostgresUserRepository(db),
	}
}

// CreateTables создаёт таблицы общих репозиториев. Вызывается из App при старте.
// Новый общий репозиторий нужно добавить и сюда.
func (r *Repositories) CreateTables(ctx context.Context) error {
	creators := []struct {
		name string
		repo interface {
			CreateTableIfNotExists(context.Context) error
		}
	}{
		{"users", r.Users},
	}
	for _, c := range creators {
		if err := c.repo.CreateTableIfNotExists(ctx); err != nil {
			return fmt.Errorf("репозиторий %s: %w", c.name, err)
		}
	}
	return nil
}
