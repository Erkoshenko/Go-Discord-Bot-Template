package examples

import (
	"context"
	"errors"
	"fmt"
)

type UserRepository interface {
	GetBalance(ctx context.Context, discordID string) (int32, error)
}

type MockUserRepository struct {
	GetBalanceFunc func(ctx context.Context, discordID string) (int32, error)
}

func (m *MockUserRepository) GetBalance(ctx context.Context, discordID string) (int32, error) {
	if m.GetBalanceFunc != nil {
		return m.GetBalanceFunc(ctx, discordID)
	}
	return 0, nil
}

func main() {
	fmt.Println("=== Запуск автономного примера с моком ===")

	// Сценарий 1: Пользователь с балансом 777
	fakeRepo1 := &MockUserRepository{
		GetBalanceFunc: func(ctx context.Context, discordID string) (int32, error) {
			if discordID == "123" {
				return 20000, nil
			}
			return 0, nil
		},
	}

	ctx := context.Background()
	balance, _ := fakeRepo1.GetBalance(ctx, "123")
	fmt.Printf("Тест 1 (успешный): баланс ID 123 = %d\n", balance)

	// Сценарий 2: База данных упала и вернула ошибку
	fakeRepo2 := &MockUserRepository{
		GetBalanceFunc: func(ctx context.Context, discordID string) (int32, error) {
			return 0, errors.New("ошибка подключения к базе данных")
		},
	}

	_, err := fakeRepo2.GetBalance(ctx, "456")
	if err != nil {
		fmt.Printf("Тест 2 (с ошибкой): поймали ожидаемую ошибку -> %v\n", err)
	}

	fmt.Println("=== Пример успешно завершен ===")
}
