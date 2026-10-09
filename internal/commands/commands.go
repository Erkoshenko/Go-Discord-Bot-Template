package commands

import (
	"bot/internal/repository"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var CommandsList = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "Проверить работу бота",
	},
}

type CommandHandler struct {
	repos *repository.Repositories
}

func NewCommandHandler(repos *repository.Repositories) *CommandHandler {
	return &CommandHandler{repos: repos}
}

func (ch *CommandHandler) RegisterCommands(session *discordgo.Session) {
	session.AddHandler(ch.onCreateInteraction)
}

func (ch *CommandHandler) RegisterDiscordCommands(s *discordgo.Session) error {
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "1456720574012461149", []*discordgo.ApplicationCommand{})
	if err != nil {
		return fmt.Errorf("ошибка при очищении команд в Discord: %w", err)
	}

	_, err = s.ApplicationCommandBulkOverwrite(s.State.User.ID, "1456720574012461149", CommandsList)
	if err != nil {
		return fmt.Errorf("ошибка регистрации команд в Discord: %w", err)
	}
	return nil
}

func (ch *CommandHandler) onCreateInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "ping":
		ch.handlePing(s, i)
	}
}

func (ch *CommandHandler) handlePing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Понг!",
		},
	})
}
