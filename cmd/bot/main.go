// Команда запуска бота. Здесь только управление жизненным циклом:
// загрузка настроек, запуск App, остановка по сигналу и перезапуск.
//
// Сигналы:
//
//	Ctrl+C / SIGTERM — корректная остановка;
//	SIGHUP           — перезапуск: заново читаются .env и config.yaml (на Linux/macOS).
package main

import (
	"bot/internal/app"
	"bot/internal/config"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("бот остановлен с ошибкой", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// ctx отменяется по Ctrl+C или SIGTERM (остановка в Docker).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)

	for {
		// Настройки читаются при каждом запуске, поэтому перезапуск подхватывает правки.
		if err := loadEnv(); err != nil {
			return err
		}
		settings, err := config.Load("configs/config.yaml")
		if err != nil {
			return fmt.Errorf("загрузка настроек: %w", err)
		}

		application := app.New(settings, log)
		runCtx, cancel := context.WithCancel(ctx)
		go func() {
			select {
			case <-hup:
				log.Info("получен SIGHUP, перезапускаюсь")
				application.RequestRestart()
			case <-runCtx.Done():
			}
		}()

		log.Info("запуск бота")
		err = application.Run(runCtx)
		cancel()

		switch {
		case errors.Is(err, app.ErrRestart):
			time.Sleep(time.Second) // дать Discord закрыть прошлое соединение
			continue
		case err != nil:
			return err
		default:
			log.Info("бот остановлен")
			return nil
		}
	}
}

// loadEnv читает .env. Файл необязателен: на сервере переменные задаёт окружение.
// Overload перезаписывает уже заданные значения, чтобы при перезапуске подхватить правки.
func loadEnv() error {
	if err := godotenv.Overload(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("не удалось прочитать .env: %w", err)
	}
	return nil
}
