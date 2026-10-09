package handlers

import (
	"bot/internal/commands"
	"bot/internal/repository"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Handler struct {
	repos      *repository.Repositories
	cmdHandler *commands.CommandHandler
}

func NewHandler(repos *repository.Repositories) *Handler {
	return &Handler{
		repos:      repos,
		cmdHandler: commands.NewCommandHandler(repos),
	}
}

func (h *Handler) RegisterAll(session *discordgo.Session) {
	session.AddHandler(h.onReady)

	h.cmdHandler.RegisterCommands(session)
}

func (h *Handler) onReady(s *discordgo.Session, r *discordgo.Ready) {
	h.cmdHandler.RegisterDiscordCommands(s)

	fmt.Printf("Бот запустился с ником %s", s.State.User.Username)
}
