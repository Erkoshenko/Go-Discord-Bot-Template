package commands

import (
	"bot/internal/repository"
	"bot/pkg/respond"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type PingCommand struct{}

func (c *PingCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Проверить работу бота",
	}
}

func (c *PingCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate, repos *repository.Repositories) error {
	if err := respond.InteractionRespond(s, i, "Понг!", true); err != nil {
		return fmt.Errorf("не удалось отправить ответ на пинг: %w", err)
	}
	return nil
}
