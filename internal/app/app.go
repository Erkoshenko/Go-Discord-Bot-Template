// Package app собирает бота из частей: база, общие репозитории, модули, сессия Discord.
package app

import (
	"bot/internal/commands"
	"bot/internal/config"
	"bot/internal/database"
	"bot/internal/module"
	"bot/internal/repository"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ErrRestart возвращается из Run, когда кто-то запросил перезапуск (RequestRestart).
// cmd/bot по этому признаку запускает приложение заново.
var ErrRestart = errors.New("запрошен перезапуск")

type App struct {
	settings config.Settings
	log      *slog.Logger
	restart  chan struct{}
}

func New(settings config.Settings, log *slog.Logger) *App {
	return &App{settings: settings, log: log, restart: make(chan struct{}, 1)}
}

// RequestRestart просит Run завершиться с ErrRestart. Безопасно вызывать из любого места
// и несколько раз подряд.
func (a *App) RequestRestart() {
	select {
	case a.restart <- struct{}{}:
	default:
	}
}

// Run собирает и запускает бота, блокируется до отмены ctx (возвращает nil)
// или до запроса перезапуска (возвращает ErrRestart). При выходе всё закрывает:
// сначала сессию Discord, потом базу.
func (a *App) Run(ctx context.Context) error {
	initCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 1. База и общие репозитории.
	db, err := database.Connect(initCtx, a.settings.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	repos := repository.NewRepositories(db)
	if err := repos.CreateTables(initCtx); err != nil {
		return err
	}

	// 2. Модули.
	mods := buildModules(module.Deps{
		DB:             db,
		Repos:          repos,
		Settings:       a.settings,
		Log:            a.log,
		RequestRestart: a.RequestRestart,
	})

	registry := commands.NewRegistry(a.log, a.settings.TestGuildID)
	intents := discordgo.IntentsGuilds
	for _, m := range mods {
		if err := m.Init(initCtx); err != nil {
			return fmt.Errorf("модуль %s: %w", m.Name(), err)
		}
		if err := registry.Add(m.Commands()...); err != nil {
			return fmt.Errorf("модуль %s: %w", m.Name(), err)
		}
		intents |= m.Intents()
		a.log.Info("модуль подключён", "module", m.Name())
	}

	// 3. Сессия Discord.
	sess, err := discordgo.New("Bot " + a.settings.DiscordToken)
	if err != nil {
		return fmt.Errorf("создание сессии Discord: %w", err)
	}
	sess.Identify.Intents = intents

	registry.Attach(sess)
	sess.AddHandler(func(_ *discordgo.Session, r *discordgo.Ready) {
		a.log.Info("бот запущен", "user", r.User.Username)
	})
	for _, m := range mods {
		m.Register(sess)
	}

	if err := sess.Open(); err != nil {
		return fmt.Errorf("подключение к Discord: %w", err)
	}
	defer sess.Close()

	// 4. Ждём остановки или запроса перезапуска.
	select {
	case <-ctx.Done():
		return nil
	case <-a.restart:
		return ErrRestart
	}
}
