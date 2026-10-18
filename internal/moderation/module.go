// Package moderation — модуль модерации: /warn /warns /unwarn /timeout /kick /ban.
//
// Слои внутри модуля:
//
//	commands_*.go, checks.go — слой Discord: аргументы, права, ответы пользователю;
//	service.go               — бизнес-логика: правила, порядок действий, автомут;
//	gateway.go               — действия в Discord (мут, кик, бан, сообщения);
//	repository.go            — SQL и таблица moderation_cases.
//
// Чтобы убрать модуль: удалите эту папку и строку moderation.New в internal/app/modules.go.
package moderation

import (
	"bot/internal/commands"
	"bot/internal/module"
	"context"

	"github.com/bwmarrin/discordgo"
)

type Module struct {
	module.Base
	repo Repository
	gw   *discordGateway
	svc  *Service
	cmd  cmdBase
}

func New(d module.Deps) *Module {
	log := d.Log.With("module", "moderation")
	cfg := d.Settings.Moderation

	repo := NewPostgresRepository(d.DB)
	gw := &discordGateway{log: log, logChannelID: cfg.LogChannelID}
	svc := NewService(repo, gw, cfg, log)

	return &Module{repo: repo, gw: gw, svc: svc, cmd: cmdBase{svc: svc, log: log}}
}

func (m *Module) Name() string { return "moderation" }

func (m *Module) Init(ctx context.Context) error {
	return m.repo.CreateTableIfNotExists(ctx)
}

// Register вызывается до подключения к Discord: передаём сессию шлюзу.
func (m *Module) Register(s *discordgo.Session) { m.gw.s = s }

func (m *Module) Commands() []commands.Command {
	return []commands.Command{
		&warnCommand{m.cmd},
		&warnsCommand{m.cmd},
		&unwarnCommand{m.cmd},
		&timeoutCommand{m.cmd},
		&kickCommand{m.cmd},
		&banCommand{m.cmd},
	}
}
