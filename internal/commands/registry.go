// Package commands — каркас слэш-команд: интерфейс Command и реестр.
// Здесь нет самих команд: каждая команда живёт в своём модуле.
package commands

import (
	"bot/pkg/respond"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// Command — одна слэш-команда.
type Command interface {
	Definition() *discordgo.ApplicationCommand
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

// Registry хранит команды всех модулей, регистрирует их в Discord
// и направляет вызовы нужной команде.
type Registry struct {
	log     *slog.Logger
	guildID string // пусто — глобальная регистрация
	cmds    map[string]Command
}

func NewRegistry(log *slog.Logger, guildID string) *Registry {
	return &Registry{log: log, guildID: guildID, cmds: make(map[string]Command)}
}

// Add добавляет команды. Одинаковые имена у разных модулей — ошибка.
func (r *Registry) Add(cmds ...Command) error {
	for _, c := range cmds {
		name := c.Definition().Name
		if _, exists := r.cmds[name]; exists {
			return fmt.Errorf("команда /%s зарегистрирована дважды", name)
		}
		r.cmds[name] = c
	}
	return nil
}

// Attach подключает реестр к сессии: регистрация команд при запуске и приём вызовов.
func (r *Registry) Attach(s *discordgo.Session) {
	s.AddHandler(r.onReady)
	s.AddHandler(r.onInteraction)
}

func (r *Registry) onReady(s *discordgo.Session, _ *discordgo.Ready) {
	defs := make([]*discordgo.ApplicationCommand, 0, len(r.cmds))
	for _, c := range r.cmds {
		defs = append(defs, c.Definition())
	}
	if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, r.guildID, defs); err != nil {
		r.log.Error("не удалось зарегистрировать команды", "err", err)
		return
	}
	r.log.Info("команды зарегистрированы", "count", len(defs))
}

func (r *Registry) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return // кнопки, модалки и автокомплит обрабатывают сами модули через Register
	}

	name := i.ApplicationCommandData().Name
	cmd, ok := r.cmds[name]
	if !ok {
		return
	}

	// Паника в одной команде не должна ронять бота.
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Error("паника в команде", "command", name, "panic", rec)
			replyError(s, i)
		}
	}()

	if err := cmd.Execute(s, i); err != nil {
		r.log.Error("ошибка в команде", "command", name, "err", err)
		// Если команда уже ответила сама, Discord отклонит повторный ответ — это нормально.
		replyError(s, i)
	}
}

func replyError(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = respond.InteractionRespond(s, i, "❌ Что-то пошло не так. Попробуйте позже.", true)
}
