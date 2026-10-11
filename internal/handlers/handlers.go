package handlers

import (
	"bot/internal/commands"
	"bot/internal/config"
	"bot/internal/repository"
	"log"

	"github.com/bwmarrin/discordgo"
)

type Handler struct {
	repos      *repository.Repositories
	cmdHandler *commands.CommandHandler
	settings   config.Settings
}

func NewHandler(repos *repository.Repositories, settings config.Settings) *Handler {
	return &Handler{
		repos:      repos,
		cmdHandler: commands.NewCommandHandler(repos, settings),
	}
}

func (h *Handler) RegisterAll(session *discordgo.Session) {
	session.AddHandler(h.onReady)

	h.cmdHandler.RegisterCommands(session)
}

func (h *Handler) onReady(s *discordgo.Session, r *discordgo.Ready) {
	err := h.cmdHandler.RegisterDiscordCommands(s)
	if err != nil {
		log.Printf("ошибка регистрации команд: %v", err)
	}

	log.Printf("Бот запустился с ником %s", s.State.User.Username)
}
