package general

import (
	"bot/pkg/respond"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type pingCommand struct{}

func (c *pingCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Проверить работу бота",
	}
}

func (c *pingCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if err := respond.InteractionRespond(s, i, "Понг!", true); err != nil {
		return fmt.Errorf("не удалось отправить ответ на пинг: %w", err)
	}
	return nil
}
