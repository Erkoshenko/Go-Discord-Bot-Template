package examples

import (
	"context"
	"os"
	"testing"

	"bot/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresUserRepository_Integration(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Пропускаем интеграционный тест: не задана переменная TEST_DATABASE_URL")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("не удалось подключиться к базе: %v", err)
	}
	defer pool.Close()

	userRepo := repository.NewPostgresUserRepository(pool)

	err = userRepo.CreateTableIfNotExists(ctx)
	if err != nil {
		t.Fatalf("ошибка создания таблицы: %v", err)
	}

	// discordID := "test_user_999"

	// err = userRepo.AddBalance(ctx, discordID, 500)
	// if err != nil { t.Fatalf(...) }

	// balance, err := userRepo.GetBalance(ctx, discordID)
	// if err != nil {
	// 	t.Fatalf("ошибка получения баланса из БД: %v", err)
	// }

	// if balance != 500 {
	// 	t.Errorf("ожидали баланс 500 в базе, а получили %d", balance)
	// }
}
