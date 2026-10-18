// Package module — контракт, который выполняет каждый модуль бота.
package module

import (
	"bot/internal/commands"
	"bot/internal/config"
	"bot/internal/repository"
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps — то, что App даёт каждому модулю при создании.
type Deps struct {
	DB             *pgxpool.Pool
	Repos          *repository.Repositories // общие репозитории (users и др.)
	Settings       config.Settings
	Log            *slog.Logger
	RequestRestart func() // попросить cmd перезапустить бота
}

// Module — самостоятельная часть бота (модерация, экономика, тикеты...).
type Module interface {
	// Name — имя модуля для логов.
	Name() string
	// Init вызывается один раз при старте: создание таблиц модуля и т.п.
	Init(ctx context.Context) error
	// Commands — слэш-команды модуля.
	Commands() []commands.Command
	// Intents — дополнительные интенты Discord, нужные модулю (IntentsGuilds уже включён).
	Intents() discordgo.Intent
	// Register вызывается до подключения к Discord: здесь модуль подписывается
	// на свои события (session.AddHandler) — сообщения, вход на сервер, кнопки.
	Register(s *discordgo.Session)
}

// Base — пустая реализация необязательных методов. Встройте её в модуль
// и переопределите только то, что нужно.
type Base struct{}

func (Base) Init(context.Context) error   { return nil }
func (Base) Commands() []commands.Command { return nil }
func (Base) Intents() discordgo.Intent    { return 0 }
func (Base) Register(*discordgo.Session)  {}
