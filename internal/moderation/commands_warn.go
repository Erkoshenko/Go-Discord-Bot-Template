package moderation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	permModerate = int64(discordgo.PermissionModerateMembers)
	permKick     = int64(discordgo.PermissionKickMembers)
	permBan      = int64(discordgo.PermissionBanMembers)

	dbTimeout = 5 * time.Second
)

// ---------- /warn ----------

type warnCommand struct{ cmdBase }

func (c *warnCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "warn",
		Description:              "Выдать предупреждение пользователю",
		DefaultMemberPermissions: permPtr(permModerate),
		DMPermission:             boolPtr(false),
		Options:                  []*discordgo.ApplicationCommandOption{userOption("Кому выдать предупреждение"), reasonOption(true)},
	}
}

func (c *warnCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	mc, err := prepare(s, i, permModerate, true)
	if mc == nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	res, err := c.svc.Warn(ctx, mc.req)
	if err != nil {
		return c.fail(s, i, err)
	}

	msg := fmt.Sprintf("⚠️ <@%s> получил предупреждение (дело #%d). Причина: %s",
		mc.target.ID, res.Case.ID, res.Case.Reason)
	if res.ActiveWarns > 0 {
		msg += fmt.Sprintf("\nДействующих предупреждений: %d", res.ActiveWarns)
	}
	if res.AutoTimeout != nil && res.AutoTimeout.ExpiresAt != nil {
		msg += fmt.Sprintf("\n🔇 Автоматически выдан мут до <t:%d:f>.", res.AutoTimeout.ExpiresAt.Unix())
	}
	return reply(s, i, msg, false)
}

// ---------- /warns ----------

type warnsCommand struct{ cmdBase }

func (c *warnsCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "warns",
		Description:              "История модерации пользователя",
		DefaultMemberPermissions: permPtr(permModerate),
		DMPermission:             boolPtr(false),
		Options:                  []*discordgo.ApplicationCommandOption{userOption("Чью историю показать")},
	}
}

func (c *warnsCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if ok, err := requireGuildPerm(s, i, permModerate); !ok {
		return err
	}
	opt, ok := optionsMap(i)["user"]
	if !ok {
		return reply(s, i, "❌ Не указан пользователь.", true)
	}
	user := opt.UserValue(s)

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	cases, err := c.svc.History(ctx, i.GuildID, user.ID, 10)
	if err != nil {
		return c.fail(s, i, err)
	}
	if len(cases) == 0 {
		return reply(s, i, fmt.Sprintf("У <@%s> чистая история.", user.ID), true)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "**История <@%s>** (последние %d):\n", user.ID, len(cases))
	for _, cs := range cases {
		line := fmt.Sprintf("#%d · %s · <t:%d:d> · <@%s> · %s",
			cs.ID, actionLabel(cs.Action), cs.CreatedAt.Unix(), cs.ModeratorID, truncate(cs.Reason, 100))
		if cs.Action == ActionWarn && !cs.Active {
			line = "~~" + line + "~~ (снято)"
		}
		b.WriteString(line + "\n")
	}
	return reply(s, i, b.String(), true)
}

func actionLabel(a Action) string {
	switch a {
	case ActionWarn:
		return "предупреждение"
	case ActionTimeout:
		return "мут"
	case ActionKick:
		return "кик"
	case ActionBan:
		return "бан"
	}
	return string(a)
}

// ---------- /unwarn ----------

type unwarnCommand struct{ cmdBase }

func (c *unwarnCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "unwarn",
		Description:              "Снять предупреждение по номеру дела",
		DefaultMemberPermissions: permPtr(permModerate),
		DMPermission:             boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "case_id",
			Description: "Номер дела (виден в /warns)",
			Required:    true,
		}},
	}
}

func (c *unwarnCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if ok, err := requireGuildPerm(s, i, permModerate); !ok {
		return err
	}
	caseID := optionsMap(i)["case_id"].IntValue()

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	ok, err := c.svc.Unwarn(ctx, i.GuildID, caseID, i.Member.User.ID)
	if err != nil {
		return c.fail(s, i, err)
	}
	if !ok {
		return reply(s, i, "❌ Предупреждение не найдено или уже снято.", true)
	}
	return reply(s, i, fmt.Sprintf("✅ Предупреждение #%d снято.", caseID), false)
}
