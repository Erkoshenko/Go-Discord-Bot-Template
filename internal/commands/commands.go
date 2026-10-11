package commands

import (
	"bot/internal/config"
	"bot/internal/repository"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

type Command interface {
	Definition() *discordgo.ApplicationCommand
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate, repos *repository.Repositories) error
}

type CommandHandler struct {
	repos    *repository.Repositories
	commands map[string]Command
	settings config.Settings
}

func NewCommandHandler(repos *repository.Repositories, settings config.Settings) *CommandHandler {
	ch := &CommandHandler{
		repos:    repos,
		commands: make(map[string]Command),
		settings: settings,
	}
	allCommands := []Command{
		&PingCommand{},
	}

	for _, cmd := range allCommands {
		ch.RegisterCommand(cmd)
	}

	return ch
}

func (ch *CommandHandler) RegisterCommand(cmd Command) {
	ch.commands[cmd.Definition().Name] = cmd
}

func (ch *CommandHandler) RegisterCommands(session *discordgo.Session) {
	session.AddHandler(ch.onCreateInteraction)
}

func (ch *CommandHandler) RegisterDiscordCommands(s *discordgo.Session) error {
	var appCommands []*discordgo.ApplicationCommand
	for _, cmd := range ch.commands {
		appCommands = append(appCommands, cmd.Definition())
	}

	// _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, ch.settings.TestGuildID, []*discordgo.ApplicationCommand{})
	// if err != nil {
	// 	return fmt.Errorf("не удалось зарегистрировать команды: %w", err)
	// }

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, ch.settings.TestGuildID, appCommands)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать команды: %w", err)
	}
	return nil
}

func (ch *CommandHandler) onCreateInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	cmdName := i.ApplicationCommandData().Name
	cmd, ok := ch.commands[cmdName]
	if !ok {
		return
	}
	if err := cmd.Execute(s, i, ch.repos); err != nil {
		log.Printf("ошибка в команде %s: %v", cmdName, err)
	}

}
