package moderation

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ---------- /timeout ----------

type timeoutCommand struct{ cmdBase }

func (c *timeoutCommand) Definition() *discordgo.ApplicationCommand {
	minVal := 1.0
	return &discordgo.ApplicationCommand{
		Name:                     "timeout",
		Description:              "Замутить пользователя на время",
		DefaultMemberPermissions: permPtr(permModerate),
		DMPermission:             boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			userOption("Кого замутить"),
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "minutes",
				Description: "На сколько минут (максимум 28 дней)",
				Required:    true,
				MinValue:    &minVal,
				MaxValue:    float64(MaxTimeout / time.Minute),
			},
			reasonOption(false),
		},
	}
}

func (c *timeoutCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	mc, err := prepare(s, i, permModerate, true)
	if mc == nil {
		return err
	}
	d := time.Duration(mc.opts["minutes"].IntValue()) * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	cs, err := c.svc.Timeout(ctx, mc.req, d)
	if err != nil {
		return c.fail(s, i, err)
	}
	return reply(s, i, fmt.Sprintf("🔇 <@%s> замучен до <t:%d:f> (дело #%d). Причина: %s",
		mc.target.ID, cs.ExpiresAt.Unix(), cs.ID, cs.Reason), false)
}

// ---------- /kick ----------

type kickCommand struct{ cmdBase }

func (c *kickCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:                     "kick",
		Description:              "Выгнать пользователя с сервера",
		DefaultMemberPermissions: permPtr(permKick),
		DMPermission:             boolPtr(false),
		Options:                  []*discordgo.ApplicationCommandOption{userOption("Кого выгнать"), reasonOption(false)},
	}
}

func (c *kickCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	mc, err := prepare(s, i, permKick, true)
	if mc == nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	cs, err := c.svc.Kick(ctx, mc.req)
	if err != nil {
		return c.fail(s, i, err)
	}
	return reply(s, i, fmt.Sprintf("👢 <@%s> выгнан (дело #%d). Причина: %s", mc.target.ID, cs.ID, cs.Reason), false)
}

// ---------- /ban ----------

type banCommand struct{ cmdBase }

func (c *banCommand) Definition() *discordgo.ApplicationCommand {
	minVal := 0.0
	return &discordgo.ApplicationCommand{
		Name:                     "ban",
		Description:              "Забанить пользователя",
		DefaultMemberPermissions: permPtr(permBan),
		DMPermission:             boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			userOption("Кого забанить"),
			reasonOption(false),
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "delete_days",
				Description: "Удалить сообщения за последние N дней (0–7)",
				MinValue:    &minVal,
				MaxValue:    7,
			},
		},
	}
}

func (c *banCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Банить можно и тех, кого уже нет на сервере.
	mc, err := prepare(s, i, permBan, false)
	if mc == nil {
		return err
	}
	days := 0
	if d, ok := mc.opts["delete_days"]; ok {
		days = int(d.IntValue())
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	cs, err := c.svc.Ban(ctx, mc.req, days)
	if err != nil {
		return c.fail(s, i, err)
	}
	return reply(s, i, fmt.Sprintf("🔨 <@%s> забанен (дело #%d). Причина: %s", mc.target.ID, cs.ID, cs.Reason), false)
}
