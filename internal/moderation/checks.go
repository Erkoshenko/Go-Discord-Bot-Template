package moderation

import (
	"bot/pkg/respond"
	"errors"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// cmdBase — общие зависимости всех команд модуля.
type cmdBase struct {
	svc *Service
	log *slog.Logger
}

// modCtx — разобранные общие аргументы модераторской команды.
type modCtx struct {
	req    Request
	target *discordgo.User
	opts   map[string]*discordgo.ApplicationCommandInteractionDataOption
}

func optionsMap(i *discordgo.InteractionCreate) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	opts := i.ApplicationCommandData().Options
	m := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(opts))
	for _, o := range opts {
		m[o.Name] = o
	}
	return m
}

func userOption(desc string) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionUser,
		Name:        "user",
		Description: desc,
		Required:    true,
	}
}

func reasonOption(required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "reason",
		Description: "Причина",
		Required:    required,
		MaxLength:   MaxReasonRunes,
	}
}

func permPtr(p int64) *int64 { return &p }

func boolPtr(b bool) *bool { return &b }

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func reply(s *discordgo.Session, i *discordgo.InteractionCreate, text string, ephemeral bool) error {
	return respond.InteractionRespond(s, i, text, ephemeral)
}

// requireGuildPerm проверяет, что команда вызвана на сервере и у модератора есть право perm.
// Если нет, сам отвечает пользователю и возвращает false.
func requireGuildPerm(s *discordgo.Session, i *discordgo.InteractionCreate, perm int64) (bool, error) {
	if i.GuildID == "" || i.Member == nil {
		return false, reply(s, i, "❌ Эта команда работает только на сервере.", true)
	}
	if i.Member.Permissions&perm == 0 {
		return false, reply(s, i, "❌ У вас недостаточно прав для этой команды.", true)
	}
	return true, nil
}

// prepare проверяет права, разбирает user и reason и проверяет иерархию ролей.
// Если что-то не так, сам отвечает пользователю и возвращает (nil, err).
func prepare(s *discordgo.Session, i *discordgo.InteractionCreate, perm int64, mustBeMember bool) (*modCtx, error) {
	if ok, err := requireGuildPerm(s, i, perm); !ok {
		return nil, err
	}

	opts := optionsMap(i)
	userOpt, ok := opts["user"]
	if !ok {
		return nil, reply(s, i, "❌ Не указан пользователь.", true)
	}
	target := userOpt.UserValue(s)

	reason := ""
	if r, ok := opts["reason"]; ok {
		reason = r.StringValue()
	}

	if msg := checkTarget(s, i, target, mustBeMember); msg != "" {
		return nil, reply(s, i, "❌ "+msg, true)
	}

	return &modCtx{
		req: Request{
			GuildID:     i.GuildID,
			TargetID:    target.ID,
			ModeratorID: i.Member.User.ID,
			Reason:      reason,
		},
		target: target,
		opts:   opts,
	}, nil
}

// checkTarget возвращает текст ошибки или "", если действие допустимо.
func checkTarget(s *discordgo.Session, i *discordgo.InteractionCreate, target *discordgo.User, mustBeMember bool) string {
	switch {
	case target.ID == i.Member.User.ID:
		return "Нельзя применить это к самому себе."
	case target.ID == s.State.User.ID:
		return "Нельзя применить это к боту."
	}

	guild, err := s.Guild(i.GuildID)
	if err != nil {
		return "Не удалось получить данные сервера."
	}
	if target.ID == guild.OwnerID {
		return "Нельзя применить это к владельцу сервера."
	}

	member, err := s.GuildMember(i.GuildID, target.ID)
	if err != nil {
		if mustBeMember {
			return "Этого пользователя нет на сервере."
		}
		return "" // бан по ID пользователя, которого нет на сервере, допустим
	}

	// Владелец может всё; остальные — только над ролями ниже своей.
	if i.Member.User.ID == guild.OwnerID {
		return ""
	}
	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		return "Не удалось получить роли сервера."
	}
	if highestRole(roles, i.Member.Roles) <= highestRole(roles, member.Roles) {
		return "Роль этого пользователя не ниже вашей."
	}
	return ""
}

func highestRole(all []*discordgo.Role, ids []string) int {
	pos := 0 // @everyone
	for _, r := range all {
		for _, id := range ids {
			if r.ID == id && r.Position > pos {
				pos = r.Position
			}
		}
	}
	return pos
}

// fail превращает ошибку сервиса в понятный ответ модератору.
// Ошибку пишет в лог и сам отвечает, поэтому возвращает только ошибку отправки ответа.
func (b cmdBase) fail(s *discordgo.Session, i *discordgo.InteractionCreate, err error) error {
	b.log.Error("ошибка модерации", "command", i.ApplicationCommandData().Name, "err", err)

	var msg string
	switch {
	case errors.Is(err, ErrInvalidDuration):
		msg = "❌ " + ErrInvalidDuration.Error() + "."
	case errors.Is(err, ErrActionFailed):
		msg = "❌ Discord отказал в действии. Проверьте права бота и что его роль выше роли пользователя."
	case errors.Is(err, ErrNotRecorded):
		msg = "⚠️ Действие выполнено, но записать его в историю не удалось."
	default:
		msg = "❌ Что-то пошло не так. Попробуйте позже."
	}
	return reply(s, i, msg, true)
}
