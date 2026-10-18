package moderation

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Gateway — всё, что сервис делает в Discord. Сервис знает только этот интерфейс,
// поэтому его можно проверить в тесте без настоящего Discord (см. service_test.go).
type Gateway interface {
	BotID() string
	Timeout(guildID, userID string, until time.Time) error
	Kick(guildID, userID, reason string) error
	Ban(guildID, userID, reason string, deleteDays int) error
	// DM пишет пользователю в личные сообщения; если они закрыты, молча ничего не делает.
	DM(userID, text string)
	// Log пишет в канал журнала модерации, если он задан в конфиге.
	Log(text string)
}

// discordGateway — реализация поверх discordgo. Сессию получает в Module.Register.
type discordGateway struct {
	s            *discordgo.Session
	log          *slog.Logger
	logChannelID string
}

func (g *discordGateway) BotID() string { return g.s.State.User.ID }

func (g *discordGateway) Timeout(guildID, userID string, until time.Time) error {
	return g.s.GuildMemberTimeout(guildID, userID, &until)
}

func (g *discordGateway) Kick(guildID, userID, reason string) error {
	return g.s.GuildMemberDeleteWithReason(guildID, userID, reason)
}

func (g *discordGateway) Ban(guildID, userID, reason string, deleteDays int) error {
	return g.s.GuildBanCreateWithReason(guildID, userID, reason, deleteDays)
}

func (g *discordGateway) DM(userID, text string) {
	ch, err := g.s.UserChannelCreate(userID)
	if err != nil {
		return
	}
	_, _ = g.s.ChannelMessageSend(ch.ID, text)
}

func (g *discordGateway) Log(text string) {
	if g.logChannelID == "" {
		return
	}
	if _, err := g.s.ChannelMessageSend(g.logChannelID, text); err != nil {
		g.log.Warn("не удалось записать в журнал", "err", fmt.Sprint(err))
	}
}
